# Weave

A CLI tool that automates Git workflows. Generate AI-powered commit messages using local LLMs and create GitFlow-compliant branch names from Jira tickets.

## Features

- **AI Commit Messages** - Generate conventional commit messages from your staged changes using Ollama
- **Smart Branch Names** - Create GitFlow-compliant branch names from Jira ticket information
- **Local & Private** - All AI processing runs locally via Ollama, your code never leaves your machine
- **Configurable** - YAML configuration with sensible defaults and automatic validation
- **Lightweight** - Single binary with zero runtime dependencies (besides Git)

## Prerequisites

- **Git** - Required for all operations
- **[Ollama](https://ollama.com)** - Required for AI commit message generation
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
