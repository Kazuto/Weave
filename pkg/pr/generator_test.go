package pr

import (
	"strings"
	"testing"

	"github.com/Kazuto/Weave/pkg/config"
)

func TestNewGenerator(t *testing.T) {
	prCfg := config.PRConfig{
		MaxDiff: 8000,
		Prompt:  "test prompt",
	}
	ollamaCfg := config.OllamaConfig{
		Model: "llama3.2",
		Host:  "http://localhost:11434",
	}

	g := NewGenerator(prCfg, ollamaCfg)

	if g == nil {
		t.Fatal("NewGenerator() returned nil")
	}

	if g.ollama == nil {
		t.Error("Generator should have ollama client")
	}

	if g.config.MaxDiff != 8000 {
		t.Errorf("Generator MaxDiff = %d, want 8000", g.config.MaxDiff)
	}
}

func TestGenerator_buildPrompt(t *testing.T) {
	prCfg := config.PRConfig{
		MaxDiff: 8000,
		Prompt:  "Branch: {{.Branch}} → {{.Base}}\nCommits:\n{{.Commits}}\nFiles:\n{{.Files}}\nDiff:\n{{.Diff}}",
	}
	ollamaCfg := config.OllamaConfig{
		Model: "llama3.2",
		Host:  "http://localhost:11434",
	}

	g := NewGenerator(prCfg, ollamaCfg)

	ctx := PRContext{
		Branch:  "feature/test",
		Base:    "main",
		Commits: "abc1234 add feature",
		Files:   "file.go\nother.go",
		Diff:    "diff --git a/file.go",
	}

	prompt := g.buildPrompt(ctx)

	checks := []struct {
		name     string
		contains string
	}{
		{"branch", "feature/test"},
		{"base", "main"},
		{"commits", "abc1234 add feature"},
		{"files", "file.go"},
		{"diff", "diff --git a/file.go"},
	}

	for _, check := range checks {
		if !strings.Contains(prompt, check.contains) {
			t.Errorf("Prompt should contain %s (%q)", check.name, check.contains)
		}
	}
}

func TestGenerator_buildPrompt_WithTemplate(t *testing.T) {
	prCfg := config.PRConfig{
		MaxDiff: 8000,
		Prompt:  "{{if .Template}}Use template:\n{{.Template}}{{else}}Default format{{end}}\nBranch: {{.Branch}}",
	}
	ollamaCfg := config.OllamaConfig{
		Model: "llama3.2",
		Host:  "http://localhost:11434",
	}

	g := NewGenerator(prCfg, ollamaCfg)

	t.Run("with template", func(t *testing.T) {
		ctx := PRContext{
			Branch:   "feature/test",
			Base:     "main",
			Template: "## Description\n## Changes",
		}

		prompt := g.buildPrompt(ctx)

		if !strings.Contains(prompt, "Use template:") {
			t.Error("Prompt should contain 'Use template:' when template is provided")
		}
		if !strings.Contains(prompt, "## Description") {
			t.Error("Prompt should contain template content")
		}
		if strings.Contains(prompt, "Default format") {
			t.Error("Prompt should NOT contain else branch when template is provided")
		}
	})

	t.Run("without template", func(t *testing.T) {
		ctx := PRContext{
			Branch: "feature/test",
			Base:   "main",
		}

		prompt := g.buildPrompt(ctx)

		if strings.Contains(prompt, "Use template:") {
			t.Error("Prompt should NOT contain 'Use template:' when no template")
		}
		if !strings.Contains(prompt, "Default format") {
			t.Error("Prompt should contain else branch when no template")
		}
	})
}

func TestExpandTemplateConditional(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		template string
		want     string
	}{
		{
			name:     "no conditional",
			prompt:   "Simple prompt with {{.Template}}",
			template: "my template",
			want:     "Simple prompt with my template",
		},
		{
			name:     "conditional with template present",
			prompt:   "Before {{if .Template}}HAS: {{.Template}}{{else}}NO TEMPLATE{{end}} After",
			template: "content",
			want:     "Before HAS: content After",
		},
		{
			name:     "conditional without template",
			prompt:   "Before {{if .Template}}HAS: {{.Template}}{{else}}NO TEMPLATE{{end}} After",
			template: "",
			want:     "Before NO TEMPLATE After",
		},
		{
			name:     "conditional without else - template present",
			prompt:   "Before {{if .Template}}HAS: {{.Template}}{{end}} After",
			template: "content",
			want:     "Before HAS: content After",
		},
		{
			name:     "conditional without else - no template",
			prompt:   "Before {{if .Template}}HAS: {{.Template}}{{end}} After",
			template: "",
			want:     "Before  After",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandTemplateConditional(tt.prompt, tt.template)
			if got != tt.want {
				t.Errorf("expandTemplateConditional() = %q, want %q", got, tt.want)
			}
		})
	}
}
