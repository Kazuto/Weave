package config

import "fmt"

// ValidationResult holds the result of configuration validation
type ValidationResult struct {
	Errors   []error
	Warnings []string
	Fixed    bool
}

// IsValid returns true if there are no validation errors
func (r *ValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

// ValidateAndFix validates the configuration and fixes invalid values with defaults
func ValidateAndFix(config *Config) *ValidationResult {
	if config == nil {
		return &ValidationResult{
			Errors: []error{fmt.Errorf("configuration cannot be nil")},
		}
	}

	result := &ValidationResult{}

	// Add field validations here as Config struct grows

	return result
}

// ValidateStrict performs strict validation without fixing values
func ValidateStrict(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	// Add strict field validations here as Config struct grows

	return nil
}
