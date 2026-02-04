package config

import (
	"fmt"
	"strings"
	"testing"
)

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
			},
			expectValid:    true,
			expectFixed:    true,
			expectErrors:   0,
			expectWarnings: 1,
		},
		{
			name: "fixes empty types",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   60,
					DefaultType: "feature",
					Types:       map[string]string{},
					Sanitization: SanitizationConfig{
						Separator: "-",
					},
				},
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
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "max_length")
			},
		},
		{
			name: "invalid max_length - too large",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   300,
					DefaultType: "feature",
					Types:       map[string]string{"feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "-",
					},
				},
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "max_length")
			},
		},
		{
			name: "empty types",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   60,
					DefaultType: "feature",
					Types:       map[string]string{},
					Sanitization: SanitizationConfig{
						Separator: "-",
					},
				},
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "types")
			},
		},
		{
			name: "empty default_type",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   60,
					DefaultType: "",
					Types:       map[string]string{"feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "-",
					},
				},
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "default_type")
			},
		},
		{
			name: "default_type not in types",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   60,
					DefaultType: "nonexistent",
					Types:       map[string]string{"feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "-",
					},
				},
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "default_type")
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
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "separator")
			},
		},
		{
			name: "separator too long",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   60,
					DefaultType: "feature",
					Types:       map[string]string{"feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "------",
					},
				},
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "separator")
			},
		},
		{
			name: "problematic separator character",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   60,
					DefaultType: "feature",
					Types:       map[string]string{"feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "/",
					},
				},
			},
			wantErr: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "separator")
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

	if config.Branch.MaxLength != 60 {
		t.Errorf("Default MaxLength = %d, want 60", config.Branch.MaxLength)
	}

	if config.Branch.DefaultType != "feature" {
		t.Errorf("Default DefaultType = %s, want 'feature'", config.Branch.DefaultType)
	}

	expectedTypes := []string{"feature", "hotfix", "refactor", "support"}
	for _, branchType := range expectedTypes {
		if _, exists := config.Branch.Types[branchType]; !exists {
			t.Errorf("Default Types missing '%s'", branchType)
		}
	}

	if config.Branch.Sanitization.Separator != "-" {
		t.Errorf("Default Separator = %s, want '-'", config.Branch.Sanitization.Separator)
	}

	if !config.Branch.Sanitization.Lowercase {
		t.Error("Default Lowercase should be true")
	}

	if config.Branch.Sanitization.RemoveUmlauts {
		t.Error("Default RemoveUmlauts should be false")
	}

	// Verify default config passes strict validation
	if err := ValidateStrict(config); err != nil {
		t.Errorf("Default config should pass strict validation: %v", err)
	}
}
