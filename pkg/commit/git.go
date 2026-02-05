package commit

import (
	"os/exec"
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
	cmd := exec.Command("git", "commit", "-m", message)
	return cmd.Run()
}
