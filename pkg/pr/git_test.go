package pr

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRef(t *testing.T) {
	tests := []struct {
		ref     string
		wantErr bool
	}{
		{"main", false},
		{"feature/my-branch", false},
		{"release/1.0.0", false},
		{"HEAD", false},
		{"my_branch", false},
		{"", true},
		{"branch name", true},
		{"branch;echo", true},
		{"branch`cmd`", true},
		{"branch$(cmd)", true},
		{"branch&cmd", true},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			err := validateRef(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRef(%q) error = %v, wantErr %v", tt.ref, err, tt.wantErr)
			}
		})
	}
}

func setupGitRepo(t *testing.T) (string, func()) {
	t.Helper()
	tempDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			t.Fatalf("failed to run %v: %v", args, err)
		}
	}

	return tempDir, func() { _ = os.Chdir(originalDir) }
}

func TestGetCurrentBranch(t *testing.T) {
	tempDir, cleanup := setupGitRepo(t)
	defer cleanup()

	// Create initial commit so HEAD exists
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("init"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "init").Run()

	branch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error: %v", err)
	}

	// Default branch could be main or master depending on git config
	if branch == "" {
		t.Error("GetCurrentBranch() returned empty string")
	}
}

func TestDetectBaseBranch(t *testing.T) {
	tempDir, cleanup := setupGitRepo(t)
	defer cleanup()

	// Create initial commit on main
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("init"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "init").Run()

	// Create a "main" branch to ensure detection works
	_ = exec.Command("git", "branch", "-M", "main").Run()

	base := DetectBaseBranch()
	if base == "" {
		t.Error("DetectBaseBranch() returned empty string")
	}
}

func TestGetCommitsBetween(t *testing.T) {
	tempDir, cleanup := setupGitRepo(t)
	defer cleanup()

	// Create initial commit on main
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("init"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "init").Run()
	_ = exec.Command("git", "branch", "-M", "main").Run()

	// Create feature branch with a commit
	_ = exec.Command("git", "checkout", "-b", "feature/test").Run()
	if err := os.WriteFile(testFile, []byte("changed"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "add feature").Run()

	commits, err := GetCommitsBetween("main", "feature/test")
	if err != nil {
		t.Fatalf("GetCommitsBetween() error: %v", err)
	}

	if commits == "" {
		t.Error("GetCommitsBetween() returned empty string, expected commits")
	}

	if len(commits) < 5 {
		t.Errorf("GetCommitsBetween() returned too short: %q", commits)
	}
}

func TestGetDiffBetween(t *testing.T) {
	tempDir, cleanup := setupGitRepo(t)
	defer cleanup()

	// Create initial commit on main
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("init"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "init").Run()
	_ = exec.Command("git", "branch", "-M", "main").Run()

	// Create feature branch with a change
	_ = exec.Command("git", "checkout", "-b", "feature/diff-test").Run()
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "modify file").Run()

	diff, err := GetDiffBetween("main", "feature/diff-test")
	if err != nil {
		t.Fatalf("GetDiffBetween() error: %v", err)
	}

	if diff == "" {
		t.Error("GetDiffBetween() returned empty string, expected diff output")
	}
}

func TestGetChangedFilesBetween(t *testing.T) {
	tempDir, cleanup := setupGitRepo(t)
	defer cleanup()

	// Create initial commit on main
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("init"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "init").Run()
	_ = exec.Command("git", "branch", "-M", "main").Run()

	// Create feature branch with changes
	_ = exec.Command("git", "checkout", "-b", "feature/files-test").Run()
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	newFile := filepath.Join(tempDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new file"), 0644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "add files").Run()

	files, err := GetChangedFilesBetween("main", "feature/files-test")
	if err != nil {
		t.Fatalf("GetChangedFilesBetween() error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("GetChangedFilesBetween() returned %d files, want 2: %v", len(files), files)
	}
}

func TestParseGitHubRepo(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		wantOwner string
		wantRepo  string
		wantOk    bool
	}{
		{
			name:      "HTTPS with .git",
			remoteURL: "https://github.com/Kazuto/Weave.git",
			wantOwner: "Kazuto",
			wantRepo:  "Weave",
			wantOk:    true,
		},
		{
			name:      "HTTPS without .git",
			remoteURL: "https://github.com/Kazuto/Weave",
			wantOwner: "Kazuto",
			wantRepo:  "Weave",
			wantOk:    true,
		},
		{
			name:      "SSH with .git",
			remoteURL: "git@github.com:Kazuto/Weave.git",
			wantOwner: "Kazuto",
			wantRepo:  "Weave",
			wantOk:    true,
		},
		{
			name:      "SSH without .git",
			remoteURL: "git@github.com:Kazuto/Weave",
			wantOwner: "Kazuto",
			wantRepo:  "Weave",
			wantOk:    true,
		},
		{
			name:      "Non-GitHub URL",
			remoteURL: "https://gitlab.com/user/repo.git",
			wantOwner: "",
			wantRepo:  "",
			wantOk:    false,
		},
		{
			name:      "Invalid URL",
			remoteURL: "not-a-url",
			wantOwner: "",
			wantRepo:  "",
			wantOk:    false,
		},
		{
			name:      "Empty URL",
			remoteURL: "",
			wantOwner: "",
			wantRepo:  "",
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, ok := ParseGitHubRepo(tt.remoteURL)
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
			if ok != tt.wantOk {
				t.Errorf("ok = %v, want %v", ok, tt.wantOk)
			}
		})
	}
}

func TestBuildGitHubPRURL(t *testing.T) {
	url := BuildGitHubPRURL("Kazuto", "Weave", "main", "feature/test", "## Summary\nTest PR")

	if !strings.HasPrefix(url, "https://github.com/Kazuto/Weave/compare/main...feature/test?") {
		t.Errorf("URL has wrong base: %s", url)
	}
	if !strings.Contains(url, "expand=1") {
		t.Error("URL missing expand=1 parameter")
	}
	if !strings.Contains(url, "body=") {
		t.Error("URL missing body parameter")
	}
}

func TestGetCommitsBetween_NoDifference(t *testing.T) {
	tempDir, cleanup := setupGitRepo(t)
	defer cleanup()

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("init"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "init").Run()
	_ = exec.Command("git", "branch", "-M", "main").Run()

	commits, err := GetCommitsBetween("main", "main")
	if err != nil {
		t.Fatalf("GetCommitsBetween() error: %v", err)
	}

	if commits != "" {
		t.Errorf("GetCommitsBetween() with same branch returned %q, want empty", commits)
	}
}
