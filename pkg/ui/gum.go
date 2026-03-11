package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[91m"
	colorGreen  = "\033[92m"
	colorCyan   = "\033[96m"
	colorPurple = "\033[95m"
	colorBold   = "\033[1m"
)

// colorize wraps text in ANSI color codes if output supports it
func colorize(text, color string) string {
	if !shouldColorize() {
		return text
	}
	return color + text + colorReset
}

// shouldColorize checks if we should output colors
func shouldColorize() bool {
	// Disable colors if NO_COLOR is set
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Force colors if FORCE_COLOR is set
	if os.Getenv("FORCE_COLOR") != "" || os.Getenv("CLICOLOR_FORCE") != "" {
		return true
	}

	// Enable colors for common terminals
	term := os.Getenv("TERM")
	if strings.Contains(term, "color") || strings.Contains(term, "xterm") ||
	   strings.Contains(term, "screen") || strings.Contains(term, "tmux") {
		return true
	}

	// Check if stdout or stderr is a terminal
	return isTerminal(os.Stdout) || isTerminal(os.Stderr)
}

// isTerminal checks if the given file is a terminal
func isTerminal(f *os.File) bool {
	fileInfo, err := f.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Style applies gum styling to text
func Style(text string, options ...string) string {
	// For compatibility, keep this function but use colorize internally
	// This is mainly used by FormatHeader with bold+purple
	if len(options) >= 2 && options[0] == "--bold" {
		return colorize(text, colorBold+colorPurple)
	}
	return text
}

// FormatHeader creates a styled header
func FormatHeader(text string) string {
	return colorize(text, colorBold+colorPurple)
}

// FormatSuccess creates a success message
func FormatSuccess(text string) string {
	return colorize("✓ "+text, colorGreen)
}

// FormatError creates an error message
func FormatError(text string) string {
	return colorize("✗ "+text, colorRed)
}

// FormatInfo creates an info message
func FormatInfo(text string) string {
	return colorize(text, colorCyan)
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
