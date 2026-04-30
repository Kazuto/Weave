package commit

import (
	"os/exec"
	"strconv"
	"strings"
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
		// Get commits unique to current branch (not in base branch)
		cmd = exec.Command("git", "log", baseBranch+".."+"HEAD", "-n", strconv.Itoa(count), "--pretty=format:%s") // #nosec G204 -- count is an integer
	} else {
		// Get last n commits from current branch
		cmd = exec.Command("git", "log", "-n", strconv.Itoa(count), "--pretty=format:%s") // #nosec G204 -- count is an integer
	}

	output, err := cmd.Output()
	if err != nil {
		// If there are no commits yet (new repo) or base branch doesn't exist, fallback to recent commits
		cmd = exec.Command("git", "log", "-n", strconv.Itoa(count), "--pretty=format:%s")
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
