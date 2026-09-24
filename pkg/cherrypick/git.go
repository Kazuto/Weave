package cherrypick

import (
	"fmt"
	"os"
	"os/exec"
)

// ExecuteCherrypick performs the full process of updating the base branch, creating a new branch, and cherry-picking commits.
func ExecuteCherrypick(baseBranch, prodBranch string, commits []string) error {
	// Fetch and pull latest changes for base branch
	
	// git checkout baseBranch
	checkoutBase := exec.Command("git", "checkout", baseBranch)
	checkoutBase.Stdout = os.Stdout
	checkoutBase.Stderr = os.Stderr
	if err := checkoutBase.Run(); err != nil {
		return fmt.Errorf("Failed to checkout base branch %s: %w", baseBranch, err)
	}

	// git pull origin baseBranch (assuming origin)
	pullBase := exec.Command("git", "pull", "origin", baseBranch)
	pullBase.Stdout = os.Stdout
	pullBase.Stderr = os.Stderr
	if err := pullBase.Run(); err != nil {
		return fmt.Errorf("Failed to pull latest changes for %s: %w", baseBranch, err)
	}

	// Create prod branch from updated base
	args := []string{"checkout", "-b", prodBranch, baseBranch}
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to create branch %s from %s: %w", prodBranch, baseBranch, err)
	}

	return CherryPickCommits(commits)
}

// CreateProdBranch creates a new branch named {TICKET-ID}-prod starting from the base branch.
func CreateProdBranch(ticketID, baseBranch string) error {
	branchName := fmt.Sprintf("%s-prod", ticketID)
	// git checkout -b {branchName} {baseBranch}
	args := []string{"checkout", "-b", branchName, baseBranch}
	cmd := exec.Command("git", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to create branch %s from %s: %s (%w)", branchName, baseBranch, string(output), err)
	}
	return nil
}

// CherryPickCommits cherry-picks the provided list of commit SHAs.
func CherryPickCommits(commits []string) error {
	if len(commits) == 0 {
		return nil
	}

	for _, sha := range commits {
		args := append([]string{"cherry-pick"}, sha)
		cmd := exec.Command("git", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Cherry-pick failed at commit %s: %w", sha, err)
		}
	}
	return nil
}

// DetectBaseBranch tries to find a common production branch.
func DetectBaseBranch() string {
	branches := []string{"main", "master", "prod"}
	for _, b := range branches {
		if branchExists(b) {
			return b
		}
	}
	return ""
}

func branchExists(branch string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "refs/heads/"+branch)
	return cmd.Run() == nil
}
