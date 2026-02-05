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
