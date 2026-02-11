package config

import (
	"fmt"
	"strings"
)

type ValidationResult struct {
	Errors   []error
	Warnings []string
	Fixed    bool
}

func (r *ValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

func ValidateAndFix(config *Config) *ValidationResult {
	if config == nil {
		return &ValidationResult{
			Errors: []error{fmt.Errorf("configuration cannot be nil")},
		}
	}

	result := &ValidationResult{}
	defaults := GetDefaultConfig()

	// Validate and fix max_length
	if config.Branch.MaxLength < 10 || config.Branch.MaxLength > 200 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("branch.max_length %d is out of range (10-200), using default %d",
				config.Branch.MaxLength, defaults.Branch.MaxLength))
		config.Branch.MaxLength = defaults.Branch.MaxLength
		result.Fixed = true
	}

	// Validate and fix types
	if len(config.Branch.Types) == 0 {
		result.Warnings = append(result.Warnings, "branch.types is empty, using defaults")
		config.Branch.Types = make(map[string]string)
		for k, v := range defaults.Branch.Types {
			config.Branch.Types[k] = v
		}
		result.Fixed = true
	} else {
		for key, value := range config.Branch.Types {
			if key == "" {
				result.Errors = append(result.Errors, fmt.Errorf("branch.types key cannot be empty"))
			}
			if value == "" {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("branch.types value for key '%s' is empty, using key as value", key))
				config.Branch.Types[key] = key
				result.Fixed = true
			}
		}
	}

	// Validate and fix default_type
	if config.Branch.DefaultType == "" {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("branch.default_type is empty, using default '%s'", defaults.Branch.DefaultType))
		config.Branch.DefaultType = defaults.Branch.DefaultType
		result.Fixed = true
	} else if _, exists := config.Branch.Types[config.Branch.DefaultType]; !exists {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("branch.default_type '%s' does not exist in branch.types, using default '%s'",
				config.Branch.DefaultType, defaults.Branch.DefaultType))
		config.Branch.DefaultType = defaults.Branch.DefaultType
		result.Fixed = true
	}

	// Validate and fix sanitization.separator
	if config.Branch.Sanitization.Separator == "" {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("branch.sanitization.separator is empty, using default '%s'", defaults.Branch.Sanitization.Separator))
		config.Branch.Sanitization.Separator = defaults.Branch.Sanitization.Separator
		result.Fixed = true
	}

	if len(config.Branch.Sanitization.Separator) > 5 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("branch.sanitization.separator '%s' is too long (>5 chars), using default '%s'",
				config.Branch.Sanitization.Separator, defaults.Branch.Sanitization.Separator))
		config.Branch.Sanitization.Separator = defaults.Branch.Sanitization.Separator
		result.Fixed = true
	}

	problematicChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	for _, char := range problematicChars {
		if strings.Contains(config.Branch.Sanitization.Separator, char) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("branch.sanitization.separator '%s' contains problematic character '%s', using default '%s'",
					config.Branch.Sanitization.Separator, char, defaults.Branch.Sanitization.Separator))
			config.Branch.Sanitization.Separator = defaults.Branch.Sanitization.Separator
			result.Fixed = true
			break
		}
	}

	// Validate and fix commit.ollama.model
	if config.Commit.Ollama.Model == "" {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("commit.ollama.model is empty, using default '%s'", defaults.Commit.Ollama.Model))
		config.Commit.Ollama.Model = defaults.Commit.Ollama.Model
		result.Fixed = true
	}

	// Validate and fix commit.ollama.host
	if config.Commit.Ollama.Host == "" {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("commit.ollama.host is empty, using default '%s'", defaults.Commit.Ollama.Host))
		config.Commit.Ollama.Host = defaults.Commit.Ollama.Host
		result.Fixed = true
	}

	// Validate and fix commit.ollama.temperature
	if config.Commit.Ollama.Temperature < 0 || config.Commit.Ollama.Temperature > 2 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("commit.ollama.temperature %.2f is out of range (0-2), using default %.2f",
				config.Commit.Ollama.Temperature, defaults.Commit.Ollama.Temperature))
		config.Commit.Ollama.Temperature = defaults.Commit.Ollama.Temperature
		result.Fixed = true
	}

	// Validate and fix commit.ollama.top_p
	if config.Commit.Ollama.TopP < 0 || config.Commit.Ollama.TopP > 1 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("commit.ollama.top_p %.2f is out of range (0-1), using default %.2f",
				config.Commit.Ollama.TopP, defaults.Commit.Ollama.TopP))
		config.Commit.Ollama.TopP = defaults.Commit.Ollama.TopP
		result.Fixed = true
	}

	// Validate and fix commit.ollama.max_diff
	if config.Commit.Ollama.MaxDiff < 100 || config.Commit.Ollama.MaxDiff > 100000 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("commit.ollama.max_diff %d is out of range (100-100000), using default %d",
				config.Commit.Ollama.MaxDiff, defaults.Commit.Ollama.MaxDiff))
		config.Commit.Ollama.MaxDiff = defaults.Commit.Ollama.MaxDiff
		result.Fixed = true
	}

	// Validate and fix commit.types
	if len(config.Commit.Types) == 0 {
		result.Warnings = append(result.Warnings, "commit.types is empty, using defaults")
		config.Commit.Types = defaults.Commit.Types
		result.Fixed = true
	}

	// Validate and fix commit.prompt
	if config.Commit.Prompt == "" {
		result.Warnings = append(result.Warnings, "commit.prompt is empty, using default")
		config.Commit.Prompt = defaults.Commit.Prompt
		result.Fixed = true
	}

	// Validate and fix pr.max_diff
	if config.PR.MaxDiff < 100 || config.PR.MaxDiff > 100000 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("pr.max_diff %d is out of range (100-100000), using default %d",
				config.PR.MaxDiff, defaults.PR.MaxDiff))
		config.PR.MaxDiff = defaults.PR.MaxDiff
		result.Fixed = true
	}

	// Validate and fix pr.prompt
	if config.PR.Prompt == "" {
		result.Warnings = append(result.Warnings, "pr.prompt is empty, using default")
		config.PR.Prompt = defaults.PR.Prompt
		result.Fixed = true
	}

	return result
}

func ValidateStrict(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	// Validate max_length
	if config.Branch.MaxLength < 10 || config.Branch.MaxLength > 200 {
		return fmt.Errorf("branch.max_length must be between 10 and 200")
	}

	// Validate types
	if len(config.Branch.Types) == 0 {
		return fmt.Errorf("branch.types cannot be empty")
	}

	for key, value := range config.Branch.Types {
		if key == "" {
			return fmt.Errorf("branch.types key cannot be empty")
		}
		if value == "" {
			return fmt.Errorf("branch.types value for key '%s' cannot be empty", key)
		}
	}

	// Validate default_type
	if config.Branch.DefaultType == "" {
		return fmt.Errorf("branch.default_type cannot be empty")
	}

	if _, exists := config.Branch.Types[config.Branch.DefaultType]; !exists {
		return fmt.Errorf("branch.default_type '%s' must exist in branch.types", config.Branch.DefaultType)
	}

	// Validate sanitization.separator
	if config.Branch.Sanitization.Separator == "" {
		return fmt.Errorf("branch.sanitization.separator cannot be empty")
	}

	if len(config.Branch.Sanitization.Separator) > 5 {
		return fmt.Errorf("branch.sanitization.separator cannot be longer than 5 characters")
	}

	problematicChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	for _, char := range problematicChars {
		if strings.Contains(config.Branch.Sanitization.Separator, char) {
			return fmt.Errorf("branch.sanitization.separator cannot contain '%s'", char)
		}
	}

	// Validate commit.ollama.model
	if config.Commit.Ollama.Model == "" {
		return fmt.Errorf("commit.ollama.model cannot be empty")
	}

	// Validate commit.ollama.host
	if config.Commit.Ollama.Host == "" {
		return fmt.Errorf("commit.ollama.host cannot be empty")
	}

	// Validate commit.ollama.temperature
	if config.Commit.Ollama.Temperature < 0 || config.Commit.Ollama.Temperature > 2 {
		return fmt.Errorf("commit.ollama.temperature must be between 0 and 2")
	}

	// Validate commit.ollama.top_p
	if config.Commit.Ollama.TopP < 0 || config.Commit.Ollama.TopP > 1 {
		return fmt.Errorf("commit.ollama.top_p must be between 0 and 1")
	}

	// Validate commit.ollama.max_diff
	if config.Commit.Ollama.MaxDiff < 100 || config.Commit.Ollama.MaxDiff > 100000 {
		return fmt.Errorf("commit.ollama.max_diff must be between 100 and 100000")
	}

	// Validate commit.types
	if len(config.Commit.Types) == 0 {
		return fmt.Errorf("commit.types cannot be empty")
	}

	// Validate commit.prompt
	if config.Commit.Prompt == "" {
		return fmt.Errorf("commit.prompt cannot be empty")
	}

	// Validate pr.max_diff
	if config.PR.MaxDiff < 100 || config.PR.MaxDiff > 100000 {
		return fmt.Errorf("pr.max_diff must be between 100 and 100000")
	}

	// Validate pr.prompt
	if config.PR.Prompt == "" {
		return fmt.Errorf("pr.prompt cannot be empty")
	}

	return nil
}
