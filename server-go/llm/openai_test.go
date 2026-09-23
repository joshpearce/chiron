package llm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIChainUsesBearerKeyAndJSONSchema(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "missing key", http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/models" {
			w.Write([]byte(`{"data":[{"id":"model"}]}`))
			return
		}
		calls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		format := body["response_format"].(map[string]any)
		if format["type"] != "json_schema" {
			t.Fatalf("response format = %#v", format)
		}
		jsonSchema := format["json_schema"].(map[string]any)
		if strict, ok := jsonSchema["strict"].(bool); !ok || !strict {
			t.Fatalf("strict must be JSON true, got %#v", jsonSchema["strict"])
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{
			map[string]any{"message": map[string]any{"content": `{"verdict":"correct"}`}},
		}})
	}))
	defer srv.Close()

	chain := NewOpenAIChain([]Upstream{{Name: "hosted", BaseURL: srv.URL, Model: "model", Enabled: true}},
		Config{APIKey: "secret", TimeoutS: 10, MaxTokens: map[string]int{"grader": 50}})
	var out struct {
		Verdict string `json:"verdict"`
	}
	err := chain.Structured("grader", "system", "user", map[string]any{
		"type": "object", "properties": map[string]any{"verdict": map[string]any{"type": "string"}},
		"required": []string{"verdict"}, "additionalProperties": false,
	}, "grade", &out)
	if err != nil || out.Verdict != "correct" || calls != 1 {
		t.Fatalf("out=%+v calls=%d err=%v", out, calls, err)
	}
}

func TestVisionUsesBearerKeyAndImageInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "missing key", http.StatusUnauthorized)
			return
		}
		var body struct {
			Messages []struct {
				Content []map[string]any `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		url := body.Messages[0].Content[1]["image_url"].(map[string]any)["url"].(string)
		if !strings.HasPrefix(url, "data:image/png;base64,") {
			t.Fatalf("image URL = %.40q", url)
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{
			map[string]any{"message": map[string]any{"content": "42"}},
		}})
	}))
	defer srv.Close()

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.Black)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if _, err := base64.StdEncoding.DecodeString(base64.StdEncoding.EncodeToString(buf.Bytes())); err != nil {
		t.Fatal(err)
	}
	got, err := TranscribeWithKey(srv.URL, "vision", "item", "secret", buf.Bytes())
	if err != nil || got != "42" {
		t.Fatalf("transcription=%q err=%v", got, err)
	}
}
