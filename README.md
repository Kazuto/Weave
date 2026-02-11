# Weave

[![Test](https://github.com/Kazuto/Weave/actions/workflows/ci.yml/badge.svg)](https://github.com/Kazuto/Weave/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Kazuto/Weave)](https://goreportcard.com/report/github.com/Kazuto/Weave)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A CLI tool that automates Git workflows. Generate AI-powered commit messages and pull request descriptions using local LLMs, and create GitFlow-compliant branch names from Jira tickets.

## Features

- **AI Commit Messages** - Generate conventional commit messages from your staged changes using Ollama
- **AI PR Descriptions** - Generate pull request descriptions from branch commits, with optional PR template support
- **Smart Branch Names** - Create GitFlow-compliant branch names from Jira ticket information
- **Local & Private** - All AI processing runs locally via Ollama, your code never leaves your machine
- **Configurable** - YAML configuration with sensible defaults and automatic validation
- **Lightweight** - Single binary with zero runtime dependencies (besides Git)

## Prerequisites

- **Git** - Required for all operations
- **[Ollama](https://ollama.com)** - Required for AI commit message and PR description generation
- **[Jira CLI](https://github.com/ankitpokhrel/jira-cli)** (optional) - For automatic ticket title fetching

## Installation

### Using Go Install

```bash
go install github.com/Kazuto/Weave@latest
```

### Build from Source

```bash
git clone https://github.com/Kazuto/Weave.git
cd Weave
make build
```

To install system-wide:

```bash
make install
```

### Binary Releases

Download pre-built binaries from the [releases page](https://github.com/Kazuto/Weave/releases).

Available platforms:

- **Linux** (amd64, arm64)
- **macOS** (amd64, arm64/Apple Silicon)
- **Windows** (amd64)

```bash
# Example: Linux amd64
curl -L https://github.com/Kazuto/Weave/releases/latest/download/weave-linux-amd64 -o weave
chmod +x weave
sudo mv weave /usr/local/bin/
```

### Verify Installation

```bash
weave version
weave help
```

## Usage

```
weave <command> [options]

Commands:
  commit      Generate an AI-powered commit message using Ollama
  branch      Generate a branch name from a Jira ticket
  pr          Generate an AI-powered pull request description
  version     Show version information
  help        Show this help message
```

### Commit

Generate a commit message from your staged changes using a local LLM.

```bash
# Generate commit message for staged changes
weave commit

# Auto-commit without prompting
weave commit -y

# Use all changes (not just staged)
weave commit --staged=false
```

**Workflow:**
1. Weave analyzes your staged diff and changed files
2. Sends the diff to Ollama for commit message generation
3. Displays the generated message in Conventional Commits format
4. Prompts you to accept (commits) or reject (copies to clipboard)

**Example output:**

```
✅ Checking Ollama connection
✅ Checking if model 'llama3.2' is available
📝 Found changes in 3 file(s)
✅ Generating commit message using llama3.2

============================================================
Generated commit message:
============================================================
feat(Auth): Add OAuth2 login flow

- Implement token refresh middleware
- Add login/logout API endpoints
============================================================

Use this commit message? [y/N]:
```

### Branch

Generate a GitFlow-compliant branch name from a Jira ticket.

```bash
# Fetch title from Jira automatically
weave branch PROJ-123

# Specify branch type
weave branch PROJ-123 --type hotfix

# Provide title manually (skips Jira lookup)
weave branch PROJ-123 --title "Add user profile dashboard"
```

**Supported branch types:**

| Type | Prefix | Purpose |
|------|--------|---------|
| `feature` | `feature/` | New features and enhancements |
| `hotfix` | `hotfix/` | Critical bug fixes for production |
| `refactor` | `refactor/` | Code improvements without changing functionality |
| `support` | `support/` | Maintenance and support tasks |

**Example output:**

```
Generated branch name:
feature/PROJ-123-add-user-profile-dashboard

Create branch with:
  git checkout -b feature/PROJ-123-add-user-profile-dashboard
```

### PR

Generate an AI-powered pull request description from your branch's commits.

```bash
# Auto-detect base branch (main/master)
weave pr

# Specify base branch
weave pr --base develop

# Auto-open in browser without prompting
weave pr -y
```

**Workflow:**
1. Weave compares your current branch against the base branch
2. Collects commits, changed files, and the diff between branches
3. If a `PULL_REQUEST_TEMPLATE.md` exists in the repo, uses it as a structural guide
4. Generates a PR description using Ollama
5. Offers to open the GitHub PR creation page in your browser or copy to clipboard

**Example output:**

```
Comparing feature/add-auth → main
📝 Found 3 commit(s) changing 5 file(s)
📋 Using PR template from repository
✅ Generating PR description using llama3.2

============================================================
Generated PR description:
============================================================
## Summary
Add OAuth2 authentication flow with token refresh support.

## Changes
- Implement login/logout API endpoints
- Add token refresh middleware
- Create auth configuration module

## Test Plan
- Verify login flow with valid credentials
- Test token refresh after expiration
============================================================

  1. Open in browser
  2. Copy to clipboard
  3. Do nothing

Select an option:
```

If a GitHub remote is detected, option 1 opens the "New Pull Request" page with the description pre-filled. Otherwise, copy to clipboard is offered as the primary action.

## Configuration

Weave automatically creates a configuration file at `~/.config/weave/config.yaml` on first run. No manual setup required.

### Configuration Options

```yaml
branch:
  max_length: 60              # Branch name max length (10-200)
  default_type: feature       # Default branch type
  types:
    feature: feature
    hotfix: hotfix
    refactor: refactor
    support: support
  sanitization:
    separator: "-"            # Replace spaces/special chars
    lowercase: true           # Convert to lowercase
    remove_umlauts: false     # Remove German umlauts

commit:
  ollama:
    model: llama3.2           # Ollama model to use
    host: http://localhost:11434
    temperature: 0.3          # Generation temperature (0-2)
    top_p: 0.9                # Top-p sampling (0-1)
    max_diff: 4000            # Max diff characters to send
  types:                      # Conventional commit types
    - feat
    - fix
    - docs
    - style
    - refactor
    - perf
    - test
    - chore
    - ci
    - build
  prompt: |                   # Custom prompt template
    ...                       # Supports {{.Types}}, {{.Files}}, {{.Diff}}

pr:
  default_base: ""            # Base branch (empty = auto-detect main/master)
  max_diff: 8000              # Max diff characters to send (100-100000)
  prompt: |                   # Custom prompt template
    ...                       # Supports {{.Branch}}, {{.Base}}, {{.Commits}},
                              # {{.Files}}, {{.Diff}}, {{.Template}}
```

### Setting Up Ollama

Install Ollama and pull a model:

```bash
# Install Ollama (macOS)
brew install ollama

# Start the Ollama server
ollama serve

# Pull the default model
ollama pull llama3.2
```

### Setting Up Jira CLI (Optional)

Required only for automatic ticket title fetching:

```bash
# Install
brew install ankitpokhrel/jira-cli/jira-cli

# Configure
jira init
```

## Development

### Setup

```bash
git clone https://github.com/Kazuto/Weave.git
cd Weave
make dev-setup
```

### Common Commands

```bash
make test               # Run tests
make test-coverage      # Run tests with coverage report
make test-race          # Run tests with race detection
make lint               # Run code linter
make fmt                # Format code
make security           # Run security checks
make build              # Build for current platform
make build-all          # Build for all platforms
make help               # Show all available commands
```

### Running Locally

```bash
# Run without building
go run main.go commit
go run main.go branch PROJ-123 --type feature --title "My feature"

# Build and run
make build
./weave commit
```

## Troubleshooting

### "Cannot connect to Ollama"

Ensure Ollama is running:

```bash
ollama serve
```

### "Model not available"

Pull the configured model:

```bash
ollama pull llama3.2
```

### "No staged changes found"

Stage your changes first:

```bash
git add <files>
weave commit
```

### "No commits found between branches"

Ensure your branch has commits ahead of the base branch:

```bash
git log main..HEAD --oneline
```

### "Open in browser" not shown

Weave needs a GitHub origin remote. Verify with:

```bash
git remote get-url origin
```

### "Jira CLI not found"

Either install Jira CLI or provide a title manually:

```bash
weave branch PROJ-123 --title "My branch title"
```

### Reset Configuration

```bash
rm ~/.config/weave/config.yaml
weave commit  # Recreates with defaults
```

## License

MIT License - see [LICENSE](LICENSE) for details.
