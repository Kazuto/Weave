package llm

import (
	"fmt"

	"github.com/Kazuto/Weave/pkg/config"
)

// NewProvider creates a new LLM provider based on the configuration
func NewProvider(cfg config.LLMConfig) (Provider, error) {
	provider := cfg.Provider
	if provider == "" {
		provider = "ollama" // default to ollama for backward compatibility
	}

	switch ProviderType(provider) {
	case ProviderOllama:
		return NewOllamaClient(cfg.Ollama), nil
	case ProviderOpenAI:
		return NewOpenAIClient(cfg.OpenAI), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s (supported: ollama, openai)", provider)
	}
}
