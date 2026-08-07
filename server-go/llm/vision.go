package llm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Transcribe sends a handwriting image to a vision-capable model on an
// OpenAI-compatible server and returns the plain-text transcription. It is a
// separate, direct call rather than part of the role chain: transcription is
// mechanical, needs no provider fallback, and the vision model is loaded on
// demand next to the text model.
// The tag parameter is for logging only. The question is deliberately NOT
// given to the model: with the question in the prompt, a small vision model
// answers it instead of transcribing the scribble - "I don't know" tick
// marks came back as correct answers that way.
func Transcribe(baseURL, model, tag string, pngData []byte) (string, error) {
	_ = tag
	prompt := "Transcribe the handwriting in this image exactly, " +
		"preserving numbers, signs, and symbols. Output ONLY the transcription " +
		"with no commentary. If the image is blank, or contains only a tick, " +
		"check mark, X, or a mark in a small box rather than written words or " +
		"numbers, output exactly: [no answer]"
	body, err := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]any{{
			"role": "user",
			"content": []map[string]any{
				{"type": "text", "text": prompt},
				{"type": "image_url", "image_url": map[string]string{
					"url": "data:image/png;base64," +
						base64.StdEncoding.EncodeToString(pngData),
				}},
			},
		}},
		"temperature": 0.0,
		"max_tokens":  500,
	})
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(baseURL+"/chat/completions", "application/json",
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vision model: http %d: %.200s", resp.StatusCode, raw)
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Choices) == 0 {
		return "", fmt.Errorf("vision model: unreadable response")
	}
	return envelope.Choices[0].Message.Content, nil
}
