package config

func getDefaultPRPrompt() string {
	return `Generate a pull request description for the following changes.

Branch: {{.Branch}} → {{.Base}}

Commits:
{{.Commits}}

Changed files:
{{.Files}}

Diff (truncated):
{{.Diff}}

{{if .Template}}Use the following PR template as a structural guide. Fill in each section based on the actual changes:

{{.Template}}
{{else}}Format the description as:

## Summary
A concise overview of what this PR does and why.

## Changes
- Bullet points describing specific changes

## Test Plan
- How to verify these changes work correctly
{{end}}
Generate ONLY the PR description, nothing else. Be concise and specific.`
}

func GetDefaultConfig() *Config {
	return &Config{
		Branch: BranchConfig{
			MaxLength:   60,
			DefaultType: "feature",
			Types: map[string]string{
				"feature":  "feature",
				"hotfix":   "hotfix",
				"refactor": "refactor",
				"support":  "support",
			},
			Sanitization: SanitizationConfig{
				Separator:     "-",
				Lowercase:     true,
				RemoveUmlauts: false,
			},
		},
		Commit: CommitConfig{
			Ollama: OllamaConfig{
				Model:       "llama3.2",
				Host:        "http://localhost:11434",
				Temperature: 0.3,
				TopP:        0.9,
				MaxDiff:     4000,
			},
			Types: []string{
				"feat",
				"fix",
				"docs",
				"style",
				"refactor",
				"perf",
				"test",
				"chore",
				"ci",
				"build",
			},
			Prompt: `Based on the following git diff, generate a commit message in Conventional Commits format.

Format:
<type>(<scope>): <short description>

- <bullet point describing a specific change>
- <bullet point describing another change>

Types: {{.Types}}
Scope: The module/component affected in PascalCase (e.g., CI, API, Auth, Core)

Rules:
- First line: type(Scope): Capitalized short description
- Blank line after the first line
- IMPORTANT: Use MINIMAL bullet points. One bullet per distinct change.
- Single file change = 1-3 bullets MAX
- Never repeat information from the first line
- Never add filler bullets like "Add necessary configurations" or "Update file structure"
- Each bullet must describe a UNIQUE, SPECIFIC change

Changed files:
{{.Files}}

Git diff:
{{.Diff}}

Generate ONLY the commit message, nothing else. Be concise and specific.`,
		},
		PR: PRConfig{
			DefaultBase: "",
			MaxDiff:     8000,
			Prompt:      getDefaultPRPrompt(),
		},
	}
}
