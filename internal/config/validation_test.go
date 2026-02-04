package config

import (
	"fmt"
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
		name         string
		config       *Config
		expectValid  bool
		expectFixed  bool
		expectErrors int
	}{
		{
			name:         "nil config",
			config:       nil,
			expectValid:  false,
			expectFixed:  false,
			expectErrors: 1,
		},
		{
			name:         "valid empty config",
			config:       &Config{},
			expectValid:  true,
			expectFixed:  false,
			expectErrors: 0,
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
		})
	}
}

func TestValidateStrict(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid empty config",
			config:  &Config{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStrict(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStrict() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
