package branch

import (
	"testing"
)

func TestJiraClient_parseJSONTitle(t *testing.T) {
	client := NewJiraClient()

	tests := []struct {
		name      string
		input     string
		expected  string
		wantError bool
	}{
		{
			name:      "valid JSON with summary",
			input:     `{"fields":{"summary":"Add user authentication"}}`,
			expected:  "Add user authentication",
			wantError: false,
		},
		{
			name:      "empty summary",
			input:     `{"fields":{"summary":""}}`,
			expected:  "",
			wantError: false,
		},
		{
			name:      "invalid JSON",
			input:     `{invalid json}`,
			expected:  "",
			wantError: true,
		},
		{
			name:      "missing fields",
			input:     `{}`,
			expected:  "",
			wantError: false,
		},
		{
			name:      "complex summary with special characters",
			input:     `{"fields":{"summary":"Fix: Login (issue) with \"quotes\""}}`,
			expected:  `Fix: Login (issue) with "quotes"`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.parseJSONTitle(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("parseJSONTitle() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if result != tt.expected {
				t.Errorf("parseJSONTitle() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewJiraClient(t *testing.T) {
	client := NewJiraClient()
	if client == nil {
		t.Error("NewJiraClient() returned nil")
	}
}

func TestIsJiraAvailable(t *testing.T) {
	// Just verify it doesn't panic - actual availability depends on system
	_ = IsJiraAvailable()
}

func TestJiraClient_IsAvailable(t *testing.T) {
	client := NewJiraClient()
	// Verify method returns same result as package function
	if client.IsAvailable() != IsJiraAvailable() {
		t.Error("IsAvailable() should match IsJiraAvailable()")
	}
}
