package branch

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckoutBranch(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "weave-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Initialize a git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for the test repo
	configCmds := [][]string{
		{"git", "config", "user.name", "Test User"},
		{"git", "config", "user.email", "test@example.com"},
	}
	for _, args := range configCmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to configure git: %v", err)
		}
	}

	// Create an initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	addCmd := exec.Command("git", "add", "test.txt")
	addCmd.Dir = tmpDir
	if err := addCmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "Initial commit")
	commitCmd.Dir = tmpDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Change to the test directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Test successful branch creation
	branchName := "feature/TEST-123-test-branch"
	if err := CheckoutBranch(branchName); err != nil {
		t.Errorf("CheckoutBranch() failed: %v", err)
	}

	// Verify we're on the new branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	currentBranch := string(output)
	if currentBranch[:len(currentBranch)-1] != branchName {
		t.Errorf("Expected to be on branch %q, got %q", branchName, currentBranch)
	}

	// Test error when branch already exists
	if err := CheckoutBranch(branchName); err == nil {
		t.Error("Expected error when creating existing branch, got nil")
	}
}
