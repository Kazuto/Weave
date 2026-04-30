package llm

import (
	"testing"

	"github.com/Kazuto/Weave/pkg/config"
)

func TestNewOpenAIClient(t *testing.T) {
	cfg := config.OpenAIConfig{
		Model:       "gpt-4",
		Host:        "http://localhost:1234",
		APIKey:      "",
		Temperature: 0.3,
		TopP:        0.9,
		MaxDiff:     4000,
	}

	client := NewOpenAIClient(cfg)

	if client == nil {
		t.Fatal("NewOpenAIClient() returned nil")
	}

	if client.config.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got %s", client.config.Model)
	}

	if client.config.Host != "http://localhost:1234" {
		t.Errorf("Expected host 'http://localhost:1234', got %s", client.config.Host)
	}
}
