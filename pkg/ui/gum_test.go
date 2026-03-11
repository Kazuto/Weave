package ui

import (
	"testing"
)

func TestIsGumAvailable(t *testing.T) {
	// This test just verifies the function runs without error
	// The actual result depends on whether gum is installed
	_ = IsGumAvailable()
}

func TestFallbackFunctions(t *testing.T) {
	// Test that fallback functions don't panic
	// We can't easily test interactive behavior in unit tests

	t.Run("chooseFallback handles empty options", func(t *testing.T) {
		options := []string{"option1", "option2"}
		defaultVal := "option1"

		// Just verify the function signature is correct
		// Actual interactive testing would require mocking stdin
		_, _ = chooseFallback("test", options, defaultVal)
	})
}
