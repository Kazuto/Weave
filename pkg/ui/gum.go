package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorPurple = lipgloss.NewStyle().Foreground(lipgloss.Color("#B388FF")).Bold(true)
	colorGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	colorRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	colorYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	colorCyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

// FormatHeader creates a styled header
func FormatHeader(text string) string {
	return colorPurple.Render(text)
}

// FormatSuccess creates a success message
func FormatSuccess(text string) string {
	return colorGreen.Render("✓ " + text)
}

// FormatError creates an error message
func FormatError(text string) string {
	return colorRed.Render("✗ " + text)
}

func FormatWarning(text string) string {
	return colorYellow.Render("⚠ " + text)
}

// FormatInfo creates an info message
func FormatInfo(text string) string {
	return colorCyan.Render("▸ " + text)
}

// FormatCyan creates a cyan message without a symbol
func FormatCyan(text string) string {
	return colorCyan.Render(text)
}

// Style applies styling to text.
func Style(text string, options ...string) string {
	if len(options) >= 1 && options[0] == "--bold" {
		return colorPurple.Render(text)
	}
	return text
}

// Spin executes a command with a gum spinner
func Spin(title string, command func() error) error {
	if !IsGumAvailable() {
		return command()
	}
	return command()
}

// IsGumAvailable checks if gum CLI is installed
func IsGumAvailable() bool {
	_, err := exec.LookPath("gum")
	return err == nil
}

// Confirm displays a yes/no confirmation prompt using gum
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
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Choose displays a selection menu using gum
func Choose(prompt string, options []string, defaultValue string, multi bool) (string, error) {
	if !IsGumAvailable() {
		return chooseFallback(prompt, options, defaultValue)
	}

	args := []string{"choose"}

	if multi {
		args = append(args, "--no-limit")
	}

	if prompt != "" {
		args = append(args, "--header", prompt)
	}

	args = append(args, options...)

	cmd := exec.Command("gum", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
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

func confirmFallback(prompt string, defaultValue bool) (bool, error) {
	defaultStr := "y/N"
	if defaultValue {
		defaultStr = "Y/n"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultStr)

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil && response == "" {
		return defaultValue, nil
	}
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
	_, err := fmt.Scanln(&input)
	if err != nil && input == "" {
		return defaultValue, nil
	}
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue, nil
	}

	var idx int
	if _, err := fmt.Sscanf(input, "%d", &idx); err == nil {
		if idx >= 1 && idx <= len(options) {
			return options[idx-1], nil
		}
	}

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
	_, err := fmt.Scanln(&input)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}
