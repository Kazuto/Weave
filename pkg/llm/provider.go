package llm

// Provider defines the interface that all LLM providers must implement
type Provider interface {
	// CheckConnection verifies that the provider is accessible
	CheckConnection() bool

	// IsModelAvailable checks if the configured model is available
	IsModelAvailable() bool

	// Generate creates text based on the given prompt
	Generate(prompt string) (string, error)
}

// ProviderType represents the type of LLM provider
type ProviderType string

const (
	ProviderOllama ProviderType = "ollama"
	ProviderOpenAI ProviderType = "openai"
)
