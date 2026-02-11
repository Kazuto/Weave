package pr

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// validateRef checks that a git ref contains only safe characters.
func validateRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("empty git ref")
	}
	for _, c := range ref {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '/' || c == '-' || c == '_' || c == '.') {
			return fmt.Errorf("invalid character %q in git ref %q", string(c), ref)
		}
	}
	return nil
}

func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func DetectBaseBranch() string {
	// Try symbolic-ref for the default branch
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := cmd.Output()
	if err == nil {
		ref := strings.TrimSpace(string(output))
		// refs/remotes/origin/main -> main
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// Try main
	cmd = exec.Command("git", "rev-parse", "--verify", "main")
	if cmd.Run() == nil {
		return "main"
	}

	// Fall back to master
	cmd = exec.Command("git", "rev-parse", "--verify", "master")
	if cmd.Run() == nil {
		return "master"
	}

	return "main"
}

func GetCommitsBetween(base, head string) (string, error) {
	if err := validateRef(base); err != nil {
		return "", err
	}
	if err := validateRef(head); err != nil {
		return "", err
	}
	cmd := exec.Command("git", "log", base+".."+head, "--pretty=format:%h %s") // #nosec G204 -- refs are validated above
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func GetDiffBetween(base, head string) (string, error) {
	if err := validateRef(base); err != nil {
		return "", err
	}
	if err := validateRef(head); err != nil {
		return "", err
	}
	cmd := exec.Command("git", "diff", base+"..."+head) // #nosec G204 -- refs are validated above
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// GetRemoteURL returns the URL of the "origin" remote.
func GetRemoteURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no origin remote configured")
	}
	return strings.TrimSpace(string(output)), nil
}

// ParseGitHubRepo extracts owner and repo from a GitHub remote URL.
// Supports HTTPS (https://github.com/owner/repo.git) and SSH (git@github.com:owner/repo.git).
// Returns empty strings and false if the remote is not a GitHub URL.
func ParseGitHubRepo(remoteURL string) (owner, repo string, ok bool) {
	remoteURL = strings.TrimSpace(remoteURL)

	var path string
	if after, found := strings.CutPrefix(remoteURL, "git@github.com:"); found {
		path = after
	} else if strings.Contains(remoteURL, "github.com/") {
		idx := strings.Index(remoteURL, "github.com/")
		path = remoteURL[idx+len("github.com/"):]
	} else {
		return "", "", false
	}

	path = strings.TrimSuffix(path, ".git")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// BuildGitHubPRURL constructs a GitHub "New Pull Request" URL with the description pre-filled.
func BuildGitHubPRURL(owner, repo, base, head, body string) string {
	u := fmt.Sprintf("https://github.com/%s/%s/compare/%s...%s", owner, repo, base, head)
	params := url.Values{}
	params.Set("expand", "1")
	params.Set("body", body)
	return u + "?" + params.Encode()
}

func GetChangedFilesBetween(base, head string) ([]string, error) {
	if err := validateRef(base); err != nil {
		return nil, err
	}
	if err := validateRef(head); err != nil {
		return nil, err
	}
	cmd := exec.Command("git", "diff", base+"..."+head, "--name-only") // #nosec G204 -- refs are validated above
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, f := range files {
		if f != "" {
			result = append(result, f)
		}
	}
	return result, nil
}
