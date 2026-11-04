package cherrypick

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}

	return strings.TrimSpace(string(out))
}

func commitFile(t *testing.T, dir, name string) string {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", name)
	runGit(t, dir, "commit", "-m", name)

	return runGit(t, dir, "rev-parse", "HEAD")
}

// setupStaleStaging creates a clone on main whose staging branch is one commit behind origin,
// changes into it, and returns the clone's path and the SHA of origin's staging tip.
func setupStaleStaging(t *testing.T) (string, string) {
	t.Helper()

	// Isolate from the user's git config, e.g. commit signing.
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	local := filepath.Join(root, "local")
	other := filepath.Join(root, "other")

	runGit(t, root, "init", "--bare", remote)
	runGit(t, root, "clone", remote, local)
	runGit(t, local, "symbolic-ref", "HEAD", "refs/heads/main")
	commitFile(t, local, "base.txt")
	runGit(t, local, "push", "origin", "main")
	runGit(t, local, "branch", "staging")
	runGit(t, local, "push", "origin", "staging")

	runGit(t, root, "clone", "--branch", "staging", remote, other)
	tip := commitFile(t, other, "staging.txt")
	runGit(t, other, "push", "origin", "staging")

	t.Chdir(local)

	return local, tip
}

func TestUpdateSourceBranch(t *testing.T) {
	tests := []struct {
		name        string
		prepare     func(t *testing.T, local string)
		source      string
		ref         string
		wantCurrent string
	}{
		{
			name:        "not checked out",
			source:      "staging",
			ref:         "refs/heads/staging",
			wantCurrent: "main",
		},
		{
			name: "checked out",
			prepare: func(t *testing.T, local string) {
				runGit(t, local, "checkout", "staging")
			},
			source:      "staging",
			ref:         "refs/heads/staging",
			wantCurrent: "staging",
		},
		{
			name: "missing locally",
			prepare: func(t *testing.T, local string) {
				runGit(t, local, "branch", "-D", "staging")
			},
			source:      "staging",
			ref:         "refs/heads/staging",
			wantCurrent: "main",
		},
		{
			name:        "remote-tracking ref",
			source:      "origin/staging",
			ref:         "refs/remotes/origin/staging",
			wantCurrent: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			local, tip := setupStaleStaging(t)
			if tt.prepare != nil {
				tt.prepare(t, local)
			}

			if err := UpdateSourceBranch(tt.source); err != nil {
				t.Fatalf("UpdateSourceBranch() error = %v", err)
			}

			if got := runGit(t, local, "rev-parse", tt.ref); got != tip {
				t.Errorf("%s = %s, want %s", tt.ref, got, tip)
			}
			if got := runGit(t, local, "rev-parse", "--abbrev-ref", "HEAD"); got != tt.wantCurrent {
				t.Errorf("current branch = %s, want %s", got, tt.wantCurrent)
			}
		})
	}
}

func TestUpdateSourceBranchKeepsDivergedBranch(t *testing.T) {
	for _, checkedOut := range []bool{false, true} {
		name := "not checked out"
		if checkedOut {
			name = "checked out"
		}

		t.Run(name, func(t *testing.T) {
			local, _ := setupStaleStaging(t)

			runGit(t, local, "checkout", "staging")
			localTip := commitFile(t, local, "local.txt")
			if !checkedOut {
				runGit(t, local, "checkout", "main")
			}

			if err := UpdateSourceBranch("staging"); err == nil {
				t.Error("UpdateSourceBranch() error = nil, want error for diverged branch")
			}

			if got := runGit(t, local, "rev-parse", "refs/heads/staging"); got != localTip {
				t.Errorf("staging = %s, want it left at %s", got, localTip)
			}
		})
	}
}

func TestUpdateSourceBranchRejectsInvalidRef(t *testing.T) {
	for _, ref := range []string{"", "-staging", "staging:main", "+staging"} {
		if err := UpdateSourceBranch(ref); err == nil {
			t.Errorf("UpdateSourceBranch(%q) error = nil, want error", ref)
		}
	}
}
