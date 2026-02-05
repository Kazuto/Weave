package commit

import (
	"fmt"
	"strings"

	"github.com/Kazuto/Weave/pkg/config"
)

type Generator struct {
	ollama *OllamaClient
	config config.CommitConfig
}

func NewGenerator(cfg config.CommitConfig) *Generator {
	return &Generator{
		ollama: NewOllamaClient(cfg.Ollama),
		config: cfg,
	}
}

func (g *Generator) CheckOllama() error {
	if !g.ollama.CheckConnection() {
		return fmt.Errorf("cannot connect to Ollama at %s", g.config.Ollama.Host)
	}

	if !g.ollama.IsModelAvailable() {
		return fmt.Errorf("model '%s' is not available", g.config.Ollama.Model)
	}

	return nil
}

func (g *Generator) Generate(diff string, files []string) (string, error) {
	if len(diff) > g.config.Ollama.MaxDiff {
		diff = diff[:g.config.Ollama.MaxDiff]
	}

	prompt := g.buildPrompt(diff, files)

	response, err := g.ollama.Generate(prompt)
	if err != nil {
		return "", err
	}

	return g.cleanResponse(response), nil
}

func (g *Generator) buildPrompt(diff string, files []string) string {
	prompt := g.config.Prompt
	prompt = strings.ReplaceAll(prompt, "{{.Types}}", strings.Join(g.config.Types, ", "))
	prompt = strings.ReplaceAll(prompt, "{{.Files}}", strings.Join(files, "\n"))
	prompt = strings.ReplaceAll(prompt, "{{.Diff}}", diff)
	return prompt
}

func (g *Generator) cleanResponse(response string) string {
	msg := strings.TrimSpace(response)
	msg = strings.Trim(msg, `"'`)
	return msg
}
