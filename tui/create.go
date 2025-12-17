package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.createFocus = (m.createFocus + 1) % (len(m.createInputs) + 1)
			return m, m.updateCreateFocus()
		case "shift+tab", "up":
			m.createFocus--
			if m.createFocus < 0 {
				m.createFocus = len(m.createInputs)
			}
			return m, m.updateCreateFocus()
		case "left":
			if m.createFocus == len(m.createInputs) {
				m.createType--
				if m.createType < 0 {
					m.createType = len(m.workItemTypes) - 1
				}
			}
		case "right":
			if m.createFocus == len(m.createInputs) {
				m.createType = (m.createType + 1) % len(m.workItemTypes)
			}
		case "enter":
			if m.createInputs[0].Value() != "" {
				m.loading = true
				return m, m.createWorkItem()
			}
		}
	}

	cmd := m.updateCreateInputs(msg)
	return m, cmd
}

func (m *Model) updateCreateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.createInputs))
	for i := range m.createInputs {
		if i == m.createFocus {
			cmds[i] = m.createInputs[i].Focus()
		} else {
			m.createInputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *Model) updateCreateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.createInputs))
	for i := range m.createInputs {
		m.createInputs[i], cmds[i] = m.createInputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m Model) viewCreate() string {
	var b strings.Builder

	title := titleStyle.Render("✨ Create Work Item")
	b.WriteString(title)
	b.WriteString("\n\n")

	labels := []string{"Title *", "Description", "Priority (1-4)", "Assigned To"}

	for i, label := range labels {
		style := labelStyle
		if i == m.createFocus {
			style = style.Copy().Foreground(lipgloss.Color("229"))
		}
		b.WriteString(style.Render(label))
		b.WriteString("\n")
		b.WriteString(m.createInputs[i].View())
		b.WriteString("\n\n")
	}

	typeLabel := labelStyle
	if m.createFocus == len(m.createInputs) {
		typeLabel = typeLabel.Copy().Foreground(lipgloss.Color("229"))
	}
	b.WriteString(typeLabel.Render("Type"))
	b.WriteString("\n")

	var types []string
	for i, t := range m.workItemTypes {
		if i == m.createType {
			types = append(types, selectedStyle.Render(t))
		} else {
			types = append(types, normalStyle.Foreground(lipgloss.Color("241")).Render(t))
		}
	}
	b.WriteString(strings.Join(types, " "))
	b.WriteString("\n\n")

	// Show configured area path
	if m.client != nil && m.client.AreaPath != "" {
		b.WriteString(labelStyle.Render("Area Path"))
		b.WriteString("\n")
		b.WriteString(normalStyle.Foreground(lipgloss.Color("39")).Render(m.client.AreaPath))
		b.WriteString("\n\n")
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	if m.loading {
		b.WriteString("Creating work item...")
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("tab/↑↓: navigate • ←→: change type • enter: create • esc: cancel"))

	return boxStyle.Render(b.String())
}
