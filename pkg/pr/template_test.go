package pr

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFindPRTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templatePath string
		content      string
		expectFound  bool
	}{
		{
			name:         "github directory template",
			templatePath: filepath.Join(".github", "PULL_REQUEST_TEMPLATE.md"),
			content:      "## Description\n\n## Changes",
			expectFound:  true,
		},
		{
			name:         "github directory lowercase",
			templatePath: filepath.Join(".github", "pull_request_template.md"),
			content:      "## Summary",
			expectFound:  true,
		},
		{
			name:         "root directory template",
			templatePath: "PULL_REQUEST_TEMPLATE.md",
			content:      "## Root Template",
			expectFound:  true,
		},
		{
			name:         "docs directory template",
			templatePath: filepath.Join("docs", "PULL_REQUEST_TEMPLATE.md"),
			content:      "## Docs Template",
			expectFound:  true,
		},
		{
			name:        "no template",
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get current directory: %v", err)
			}
			defer func() { _ = os.Chdir(originalDir) }()

			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("failed to change to temp directory: %v", err)
			}

			// Initialize git repo so getRepoRoot works
			if err := exec.Command("git", "init").Run(); err != nil {
				t.Fatalf("failed to init git repo: %v", err)
			}

			if tt.templatePath != "" {
				fullPath := filepath.Join(tempDir, tt.templatePath)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("failed to write template: %v", err)
				}
			}

			result := FindPRTemplate()

			if tt.expectFound && result == "" {
				t.Error("FindPRTemplate() returned empty, expected template content")
			}

			if !tt.expectFound && result != "" {
				t.Errorf("FindPRTemplate() returned %q, expected empty", result)
			}

			if tt.expectFound && result != tt.content {
				t.Errorf("FindPRTemplate() = %q, want %q", result, tt.content)
			}
		})
	}
}

func TestFindPRTemplate_PriorityOrder(t *testing.T) {
	tempDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Create templates in multiple locations
	githubDir := filepath.Join(tempDir, ".github")
	if err := os.MkdirAll(githubDir, 0755); err != nil {
		t.Fatalf("failed to create .github dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(githubDir, "PULL_REQUEST_TEMPLATE.md"), []byte("github template"), 0644); err != nil {
		t.Fatalf("failed to write github template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "PULL_REQUEST_TEMPLATE.md"), []byte("root template"), 0644); err != nil {
		t.Fatalf("failed to write root template: %v", err)
	}

	result := FindPRTemplate()
	if result != "github template" {
		t.Errorf("FindPRTemplate() = %q, want 'github template' (.github/ should have priority)", result)
	}
}
