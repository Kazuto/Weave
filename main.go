package main

import (
	"flag"
	"fmt"
	"os"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Kazuto/Weave/pkg/branch"
	"github.com/Kazuto/Weave/pkg/commit"
	"github.com/Kazuto/Weave/pkg/config"
	"github.com/Kazuto/Weave/pkg/pr"
	"github.com/Kazuto/Weave/pkg/spinner"
	"github.com/Kazuto/Weave/pkg/ui"
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
		fmt.Fprintf(os.Stderr, "Unknown command: %q\n\n", os.Args[1]) // #nosec G705 -- CLI stderr output, not web response
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
	base := fs.String("base", "", "Base branch for commit reference filtering (default: config or auto-detect)")
	fs.StringVar(base, "b", "", "Base branch (shorthand)")
	_ = fs.Parse(args) // ExitOnError handles errors

	if !commit.IsGitAvailable() {
		fmt.Fprintln(os.Stderr, ui.FormatError("Git is not installed or not in PATH"))
		os.Exit(1)
	}

	if !commit.IsGitRepository() {
		fmt.Fprintln(os.Stderr, ui.FormatError("Not a git repository"))
		os.Exit(1)
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error loading config: %v", err)))
		os.Exit(1)
	}

	// Override reference_branch if -b flag is provided
	if *base != "" {
		cfg.Commit.ReferenceBranch = *base
	}

	generator, err := commit.NewGenerator(cfg.Commit, cfg.LLM)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error creating generator: %v", err)))
		os.Exit(1)
	}

	providerType := cfg.LLM.Provider
	if providerType == "" {
		providerType = "ollama"
	}

	// Check provider connection
	spin := spinner.New(fmt.Sprintf("Checking %s connection", providerType))
	spin.Start()
	connOk := generator.CheckConnection()
	spin.Stop(connOk)
	if !connOk {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Cannot connect to %s provider", providerType)))
		os.Exit(1)
	}

	// Check model availability
	spin = spinner.New("Checking if model is available")
	spin.Start()
	modelOk := generator.CheckModel()
	spin.Stop(modelOk)
	if !modelOk {
		fmt.Fprintln(os.Stderr, ui.FormatError("Model is not available"))
		os.Exit(1)
	}

	// Analyze changes
	diff, err := commit.GetDiff(*staged)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error getting diff: %v", err)))
		os.Exit(1)
	}

	if diff == "" {
		if *staged {
			fmt.Fprintln(os.Stderr, ui.FormatError("No staged changes found. Stage changes with 'git add' first"))
		} else {
			fmt.Fprintln(os.Stderr, ui.FormatError("No changes found"))
		}
		os.Exit(1)
	}

	files, err := commit.GetChangedFiles(*staged)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error getting changed files: %v", err)))
		os.Exit(1)
	}

	fmt.Println(ui.FormatInfo(fmt.Sprintf("Found changes in %d file(s)", len(files))))

	// Generate commit message
	modelName := cfg.LLM.Ollama.Model
	if cfg.LLM.Provider == "openai" {
		modelName = cfg.LLM.OpenAI.Model
	}
	spin = spinner.New(fmt.Sprintf("Generating commit message using %s", modelName))
	spin.Start()
	message, err := generator.Generate(diff, files)
	spin.Stop(err == nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(err.Error()))
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(ui.FormatHeader("Generated commit message:"))
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println(message)
	fmt.Println(strings.Repeat("─", 60) + "\n")

	if *autoCommit {
		if err := commit.Commit(message); err != nil {
			fmt.Fprintf(os.Stderr, "Error committing: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(ui.FormatSuccess("Committed successfully!"))
		return
	}

	confirmed, err := ui.Confirm("Use this commit message?", false)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(err.Error()))
		os.Exit(1)
	}

	if confirmed {
		if err := commit.Commit(message); err != nil {
			fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error committing: %v", err)))
			os.Exit(1)
		}
		fmt.Println(ui.FormatSuccess("Committed successfully!"))
	} else {
		if err := copyToClipboard(message); err != nil {
			fmt.Println(ui.FormatInfo("Commit cancelled. Message not copied to clipboard"))
		} else {
			fmt.Println(ui.FormatInfo("Commit message copied to clipboard"))
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

func openInBrowser(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return fmt.Errorf("invalid URL: %q", rawURL)
	}

	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", parsed.String()).Run() // #nosec G204 -- URL is validated above
	case "linux":
		return exec.Command("xdg-open", parsed.String()).Run() // #nosec G204 -- URL is validated above
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", parsed.String()).Run() // #nosec G204 -- URL is validated above
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func runBranch(args []string) {
	fs := flag.NewFlagSet("branch", flag.ExitOnError)
	branchType := fs.String("type", "", "Branch type (feature, hotfix, refactor, support)")
	title := fs.String("title", "", "Custom title (skips Jira lookup)")
	autoCheckout := fs.Bool("y", false, "Automatically switch to the new branch without prompting")
	_ = fs.Parse(args) // ExitOnError handles errors

	if !commit.IsGitAvailable() {
		fmt.Fprintln(os.Stderr, ui.FormatError("Git is not installed or not in PATH"))
		os.Exit(1)
	}

	if !commit.IsGitRepository() {
		fmt.Fprintln(os.Stderr, ui.FormatError("Not a git repository"))
		os.Exit(1)
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, ui.FormatError("Ticket ID required"))
		fmt.Fprintln(os.Stderr, "Usage: weave branch <ticket-id> [--type <type>] [--title <title>]")
		os.Exit(1)
	}
	ticketID := strings.ToUpper(remaining[0])

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error loading config: %v", err)))
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
			fmt.Fprintln(os.Stderr, ui.FormatError("Jira CLI is not installed or not in PATH"))
			fmt.Fprintln(os.Stderr, "Install it from: https://github.com/ankitpokhrel/jira-cli")
			fmt.Fprintln(os.Stderr, "Alternatively, provide a title with --title flag")
			os.Exit(1)
		}

		fmt.Println(ui.FormatInfo(fmt.Sprintf("Fetching ticket %s from Jira...", ticketID)))
		jiraClient := branch.NewJiraClient()
		ticketTitle, err = jiraClient.GetTicketTitle(ticketID)
		if err != nil {
			fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error fetching ticket: %v", err)))
			os.Exit(1)
		}
		fmt.Printf("\n%s\n\n", ui.FormatInfo(fmt.Sprintf("Title: %s", ticketTitle)))
	}

	generator := branch.NewGenerator(cfg.Branch)
	branchName := generator.GenerateName(branch.BranchInfo{
		Type:     generator.GetBranchType(selectedType),
		TicketID: ticketID,
		Title:    ticketTitle,
	})

	if err := generator.ValidateName(branchName); err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Invalid branch name: %v", err)))
		os.Exit(1)
	}

	fmt.Println(ui.FormatHeader("Generated branch name:"))
	fmt.Printf("%s\n\n", ui.Style(branchName, "--foreground", "212", "--bold"))

	if *autoCheckout {
		if err := branch.CheckoutBranch(branchName); err != nil {
			fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error switching to branch: %v", err)))
			os.Exit(1)
		}
		fmt.Println(ui.FormatSuccess("Switched to new branch successfully!"))
		return
	}

	confirmed, err := ui.Confirm("Switch to this branch?", false)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(err.Error()))
		os.Exit(1)
	}

	if confirmed {
		if err := branch.CheckoutBranch(branchName); err != nil {
			fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error switching to branch: %v", err)))
			os.Exit(1)
		}
		fmt.Println(ui.FormatSuccess("Switched to new branch successfully!"))
	} else {
		if err := copyToClipboard(branchName); err != nil {
			fmt.Println(ui.FormatInfo("Branch not created. Name not copied to clipboard"))
		} else {
			fmt.Println(ui.FormatInfo("Branch name copied to clipboard"))
		}
	}
}

func runPR(args []string) {
	fs := flag.NewFlagSet("pr", flag.ExitOnError)
	base := fs.String("base", "", "Base branch to compare against (default: auto-detect)")
	fs.StringVar(base, "b", "", "Base branch (shorthand)")
	remote := fs.String("remote", "", "Target remote for PR (default: origin)")
	fs.StringVar(remote, "r", "", "Target remote (shorthand)")
	autoOpen := fs.Bool("y", false, "Automatically open in browser without prompting")
	_ = fs.Parse(args) // ExitOnError handles errors

	if !commit.IsGitAvailable() {
		fmt.Fprintln(os.Stderr, ui.FormatError("Git is not installed or not in PATH"))
		os.Exit(1)
	}

	if !commit.IsGitRepository() {
		fmt.Fprintln(os.Stderr, ui.FormatError("Not a git repository"))
		os.Exit(1)
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error loading config: %v", err)))
		os.Exit(1)
	}

	generator, err := pr.NewGenerator(cfg.PR, cfg.LLM)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error creating generator: %v", err)))
		os.Exit(1)
	}

	providerType := cfg.LLM.Provider
	if providerType == "" {
		providerType = "ollama"
	}

	// Check provider connection
	spin := spinner.New(fmt.Sprintf("Checking %s connection", providerType))
	spin.Start()
	connOk := generator.CheckConnection()
	spin.Stop(connOk)
	if !connOk {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Cannot connect to %s provider", providerType)))
		os.Exit(1)
	}

	// Check model availability
	spin = spinner.New("Checking if model is available")
	spin.Start()
	modelOk := generator.CheckModel()
	spin.Stop(modelOk)
	if !modelOk {
		fmt.Fprintln(os.Stderr, ui.FormatError("Model is not available"))
		os.Exit(1)
	}

	// Get current branch
	currentBranch, err := pr.GetCurrentBranch()
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error getting current branch: %v", err)))
		os.Exit(1)
	}

	// Determine target remote
	targetRemote := *remote
	if targetRemote == "" {
		targetRemote = cfg.PR.DefaultRemote
	}
	if targetRemote == "" {
		targetRemote = "origin"
	}

	// Determine base branch
	baseBranch := *base
	if baseBranch == "" {
		baseBranch = cfg.PR.DefaultBase
	}
	if baseBranch == "" {
		baseBranch = pr.DetectBaseBranch(targetRemote)
	}

	if currentBranch == baseBranch {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Current branch '%s' is the same as base branch '%s'", currentBranch, baseBranch)))
		os.Exit(1)
	}

	fmt.Println(ui.FormatInfo(fmt.Sprintf("Comparing %s → %s", currentBranch, baseBranch)))

	// Get commits between branches
	commits, err := pr.GetCommitsBetween(baseBranch, currentBranch)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error getting commits: %v", err)))
		os.Exit(1)
	}

	if commits == "" {
		fmt.Fprintln(os.Stderr, ui.FormatError("No commits found between branches. Nothing to describe"))
		os.Exit(1)
	}

	// Get diff and changed files
	diff, err := pr.GetDiffBetween(baseBranch, currentBranch)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error getting diff: %v", err)))
		os.Exit(1)
	}

	files, err := pr.GetChangedFilesBetween(baseBranch, currentBranch)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error getting changed files: %v", err)))
		os.Exit(1)
	}

	fmt.Println(ui.FormatInfo(fmt.Sprintf("Found %d commit(s) changing %d file(s)", len(strings.Split(commits, "\n")), len(files))))

	// Find PR template
	template := pr.FindPRTemplate()
	if template != "" {
		fmt.Println(ui.FormatInfo("Using PR template from repository"))
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

	modelName := cfg.LLM.Ollama.Model
	if cfg.LLM.Provider == "openai" {
		modelName = cfg.LLM.OpenAI.Model
	}
	spin = spinner.New(fmt.Sprintf("Generating PR description using %s", modelName))
	spin.Start()
	description, err := generator.Generate(ctx)
	spin.Stop(err == nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(err.Error()))
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(ui.FormatHeader("Generated PR description:"))
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println(description)
	fmt.Println(strings.Repeat("─", 60) + "\n")

	// Determine if we can open in browser
	canOpenBrowser := false
	var prURL string
	remoteURL, err := pr.GetRemoteURL(targetRemote)
	if err == nil {
		owner, repo, ok := pr.ParseGitHubRepo(remoteURL)
		if ok {
			// Detect fork owner for cross-fork PRs
			var headOwner string
			if targetRemote != "origin" {
				originURL, originErr := pr.GetRemoteURL("origin")
				if originErr == nil {
					forkOwner, _, forkOk := pr.ParseGitHubRepo(originURL)
					if forkOk {
						headOwner = forkOwner
					}
				}
			}
			prURL = pr.BuildGitHubPRURL(owner, repo, baseBranch, currentBranch, description, headOwner)
			canOpenBrowser = true
		}
	}

	if *autoOpen {
		if canOpenBrowser {
			if err := openInBrowser(prURL); err != nil {
				fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error opening browser: %v", err)))
				os.Exit(1)
			}
			fmt.Println(ui.FormatSuccess("Opened PR creation page in browser!"))
		} else {
			if err := copyToClipboard(description); err != nil {
				fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error copying to clipboard: %v", err)))
				os.Exit(1)
			}
			fmt.Println(ui.FormatInfo("No GitHub remote found. PR description copied to clipboard!"))
		}
		return
	}

	// Interactive menu
	var options []string
	if canOpenBrowser {
		options = []string{"Open in browser", "Copy to clipboard", "Do nothing"}
	} else {
		options = []string{"Copy to clipboard", "Do nothing"}
	}

	choice, err := ui.Choose("What would you like to do?", options, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(err.Error()))
		os.Exit(1)
	}

	switch choice {
	case "Open in browser":
		if err := openInBrowser(prURL); err != nil {
			fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error opening browser: %v", err)))
			os.Exit(1)
		}
		fmt.Println(ui.FormatSuccess("Opened PR creation page in browser!"))
	case "Copy to clipboard":
		if err := copyToClipboard(description); err != nil {
			fmt.Fprintln(os.Stderr, ui.FormatError(fmt.Sprintf("Error copying to clipboard: %v", err)))
			os.Exit(1)
		}
		fmt.Println(ui.FormatInfo("PR description copied to clipboard!"))
	}
}

func promptBranchType(types map[string]string, defaultType string) string {
	typeList := make([]string, 0, len(types))
	for key := range types {
		typeList = append(typeList, key)
	}

	choice, err := ui.Choose("Select branch type:", typeList, defaultType)
	if err != nil {
		fmt.Println(ui.FormatError(fmt.Sprintf("Error selecting branch type, using default: %s", defaultType)))
		return defaultType
	}

	if choice == "" {
		return defaultType
	}

	return choice
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
