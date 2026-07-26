package llm

import "os"

// FactoryConfig is the slice of config.yaml the factory needs.
type FactoryConfig struct {
	Provider       string     `yaml:"provider"`
	AnthropicModel string     `yaml:"anthropic_model"`
	ClaudeCLIModel string     `yaml:"claude_cli_model"`
	Upstreams      []Upstream `yaml:"upstreams"`
	LLM            Config     `yaml:"llm"`
}

// New picks a backend: "anthropic" for the direct API, "claude-cli" for
// headless Claude Code on a subscription, anything else for the
// OpenAI-compatible chain (LM Studio, the offline flight path).
//
// CHIRON_PROVIDER overrides the configured provider so a test or an authoring
// run can choose an engine without editing the config the server boots from.
func New(cfg FactoryConfig) Chain {
	provider := cfg.Provider
	if env := os.Getenv("CHIRON_PROVIDER"); env != "" {
		provider = env
	}
	switch provider {
	case "anthropic":
		return NewAnthropic(cfg.AnthropicModel)
	case "claude-cli":
		// An empty model means the per-role tiers apply.
		return &ClaudeCLI{Model: cfg.ClaudeCLIModel}
	default:
		return NewOpenAIChain(cfg.Upstreams, cfg.LLM)
	}
}
