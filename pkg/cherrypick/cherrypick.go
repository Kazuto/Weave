package cherrypick

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var ticketRegex = regexp.MustCompile(`([A-Z]+-[0-9]+)`)

// GetCurrentBranch returns the current git branch name.
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ExtractTicketID attempts to find a Jira ticket ID in the given string.
func ExtractTicketID(input string) string {
	match := ticketRegex.FindString(input)
	return match
}

// FindTicketCommits finds all commit SHAs on the staging branch that contain the ticket ID in their message.
func FindTicketCommits(stagingBranch, ticketID string) ([]string, error) {
	return GetCommitsByTicket(stagingBranch, ticketID)
}

// GetCommitsByTicket finds all commit SHAs on the staging branch that contain the ticket ID in their message.
func GetCommitsByTicket(stagingBranch, ticketID string) ([]string, error) {
	// git log stagingBranch --grep="TICKET-ID" --format=%H
	args := []string{"log", stagingBranch, fmt.Sprintf("--grep=%s", ticketID), "--format=%H"}
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Failed to find commits for ticket %s on branch %s: %w", ticketID, stagingBranch, err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var commits []string
	for _, line := range lines {
		if line != "" {
			commits = append(commits, line)
		}
	}

	// Git log returns most recent first, but cherry-pick needs them in chronological order.
	for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
		commits[i], commits[j] = commits[j], commits[i]
	}

	return commits, nil
}
