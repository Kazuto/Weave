package commit

import (
	"testing"

	"github.com/Kazuto/Weave/internal/config"
)

func TestNewOllamaClient(t *testing.T) {
	cfg := config.OllamaConfig{
		Model:       "llama3.2",
		Host:        "http://localhost:11434",
		Temperature: 0.3,
		TopP:        0.9,
		MaxDiff:     4000,
	}

	client := NewOllamaClient(cfg)

	if client == nil {
		t.Fatal("NewOllamaClient() returned nil")
	}

	if client.config.Model != "llama3.2" {
		t.Errorf("Expected model 'llama3.2', got '%s'", client.config.Model)
	}

	if client.config.Host != "http://localhost:11434" {
		t.Errorf("Expected host 'http://localhost:11434', got '%s'", client.config.Host)
	}

	if client.client == nil {
		t.Error("HTTP client should not be nil")
	}
}

func TestOllamaClient_CheckConnection_NoServer(t *testing.T) {
	cfg := config.OllamaConfig{
		Host: "http://localhost:99999", // Invalid port
	}

	client := NewOllamaClient(cfg)

	if client.CheckConnection() {
		t.Error("Expected CheckConnection() to return false for invalid host")
	}
}

func TestOllamaClient_IsModelAvailable_NoServer(t *testing.T) {
	cfg := config.OllamaConfig{
		Model: "llama3.2",
		Host:  "http://localhost:99999", // Invalid port
	}

	client := NewOllamaClient(cfg)

	if client.IsModelAvailable() {
		t.Error("Expected IsModelAvailable() to return false for invalid host")
	}
}
