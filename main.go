package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Kazuto/Weave/pkg/branch"
	"github.com/Kazuto/Weave/pkg/commit"
	"github.com/Kazuto/Weave/pkg/config"
	"github.com/Kazuto/Weave/pkg/pr"
	"github.com/Kazuto/Weave/pkg/spinner"
	"github.com/Kazuto/Weave/pkg/version"
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
	case "pr":
		runPR(os.Args[2:])
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
  pr          Generate an AI-powered pull request description
  version     Show version information
  help        Show this help message

Run 'weave <command> --help' for more information on a command.`)
}

func runCommit(args []string) {
	fs := flag.NewFlagSet("commit", flag.ExitOnError)
	staged := fs.Bool("staged", true, "Use staged changes (default: true)")
	autoCommit := fs.Bool("y", false, "Automatically commit without prompting")
	_ = fs.Parse(args) // ExitOnError handles errors

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

	generator := commit.NewGenerator(cfg.Commit)

	// Check Ollama connection
	spin := spinner.New("Checking Ollama connection")
	spin.Start()
	connOk := generator.CheckConnection()
	spin.Stop(connOk)
	if !connOk {
		fmt.Fprintf(os.Stderr, "   Cannot connect to Ollama at %s\n", cfg.Commit.Ollama.Host)
		os.Exit(1)
	}

	// Check model availability
	spin = spinner.New(fmt.Sprintf("Checking if model '%s' is available", cfg.Commit.Ollama.Model))
	spin.Start()
	modelOk := generator.CheckModel()
	spin.Stop(modelOk)
	if !modelOk {
		fmt.Fprintf(os.Stderr, "   Model '%s' is not available\n", cfg.Commit.Ollama.Model)
		os.Exit(1)
	}

	// Analyze changes
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

	fmt.Printf("📝 Found changes in %d file(s)\n", len(files))

	// Generate commit message
	spin = spinner.New(fmt.Sprintf("Generating commit message using %s", cfg.Commit.Ollama.Model))
	spin.Start()
	message, err := generator.Generate(diff, files)
	spin.Stop(err == nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Generated commit message:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(message)
	fmt.Println(strings.Repeat("=", 60) + "\n")

	if *autoCommit {
		if err := commit.Commit(message); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error committing: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Committed successfully!")
		return
	}

	fmt.Print("Use this commit message? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		if err := commit.Commit(message); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error committing: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Committed successfully!")
	} else {
		if err := copyToClipboard(message); err != nil {
			fmt.Println("Commit cancelled. Message not copied to clipboard.")
		} else {
			fmt.Println("📋 Commit message copied to clipboard")
		}
	}
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return fmt.Errorf("unsupported platform")
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func openInBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Run()
	case "linux":
		return exec.Command("xdg-open", url).Run()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func runBranch(args []string) {
	fs := flag.NewFlagSet("branch", flag.ExitOnError)
	branchType := fs.String("type", "", "Branch type (feature, hotfix, refactor, support)")
	title := fs.String("title", "", "Custom title (skips Jira lookup)")
	_ = fs.Parse(args) // ExitOnError handles errors

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

	var selectedType string
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

func runPR(args []string) {
	fs := flag.NewFlagSet("pr", flag.ExitOnError)
	base := fs.String("base", "", "Base branch to compare against (default: auto-detect)")
	autoOpen := fs.Bool("y", false, "Automatically open in browser without prompting")
	_ = fs.Parse(args) // ExitOnError handles errors

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

	generator := pr.NewGenerator(cfg.PR, cfg.Commit.Ollama)

	// Check Ollama connection
	spin := spinner.New("Checking Ollama connection")
	spin.Start()
	connOk := generator.CheckConnection()
	spin.Stop(connOk)
	if !connOk {
		fmt.Fprintf(os.Stderr, "   Cannot connect to Ollama at %s\n", cfg.Commit.Ollama.Host)
		os.Exit(1)
	}

	// Check model availability
	spin = spinner.New(fmt.Sprintf("Checking if model '%s' is available", cfg.Commit.Ollama.Model))
	spin.Start()
	modelOk := generator.CheckModel()
	spin.Stop(modelOk)
	if !modelOk {
		fmt.Fprintf(os.Stderr, "   Model '%s' is not available\n", cfg.Commit.Ollama.Model)
		os.Exit(1)
	}

	// Get current branch
	currentBranch, err := pr.GetCurrentBranch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current branch: %v\n", err)
		os.Exit(1)
	}

	// Determine base branch
	baseBranch := *base
	if baseBranch == "" {
		baseBranch = cfg.PR.DefaultBase
	}
	if baseBranch == "" {
		baseBranch = pr.DetectBaseBranch()
	}

	if currentBranch == baseBranch {
		fmt.Fprintf(os.Stderr, "Error: current branch '%s' is the same as base branch '%s'\n", currentBranch, baseBranch)
		os.Exit(1)
	}

	fmt.Printf("Comparing %s → %s\n", currentBranch, baseBranch)

	// Get commits between branches
	commits, err := pr.GetCommitsBetween(baseBranch, currentBranch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting commits: %v\n", err)
		os.Exit(1)
	}

	if commits == "" {
		fmt.Fprintln(os.Stderr, "Error: no commits found between branches. Nothing to describe.")
		os.Exit(1)
	}

	// Get diff and changed files
	diff, err := pr.GetDiffBetween(baseBranch, currentBranch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting diff: %v\n", err)
		os.Exit(1)
	}

	files, err := pr.GetChangedFilesBetween(baseBranch, currentBranch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting changed files: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📝 Found %d commit(s) changing %d file(s)\n", len(strings.Split(commits, "\n")), len(files))

	// Find PR template
	template := pr.FindPRTemplate()
	if template != "" {
		fmt.Println("📋 Using PR template from repository")
	}

	// Generate PR description
	ctx := pr.PRContext{
		Branch:   currentBranch,
		Base:     baseBranch,
		Commits:  commits,
		Files:    strings.Join(files, "\n"),
		Diff:     diff,
		Template: template,
	}

	spin = spinner.New(fmt.Sprintf("Generating PR description using %s", cfg.Commit.Ollama.Model))
	spin.Start()
	description, err := generator.Generate(ctx)
	spin.Stop(err == nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Generated PR description:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(description)
	fmt.Println(strings.Repeat("=", 60) + "\n")

	// Determine if we can open in browser
	canOpenBrowser := false
	var prURL string
	remoteURL, err := pr.GetRemoteURL()
	if err == nil {
		owner, repo, ok := pr.ParseGitHubRepo(remoteURL)
		if ok {
			prURL = pr.BuildGitHubPRURL(owner, repo, baseBranch, currentBranch, description)
			canOpenBrowser = true
		}
	}

	if *autoOpen {
		if canOpenBrowser {
			if err := openInBrowser(prURL); err != nil {
				fmt.Fprintf(os.Stderr, "Error opening browser: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Opened PR creation page in browser!")
		} else {
			if err := copyToClipboard(description); err != nil {
				fmt.Fprintf(os.Stderr, "Error copying to clipboard: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("📋 No GitHub remote found. PR description copied to clipboard!")
		}
		return
	}

	// Interactive menu
	if canOpenBrowser {
		fmt.Println("  1. Open in browser")
	}
	fmt.Println("  2. Copy to clipboard")
	fmt.Println("  3. Do nothing")

	fmt.Print("\nSelect an option: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	switch response {
	case "1":
		if !canOpenBrowser {
			fmt.Fprintln(os.Stderr, "No GitHub remote found.")
			break
		}
		if err := openInBrowser(prURL); err != nil {
			fmt.Fprintf(os.Stderr, "Error opening browser: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Opened PR creation page in browser!")
	case "2":
		if err := copyToClipboard(description); err != nil {
			fmt.Fprintf(os.Stderr, "Error copying to clipboard: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("📋 PR description copied to clipboard!")
	}
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
