# AGENTS.md

This document provides guidance for AI agents working on the `bored` codebase.

## Project Overview

**bored** is a Terminal User Interface (TUI) application for Azure DevOps Boards. It allows users to view, create, update, and manage Azure DevOps work items directly from the terminal.

## Tech Stack

- **Language**: Go 1.25
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm-architecture based TUI framework)
- **Styling**: [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Components**: [Bubbles](https://github.com/charmbracelet/bubbles) (text inputs, etc.)
- **Config**: TOML via `github.com/BurntSushi/toml`
- **Secrets**: System keychain via `github.com/zalando/go-keyring`

## Project Structure

```
bored/
├── main.go              # Entry point - initializes and runs the TUI
├── azdo/                # Azure DevOps API client
│   ├── client.go        # HTTP client for Azure DevOps REST API
│   └── client_test.go   # Tests for the API client
├── tui/                 # Terminal UI components
│   ├── model.go         # Main Bubble Tea model and message types
│   ├── board.go         # Board view (work item list)
│   ├── config.go        # Configuration view (credentials setup)
│   ├── configfile.go    # Config file management (TOML)
│   ├── create.go        # Create work item view
│   ├── detail.go        # Work item detail view
│   ├── keychain.go      # Keychain credential storage
│   └── *_test.go        # Test files
├── Justfile             # Task runner commands (like Makefile)
└── bin/                 # Build output directory
```

## Build & Run Commands

This project uses [just](https://github.com/casey/just) as a task runner:

```bash
just build      # Build to bin/bored
just run        # Run with go run
just test       # Run all tests
just test-v     # Run tests with verbose output
just fmt        # Format code
just vet        # Run go vet
just check      # Run fmt, vet, and test
just clean      # Remove build artifacts
just dev        # Build and run
just install    # Install to GOPATH/bin
```

Or use standard Go commands:

```bash
go build -o bin/bored .
go run .
go test ./...
```

## Architecture

### Bubble Tea Pattern

The application follows the Elm architecture via Bubble Tea:

1. **Model** (`tui/model.go`): Application state
2. **Update**: Handle messages and update state
3. **View**: Render the UI based on state

### Views

The app has multiple views (screens):

- `ViewConfig`: Initial setup for Azure DevOps credentials
- `ViewBoard`: Main work item list
- `ViewCreate`: Create new work items
- `ViewDetail`: View/edit work item details
- `ViewConfigFile`: Application settings

### Message Types

Async operations use message types:

- `workItemsMsg`: Results from fetching work items
- `createResultMsg`: Results from creating work items
- `connectMsg`: Connection test results
- `commentsMsg`, `updateWorkItemMsg`, etc.

## Azure DevOps API Client

The `azdo.Client` provides methods for:

- `GetWorkItems()` / `GetWorkItemsFiltered()`: Query work items via WIQL
- `CreateWorkItem()` / `CreateWorkItemWithAssignee()`: Create work items
- `UpdateWorkItem()`: Modify work item fields
- `GetComments()` / `AddComment()`: Manage comments
- `GetRelatedWorkItems()`: Get parent/child relationships
- `DeleteWorkItem()`: Delete work items
- `GetIterations()` / `UpdateWorkItemIteration()`: Manage iterations

Authentication uses Personal Access Tokens (PAT) with Basic auth.

## Configuration

### Credentials

Credentials are stored in the system keychain with the service name `bored-azdo`:

- Organization, Project, Team, Area Path
- Personal Access Token (PAT)
- Username

### App Config

Application settings are stored in `~/.config/bored/config.toml`:

```toml
default_show_all = false
max_work_items = 50
```

## Coding Conventions

1. **Error Handling**: Return errors, don't panic. Log API errors with status codes.

2. **Testing**: Test files are alongside source files (`*_test.go`). Use table-driven tests.

3. **Styling**: Use Lip Gloss styles defined in `model.go` (`titleStyle`, `selectedStyle`, etc.)

4. **Async Operations**: Return `tea.Cmd` for async work. Handle results in `Update()` via message types.

5. **State Updates**: In Bubble Tea, always return the modified model and any commands.

## Key Patterns

### Adding a New View

1. Add a new `View` constant in `model.go`
2. Create a new file (e.g., `tui/newview.go`)
3. Implement `updateNewView(msg tea.Msg)` and `viewNewView()` methods
4. Add cases in `Model.Update()` and `Model.View()`

### Adding an API Method

1. Add the method to `azdo/client.go`
2. Create response struct types as needed
3. Add a corresponding message type in `tui/model.go`
4. Create a fetch function returning `tea.Cmd`
5. Handle the message in `Model.Update()`

### Keyboard Shortcuts

Keyboard handling is in the `Update` methods. Common patterns:

- `ctrl+c`: Quit
- `esc`: Go back
- `enter`: Confirm/select
- `tab`/`shift+tab`: Navigate inputs
- Arrow keys: Navigate lists

## Testing

Run tests with:

```bash
just test       # or: go test ./...
just test-v     # verbose output
```

Test files:
- `azdo/client_test.go`: API client tests
- `tui/*_test.go`: TUI component tests

## Important Notes

1. **PAT Permissions**: The Azure DevOps PAT needs work item read/write permissions.

2. **Team Context**: When a team is specified, queries are scoped to team area paths.

3. **Work Item Relations**: Parent/child links use `System.LinkTypes.Hierarchy-Reverse` (parent) and `System.LinkTypes.Hierarchy-Forward` (child).

4. **API Version**: Uses Azure DevOps REST API version 7.0.
