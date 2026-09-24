package cherrypick

import (
	"fmt"
	"os/exec"
)

// ExecuteCherrypick performs the full process of updating the base branch, creating a new branch, and cherry-picking commits.
func ExecuteCherrypick(baseBranch, prodBranch string, commits []string) error {
	// Fetch and pull latest changes for base branch

	// git checkout baseBranch
	checkoutBase := exec.Command("git", "checkout", baseBranch)
	if output, err := checkoutBase.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to checkout base branch %s: %s (%w)", baseBranch, string(output), err)
	}

	// git pull origin baseBranch (assuming origin)
	pullBase := exec.Command("git", "pull", "origin", baseBranch)
	if output, err := pullBase.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to pull latest changes for %s: %s (%w)", baseBranch, string(output), err)
	}

	// Create prod branch from updated base
	args := []string{"checkout", "-b", prodBranch, baseBranch}
	cmd := exec.Command("git", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to create branch %s from %s: %s (%w)", prodBranch, baseBranch, string(output), err)
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

	args := append([]string{"cherry-pick"}, commits...)
	cmd := exec.Command("git", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Cherry-pick failed: %s (%w)", string(output), err)
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
