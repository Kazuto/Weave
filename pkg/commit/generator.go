package commit

import (
	"fmt"
	"strings"

	"github.com/Kazuto/Weave/pkg/config"
	"github.com/Kazuto/Weave/pkg/llm"
)

type Generator struct {
	provider  llm.Provider
	config    config.CommitConfig
	llmConfig config.LLMConfig
}

func NewGenerator(cfg config.CommitConfig, llmCfg config.LLMConfig) (*Generator, error) {
	provider, err := llm.NewProvider(llmCfg)
	if err != nil {
		return nil, err
	}

	return &Generator{
		provider:  provider,
		config:    cfg,
		llmConfig: llmCfg,
	}, nil
}

func (g *Generator) CheckProvider() error {
	if !g.provider.CheckConnection() {
		providerType := g.llmConfig.Provider
		if providerType == "" {
			providerType = "ollama"
		}
		return fmt.Errorf("cannot connect to %s provider", providerType)
	}

	if !g.provider.IsModelAvailable() {
		return fmt.Errorf("model is not available")
	}

	return nil
}

func (g *Generator) CheckConnection() bool {
	return g.provider.CheckConnection()
}

func (g *Generator) CheckModel() bool {
	return g.provider.IsModelAvailable()
}

func (g *Generator) Generate(diff string, files []string) (string, error) {
	maxDiff := llm.GetMaxDiff(g.llmConfig)
	if maxDiff > 0 && len(diff) > maxDiff {
		diff = diff[:maxDiff]
	}

	// Get recent commits for context
	recentCommits, _ := GetRecentCommitsFromBranch(g.config.ReferenceCommits, g.config.ReferenceBranch)

	prompt := g.buildPrompt(diff, files, recentCommits)

	response, err := g.provider.Generate(prompt)
	if err != nil {
		return "", err
	}

	return g.cleanResponse(response), nil
}

func (g *Generator) buildPrompt(diff string, files []string, recentCommits []string) string {
	prompt := g.config.Prompt
	prompt = strings.ReplaceAll(prompt, "{{.Types}}", strings.Join(g.config.Types, ", "))
	prompt = strings.ReplaceAll(prompt, "{{.Files}}", strings.Join(files, "\n"))
	prompt = strings.ReplaceAll(prompt, "{{.Diff}}", diff)

	// Add recent commits for context
	if len(recentCommits) > 0 {
		prompt = strings.ReplaceAll(prompt, "{{.RecentCommits}}", strings.Join(recentCommits, "\n"))
	} else {
		prompt = strings.ReplaceAll(prompt, "{{.RecentCommits}}", "No recent commits available")
	}

	return prompt
}

func (g *Generator) cleanResponse(response string) string {
	msg := strings.TrimSpace(response)
	msg = strings.Trim(msg, `"'`)
	return msg
}
