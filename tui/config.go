package tui

import (
	"fmt"
	"strings"

	"github.com/laupski/bored/azdo"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.configFocus = (m.configFocus + 1) % len(m.configInputs)
			return m, m.updateConfigFocus()
		case "shift+tab", "up":
			m.configFocus--
			if m.configFocus < 0 {
				m.configFocus = len(m.configInputs) - 1
			}
			return m, m.updateConfigFocus()
		case "enter":
			if m.configInputs[0].Value() != "" && m.configInputs[1].Value() != "" && m.configInputs[2].Value() != "" && m.configInputs[3].Value() != "" && m.configInputs[4].Value() != "" && m.configInputs[5].Value() != "" {
				org := m.configInputs[0].Value()
				project := m.configInputs[1].Value()
				team := m.configInputs[2].Value()
				areaPath := m.configInputs[3].Value()
				pat := m.configInputs[4].Value()
				username := m.configInputs[5].Value()

				m.client = azdo.NewClient(org, project, team, areaPath, pat)
				m.username = username
				m.loading = true

				// Save credentials to keychain
				if err := SaveCredentials(org, project, team, areaPath, pat, username); err != nil {
					m.keychainMessage = "Warning: Could not save to keychain"
				} else {
					m.keychainMessage = "Credentials saved to keychain"
				}

				return m, m.connect()
			}
		case "ctrl+d":
			// Clear stored credentials
			ClearCredentials()
			m.configInputs[0].SetValue("")
			m.configInputs[1].SetValue("")
			m.configInputs[2].SetValue("")
			m.configInputs[3].SetValue("")
			m.configInputs[4].SetValue("")
			m.configInputs[5].SetValue("")
			m.keychainLoaded = false
			m.keychainMessage = "Credentials cleared from keychain"
			return m, nil
		case "ctrl+f":
			// Open config file screen
			m.view = ViewConfigFile
			m.configFileFocus = 0
			m.appConfigMessage = ""
			return m, nil
		}
	}

	cmd := m.updateConfigInputs(msg)
	return m, cmd
}

func (m *Model) updateConfigFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.configInputs))
	for i := range m.configInputs {
		if i == m.configFocus {
			cmds[i] = m.configInputs[i].Focus()
		} else {
			m.configInputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *Model) updateConfigInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.configInputs))
	for i := range m.configInputs {
		m.configInputs[i], cmds[i] = m.configInputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m Model) viewConfig() string {
	var b strings.Builder

	title := titleStyle.Render("Azure DevOps TUI")
	b.WriteString(title)
	b.WriteString("\n\n")

	labels := []string{"Organization", "Project", "Team", "Area Path", "Personal Access Token", "Username"}

	for i, label := range labels {
		style := labelStyle
		if i == m.configFocus {
			style = style.Copy().Foreground(lipgloss.Color("229"))
		}
		b.WriteString(style.Render(label))
		b.WriteString("\n")
		b.WriteString(m.configInputs[i].View())
		b.WriteString("\n\n")
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	if m.loading {
		b.WriteString("Connecting...")
		b.WriteString("\n\n")
	}

	if m.keychainMessage != "" {
		b.WriteString(successStyle.Render("🔐 " + m.keychainMessage))
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("tab/↑↓: navigate • enter: connect • ctrl+d: clear keychain • ctrl+f: settings • ctrl+c: quit"))

	return boxStyle.Render(b.String())
}
