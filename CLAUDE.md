# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with this codebase.

## Project Overview

**bored** is a TUI (Terminal User Interface) application for Azure DevOps Boards, built with Go using the Bubble Tea framework. It allows users to manage work items, comments, iterations, and hierarchy relationships directly from the terminal.

## Tech Stack

- **Language**: Go 1.25
- **TUI Framework**: Bubble Tea (charmbracelet/bubbletea)
- **Styling**: Lip Gloss (charmbracelet/lipgloss)
- **Components**: Bubbles (charmbracelet/bubbles)
- **Secrets**: go-keyring for secure credential storage
- **Config**: TOML format (`~/.config/bored/config.toml`)

## Project Structure

```
├── main.go           # Application entry point
├── azdo/             # Azure DevOps API client
│   └── client.go     # HTTP client for Azure DevOps REST API
├── tui/              # Terminal UI components
│   ├── model.go      # Main Bubble Tea model
│   ├── board.go      # Board view
│   ├── detail.go     # Work item detail view
│   ├── create.go     # Work item creation
│   ├── config.go     # Configuration handling
│   ├── configfile.go # TOML config file parsing
│   └── keychain.go   # Secure credential storage
└── Justfile          # Task runner commands
```

## Common Commands

All commands use [just](https://github.com/casey/just) as the task runner:

```bash
just build      # Build binary to bin/bored
just run        # Run without building (go run)
just test       # Run all tests
just test-v     # Run tests with verbose output
just cover      # Run tests with coverage report
just cover-html # Run tests with coverage and open HTML report
just doc        # Serve godoc documentation locally
just fmt        # Format code
just vet        # Run go vet
just lint       # Run golangci-lint
just check      # Run all checks (fmt, vet, lint, test, cover)
just dev        # Build and run
just clean      # Remove build artifacts
just install    # Install to GOPATH/bin
```

## Development Workflow

1. **Before committing**: Run `just check` to ensure code passes all checks
2. **Quick iteration**: Use `just dev` to build and run in one step
3. **Testing**: Run `just test` for standard output or `just test-v` for verbose

## Architecture Notes

- The application follows the Elm architecture via Bubble Tea (Model-Update-View)
- Azure DevOps API interactions are centralized in the `azdo/` package
- UI state and components are managed in the `tui/` package
- PAT (Personal Access Token) is stored in the system keychain, not in config files
