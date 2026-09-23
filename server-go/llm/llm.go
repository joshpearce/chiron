// Package llm holds the interchangeable model backends.
//
// Three engines implement one interface: an OpenAI-compatible endpoint chain
// (LM Studio on the Mac, the offline flight path), headless Claude Code via
// `claude -p` (the sprite, billed to a subscription rather than a key), and the
// Anthropic API. Every call is structured output bounded by a hard token cap
// and a client timeout - the local MoE model has a documented infinite-loop
// pathology, and a mid-flight hang is unrecoverable.
package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Error is what every backend returns when it cannot produce a usable answer.
// Callers decide the fallback: grading degrades to "ungraded" rather than
// guessing, and chapter delivery falls back to canon.
type Error struct{ msg string }

func (e *Error) Error() string { return e.msg }

func Errorf(format string, a ...any) error { return &Error{msg: fmt.Sprintf(format, a...)} }

// IsUpstream reports whether an error came from a model backend, as opposed to
// a programming error the caller should not paper over.
func IsUpstream(err error) bool {
	var e *Error
	return errors.As(err, &e)
}

// Status is what /health reports.
type Status struct {
	Connected bool   `json:"connected"`
	Upstream  string `json:"upstream,omitempty"`
	Model     string `json:"model,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Chain is the interface roles.go talks to; it never knows which engine is behind it.
type Chain interface {
	// Structured runs one call and unmarshals the model's JSON into out.
	Structured(role, system, user string, schema map[string]any, schemaName string, out any) error
	Status() Status
}

const healthTimeout = 3 * time.Second

const (
	defaultResponseHeaderTimeout = 120 * time.Second
	defaultIdleConnTimeout       = 180 * time.Second
	defaultRequestTimeout        = 10 * time.Minute
)

type Upstream struct {
	Name    string `yaml:"name"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
	Enabled bool   `yaml:"enabled"`
}

type Config struct {
	Temperature            float64        `yaml:"temperature"`
	TimeoutS               int            `yaml:"timeout_s"`
	ResponseHeaderTimeoutS int            `yaml:"response_header_timeout_s"`
	IdleConnTimeoutS       int            `yaml:"idle_conn_timeout_s"`
	MaxTokens              map[string]int `yaml:"max_tokens"`
	// APIKeyFile is preferred for hosted OpenAI-compatible services so a
	// credential can be mounted as a secret instead of copied into YAML or an
	// environment variable. APIKey is populated at boot and never serialized.
	APIKeyFile string `yaml:"api_key_file"`
	APIKey     string `yaml:"-"`
}

// OpenAIChain is an ordered list of OpenAI-compatible endpoints; the first
// healthy one wins.
type OpenAIChain struct {
	upstreams []Upstream
	cfg       Config
	client    *http.Client

	mu       sync.Mutex
	cachedAt time.Time
	cached   *Upstream
}

func NewOpenAIChain(upstreams []Upstream, cfg Config) *OpenAIChain {
	var enabled []Upstream
	for _, u := range upstreams {
		if u.Enabled {
			enabled = append(enabled, u)
		}
	}
	return &OpenAIChain{
		upstreams: enabled,
		cfg:       cfg,
		client:    openAIHTTPClient(cfg),
	}
}

func openAIHTTPClient(cfg Config) *http.Client {
	timeout := durationSeconds(cfg.TimeoutS, defaultRequestTimeout)
	responseHeaderTimeout := durationSeconds(cfg.ResponseHeaderTimeoutS, defaultResponseHeaderTimeout)
	idleConnTimeout := durationSeconds(cfg.IdleConnTimeoutS, defaultIdleConnTimeout)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = responseHeaderTimeout
	transport.IdleConnTimeout = idleConnTimeout
	return &http.Client{Transport: transport, Timeout: timeout}
}

func durationSeconds(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func (c *OpenAIChain) healthy() *Upstream {
	c.mu.Lock()
	if c.cached != nil && time.Since(c.cachedAt) < 30*time.Second {
		up := c.cached
		c.mu.Unlock()
		return up
	}
	c.mu.Unlock()

	probe := &http.Client{Timeout: healthTimeout}
	for i := range c.upstreams {
		up := c.upstreams[i]
		req, err := http.NewRequest(http.MethodGet, up.BaseURL+"/models", nil)
		if err != nil {
			continue
		}
		c.authorize(req)
		resp, err := probe.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			c.mu.Lock()
			c.cachedAt, c.cached = time.Now(), &up
			c.mu.Unlock()
			return &up
		}
	}
	c.mu.Lock()
	c.cachedAt, c.cached = time.Now(), nil
	c.mu.Unlock()
	return nil
}

func (c *OpenAIChain) forgetHealth() {
	c.mu.Lock()
	c.cachedAt, c.cached = time.Time{}, nil
	c.mu.Unlock()
}

func (c *OpenAIChain) Status() Status {
	up := c.healthy()
	if up == nil {
		return Status{Connected: false}
	}
	return Status{Connected: true, Upstream: up.Name, Model: up.Model}
}

func (c *OpenAIChain) Structured(role, system, user string, schema map[string]any,
	schemaName string, out any) error {
	up := c.healthy()
	if up == nil {
		return Errorf("no upstream reachable")
	}
	maxTokens := c.cfg.MaxTokens[role]
	if maxTokens == 0 {
		maxTokens = 2000
	}
	temp := c.cfg.Temperature
	body, err := json.Marshal(map[string]any{
		"model": up.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": temp,
		"max_tokens":  maxTokens,
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name": schemaName, "strict": true, "schema": schema,
			},
		},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, up.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.authorize(req)
	resp, err := c.client.Do(req)
	if err != nil {
		c.forgetHealth() // force a re-probe on the next call
		return Errorf("%s: %v", up.Name, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		c.forgetHealth()
		return Errorf("%s: %v", up.Name, err)
	}
	if resp.StatusCode != http.StatusOK {
		c.forgetHealth()
		return Errorf("%s: http %d: %.200s", up.Name, resp.StatusCode, raw)
	}

	var envelope struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Choices) == 0 {
		return Errorf("%s: unreadable response envelope", up.Name)
	}
	msg := envelope.Choices[0].Message
	// Thinking models (Qwen3.x) can land the JSON in reasoning_content with
	// content empty; take whichever field yields valid JSON.
	for _, text := range []string{msg.Content, msg.ReasoningContent} {
		if strings.TrimSpace(text) == "" {
			continue
		}
		if err := unmarshalLoose(text, out); err == nil {
			return nil
		}
	}
	return Errorf("%s: unparseable structured output", up.Name)
}

func (c *OpenAIChain) authorize(req *http.Request) {
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
}

// unmarshalLoose accepts either a bare JSON object or one wrapped in prose or
// code fences, which is what prompt-enforced JSON actually looks like in
// practice.
func unmarshalLoose(text string, out any) error {
	text = strings.TrimSpace(text)
	if err := json.Unmarshal([]byte(text), out); err == nil {
		return nil
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return Errorf("no JSON object in output")
	}
	return json.Unmarshal([]byte(text[start:end+1]), out)
}
