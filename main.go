// Package main provides the entry point for bored, a TUI application
// for managing Azure DevOps Boards work items from the terminal.
package main

import (
	"fmt"
	"os"

	"github.com/laupski/bored/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(tui.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
