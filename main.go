package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Kazuto/Weave/internal/branch"
	"github.com/Kazuto/Weave/internal/commit"
	"github.com/Kazuto/Weave/internal/config"
	"github.com/Kazuto/Weave/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "commit":
		runCommit(os.Args[2:])
	case "branch":
		runBranch(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("weave %s\n", version.Version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`weave - Git workflow automation tool

Usage:
  weave <command> [options]

Commands:
  commit      Generate an AI-powered commit message using Ollama
  branch      Generate a branch name from a Jira ticket
  version     Show version information
  help        Show this help message

Run 'weave <command> --help' for more information on a command.`)
}

func runCommit(args []string) {
	fs := flag.NewFlagSet("commit", flag.ExitOnError)
	staged := fs.Bool("staged", true, "Use staged changes (default: true)")
	execute := fs.Bool("execute", false, "Execute the commit after generating message")
	fs.Parse(args)

	if !commit.IsGitAvailable() {
		fmt.Fprintln(os.Stderr, "Error: git is not installed or not in PATH")
		os.Exit(1)
	}

	if !commit.IsGitRepository() {
		fmt.Fprintln(os.Stderr, "Error: not a git repository")
		os.Exit(1)
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	diff, err := commit.GetDiff(*staged)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting diff: %v\n", err)
		os.Exit(1)
	}

	if diff == "" {
		if *staged {
			fmt.Fprintln(os.Stderr, "Error: no staged changes found. Stage changes with 'git add' first.")
		} else {
			fmt.Fprintln(os.Stderr, "Error: no changes found.")
		}
		os.Exit(1)
	}

	files, err := commit.GetChangedFiles(*staged)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting changed files: %v\n", err)
		os.Exit(1)
	}

	generator := commit.NewGenerator(cfg.Commit)
	if err := generator.CheckOllama(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generating commit message...")
	message, err := generator.Generate(diff, files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating commit message: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nGenerated commit message:\n%s\n\n", message)

	if *execute {
		if err := commit.Commit(message); err != nil {
			fmt.Fprintf(os.Stderr, "Error committing: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Commit created successfully!")
	} else {
		fmt.Println("Run with --execute to create the commit, or copy the message above.")
	}
}

func runBranch(args []string) {
	fs := flag.NewFlagSet("branch", flag.ExitOnError)
	branchType := fs.String("type", "", "Branch type (feature, hotfix, refactor, support)")
	title := fs.String("title", "", "Custom title (skips Jira lookup)")
	fs.Parse(args)

	if !commit.IsGitAvailable() {
		fmt.Fprintln(os.Stderr, "Error: git is not installed or not in PATH")
		os.Exit(1)
	}

	if !commit.IsGitRepository() {
		fmt.Fprintln(os.Stderr, "Error: not a git repository")
		os.Exit(1)
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "Error: ticket ID required")
		fmt.Fprintln(os.Stderr, "\nUsage: weave branch <ticket-id> [--type <type>] [--title <title>]")
		os.Exit(1)
	}
	ticketID := strings.ToUpper(remaining[0])

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	selectedType := cfg.Branch.DefaultType
	if *branchType != "" {
		selectedType = *branchType
	} else {
		selectedType = promptBranchType(cfg.Branch.Types, cfg.Branch.DefaultType)
	}

	var ticketTitle string
	if *title != "" {
		ticketTitle = *title
	} else {
		if !branch.IsJiraAvailable() {
			fmt.Fprintln(os.Stderr, "Error: jira CLI is not installed or not in PATH")
			fmt.Fprintln(os.Stderr, "Install it from: https://github.com/ankitpokhrel/jira-cli")
			fmt.Fprintln(os.Stderr, "\nAlternatively, provide a title with --title flag")
			os.Exit(1)
		}

		fmt.Printf("Fetching ticket %s from Jira...\n", ticketID)
		jiraClient := branch.NewJiraClient()
		ticketTitle, err = jiraClient.GetTicketTitle(ticketID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching ticket: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Title: %s\n\n", ticketTitle)
	}

	generator := branch.NewGenerator(cfg.Branch)
	branchName := generator.GenerateName(branch.BranchInfo{
		Type:     generator.GetBranchType(selectedType),
		TicketID: ticketID,
		Title:    ticketTitle,
	})

	if err := generator.ValidateName(branchName); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid branch name: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated branch name:\n%s\n\n", branchName)
	fmt.Printf("Create branch with:\n  git checkout -b %s\n", branchName)
}

func promptBranchType(types map[string]string, defaultType string) string {
	fmt.Println("Select branch type:")

	typeList := make([]string, 0, len(types))
	for key := range types {
		typeList = append(typeList, key)
	}

	for i, t := range typeList {
		marker := ""
		if t == defaultType {
			marker = " (default)"
		}
		fmt.Printf("  %d. %s%s\n", i+1, t, marker)
	}

	fmt.Print("\nEnter number or type name [" + defaultType + "]: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultType
	}

	var idx int
	if _, err := fmt.Sscanf(input, "%d", &idx); err == nil {
		if idx >= 1 && idx <= len(typeList) {
			return typeList[idx-1]
		}
	}

	if _, ok := types[input]; ok {
		return input
	}

	fmt.Printf("Invalid selection, using default: %s\n", defaultType)
	return defaultType
}

func loadConfig() (*config.Config, error) {
	manager := config.NewConfigManager()

	if err := manager.EnsureExists(); err != nil {
		return nil, err
	}

	cfg, err := manager.Load()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
