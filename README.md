# Weave

[![Test](https://github.com/Kazuto/Weave/actions/workflows/ci.yml/badge.svg)](https://github.com/Kazuto/Weave/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Kazuto/Weave)](https://goreportcard.com/report/github.com/Kazuto/Weave)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A CLI tool that automates Git workflows. Generate AI-powered commit messages and pull request descriptions using local or remote LLMs, and create GitFlow-compliant branch names from Jira tickets.

## Features

- **AI Commit Messages** - Generate conventional commit messages from your staged changes
- **AI PR Descriptions** - Generate pull request descriptions from branch commits, with optional PR template support
- **Smart Branch Names** - Create GitFlow-compliant branch names from Jira ticket information
- **Flexible LLM Support** - Use Ollama locally, any OpenAI-compatible API, or GitHub Copilot
- **Configurable** - YAML configuration with sensible defaults and automatic validation
- **Lightweight** - Single binary with zero runtime dependencies (besides Git)

## Prerequisites

- **Git** - Required for all operations
- An LLM provider (see [Configuration](#configuration))
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
  commit      Generate an AI-powered commit message
  branch      Generate a branch name from a Jira ticket
  pr          Generate an AI-powered pull request description
  version     Show version information
  help        Show this help message
```

### Commit

Generate a commit message from your staged changes.

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
2. Sends the diff to your configured LLM provider
3. Displays the generated message in Conventional Commits format
4. Prompts you to accept (commits) or reject (copies to clipboard)

**Example output:**

```
✓ Checking openai connection
✓ Checking if model 'claude-haiku-4.5' is available
▸ Found changes in 3 file(s)
✓ Generating commit message using claude-haiku-4.5

────────────────────────────────────────────────────────────
Generated commit message:
────────────────────────────────────────────────────────────
feat(Auth): Add OAuth2 login flow

- Implement token refresh middleware
- Add login/logout API endpoints
────────────────────────────────────────────────────────────

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

# Auto-switch branch without prompting
weave branch PROJ-123 -y
```

**Supported branch types:**

| Type       | Prefix      | Purpose                                          |
| ---------- | ----------- | ------------------------------------------------ |
| `feature`  | `feature/`  | New features and enhancements                    |
| `hotfix`   | `hotfix/`   | Critical bug fixes for production                |
| `refactor` | `refactor/` | Code improvements without changing functionality |
| `support`  | `support/`  | Maintenance and support tasks                    |

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
weave pr -b develop

# Target a different remote (e.g., upstream for fork-based workflows)
weave pr --remote upstream
weave pr -r upstream

# Auto-open in browser without prompting
weave pr -y
```

**Workflow:**

1. Weave compares your current branch against the base branch
2. Collects commits, changed files, and the diff between branches
3. If a `PULL_REQUEST_TEMPLATE.md` exists in the repo, uses it as a structural guide
4. Generates a PR description using your configured LLM provider
5. Offers to open the GitHub PR creation page in your browser or copy to clipboard

**Example output:**

```
▸ Comparing feature/add-auth → main
▸ Found 3 commit(s) changing 5 file(s)
▸ Using PR template from repository
✓ Generating PR description using claude-haiku-4.5

────────────────────────────────────────────────────────────
Generated PR description:
────────────────────────────────────────────────────────────
## Summary
Add OAuth2 authentication flow with token refresh support.

## Changes
- Implement login/logout API endpoints
- Add token refresh middleware
- Create auth configuration module

## Test Plan
- Verify login flow with valid credentials
- Test token refresh after expiration
────────────────────────────────────────────────────────────

  1. Open in browser
  2. Copy to clipboard
  3. Do nothing

Select an option:
```

If a GitHub remote is detected, option 1 opens the "New Pull Request" page with the description pre-filled. For fork-based workflows using `--remote upstream`, Weave automatically constructs cross-fork PR URLs. Otherwise, copy to clipboard is offered as the primary action.

## Configuration

Weave automatically creates a configuration file at `~/.config/weave/config.yaml` on first run. No manual setup required.

### Configuration Options

```yaml
branch:
  max_length: 60            # Branch name max length (10-200)
  default_type: feature     # Default branch type
  types:
    feature: feature
    hotfix: hotfix
    refactor: refactor
    support: support
  sanitization:
    separator: "-"          # Replace spaces/special chars
    lowercase: true         # Convert to lowercase
    remove_umlauts: false   # Remove German umlauts
  enable_gitflow: true      # Enable gitflow branch naming

commit:
  types:                    # Conventional commit types
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
  prompt: |                 # Custom prompt template
    ...                     # Supports {{.Types}}, {{.Files}}, {{.Diff}}, {{.RecentCommits}}

pr:
  default_base: ""          # Base branch (empty = auto-detect main/master)
  default_remote: ""        # Target remote (empty = origin)
  prompt: |                 # Custom prompt template
    ...                     # Supports {{.Branch}}, {{.Base}}, {{.Commits}},
                            # {{.Files}}, {{.Diff}}, {{.Template}}

llm:
  provider: ollama          # "ollama" or "openai"
  max_diff: 4000            # Max diff characters to send to the LLM (100-100000)
  ollama:
    model: llama3.2
    host: http://localhost:11434
    temperature: 0.3        # Generation temperature (0-2)
    top_p: 0.9              # Top-p sampling (0-1)
    timeout: 0              # Request timeout in seconds (0 = default 300s)
  openai:
    model: gpt-4o
    host: http://localhost:1234/v1  # Any OpenAI-compatible endpoint
    api_key: ""
    temperature: 0.7
    top_p: 0.9
    timeout: 0
```

### LLM Providers

Weave supports two provider types: `ollama` for local models, and `openai` for any OpenAI-compatible API.

#### Ollama (local)

Install Ollama and pull a model:

```bash
# Install Ollama (macOS)
brew install ollama

# Start the Ollama server
ollama serve

# Pull a model
ollama pull llama3.2
```

```yaml
llm:
  provider: ollama
  max_diff: 4000
  ollama:
    model: llama3.2
    host: http://localhost:11434
    temperature: 0.3
    top_p: 0.9
```

#### OpenAI

```yaml
llm:
  provider: openai
  max_diff: 4000
  openai:
    model: gpt-4o
    host: https://api.openai.com/v1
    api_key: sk-...
    temperature: 0.7
    top_p: 0.9
```

#### GitHub Copilot

Use GitHub Copilot as your LLM provider with any model available on your plan. Requires a GitHub token with Copilot access (a PAT or the token from `gh auth token`).

```yaml
llm:
  provider: openai
  max_diff: 4000
  openai:
    model: claude-haiku-4.5
    host: https://api.githubcopilot.com
    api_key: <your_github_token>
    temperature: 0.3
    top_p: 0.9
```

Available Copilot models (as of August 2026) include `claude-haiku-4.5`, `claude-sonnet-4.6`, `claude-opus-4.6`, `gpt-4.1`, `gemini-3.6-flash`, and others. Check the available models for your account:

```bash
curl -s -H "Authorization: Bearer $(gh auth token)" \
  https://api.githubcopilot.com/models | \
  python3 -c "import json,sys; [print(m['id']) for m in json.load(sys.stdin)['data']]"
```

#### OpenAI-compatible local servers (LM Studio, llama.cpp)

```yaml
llm:
  provider: openai
  max_diff: 4000
  openai:
    model: your-model-name
    host: http://localhost:1234/v1
    temperature: 0.3
    top_p: 0.9
```

> **Note:** Include `/v1` in the host for standard OpenAI-compatible servers. For GitHub Copilot, omit it — the API uses a different path structure.

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

### "Cannot connect to provider"

For Ollama, ensure it is running:

```bash
ollama serve
```

For OpenAI-compatible providers, verify the host and API key are correct.

### "Model not available"

For Ollama, pull the configured model:

```bash
ollama pull llama3.2
```

For GitHub Copilot, verify the model ID matches exactly what the API returns (e.g. `claude-sonnet-4.6`, not `claude-sonnet-4-6`).

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
