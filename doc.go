// Package main is the entry point for bored, a terminal user interface (TUI)
// application for managing Azure DevOps Boards work items.
//
// # Installation
//
// Install bored using Go:
//
//	go install github.com/laupski/bored@latest
//
// # Features
//
// Bored provides a full-featured terminal interface for Azure DevOps including:
//
//   - View and manage work items (Bug, Task, User Story, Feature, Epic)
//   - Create, edit, and delete work items
//   - Add and view comments with @mention highlighting
//   - Manage parent/child hierarchy relationships
//   - Change iterations/sprints
//   - Update planning fields (Story Points, Estimates, etc.)
//   - Filter between "My Items" and "All Items"
//   - Vim-style keyboard navigation (j/k, h/l)
//   - Change notifications with system sound alerts
//
// # Configuration
//
// Credentials are stored securely in the system keychain.
// Application settings are stored in a TOML config file:
//
//   - Linux: ~/.config/bored/config.toml
//   - macOS: ~/Library/Application Support/bored/config.toml
//   - Windows: %APPDATA%\bored\config.toml
//
// # Packages
//
// The application is organized into the following packages:
//
//   - [github.com/laupski/bored/azdo]: Azure DevOps REST API client
//   - [github.com/laupski/bored/tui]: Terminal UI components using Bubble Tea
package main
