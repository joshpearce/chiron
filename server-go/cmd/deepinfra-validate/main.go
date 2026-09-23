// Command deepinfra-validate checks Chiron's production DeepInfra contract.
// It never prints credentials and only reads them from the standard Chiron
// environment variable or secret-file setting.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"

	"github.com/mjbraun/chiron/server/llm"
)

const (
	baseURL = "https://api.deepinfra.com/v1/openai"
	model   = "deepseek-ai/DeepSeek-V4-Flash-0731"
)

func main() {
	key, err := credential()
	if err != nil {
		fail(err)
	}
	cfg := llm.Config{
		APIKey: key, Temperature: 0,
		TimeoutS: 1800, ResponseHeaderTimeoutS: 120, IdleConnTimeoutS: 180,
		MaxTokens: map[string]int{"grader": 256, "author": 512},
	}
	chain := llm.NewOpenAIChain([]llm.Upstream{{
		Name: "deepinfra", BaseURL: baseURL, Model: model, Enabled: true,
	}}, cfg)

	var grade struct {
		Verdict string `json:"verdict"`
		Reason  string `json:"reason"`
	}
	gradeSchema := objectSchema(map[string]any{
		"verdict": map[string]any{"type": "string", "enum": []string{"correct", "incorrect"}},
		"reason":  map[string]any{"type": "string"},
	}, "verdict", "reason")
	if err := chain.Structured("grader", "Grade mathematical answers exactly.",
		"Question: What is 6 * 7? Learner answer: 42", gradeSchema, "chiron_grade", &grade); err != nil {
		fail(fmt.Errorf("structured grading: %w", err))
	}
	if grade.Verdict != "correct" {
		fail(fmt.Errorf("structured grading returned verdict %q", grade.Verdict))
	}
	fmt.Println("structured grading: ok")

	var generated struct {
		Title string   `json:"title"`
		Beats []string `json:"beats"`
	}
	generationSchema := objectSchema(map[string]any{
		"title": map[string]any{"type": "string"},
		"beats": map[string]any{
			"type": "array", "items": map[string]any{"type": "string"}, "minItems": 2,
		},
	}, "title", "beats")
	if err := chain.Structured("author", "Write a compact adaptive-textbook outline.",
		"Generate a two-beat lesson that corrects the misconception that correlation proves causation.",
		generationSchema, "chiron_generation", &generated); err != nil {
		fail(fmt.Errorf("structured generation: %w", err))
	}
	if generated.Title == "" || len(generated.Beats) < 2 {
		fail(fmt.Errorf("structured generation returned incomplete content"))
	}
	fmt.Println("structured generation: ok")

	imageData, err := samplePNG()
	if err != nil {
		fail(err)
	}
	text, err := llm.TranscribeWithKey(baseURL, model, "deepinfra-validation", key, imageData)
	if err != nil {
		fail(fmt.Errorf("image input incompatible: %w", err))
	}
	if strings.TrimSpace(text) == "" {
		fail(fmt.Errorf("image input returned empty content"))
	}
	fmt.Println("image input: ok")
	fmt.Printf("model: %s\n", model)
}

func credential() (string, error) {
	if key := strings.TrimSpace(os.Getenv("CHIRON_LLM_API_KEY")); key != "" {
		return key, nil
	}
	path := strings.TrimSpace(os.Getenv("CHIRON_LLM_API_KEY_FILE"))
	if path == "" {
		return "", fmt.Errorf("set CHIRON_LLM_API_KEY_FILE or CHIRON_LLM_API_KEY")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read credential file: %w", err)
	}
	if key := strings.TrimSpace(string(raw)); key != "" {
		return key, nil
	}
	return "", fmt.Errorf("credential file is empty")
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	return map[string]any{
		"type": "object", "properties": properties, "required": required,
		"additionalProperties": false,
	}
}

func samplePNG() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.White)
		}
	}
	for x := 8; x < 56; x++ {
		img.Set(x, 16, color.Black)
		img.Set(x, 17, color.Black)
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, fmt.Errorf("encode validation image: %w", err)
	}
	return out.Bytes(), nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
