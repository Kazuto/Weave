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
	shas, _, err := GetCommitsWithMessages(stagingBranch, ticketID)
	return shas, err
}

// GetCommitsByTicket finds all commit SHAs on the staging branch that contain the ticket ID in their message.
func GetCommitsByTicket(stagingBranch, ticketID string) ([]string, error) {
	return FindTicketCommits(stagingBranch, ticketID)
}

// GetCommitsWithMessages finds all commit SHAs and their messages on the staging branch that contain the ticket ID.
func GetCommitsWithMessages(stagingBranch, ticketID string) ([]string, []string, error) {
	// git log stagingBranch --grep="TICKET-ID" --format="%H|%s"
	args := []string{"log", stagingBranch, fmt.Sprintf("--grep=%s", ticketID), "--format=%H|%s"}
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to find commits for ticket %s on branch %s: %w", ticketID, stagingBranch, err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var shas []string
	var messages []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			shas = append(shas, parts[0])
			messages = append(messages, parts[1])
		}
	}

	// No need to reverse; git log is newest to oldest by default.
	// We remove the reversal loop to maintain newest-to-oldest order.
	return shas, messages, nil
}

// GetCommitsBetween branches finds all commit SHAs and their messages that are on the staging branch but not on the base branch, ignoring commits that are already present on the base branch (cherry-picks).
func GetCommitsBetween(baseBranch, stagingBranch string, limit int) ([]string, []string, error) {
	// git log baseBranch...stagingBranch --cherry-pick --right-only --no-merges --format="%H|%s"
	args := []string{"log", fmt.Sprintf("%s...%s", baseBranch, stagingBranch), "--cherry-pick", "--right-only", "--no-merges", "--format=%H|%s"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("-n %d", limit))
	}
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to find commits between %s and %s: %w", baseBranch, stagingBranch, err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var shas []string
	var messages []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			shas = append(shas, parts[0])
			messages = append(messages, parts[1])
		}
	}

	// No need to reverse; git log is newest to oldest by default.
	// We remove the reversal loop to maintain newest-to-oldest order.
	return shas, messages, nil
}
