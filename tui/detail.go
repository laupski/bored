package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			if !m.commentsExpanded {
				m.detailFocus = (m.detailFocus + 1) % len(m.detailInputs)
				return m, m.updateDetailFocus()
			}
			return m, nil
		case "shift+tab", "up":
			if !m.commentsExpanded {
				m.detailFocus--
				if m.detailFocus < 0 {
					m.detailFocus = len(m.detailInputs) - 1
				}
				return m, m.updateDetailFocus()
			}
			return m, nil
		case "ctrl+s":
			// Save changes to title/state/assignee/tags
			title := m.detailInputs[0].Value()
			state := m.detailInputs[1].Value()
			assignedTo := m.detailInputs[2].Value()
			tags := m.detailInputs[3].Value()
			m.loading = true
			return m, m.updateWorkItem(m.selectedItem.ID, title, state, assignedTo, tags)
		case "enter":
			// If on comment field and there's text, add the comment
			if m.detailFocus == 4 && m.detailInputs[4].Value() != "" {
				m.loading = true
				return m, m.addComment(m.selectedItem.ID, m.detailInputs[4].Value())
			}
			return m, nil
		case "v":
			// Toggle comments expanded/collapsed
			m.commentsExpanded = !m.commentsExpanded
			m.commentScroll = 0
			return m, nil
		case "j":
			// Scroll comments down when expanded
			if m.commentsExpanded && m.commentScroll < len(m.comments)-1 {
				m.commentScroll++
			}
			return m, nil
		case "k":
			// Scroll comments up when expanded
			if m.commentsExpanded && m.commentScroll > 0 {
				m.commentScroll--
			}
			return m, nil
		}
	}

	if !m.commentsExpanded {
		cmd := m.updateDetailInputs(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) updateDetailFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.detailInputs))
	for i := range m.detailInputs {
		if i == m.detailFocus {
			cmds[i] = m.detailInputs[i].Focus()
		} else {
			m.detailInputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *Model) updateDetailInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.detailInputs))
	for i := range m.detailInputs {
		m.detailInputs[i], cmds[i] = m.detailInputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m Model) viewDetail() string {
	if m.selectedItem == nil {
		return "No work item selected"
	}

	var b strings.Builder

	wi := m.selectedItem

	// Header
	header := titleStyle.Render(fmt.Sprintf("📝 %s #%d", wi.Fields.WorkItemType, wi.ID))
	b.WriteString(header)
	b.WriteString("\n\n")

	// Editable fields with helper text
	labels := []string{"Title", "State", "Assigned To", "Tags", "Add Comment"}
	hints := []string{
		"",
		"(New, Active, Resolved, Closed, Done)",
		"(email address)",
		"(semicolon-separated: tag1; tag2)",
		"",
	}

	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)

	for i, label := range labels {
		style := labelStyle
		if i == m.detailFocus {
			style = style.Copy().Foreground(lipgloss.Color("229"))
		}
		b.WriteString(style.Render(label))
		if hints[i] != "" {
			b.WriteString(" ")
			b.WriteString(hintStyle.Render(hints[i]))
		}
		b.WriteString("\n")
		b.WriteString(m.detailInputs[i].View())
		b.WriteString("\n\n")
	}

	// Work item details (read-only)
	detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	b.WriteString(labelStyle.Render("Details"))
	b.WriteString("\n")
	b.WriteString(detailStyle.Render(fmt.Sprintf("Area Path: %s", wi.Fields.AreaPath)))
	b.WriteString("\n")
	b.WriteString(detailStyle.Render(fmt.Sprintf("Type: %s", wi.Fields.WorkItemType)))
	b.WriteString("\n\n")

	// Comments section
	commentHeaderStyle := labelStyle.Copy()
	if m.commentsExpanded {
		commentHeaderStyle = commentHeaderStyle.Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229"))
	}

	if m.commentsExpanded {
		b.WriteString(commentHeaderStyle.Render(fmt.Sprintf("▼ Comments (%d)", len(m.comments))))
		b.WriteString(" ")
		b.WriteString(hintStyle.Render("(v: collapse, j/k: scroll)"))
	} else {
		b.WriteString(labelStyle.Render(fmt.Sprintf("▶ Comments (%d)", len(m.comments))))
		b.WriteString(" ")
		b.WriteString(hintStyle.Render("(v: expand)"))
	}
	b.WriteString("\n")

	if len(m.comments) == 0 {
		b.WriteString(detailStyle.Render("No comments"))
		b.WriteString("\n")
	} else if !m.commentsExpanded {
		// Collapsed: show summary of most recent comment
		lastComment := m.comments[len(m.comments)-1]
		dateStr := ""
		if t, err := time.Parse(time.RFC3339, lastComment.CreatedDate); err == nil {
			dateStr = t.Format("Jan 02")
		}
		summary := fmt.Sprintf("Latest: %s (%s)", lastComment.CreatedBy.DisplayName, dateStr)
		b.WriteString(detailStyle.Render(summary))
		b.WriteString("\n")
	} else {
		// Expanded: show comments with scroll
		commentStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			MarginBottom(1)

		// Show 5 comments starting from scroll position
		maxVisible := 5
		start := m.commentScroll
		end := start + maxVisible
		if end > len(m.comments) {
			end = len(m.comments)
		}

		// Show scroll indicator
		if len(m.comments) > maxVisible {
			scrollInfo := fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(m.comments))
			b.WriteString(detailStyle.Render(scrollInfo))
			b.WriteString("\n")
		}

		for i := start; i < end; i++ {
			c := m.comments[i]
			dateStr := ""
			if t, err := time.Parse(time.RFC3339, c.CreatedDate); err == nil {
				dateStr = t.Format("Jan 02, 15:04")
			}
			header := fmt.Sprintf("%s - %s", c.CreatedBy.DisplayName, dateStr)
			text := c.Text
			// Strip HTML tags (basic)
			text = strings.ReplaceAll(text, "<div>", "")
			text = strings.ReplaceAll(text, "</div>", "")
			text = strings.ReplaceAll(text, "<br>", "\n")
			if len(text) > 200 {
				text = text[:197] + "..."
			}
			b.WriteString(commentStyle.Render(fmt.Sprintf("%s\n%s", header, text)))
			b.WriteString("\n")
		}
	}

	// Error/success messages
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}
	if m.message != "" {
		b.WriteString(successStyle.Render(m.message))
		b.WriteString("\n")
	}

	if m.loading {
		b.WriteString("Loading...")
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.commentsExpanded {
		b.WriteString(helpStyle.Render("v: collapse comments • j/k: scroll • esc: back"))
	} else {
		b.WriteString(helpStyle.Render("tab/↑↓: navigate • ctrl+s: save • enter: add comment • v: view comments • esc: back"))
	}

	return boxStyle.Render(b.String())
}
