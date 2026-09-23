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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
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

func TestOpenAIClientLivenessBudgets(t *testing.T) {
	chain := NewOpenAIChain(nil, Config{
		TimeoutS: 1800, ResponseHeaderTimeoutS: 120, IdleConnTimeoutS: 180,
	})
	if got, want := chain.client.Timeout, 30*time.Minute; got != want {
		t.Fatalf("whole request timeout = %v, want %v", got, want)
	}
	transport, ok := chain.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", chain.client.Transport)
	}
	if got, want := transport.ResponseHeaderTimeout, 2*time.Minute; got != want {
		t.Errorf("response header timeout = %v, want %v", got, want)
	}
	if got, want := transport.IdleConnTimeout, 3*time.Minute; got != want {
		t.Errorf("idle connection timeout = %v, want %v", got, want)
	}
}

func TestOpenAIClientDefaultsRetainSafeLongRequest(t *testing.T) {
	client := openAIHTTPClient(Config{})
	if client.Timeout < 10*time.Minute {
		t.Fatalf("default request timeout = %v, want at least 10m", client.Timeout)
	}
}

func TestNASConfigPinsMiDineroProductionContract(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "deploy", "nas", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		VisionModel string     `yaml:"vision_model"`
		Upstreams   []Upstream `yaml:"upstreams"`
		LLM         Config     `yaml:"llm"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	const model = "deepseek-ai/DeepSeek-V4-Flash-0731"
	const baseURL = "https://api.deepinfra.com/v1/openai"
	if cfg.VisionModel != model {
		t.Errorf("vision model = %q, want %q", cfg.VisionModel, model)
	}
	if len(cfg.Upstreams) != 1 || !cfg.Upstreams[0].Enabled ||
		cfg.Upstreams[0].BaseURL != baseURL || cfg.Upstreams[0].Model != model {
		t.Errorf("production upstream = %+v, want enabled %s at %s", cfg.Upstreams, model, baseURL)
	}
	if cfg.LLM.ResponseHeaderTimeoutS != 120 || cfg.LLM.IdleConnTimeoutS != 180 || cfg.LLM.TimeoutS != 1800 {
		t.Errorf("liveness budgets = header:%ds idle:%ds whole:%ds, want 120/180/1800",
			cfg.LLM.ResponseHeaderTimeoutS, cfg.LLM.IdleConnTimeoutS, cfg.LLM.TimeoutS)
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
