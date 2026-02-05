package branch

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

func IsJiraAvailable() bool {
	_, err := exec.LookPath("jira")
	return err == nil
}

type JiraClient struct{}

func NewJiraClient() *JiraClient {
	return &JiraClient{}
}

func (c *JiraClient) IsAvailable() bool {
	return IsJiraAvailable()
}

func (c *JiraClient) GetTicketTitle(ticketID string) (string, error) {
	if !c.IsAvailable() {
		return "", fmt.Errorf("jira CLI not found - please install jira CLI or provide title manually")
	}

	cmd := exec.Command("jira", "issue", "view", ticketID, "--raw")
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			stderr := string(exitError.Stderr)
			if strings.Contains(stderr, "not found") || strings.Contains(stderr, "does not exist") {
				return "", fmt.Errorf("ticket %s not found", ticketID)
			}
			if strings.Contains(stderr, "authentication") || strings.Contains(stderr, "unauthorized") {
				return "", fmt.Errorf("authentication failed - please run 'jira init' to configure credentials")
			}
			return "", fmt.Errorf("failed to fetch ticket: %s", stderr)
		}
		return "", fmt.Errorf("failed to execute jira command: %v", err)
	}

	title, err := c.parseJSONTitle(string(output))
	if err != nil {
		return "", fmt.Errorf("failed to parse ticket title: %v", err)
	}

	if title == "" {
		return "", fmt.Errorf("ticket title is empty")
	}

	return title, nil
}

func (c *JiraClient) parseJSONTitle(output string) (string, error) {
	var ticket struct {
		Fields struct {
			Summary string `json:"summary"`
		} `json:"fields"`
	}

	if err := json.Unmarshal([]byte(output), &ticket); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %v", err)
	}

	return ticket.Fields.Summary, nil
}
