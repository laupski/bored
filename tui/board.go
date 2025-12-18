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
		// Handle delete confirmation mode
		if m.deletingWorkItem {
			switch msg.String() {
			case "esc":
				m.deletingWorkItem = false
				m.deleteConfirmInput = ""
				return m, nil
			case "enter":
				// Check if the typed text matches the title
				if m.deleteConfirmInput == m.deleteWorkItemTitle {
					m.loading = true
					m.deletingWorkItem = false
					return m, m.deleteWorkItem(m.deleteWorkItemID)
				}
				// Wrong title - show error
				m.err = fmt.Errorf("title does not match - deletion cancelled")
				m.deletingWorkItem = false
				m.deleteConfirmInput = ""
				return m, nil
			case "backspace":
				if len(m.deleteConfirmInput) > 0 {
					m.deleteConfirmInput = m.deleteConfirmInput[:len(m.deleteConfirmInput)-1]
				}
				return m, nil
			default:
				// Add character to input
				if len(msg.String()) == 1 {
					m.deleteConfirmInput += msg.String()
				} else if msg.String() == "space" {
					m.deleteConfirmInput += " "
				}
				return m, nil
			}
		}

		// Calculate page info
		maxVisible := m.height - 12
		if maxVisible < 5 {
			maxVisible = 10
		}
		pageSize := maxVisible
		totalPages := (len(m.workItems) + pageSize - 1) / pageSize
		if totalPages == 0 {
			totalPages = 1
		}
		currentPage := m.cursor / pageSize

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.workItems)-1 {
				m.cursor++
			}
		case "left", "h", "pgup":
			// Previous page
			if currentPage > 0 {
				m.cursor = (currentPage - 1) * pageSize
			}
		case "right", "l", "pgdown":
			// Next page
			if currentPage < totalPages-1 {
				m.cursor = (currentPage + 1) * pageSize
				if m.cursor >= len(m.workItems) {
					m.cursor = len(m.workItems) - 1
				}
			}
		case "home":
			m.cursor = 0
		case "end":
			if len(m.workItems) > 0 {
				m.cursor = len(m.workItems) - 1
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
				m.parentItem = nil
				m.childItems = nil
				m.relatedExpanded = false
				m.relatedCursor = 0
				m.commentsExpanded = false
				m.commentScroll = 0
				m.iterationExpanded = false
				m.iterationCursor = 0
				m.err = nil
				m.message = ""
				return m, tea.Batch(m.fetchComments(wi.ID), m.fetchRelatedItems(wi.ID))
			}
			return m, nil
		case "c", "n":
			m.view = ViewCreate
			m.createFocus = 0
			m.createInputs[0].Focus()
			for i := 1; i < len(m.createInputs); i++ {
				m.createInputs[i].Blur()
			}
			// Auto-populate assignee with username
			m.createInputs[3].SetValue(m.username)
			m.err = nil
			m.message = ""
			return m, nil
		case "d":
			// Start delete confirmation for selected work item
			if len(m.workItems) > 0 && m.cursor < len(m.workItems) {
				wi := m.workItems[m.cursor]
				m.deletingWorkItem = true
				m.deleteWorkItemID = wi.ID
				m.deleteWorkItemTitle = wi.Fields.Title
				m.deleteConfirmInput = ""
				m.err = nil
			}
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
	} else if len(m.workItems) == 0 && m.err == nil {
		b.WriteString("No work items found.")
		b.WriteString("\n")
	} else {
		// Column definitions: ID, Type, Title, Assigned To, State, Area Path, Tags, Comments, Related, Activity Date
		colID := lipgloss.NewStyle().Width(10).Align(lipgloss.Left).MarginRight(2)
		colType := lipgloss.NewStyle().Width(12).Align(lipgloss.Left)
		colTitle := lipgloss.NewStyle().Width(35).Align(lipgloss.Left)
		colAssigned := lipgloss.NewStyle().Width(25).Align(lipgloss.Left)
		colState := lipgloss.NewStyle().Width(12).Align(lipgloss.Left)
		colArea := lipgloss.NewStyle().Width(18).Align(lipgloss.Left)
		colTags := lipgloss.NewStyle().Width(15).Align(lipgloss.Left)
		colComments := lipgloss.NewStyle().Width(4).Align(lipgloss.Left)
		colRelated := lipgloss.NewStyle().Width(4).Align(lipgloss.Left)
		colActivity := lipgloss.NewStyle().Width(14).Align(lipgloss.Left)

		headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		headerRow := lipgloss.JoinHorizontal(
			lipgloss.Top,
			colID.Inherit(headerStyle).Render("ID"),
			colType.Inherit(headerStyle).Render("Type"),
			colTitle.Inherit(headerStyle).Render("Title"),
			colAssigned.Inherit(headerStyle).Render("Assigned To"),
			colState.Inherit(headerStyle).Render("State"),
			colArea.Inherit(headerStyle).Render("Area Path"),
			colTags.Inherit(headerStyle).Render("Tags"),
			colComments.Inherit(headerStyle).Render("💬"),
			colRelated.Inherit(headerStyle).Render("🔗"),
			colActivity.Inherit(headerStyle).Render("Activity"),
		)
		b.WriteString(headerRow)
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", 152))
		b.WriteString("\n")

		// Calculate pagination
		maxVisible := m.height - 12
		if maxVisible < 5 {
			maxVisible = 10
		}
		pageSize := maxVisible
		totalPages := (len(m.workItems) + pageSize - 1) / pageSize
		if totalPages == 0 {
			totalPages = 1
		}
		currentPage := m.cursor / pageSize

		start := currentPage * pageSize
		end := start + pageSize
		if end > len(m.workItems) {
			end = len(m.workItems)
		}

		// Show page indicator
		if totalPages > 1 {
			pageInfo := fmt.Sprintf("Page %d of %d (%d items)", currentPage+1, totalPages, len(m.workItems))
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(pageInfo))
			b.WriteString("\n\n")
		}

		for i := start; i < end; i++ {
			wi := m.workItems[i]

			id := fmt.Sprintf("#%d", wi.ID)

			wiType := wi.Fields.WorkItemType
			if len(wiType) > 11 {
				wiType = wiType[:8] + "..."
			}

			title := wi.Fields.Title
			if len(title) > 34 {
				title = title[:31] + "..."
			}

			assignedTo := ""
			if wi.Fields.AssignedTo != nil {
				assignedTo = wi.Fields.AssignedTo.DisplayName
				if len(assignedTo) > 24 {
					assignedTo = assignedTo[:21] + "..."
				}
			}

			state := wi.Fields.State

			areaPath := wi.Fields.AreaPath
			// Show only the last part of area path
			if idx := strings.LastIndex(areaPath, "\\"); idx >= 0 {
				areaPath = areaPath[idx+1:]
			}
			if len(areaPath) > 17 {
				areaPath = areaPath[:14] + "..."
			}

			tags := wi.Fields.Tags
			if len(tags) > 14 {
				tags = tags[:11] + "..."
			}

			comments := fmt.Sprintf("%d", wi.Fields.CommentCount)

			// Count hierarchy relations (parent + children)
			relatedCount := 0
			for _, rel := range wi.Relations {
				if rel.Rel == "System.LinkTypes.Hierarchy-Reverse" || rel.Rel == "System.LinkTypes.Hierarchy-Forward" {
					relatedCount++
				}
			}
			related := fmt.Sprintf("%d", relatedCount)

			activityDate := ""
			if wi.Fields.ChangedDate != "" {
				if t, err := time.Parse(time.RFC3339, wi.Fields.ChangedDate); err == nil {
					activityDate = t.Format("Jan 02 '06")
				}
			}

			row := lipgloss.JoinHorizontal(
				lipgloss.Top,
				colID.Render(id),
				colType.Render(wiType),
				colTitle.Render(title),
				colAssigned.Render(assignedTo),
				colState.Render(state),
				colArea.Render(areaPath),
				colTags.Render(tags),
				colComments.Render(comments),
				colRelated.Render(related),
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

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(successStyle.Render(m.message))
	}

	b.WriteString("\n")

	// Show delete confirmation dialog
	if m.deletingWorkItem {
		deleteStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(0, 1)
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

		deletePrompt := fmt.Sprintf("⚠️  DELETE #%d\n\n", m.deleteWorkItemID)
		deletePrompt += warningStyle.Render("To confirm deletion, type the title:") + "\n"
		deletePrompt += fmt.Sprintf("\"%s\"\n\n", m.deleteWorkItemTitle)
		deletePrompt += fmt.Sprintf("Your input: %s_\n\n", m.deleteConfirmInput)
		deletePrompt += "enter: confirm • esc: cancel"
		b.WriteString(deleteStyle.Render(deletePrompt))
		b.WriteString("\n")
	} else {
		helpText := "↑/k ↓/j: navigate • ←/h →/l: page • c/n: create • d: delete • r: refresh"
		if m.username != "" {
			if m.showAll {
				helpText += " • a: show mine"
			} else {
				helpText += " • a: show all"
			}
		}
		helpText += " • e: edit • o: open • q: quit"
		b.WriteString(helpStyle.Render(helpText))
	}

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
