package llm

import (
	"testing"

	"github.com/Kazuto/Weave/pkg/config"
)

func TestNewProvider_Ollama(t *testing.T) {
	cfg := config.LLMConfig{
		Provider: "ollama",
		Ollama: config.OllamaConfig{
			Model:       "llama3.2",
			Host:        "http://localhost:11434",
			Temperature: 0.3,
			TopP:        0.9,
			MaxDiff:     4000,
		},
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil")
	}

	if _, ok := provider.(*OllamaClient); !ok {
		t.Error("Expected OllamaClient provider")
	}
}

func TestNewProvider_OpenAI(t *testing.T) {
	cfg := config.LLMConfig{
		Provider: "openai",
		OpenAI: config.OpenAIConfig{
			Model:       "gpt-4",
			Host:        "http://localhost:1234",
			Temperature: 0.7,
			TopP:        0.9,
			MaxDiff:     4000,
		},
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil")
	}

	if _, ok := provider.(*OpenAIClient); !ok {
		t.Error("Expected OpenAIClient provider")
	}
}

func TestNewProvider_DefaultsToOllama(t *testing.T) {
	cfg := config.LLMConfig{
		Provider: "", // empty should default to ollama
		Ollama: config.OllamaConfig{
			Model: "llama3.2",
			Host:  "http://localhost:11434",
		},
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if _, ok := provider.(*OllamaClient); !ok {
		t.Error("Empty provider should default to OllamaClient")
	}
}

func TestNewProvider_InvalidProvider(t *testing.T) {
	cfg := config.LLMConfig{
		Provider: "invalid",
	}

	_, err := NewProvider(cfg)
	if err == nil {
		t.Error("Expected error for invalid provider")
	}
}

func TestGetMaxDiff(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.LLMConfig
		expected int
	}{
		{
			name: "Ollama provider",
			cfg: config.LLMConfig{
				Provider: "ollama",
				Ollama: config.OllamaConfig{
					MaxDiff: 5000,
				},
			},
			expected: 5000,
		},
		{
			name: "OpenAI provider",
			cfg: config.LLMConfig{
				Provider: "openai",
				OpenAI: config.OpenAIConfig{
					MaxDiff: 3000,
				},
			},
			expected: 3000,
		},
		{
			name: "Default to Ollama",
			cfg: config.LLMConfig{
				Provider: "",
				Ollama: config.OllamaConfig{
					MaxDiff: 4000,
				},
			},
			expected: 4000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMaxDiff(tt.cfg)
			if result != tt.expected {
				t.Errorf("GetMaxDiff() = %d, want %d", result, tt.expected)
			}
		})
	}
}
