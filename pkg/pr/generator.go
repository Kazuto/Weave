package pr

import (
	"strings"

	"github.com/Kazuto/Weave/pkg/config"
	"github.com/Kazuto/Weave/pkg/llm"
)

type PRContext struct {
	Branch   string
	Base     string
	Commits  string
	Files    string
	Diff     string
	Template string
}

type Generator struct {
	provider llm.Provider
	config   config.PRConfig
}

func NewGenerator(prCfg config.PRConfig, llmCfg config.LLMConfig) (*Generator, error) {
	provider, err := llm.NewProvider(llmCfg)
	if err != nil {
		return nil, err
	}

	return &Generator{
		provider: provider,
		config:   prCfg,
	}, nil
}

func (g *Generator) CheckConnection() bool {
	return g.provider.CheckConnection()
}

func (g *Generator) CheckModel() bool {
	return g.provider.IsModelAvailable()
}

func (g *Generator) Generate(ctx PRContext) (string, error) {
	if g.config.MaxDiff > 0 && len(ctx.Diff) > g.config.MaxDiff {
		ctx.Diff = ctx.Diff[:g.config.MaxDiff]
	}

	prompt := g.buildPrompt(ctx)

	response, err := g.provider.Generate(prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(response), nil
}

func (g *Generator) buildPrompt(ctx PRContext) string {
	prompt := g.config.Prompt

	// Handle {{if .Template}}...{{else}}...{{end}} conditional
	prompt = expandTemplateConditional(prompt, ctx.Template)

	prompt = strings.ReplaceAll(prompt, "{{.Branch}}", ctx.Branch)
	prompt = strings.ReplaceAll(prompt, "{{.Base}}", ctx.Base)
	prompt = strings.ReplaceAll(prompt, "{{.Commits}}", ctx.Commits)
	prompt = strings.ReplaceAll(prompt, "{{.Files}}", ctx.Files)
	prompt = strings.ReplaceAll(prompt, "{{.Diff}}", ctx.Diff)

	return prompt
}

func expandTemplateConditional(prompt, template string) string {
	ifTag := "{{if .Template}}"
	elseTag := "{{else}}"
	endTag := "{{end}}"

	ifIdx := strings.Index(prompt, ifTag)
	if ifIdx == -1 {
		// No conditional, just replace the placeholder directly
		return strings.ReplaceAll(prompt, "{{.Template}}", template)
	}

	endIdx := strings.Index(prompt, endTag)
	if endIdx == -1 {
		return strings.ReplaceAll(prompt, "{{.Template}}", template)
	}

	before := prompt[:ifIdx]
	after := prompt[endIdx+len(endTag):]
	block := prompt[ifIdx+len(ifTag) : endIdx]

	elseIdx := strings.Index(block, elseTag)

	var selected string
	if template != "" {
		if elseIdx != -1 {
			selected = block[:elseIdx]
		} else {
			selected = block
		}
		selected = strings.ReplaceAll(selected, "{{.Template}}", template)
	} else {
		if elseIdx != -1 {
			selected = block[elseIdx+len(elseTag):]
		} else {
			selected = ""
		}
	}

	return before + selected + after
}
