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
	}
}
