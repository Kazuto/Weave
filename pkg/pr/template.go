package pr

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func FindPRTemplate() string {
	root := getRepoRoot()
	if root == "" {
		return ""
	}

	paths := []string{
		filepath.Join(root, ".github", "PULL_REQUEST_TEMPLATE.md"),
		filepath.Join(root, ".github", "pull_request_template.md"),
		filepath.Join(root, "PULL_REQUEST_TEMPLATE.md"),
		filepath.Join(root, "pull_request_template.md"),
		filepath.Join(root, "docs", "PULL_REQUEST_TEMPLATE.md"),
		filepath.Join(root, "docs", "pull_request_template.md"),
	}

	for _, p := range paths {
		data, err := os.ReadFile(filepath.Clean(p))
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	return ""
}

func getRepoRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
