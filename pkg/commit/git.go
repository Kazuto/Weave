package commit

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"unicode"
)

func IsGitAvailable() bool {
	cmd := exec.Command("git", "--version")
	return cmd.Run() == nil
}

func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

func GetDiff(staged bool) (string, error) {
	var cmd *exec.Cmd
	if staged {
		cmd = exec.Command("git", "diff", "--cached")
	} else {
		cmd = exec.Command("git", "diff")
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func GetChangedFiles(staged bool) ([]string, error) {
	var cmd *exec.Cmd
	if staged {
		cmd = exec.Command("git", "diff", "--cached", "--name-only")
	} else {
		cmd = exec.Command("git", "diff", "--name-only")
	}

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

func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message) // #nosec G204 -- message is passed as a separate argument, not interpreted by shell
	return cmd.Run()
}

// isSafeChar checks if a character is alphanumeric or in the extra set
func isSafeChar(c rune, extra string) bool {
	if unicode.IsLetter(c) || unicode.IsDigit(c) {
		return true
	}

	for _, e := range extra {
		if c == e {
			return true
		}
	}

	return false
}

// validateRef checks that a git ref contains only safe characters
func validateRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("empty git ref")
	}

	for _, c := range ref {
		if !isSafeChar(c, "/-_.") {
			return fmt.Errorf("invalid character %q in git ref %q", c, ref)
		}
	}

	return nil
}

func GetRecentCommits(count int) ([]string, error) {
	return GetRecentCommitsFromBranch(count, "")
}

func GetRecentCommitsFromBranch(count int, baseBranch string) ([]string, error) {
	if count <= 0 {
		return []string{}, nil
	}

	// Auto-detect base branch if not specified
	if baseBranch == "" {
		baseBranch = detectBaseBranch()
	}

	var cmd *exec.Cmd
	if baseBranch != "" {
		// Validate the base branch to prevent command injection
		if err := validateRef(baseBranch); err != nil {
			// If validation fails, fallback to recent commits
			baseBranch = ""
		}
	}

	if baseBranch != "" {
		// Get commits unique to current branch (not in base branch)
		cmd = exec.Command("git", "log", baseBranch+".."+"HEAD", "-n", strconv.Itoa(count), "--pretty=format:%s") // #nosec G204 -- baseBranch is validated above
	} else {
		// Get last n commits from current branch
		cmd = exec.Command("git", "log", "-n", strconv.Itoa(count), "--pretty=format:%s") // #nosec G204 -- count is an integer
	}

	output, err := cmd.Output()
	if err != nil {
		// If there are no commits yet (new repo) or base branch doesn't exist, fallback to recent commits
		cmd = exec.Command("git", "log", "-n", strconv.Itoa(count), "--pretty=format:%s") // #nosec G204 -- count is an integer
		output, err = cmd.Output()
		if err != nil {
			return []string{}, nil
		}
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, c := range commits {
		if c != "" {
			result = append(result, c)
		}
	}
	return result, nil
}

func detectBaseBranch() string {
	// Try main
	cmd := exec.Command("git", "rev-parse", "--verify", "main")
	if cmd.Run() == nil {
		return "main"
	}

	// Try master
	cmd = exec.Command("git", "rev-parse", "--verify", "master")
	if cmd.Run() == nil {
		return "master"
	}

	// No base branch found
	return ""
}
