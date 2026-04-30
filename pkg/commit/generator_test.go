package commit

import (
	"strings"
	"testing"

	"github.com/Kazuto/Weave/pkg/config"
)

func TestGenerator_buildPrompt(t *testing.T) {
	cfg := config.CommitConfig{
		Types:  []string{"feat", "fix", "docs"},
		Prompt: "Types: {{.Types}}\nFiles: {{.Files}}\nDiff: {{.Diff}}",
	}
	llmCfg := config.LLMConfig{
		Provider: "ollama",
		Ollama: config.OllamaConfig{
			Model:       "llama3.2",
			Host:        "http://localhost:11434",
			Temperature: 0.3,
			TopP:        0.9,
			MaxDiff:     4000,
		},
	}

	g, err := NewGenerator(cfg, llmCfg)
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	diff := "diff --git a/file.go"
	files := []string{"file.go", "other.go"}

	prompt := g.buildPrompt(diff, files)

	if !strings.Contains(prompt, "feat, fix, docs") {
		t.Error("Prompt should contain commit types")
	}

	if !strings.Contains(prompt, "file.go") {
		t.Error("Prompt should contain changed files")
	}

	if !strings.Contains(prompt, diff) {
		t.Error("Prompt should contain diff")
	}
}

func TestGenerator_cleanResponse(t *testing.T) {
	cfg := config.CommitConfig{
		Types: []string{},
	}
	llmCfg := config.LLMConfig{
		Provider: "ollama",
		Ollama:   config.OllamaConfig{},
	}

	g, err := NewGenerator(cfg, llmCfg)
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "  feat(api): add endpoint  ",
			expected: "feat(api): add endpoint",
		},
		{
			input:    `"feat(api): add endpoint"`,
			expected: "feat(api): add endpoint",
		},
		{
			input:    `'feat(api): add endpoint'`,
			expected: "feat(api): add endpoint",
		},
		{
			input:    "\n\nfeat(api): add endpoint\n\n",
			expected: "feat(api): add endpoint",
		},
	}

	for _, tt := range tests {
		result := g.cleanResponse(tt.input)
		if result != tt.expected {
			t.Errorf("cleanResponse(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNewGenerator(t *testing.T) {
	cfg := config.CommitConfig{
		Types: []string{"feat", "fix"},
	}
	llmCfg := config.LLMConfig{
		Provider: "ollama",
		Ollama: config.OllamaConfig{
			Model:       "llama3.2",
			Host:        "http://localhost:11434",
			Temperature: 0.3,
			TopP:        0.9,
			MaxDiff:     4000,
		},
	}

	g, err := NewGenerator(cfg, llmCfg)
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	if g == nil {
		t.Fatal("NewGenerator() returned nil")
	}

	if g.provider == nil {
		t.Error("Generator should have provider")
	}

	if len(g.config.Types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(g.config.Types))
	}
}
