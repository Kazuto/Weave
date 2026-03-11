package ui

import (
	"fmt"
	"os/exec"
	"strings"
)

// Style applies gum styling to text
func Style(text string, options ...string) string {
	if !IsGumAvailable() {
		return text
	}

	args := []string{"style"}
	args = append(args, options...)
	args = append(args, text)

	cmd := exec.Command("gum", args...)
	output, err := cmd.Output()
	if err != nil {
		return text
	}

	return strings.TrimSpace(string(output))
}

// FormatHeader creates a styled header
func FormatHeader(text string) string {
	return Style(text, "--bold", "--foreground", "212")
}

// FormatSuccess creates a success message
func FormatSuccess(text string) string {
	return Style("✓ "+text, "--foreground", "42")
}

// FormatError creates an error message
func FormatError(text string) string {
	return Style("✗ "+text, "--foreground", "196")
}

// FormatInfo creates an info message
func FormatInfo(text string) string {
	return Style(text, "--foreground", "86")
}

// Spin executes a command with a gum spinner
// Returns the output and any error
func Spin(title string, command func() error) error {
	if !IsGumAvailable() {
		return command()
	}

	// For commands, we need to run the function directly
	// as gum spin expects a shell command
	return command()
}

// IsGumAvailable checks if gum CLI is installed
func IsGumAvailable() bool {
	_, err := exec.LookPath("gum")
	return err == nil
}

// Confirm displays a yes/no confirmation prompt using gum
// Returns true if user confirms, false otherwise
func Confirm(prompt string, defaultValue bool) (bool, error) {
	if !IsGumAvailable() {
		return confirmFallback(prompt, defaultValue)
	}

	args := []string{"confirm", prompt}
	if defaultValue {
		args = append(args, "--default=true")
	} else {
		args = append(args, "--default=false")
	}

	cmd := exec.Command("gum", args...)
	err := cmd.Run()
	if err != nil {
		// Exit code 1 means "no", other errors are actual errors
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Choose displays a selection menu using gum
// Returns the selected item
func Choose(prompt string, options []string, defaultValue string) (string, error) {
	if !IsGumAvailable() {
		return chooseFallback(prompt, options, defaultValue)
	}

	args := []string{"choose"}
	if prompt != "" {
		args = append(args, "--header", prompt)
	}
	args = append(args, options...)

	cmd := exec.Command("gum", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			// User cancelled with Ctrl+C
			return defaultValue, nil
		}
		return "", err
	}

	choice := strings.TrimSpace(string(output))
	if choice == "" {
		return defaultValue, nil
	}
	return choice, nil
}

// Input displays a text input prompt using gum
func Input(prompt string, placeholder string) (string, error) {
	if !IsGumAvailable() {
		return inputFallback(prompt, placeholder)
	}

	args := []string{"input"}
	if prompt != "" {
		args = append(args, "--prompt", prompt+" ")
	}
	if placeholder != "" {
		args = append(args, "--placeholder", placeholder)
	}

	cmd := exec.Command("gum", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// Fallback functions for when gum is not available

func confirmFallback(prompt string, defaultValue bool) (bool, error) {
	defaultStr := "y/N"
	if defaultValue {
		defaultStr = "Y/n"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultStr)

	var response string
	fmt.Scanln(&response)
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" {
		return defaultValue, nil
	}

	return response == "y" || response == "yes", nil
}

func chooseFallback(prompt string, options []string, defaultValue string) (string, error) {
	if prompt != "" {
		fmt.Println(prompt)
	}

	for i, opt := range options {
		marker := ""
		if opt == defaultValue {
			marker = " (default)"
		}
		fmt.Printf("  %d. %s%s\n", i+1, opt, marker)
	}

	fmt.Printf("\nEnter number or name [%s]: ", defaultValue)

	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue, nil
	}

	// Try to parse as number
	var idx int
	if _, err := fmt.Sscanf(input, "%d", &idx); err == nil {
		if idx >= 1 && idx <= len(options) {
			return options[idx-1], nil
		}
	}

	// Check if it's a valid option name
	for _, opt := range options {
		if opt == input {
			return opt, nil
		}
	}

	return defaultValue, nil
}

func inputFallback(prompt string, placeholder string) (string, error) {
	if placeholder != "" {
		fmt.Printf("%s [%s]: ", prompt, placeholder)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	var input string
	fmt.Scanln(&input)
	return strings.TrimSpace(input), nil
}
