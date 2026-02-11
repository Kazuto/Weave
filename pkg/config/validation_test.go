package config

import (
	"fmt"
	"strings"
	"testing"
)

// validCommitConfig returns a valid commit config for testing
func validCommitConfig() CommitConfig {
	return GetDefaultConfig().Commit
}

// validPRConfig returns a valid PR config for testing
func validPRConfig() PRConfig {
	return GetDefaultConfig().PR
}

func TestValidationResult_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		result *ValidationResult
		want   bool
	}{
		{
			name: "no errors",
			result: &ValidationResult{
				Errors: []error{},
			},
			want: true,
		},
		{
			name: "nil errors slice",
			result: &ValidationResult{
				Errors: nil,
			},
			want: true,
		},
		{
			name: "with errors",
			result: &ValidationResult{
				Errors: []error{
					fmt.Errorf("test error"),
				},
			},
			want: false,
		},
		{
			name: "with warnings but no errors",
			result: &ValidationResult{
				Errors:   []error{},
				Warnings: []string{"warning 1", "warning 2"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAndFix(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		expectValid    bool
		expectFixed    bool
		expectErrors   int
		expectWarnings int
	}{
		{
			name:         "nil config",
			config:       nil,
			expectValid:  false,
			expectFixed:  false,
			expectErrors: 1,
		},
		{
			name:           "valid config",
			config:         GetDefaultConfig(),
			expectValid:    true,
			expectFixed:    false,
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name: "fixes invalid max_length - too small",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   5,
					DefaultType: "feature",
					Types:       map[string]string{"feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "-",
					},
				},
				Commit: validCommitConfig(),
				PR:     validPRConfig(),
			},
			expectValid:    true,
			expectFixed:    true,
			expectErrors:   0,
			expectWarnings: 1,
		},
		{
			name: "fixes invalid max_length - too large",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   300,
					DefaultType: "feature",
					Types:       map[string]string{"feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "-",
					},
				},
				Commit: validCommitConfig(),
				PR:     validPRConfig(),
			},
			expectValid:    true,
			expectFixed:    true,
			expectErrors:   0,
			expectWarnings: 1,
		},
		{
			name: "fixes empty branch types",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   60,
					DefaultType: "feature",
					Types:       map[string]string{},
					Sanitization: SanitizationConfig{
						Separator: "-",
					},
				},
				Commit: validCommitConfig(),
				PR:     validPRConfig(),
			},
			expectValid:    true,
			expectFixed:    true,
			expectErrors:   0,
			expectWarnings: 1,
		},
		{
			name: "error on empty type key",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   60,
					DefaultType: "feature",
					Types:       map[string]string{"": "empty", "feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "-",
					},
				},
				Commit: validCommitConfig(),
				PR:     validPRConfig(),
			},
			expectValid:  false,
			expectFixed:  false,
			expectErrors: 1,
		},
		{
			name: "fixes empty separator",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   60,
					DefaultType: "feature",
					Types:       map[string]string{"feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "",
					},
				},
				Commit: validCommitConfig(),
				PR:     validPRConfig(),
			},
			expectValid:    true,
			expectFixed:    true,
			expectErrors:   0,
			expectWarnings: 1,
		},
		{
			name: "fixes problematic separator",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   60,
					DefaultType: "feature",
					Types:       map[string]string{"feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "/",
					},
				},
				Commit: validCommitConfig(),
				PR:     validPRConfig(),
			},
			expectValid:    true,
			expectFixed:    true,
			expectErrors:   0,
			expectWarnings: 1,
		},
		{
			name: "fixes empty ollama model",
			config: &Config{
				Branch: GetDefaultConfig().Branch,
				Commit: CommitConfig{
					Ollama: OllamaConfig{
						Model:       "",
						Host:        "http://localhost:11434",
						Temperature: 0.3,
						TopP:        0.9,
						MaxDiff:     4000,
					},
					Types: []string{"feat", "fix"},
				},
				PR: validPRConfig(),
			},
			expectValid:    true,
			expectFixed:    true,
			expectErrors:   0,
			expectWarnings: 1,
		},
		{
			name: "fixes invalid temperature",
			config: &Config{
				Branch: GetDefaultConfig().Branch,
				Commit: CommitConfig{
					Ollama: OllamaConfig{
						Model:       "llama3.2",
						Host:        "http://localhost:11434",
						Temperature: 5.0,
						TopP:        0.9,
						MaxDiff:     4000,
					},
					Types: []string{"feat", "fix"},
				},
				PR: validPRConfig(),
			},
			expectValid:    true,
			expectFixed:    true,
			expectErrors:   0,
			expectWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateAndFix(tt.config)

			if result.IsValid() != tt.expectValid {
				t.Errorf("ValidateAndFix() IsValid() = %v, want %v", result.IsValid(), tt.expectValid)
			}

			if result.Fixed != tt.expectFixed {
				t.Errorf("ValidateAndFix() Fixed = %v, want %v", result.Fixed, tt.expectFixed)
			}

			if len(result.Errors) != tt.expectErrors {
				t.Errorf("ValidateAndFix() errors count = %d, want %d", len(result.Errors), tt.expectErrors)
			}

			if tt.expectWarnings > 0 && len(result.Warnings) < tt.expectWarnings {
				t.Errorf("ValidateAndFix() warnings count = %d, want at least %d", len(result.Warnings), tt.expectWarnings)
			}

			// If fixed and valid, verify strict validation passes
			if tt.config != nil && result.Fixed && result.IsValid() {
				if err := ValidateStrict(tt.config); err != nil {
					t.Errorf("Fixed config should pass strict validation: %v", err)
				}
			}
		})
	}
}

func TestValidateStrict(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		wantErr    bool
		errorCheck func(error) bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid config",
			config:  GetDefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid max_length - too small",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   5,
					DefaultType: "feature",
					Types:       map[string]string{"feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "-",
					},
				},
				Commit: validCommitConfig(),
				PR:     validPRConfig(),
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "max_length")
			},
		},
		{
			name: "empty branch types",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   60,
					DefaultType: "feature",
					Types:       map[string]string{},
					Sanitization: SanitizationConfig{
						Separator: "-",
					},
				},
				Commit: validCommitConfig(),
				PR:     validPRConfig(),
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "types")
			},
		},
		{
			name: "empty separator",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   60,
					DefaultType: "feature",
					Types:       map[string]string{"feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "",
					},
				},
				Commit: validCommitConfig(),
				PR:     validPRConfig(),
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "separator")
			},
		},
		{
			name: "empty ollama model",
			config: &Config{
				Branch: GetDefaultConfig().Branch,
				Commit: CommitConfig{
					Ollama: OllamaConfig{
						Model:       "",
						Host:        "http://localhost:11434",
						Temperature: 0.3,
						TopP:        0.9,
						MaxDiff:     4000,
					},
					Types: []string{"feat"},
				},
				PR: validPRConfig(),
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "commit.ollama.model")
			},
		},
		{
			name: "invalid temperature",
			config: &Config{
				Branch: GetDefaultConfig().Branch,
				Commit: CommitConfig{
					Ollama: OllamaConfig{
						Model:       "llama3.2",
						Host:        "http://localhost:11434",
						Temperature: 5.0,
						TopP:        0.9,
						MaxDiff:     4000,
					},
					Types: []string{"feat"},
				},
				PR: validPRConfig(),
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "temperature")
			},
		},
		{
			name: "empty commit types",
			config: &Config{
				Branch: GetDefaultConfig().Branch,
				Commit: CommitConfig{
					Ollama: OllamaConfig{
						Model:       "llama3.2",
						Host:        "http://localhost:11434",
						Temperature: 0.3,
						TopP:        0.9,
						MaxDiff:     4000,
					},
					Types: []string{},
				},
				PR: validPRConfig(),
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "commit.types")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStrict(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStrict() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.errorCheck != nil && err != nil && !tt.errorCheck(err) {
				t.Errorf("ValidateStrict() error check failed: %v", err)
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	if config == nil {
		t.Fatal("GetDefaultConfig() returned nil")
	}

	// Branch defaults
	if config.Branch.MaxLength != 60 {
		t.Errorf("Default Branch.MaxLength = %d, want 60", config.Branch.MaxLength)
	}

	if config.Branch.DefaultType != "feature" {
		t.Errorf("Default Branch.DefaultType = %s, want 'feature'", config.Branch.DefaultType)
	}

	expectedBranchTypes := []string{"feature", "hotfix", "refactor", "support"}
	for _, branchType := range expectedBranchTypes {
		if _, exists := config.Branch.Types[branchType]; !exists {
			t.Errorf("Default Branch.Types missing '%s'", branchType)
		}
	}

	if config.Branch.Sanitization.Separator != "-" {
		t.Errorf("Default Branch.Sanitization.Separator = %s, want '-'", config.Branch.Sanitization.Separator)
	}

	// Commit defaults
	if config.Commit.Ollama.Model != "llama3.2" {
		t.Errorf("Default Commit.Ollama.Model = %s, want 'llama3.2'", config.Commit.Ollama.Model)
	}

	if config.Commit.Ollama.Host != "http://localhost:11434" {
		t.Errorf("Default Commit.Ollama.Host = %s, want 'http://localhost:11434'", config.Commit.Ollama.Host)
	}

	if config.Commit.Ollama.Temperature != 0.3 {
		t.Errorf("Default Commit.Ollama.Temperature = %f, want 0.3", config.Commit.Ollama.Temperature)
	}

	if config.Commit.Ollama.MaxDiff != 4000 {
		t.Errorf("Default Commit.Ollama.MaxDiff = %d, want 4000", config.Commit.Ollama.MaxDiff)
	}

	expectedCommitTypes := []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "chore", "ci", "build"}
	if len(config.Commit.Types) != len(expectedCommitTypes) {
		t.Errorf("Default Commit.Types length = %d, want %d", len(config.Commit.Types), len(expectedCommitTypes))
	}

	// Verify default config passes strict validation
	if err := ValidateStrict(config); err != nil {
		t.Errorf("Default config should pass strict validation: %v", err)
	}
}
