package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) updateBoard(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.workItems)-1 {
				m.cursor++
			}
		case "r":
			m.loading = true
			m.err = nil
			return m, m.fetchWorkItems()
		case "a":
			// Toggle show all / my items filter
			m.showAll = !m.showAll
			m.loading = true
			m.cursor = 0
			return m, m.fetchWorkItems()
		case "o":
			// Open selected work item in browser
			if len(m.workItems) > 0 && m.cursor < len(m.workItems) {
				wi := m.workItems[m.cursor]
				url := fmt.Sprintf("https://dev.azure.com/%s/%s/_workitems/edit/%d",
					m.client.Organization, m.client.Project, wi.ID)
				openBrowser(url)
			}
			return m, nil
		case "e", "enter":
			// Open detail/edit view
			if len(m.workItems) > 0 && m.cursor < len(m.workItems) {
				wi := m.workItems[m.cursor]
				m.selectedItem = &wi
				m.view = ViewDetail
				m.detailInputs[0].SetValue(wi.Fields.Title)
				m.detailInputs[1].SetValue(wi.Fields.State)
				// Populate Assigned To field
				assignedTo := ""
				if wi.Fields.AssignedTo != nil {
					assignedTo = wi.Fields.AssignedTo.UniqueName
				}
				m.detailInputs[2].SetValue(assignedTo)
				m.detailInputs[3].SetValue(wi.Fields.Tags)
				m.detailInputs[4].SetValue("")
				m.detailFocus = 0
				m.detailInputs[0].Focus()
				m.comments = nil
				m.err = nil
				m.message = ""
				return m, m.fetchComments(wi.ID)
			}
			return m, nil
		case "c", "n":
			m.view = ViewCreate
			m.createFocus = 0
			m.createInputs[0].Focus()
			for i := 1; i < len(m.createInputs); i++ {
				m.createInputs[i].Blur()
			}
			m.err = nil
			m.message = ""
			return m, nil
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) viewBoard() string {
	var b strings.Builder

	filterStatus := ""
	if m.username != "" {
		if m.showAll {
			filterStatus = " (showing all)"
		} else {
			filterStatus = fmt.Sprintf(" (filtered: %s)", m.username)
		}
	}
	header := titleStyle.Render(fmt.Sprintf("📋 Work Items - %s/%s%s", m.client.Organization, m.client.Project, filterStatus))
	b.WriteString(header)
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString("Loading work items...")
		b.WriteString("\n")
	} else if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	} else if len(m.workItems) == 0 {
		b.WriteString("No work items found.")
		b.WriteString("\n")
	} else {
		// Column definitions: ID, Title, Assigned To, State, Area Path, Tags, Comments, Activity Date
		colID := lipgloss.NewStyle().Width(7).Align(lipgloss.Left)
		colTitle := lipgloss.NewStyle().Width(30).Align(lipgloss.Left)
		colAssigned := lipgloss.NewStyle().Width(15).Align(lipgloss.Left)
		colState := lipgloss.NewStyle().Width(10).Align(lipgloss.Left)
		colArea := lipgloss.NewStyle().Width(15).Align(lipgloss.Left)
		colTags := lipgloss.NewStyle().Width(15).Align(lipgloss.Left)
		colComments := lipgloss.NewStyle().Width(4).Align(lipgloss.Left)
		colActivity := lipgloss.NewStyle().Width(12).Align(lipgloss.Left)

		headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		headerRow := lipgloss.JoinHorizontal(
			lipgloss.Top,
			colID.Inherit(headerStyle).Render("ID"),
			colTitle.Inherit(headerStyle).Render("Title"),
			colAssigned.Inherit(headerStyle).Render("Assigned To"),
			colState.Inherit(headerStyle).Render("State"),
			colArea.Inherit(headerStyle).Render("Area Path"),
			colTags.Inherit(headerStyle).Render("Tags"),
			colComments.Inherit(headerStyle).Render("💬"),
			colActivity.Inherit(headerStyle).Render("Activity"),
		)
		b.WriteString(headerRow)
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", 110))
		b.WriteString("\n")

		start := 0
		end := len(m.workItems)
		maxVisible := m.height - 12
		if maxVisible < 5 {
			maxVisible = 10
		}

		if len(m.workItems) > maxVisible {
			if m.cursor >= maxVisible {
				start = m.cursor - maxVisible + 1
			}
			end = start + maxVisible
			if end > len(m.workItems) {
				end = len(m.workItems)
			}
		}

		for i := start; i < end; i++ {
			wi := m.workItems[i]

			id := fmt.Sprintf("#%d", wi.ID)

			title := wi.Fields.Title
			if len(title) > 28 {
				title = title[:25] + "..."
			}

			assignedTo := ""
			if wi.Fields.AssignedTo != nil {
				assignedTo = wi.Fields.AssignedTo.DisplayName
				if len(assignedTo) > 13 {
					assignedTo = assignedTo[:10] + "..."
				}
			}

			state := wi.Fields.State

			areaPath := wi.Fields.AreaPath
			// Show only the last part of area path
			if idx := strings.LastIndex(areaPath, "\\"); idx >= 0 {
				areaPath = areaPath[idx+1:]
			}
			if len(areaPath) > 13 {
				areaPath = areaPath[:10] + "..."
			}

			tags := wi.Fields.Tags
			if len(tags) > 13 {
				tags = tags[:10] + "..."
			}

			comments := fmt.Sprintf("%d", wi.Fields.CommentCount)

			activityDate := ""
			if wi.Fields.ChangedDate != "" {
				if t, err := time.Parse(time.RFC3339, wi.Fields.ChangedDate); err == nil {
					activityDate = t.Format("Jan 02")
				}
			}

			stateStyle := colState
			if color, ok := stateColors[state]; ok {
				stateStyle = stateStyle.Foreground(color)
			}

			row := lipgloss.JoinHorizontal(
				lipgloss.Top,
				colID.Render(id),
				colTitle.Render(title),
				colAssigned.Render(assignedTo),
				stateStyle.Render(state),
				colArea.Render(areaPath),
				colTags.Render(tags),
				colComments.Render(comments),
				colActivity.Render(activityDate),
			)

			if i == m.cursor {
				b.WriteString(selectedStyle.Render(row))
			} else {
				b.WriteString(normalStyle.Render(row))
			}
			b.WriteString("\n")
		}
	}

	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(successStyle.Render(m.message))
	}

	b.WriteString("\n")
	helpText := "↑/k ↓/j: navigate • c/n: create • r: refresh"
	if m.username != "" {
		if m.showAll {
			helpText += " • a: show mine"
		} else {
			helpText += " • a: show all"
		}
	}
	helpText += " • e: edit • o: open • q: quit"
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
