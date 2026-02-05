package branch

import (
	"testing"

	"github.com/Kazuto/Weave/pkg/config"
)

func testBranchConfig() config.BranchConfig {
	return config.BranchConfig{
		MaxLength:   60,
		DefaultType: "feature",
		Types: map[string]string{
			"feature": "feature",
			"bugfix":  "bugfix",
			"hotfix":  "hotfix",
		},
		Sanitization: config.SanitizationConfig{
			Separator:     "-",
			Lowercase:     true,
			RemoveUmlauts: true,
		},
	}
}

func TestGenerator_GenerateName(t *testing.T) {
	generator := NewGenerator(testBranchConfig())

	tests := []struct {
		name     string
		info     BranchInfo
		expected string
	}{
		{
			name: "basic branch generation",
			info: BranchInfo{
				Type:     "feature",
				TicketID: "STR-123",
				Title:    "Add user authentication",
			},
			expected: "feature/STR-123-add-user-authentication",
		},
		{
			name: "title with special characters",
			info: BranchInfo{
				Type:     "bugfix",
				TicketID: "BUG-456",
				Title:    "Fix: Login (issue) with \"quotes\"",
			},
			expected: "bugfix/BUG-456-fix-login-issue-with-quotes",
		},
		{
			name: "empty title uses ticket ID",
			info: BranchInfo{
				Type:     "hotfix",
				TicketID: "HOT-999",
				Title:    "",
			},
			expected: "hotfix/HOT-999-hot-999",
		},
		{
			name: "missing type returns empty",
			info: BranchInfo{
				Type:     "",
				TicketID: "TEST-123",
				Title:    "Some title",
			},
			expected: "",
		},
		{
			name: "missing ticket ID returns empty",
			info: BranchInfo{
				Type:     "feature",
				TicketID: "",
				Title:    "Some title",
			},
			expected: "",
		},
		{
			name: "title with German umlauts",
			info: BranchInfo{
				Type:     "feature",
				TicketID: "GER-123",
				Title:    "Füge Benutzerverwaltung hinzü",
			},
			expected: "feature/GER-123-fuege-benutzerverwaltung-hinzue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.GenerateName(tt.info)
			if result != tt.expected {
				t.Errorf("GenerateName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerator_ValidateName(t *testing.T) {
	generator := NewGenerator(testBranchConfig())

	tests := []struct {
		name       string
		branchName string
		wantError  bool
	}{
		{
			name:       "valid branch name",
			branchName: "feature/STR-123-add-user-auth",
			wantError:  false,
		},
		{
			name:       "empty branch name",
			branchName: "",
			wantError:  true,
		},
		{
			name:       "branch name with spaces",
			branchName: "feature/STR-123 add user auth",
			wantError:  true,
		},
		{
			name:       "branch name starting with dot",
			branchName: ".feature/STR-123-add-user-auth",
			wantError:  true,
		},
		{
			name:       "branch name with double dots",
			branchName: "feature/STR..123-add-user-auth",
			wantError:  true,
		},
		{
			name:       "branch name with double slashes",
			branchName: "feature//STR-123-add-user-auth",
			wantError:  true,
		},
		{
			name:       "valid branch with dots in version",
			branchName: "feature/STR-123-add-v2.1.0-support",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.ValidateName(tt.branchName)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateName() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestGenerator_GetBranchType(t *testing.T) {
	generator := NewGenerator(testBranchConfig())

	tests := []struct {
		name     string
		typeKey  string
		expected string
	}{
		{
			name:     "feature type",
			typeKey:  "feature",
			expected: "feature",
		},
		{
			name:     "bugfix type",
			typeKey:  "bugfix",
			expected: "bugfix",
		},
		{
			name:     "unknown type returns default",
			typeKey:  "unknown",
			expected: "feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.GetBranchType(tt.typeKey)
			if result != tt.expected {
				t.Errorf("GetBranchType() = %v, want %v", result, tt.expected)
			}
		})
	}
}
