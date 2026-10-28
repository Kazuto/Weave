package cherrypick

import (
	"testing"
)

func TestExtractTicketID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Valid Ticket ID", "feature/PROJ-123-add-login", "PROJ-123"},
		{"Ticket ID in middle", "fix/WEAVE-456-bug", "WEAVE-456"},
		{"No Ticket ID", "feature/add-login", ""},
		{"Multiple IDs", "PROJ-123 and PROJ-456", "PROJ-123"},
		{"Lowercase", "feature/proj-123", ""}, // Regex expects uppercase [A-Z]
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractTicketID(tt.input); got != tt.want {
				t.Errorf("ExtractTicketID() = %v, want %v", got, tt.want)
			}
		})
	}
}
