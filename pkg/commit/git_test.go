package commit

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsGitAvailable(t *testing.T) {
	// Git should be available on the test system
	if !IsGitAvailable() {
		t.Skip("Git is not available on this system")
	}
}

func TestIsGitRepository(t *testing.T) {
	// Current directory should be a git repository
	if !IsGitRepository() {
		t.Skip("Not running in a git repository")
	}
}

func TestIsGitRepository_NotARepo(t *testing.T) {
	tempDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	if IsGitRepository() {
		t.Error("Expected false for non-git directory")
	}
}

func TestGetDiff_EmptyRepo(t *testing.T) {
	tempDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	_ = exec.Command("git", "config", "user.email", "test@test.com").Run()
	_ = exec.Command("git", "config", "user.name", "Test").Run()

	// No changes, should return empty diff
	diff, err := GetDiff(true)
	if err != nil {
		t.Fatalf("GetDiff() error: %v", err)
	}

	if diff != "" {
		t.Errorf("Expected empty diff, got: %s", diff)
	}
}

func TestGetChangedFiles_WithChanges(t *testing.T) {
	tempDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user
	_ = exec.Command("git", "config", "user.email", "test@test.com").Run()
	_ = exec.Command("git", "config", "user.name", "Test").Run()

	// Create and commit a file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	_ = exec.Command("git", "add", "test.txt").Run()
	_ = exec.Command("git", "commit", "-m", "initial").Run()

	// Modify the file and stage it
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}
	_ = exec.Command("git", "add", "test.txt").Run()

	files, err := GetChangedFiles(true)
	if err != nil {
		t.Fatalf("GetChangedFiles() error: %v", err)
	}

	if len(files) != 1 || files[0] != "test.txt" {
		t.Errorf("Expected [test.txt], got: %v", files)
	}
}

func TestGetRecentCommits(t *testing.T) {
	if !IsGitRepository() {
		t.Skip("Not a git repository")
	}

	commits, err := GetRecentCommits(5)
	if err != nil {
		t.Errorf("GetRecentCommits() error = %v", err)
	}

	// Should return at most 5 commits
	if len(commits) > 5 {
		t.Errorf("Expected at most 5 commits, got %d", len(commits))
	}

	// Each commit should be non-empty
	for i, commit := range commits {
		if commit == "" {
			t.Errorf("Commit %d is empty", i)
		}
	}
}

func TestGetRecentCommitsFromBranch(t *testing.T) {
	if !IsGitRepository() {
		t.Skip("Not a git repository")
	}

	// Test with auto-detection (empty base branch)
	commits, err := GetRecentCommitsFromBranch(5, "")
	if err != nil {
		t.Errorf("GetRecentCommitsFromBranch() error = %v", err)
	}

	if len(commits) > 5 {
		t.Errorf("Expected at most 5 commits, got %d", len(commits))
	}

	// Test with explicit base branch
	commits, err = GetRecentCommitsFromBranch(5, "master")
	if err != nil {
		t.Errorf("GetRecentCommitsFromBranch(master) error = %v", err)
	}

	// Should not fail even if master doesn't exist (fallback to recent commits)
	if len(commits) > 5 {
		t.Errorf("Expected at most 5 commits, got %d", len(commits))
	}
}

func TestGetRecentCommits_Zero(t *testing.T) {
	commits, err := GetRecentCommits(0)
	if err != nil {
		t.Errorf("GetRecentCommits(0) error = %v", err)
	}

	if len(commits) != 0 {
		t.Errorf("Expected 0 commits, got %d", len(commits))
	}
}

func TestGetRecentCommits_Negative(t *testing.T) {
	commits, err := GetRecentCommits(-1)
	if err != nil {
		t.Errorf("GetRecentCommits(-1) error = %v", err)
	}

	if len(commits) != 0 {
		t.Errorf("Expected 0 commits, got %d", len(commits))
	}
}

func TestGetRecentCommitsFromBranch_InvalidRef(t *testing.T) {
	if !IsGitRepository() {
		t.Skip("Not a git repository")
	}

	// Test with invalid characters (command injection attempt)
	commits, err := GetRecentCommitsFromBranch(5, "main; echo hacked")
	if err != nil {
		t.Errorf("GetRecentCommitsFromBranch() should not error, got %v", err)
	}

	// Should fallback to recent commits when validation fails
	if len(commits) > 5 {
		t.Errorf("Expected at most 5 commits, got %d", len(commits))
	}
}

func TestValidateRef(t *testing.T) {
	tests := []struct {
		name    string
		ref     string
		wantErr bool
	}{
		{"valid branch", "main", false},
		{"valid with slash", "feature/test", false},
		{"valid with hyphen", "my-branch", false},
		{"valid with underscore", "my_branch", false},
		{"valid with dot", "release/1.0.0", false},
		{"empty ref", "", true},
		{"command injection semicolon", "main; rm -rf /", true},
		{"command injection pipe", "main | cat", true},
		{"command injection ampersand", "main & echo test", true},
		{"command injection backtick", "main`whoami`", true},
		{"command injection dollar", "main$(whoami)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRef(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRef(%q) error = %v, wantErr %v", tt.ref, err, tt.wantErr)
			}
		})
	}
}
