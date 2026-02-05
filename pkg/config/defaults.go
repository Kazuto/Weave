package config

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

Format: <type>(<scope>): <description>

Types: {{.Types}}
Scope: The module/component affected (e.g., Core, Authentication, API, etc.)

Changed files:
{{.Files}}

Git diff:
{{.Diff}}

Generate ONLY the commit message in the format specified, nothing else. Be concise and specific.`,
		},
	}
}
