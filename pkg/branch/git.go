package branch

import (
	"fmt"
	"os/exec"
)

func CheckoutBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName) // #nosec G204 -- branchName is validated by ValidateName
	if err := cmd.Run(); err != nil {
		// Check if branch already exists
		checkCmd := exec.Command("git", "rev-parse", "--verify", branchName)
		if checkCmd.Run() == nil {
			return fmt.Errorf("branch '%s' already exists", branchName)
		}
		return fmt.Errorf("failed to create branch: %v", err)
	}
	return nil
}
