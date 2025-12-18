package tui

import (
	"fmt"

	"regexp"
	"strings"
	"time"

	"github.com/laupski/bored/azdo"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// parseMentions extracts @mentions from comment HTML and returns formatted text
// with highlighted mentions. The format is:
// <a href=\"#\" data-vss-mention=\"version:2.0,{user-guid}\">@Display Name</a>
func parseMentions(text string, orgURL string) string {
	// Regex to match mention anchor tags
	mentionRegex := regexp.MustCompile(`<a[^>]*data-vss-mention=\"version:[^,]*,([^\"]*)\"[^>]*>@([^<]*)</a>`)

	// Style for mentions - just color and bold, no hyperlinking
	// OSC 8 hyperlinks are incompatible with bubbletea/lipgloss rendering
	mentionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	result := mentionRegex.ReplaceAllStringFunc(text, func(match string) string {
		submatches := mentionRegex.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			displayName := submatches[2]
			// Just style the mention without hyperlinking
			return mentionStyle.Render("@" + displayName)
		}
		return match
	})

	return result
}

// parseURLs finds URLs in text and highlights them (no hyperlinking to avoid terminal issues)
func parseURLs(text string) string {
	// Regex to match URLs (http, https, ftp) - more restrictive to avoid matching escape sequences
	urlRegex := regexp.MustCompile(`(https?://[^\s<>"\x1b]+)`)

	// Style for URLs - using blue color
	urlStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33"))

	// Replace URLs with styled versions (no OSC 8 hyperlinks)
	result := urlRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Clean up any trailing punctuation that might have been captured
		cleanURL := strings.TrimRight(match, ".,;:!?)")
		trailingChars := match[len(cleanURL):]

		// Just style the URL text without hyperlinking
		return urlStyle.Render(cleanURL) + trailingChars
	})

	return result
}

// parseHTMLLinks extracts URLs from HTML anchor tags and highlights them (no hyperlinking)
func parseHTMLLinks(text string) string {
	// Regex to match HTML anchor tags with href (but not vss-mention tags)
	linkRegex := regexp.MustCompile(`<a[^>]*href=\"([^\"]+)\"[^>]*>([^<]*)</a>`)

	// Style for links - blue color
	linkStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33"))

	result := linkRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Skip mention tags (they're handled separately)
		if strings.Contains(match, "data-vss-mention") {
			return match
		}

		submatches := linkRegex.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			url := submatches[1]
			linkText := submatches[2]

			// Skip anchor-only links
			if url == "#" || url == "" {
				return linkText
			}

			// Just style the link text without hyperlinking
			// Show both the text and URL if they differ
			if linkText != url && linkText != "" {
				return linkStyle.Render(linkText) + " (" + linkStyle.Render(url) + ")"
			}
			return linkStyle.Render(url)
		}
		return match
	})

	return result
}

// stripHTMLTags removes common HTML tags from text while preserving mentions and URLs
func stripHTMLTags(text string, orgURL string) string {
	// First, process mentions to preserve them
	text = parseMentions(text, orgURL)

	// Then process HTML anchor tags with URLs (before stripping tags)
	text = parseHTMLLinks(text)

	// Strip common HTML tags
	text = strings.ReplaceAll(text, "<div>", "")
	text = strings.ReplaceAll(text, "</div>", "")
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "<p>", "")
	text = strings.ReplaceAll(text, "</p>", "\n")

	// Remove any remaining HTML tags (but not our OSC 8 sequences)
	tagRegex := regexp.MustCompile(`<[^>]+>`)
	text = tagRegex.ReplaceAllString(text, "")

	// Finally, process any plain-text URLs that weren't in anchor tags
	text = parseURLs(text)

	return strings.TrimSpace(text)
}

func (m Model) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle planning edit mode
		if m.planningExpanded {
			fieldCount := len(m.planningFields)
			if fieldCount == 0 {
				fieldCount = 1 // Avoid division by zero
			}
			switch msg.String() {
			case "esc", "ctrl+g":
				m.planningExpanded = false
				return m, nil
			case "tab", "down":
				m.planningFocus = (m.planningFocus + 1) % fieldCount
				return m, m.updatePlanningFocus()
			case "shift+tab", "up":
				m.planningFocus--
				if m.planningFocus < 0 {
					m.planningFocus = fieldCount - 1
				}
				return m, m.updatePlanningFocus()
			case "enter":
				// Save planning fields dynamically
				return m, m.savePlanningFieldsDynamic()
			}
			// Update the focused planning input
			cmd := m.updatePlanningInputs(msg)
			return m, cmd
		}

		// Handle iteration selection mode
		if m.iterationExpanded {
			switch msg.String() {
			case "esc", "ctrl+t":
				m.iterationExpanded = false
				return m, nil
			case "up":
				if m.iterationCursor > 0 {
					m.iterationCursor--
				}
				return m, nil
			case "down":
				if m.iterationCursor < len(m.iterations)-1 {
					m.iterationCursor++
				}
				return m, nil
			case "enter":
				if m.iterationCursor < len(m.iterations) {
					// Get iteration path from reordered display list
					displayOrder := m.getIterationDisplayOrder()
					if m.iterationCursor < len(displayOrder) {
						m.loading = true
						return m, m.updateIteration(m.selectedItem.ID, displayOrder[m.iterationCursor].Path)
					}
				}
				return m, nil
			}
			return m, nil
		}

		// Handle create related mode input
		if m.creatingRelated {
			switch msg.String() {
			case "esc":
				m.creatingRelated = false
				m.createRelatedTitle = ""
				return m, nil
			case "enter":
				if m.createRelatedTitle != "" {
					wiType := "Task"
					if m.createRelatedType < len(m.workItemTypes) {
						wiType = m.workItemTypes[m.createRelatedType]
					}
					m.loading = true
					m.creatingRelated = false
					return m, m.createRelatedItem(m.selectedItem.ID, m.createRelatedAsChild, m.createRelatedTitle, wiType, m.createRelatedAssignee)
				}
				return m, nil
			case "tab":
				// Toggle between title and assignee fields
				m.createRelatedFocus = (m.createRelatedFocus + 1) % 2
				return m, nil
			case "left":
				if m.createRelatedType > 0 {
					m.createRelatedType--
				} else {
					m.createRelatedType = len(m.workItemTypes) - 1
				}
				return m, nil
			case "right":
				m.createRelatedType = (m.createRelatedType + 1) % len(m.workItemTypes)
				return m, nil
			case "backspace":
				if m.createRelatedFocus == 0 && len(m.createRelatedTitle) > 0 {
					m.createRelatedTitle = m.createRelatedTitle[:len(m.createRelatedTitle)-1]
				} else if m.createRelatedFocus == 1 && len(m.createRelatedAssignee) > 0 {
					m.createRelatedAssignee = m.createRelatedAssignee[:len(m.createRelatedAssignee)-1]
				}
				return m, nil
			default:
				// Add character to the focused field
				if len(msg.String()) == 1 {
					if m.createRelatedFocus == 0 {
						m.createRelatedTitle += msg.String()
					} else {
						m.createRelatedAssignee += msg.String()
					}
				} else if msg.String() == "space" {
					if m.createRelatedFocus == 0 {
						m.createRelatedTitle += " "
					} else {
						m.createRelatedAssignee += " "
					}
				}
				return m, nil
			}
		}

		switch msg.String() {
		case "tab", "down":
			if !m.commentsExpanded && !m.relatedExpanded {
				m.detailFocus = (m.detailFocus + 1) % len(m.detailInputs)
				return m, m.updateDetailFocus()
			} else if m.relatedExpanded {
				// Navigate through related items
				maxCursor := len(m.childItems)
				if m.parentItem != nil {
					maxCursor++ // Account for parent
				}
				if maxCursor > 0 {
					m.relatedCursor = (m.relatedCursor + 1) % maxCursor
				}
			}
			return m, nil
		case "shift+tab", "up":
			if !m.commentsExpanded && !m.relatedExpanded {
				m.detailFocus--
				if m.detailFocus < 0 {
					m.detailFocus = len(m.detailInputs) - 1
				}
				return m, m.updateDetailFocus()
			} else if m.relatedExpanded {
				// Navigate through related items
				maxCursor := len(m.childItems)
				if m.parentItem != nil {
					maxCursor++ // Account for parent
				}
				if maxCursor > 0 {
					m.relatedCursor--
					if m.relatedCursor < 0 {
						m.relatedCursor = maxCursor - 1
					}
				}
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
			// If related items expanded, navigate to selected item
			if m.relatedExpanded {
				var targetItem *azdo.WorkItem
				if m.parentItem != nil {
					if m.relatedCursor == 0 {
						targetItem = m.parentItem
					} else if m.relatedCursor-1 < len(m.childItems) {
						targetItem = &m.childItems[m.relatedCursor-1]
					}
				} else if m.relatedCursor < len(m.childItems) {
					targetItem = &m.childItems[m.relatedCursor]
				}
				if targetItem != nil {
					return m.navigateToWorkItem(targetItem)
				}
				return m, nil
			}
			// If on comment field and there's text, add the comment
			if m.detailFocus == 4 && m.detailInputs[4].Value() != "" {
				m.loading = true
				return m, m.addComment(m.selectedItem.ID, m.detailInputs[4].Value())
			}
			return m, nil
		case "ctrl+r":
			// Toggle related items expanded/collapsed
			m.relatedExpanded = !m.relatedExpanded
			m.relatedCursor = 0
			// Auto-collapse other sections
			if m.relatedExpanded {
				m.commentsExpanded = false
				m.iterationExpanded = false
			}
			return m, nil
		case "d", "delete":
			// Remove the selected link when in related items view (only when related is expanded, otherwise let "d" pass through to input)
			if m.relatedExpanded && !m.creatingRelated && !m.confirmingDelete {
				var targetID int
				var isParent bool
				if m.parentItem != nil {
					if m.relatedCursor == 0 {
						// Removing parent link
						targetID = m.parentItem.ID
						isParent = true
					} else if m.relatedCursor-1 < len(m.childItems) {
						// Removing child link
						targetID = m.childItems[m.relatedCursor-1].ID
						isParent = false
					}
				} else if m.relatedCursor < len(m.childItems) {
					// No parent, removing child link
					targetID = m.childItems[m.relatedCursor].ID
					isParent = false
				}
				if targetID > 0 {
					// Start confirmation
					m.confirmingDelete = true
					m.confirmDeleteTargetID = targetID
					m.confirmDeleteIsParent = isParent
				}
				return m, nil
			}
		case "y":
			// Confirm delete (only when confirming)
			if m.confirmingDelete {
				m.loading = true
				m.confirmingDelete = false
				return m, m.removeLink(m.selectedItem.ID, m.confirmDeleteTargetID, m.confirmDeleteIsParent)
			}
		case "n":
			// Cancel delete confirmation (only when confirming, otherwise let "n" pass through to input)
			if m.confirmingDelete {
				m.confirmingDelete = false
				return m, nil
			}
		case "ctrl+e":
			// Toggle comments expanded/collapsed
			m.commentsExpanded = !m.commentsExpanded
			m.commentScroll = 0
			// Auto-collapse other sections
			if m.commentsExpanded {
				m.relatedExpanded = false
				m.iterationExpanded = false
			}
			return m, nil
		case "ctrl+n":
			// Scroll comments down when expanded, or create child when in related mode
			if m.commentsExpanded && m.commentScroll < len(m.comments)-1 {
				m.commentScroll++
			} else if m.relatedExpanded && !m.creatingRelated {
				// Start creating a child item
				m.creatingRelated = true
				m.createRelatedAsChild = true
				m.createRelatedTitle = ""
				m.createRelatedType = 0
				m.createRelatedAssignee = m.username
				m.createRelatedFocus = 0
			}
			return m, nil
		case "ctrl+p":
			// Scroll comments up when expanded, or create parent when in related mode
			if m.commentsExpanded && m.commentScroll > 0 {
				m.commentScroll--
			} else if m.relatedExpanded && !m.creatingRelated {
				// Start creating a parent item
				m.creatingRelated = true
				m.createRelatedAsChild = false
				m.createRelatedTitle = ""
				m.createRelatedType = 0
				m.createRelatedAssignee = m.username
				m.createRelatedFocus = 0
			}
			return m, nil
		case "ctrl+t":
			// Toggle iteration selection (ctrl+t for timeline/sprint)
			if !m.iterationExpanded {
				m.iterationExpanded = true
				m.iterationCursor = 0
				// Auto-collapse other sections
				m.commentsExpanded = false
				m.relatedExpanded = false
				m.planningExpanded = false
				// Find current iteration in list to set cursor
				for i, iter := range m.iterations {
					if iter.Path == m.selectedItem.Fields.IterationPath {
						m.iterationCursor = i
						break
					}
				}
				// Fetch iterations if not already loaded
				if len(m.iterations) == 0 {
					return m, m.fetchIterations()
				}
			} else {
				m.iterationExpanded = false
			}
			return m, nil
		case "ctrl+g":
			// Toggle planning section (ctrl+g for planning Goals/estimates)
			if !m.planningExpanded {
				m.planningExpanded = true
				m.planningFocus = 0
				// Auto-collapse other sections
				m.commentsExpanded = false
				m.relatedExpanded = false
				m.iterationExpanded = false
				// Fetch available planning fields for this work item type
				// and load current values into inputs
				if m.selectedItem != nil {
					return m, tea.Batch(
						m.fetchPlanningFields(m.selectedItem.Fields.WorkItemType),
						m.updatePlanningFocus(),
					)
				}
				return m, m.updatePlanningFocus()
			} else {
				m.planningExpanded = false
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

// navigateToWorkItem switches the detail view to a different work item
func (m Model) navigateToWorkItem(wi *azdo.WorkItem) (tea.Model, tea.Cmd) {
	m.selectedItem = wi
	m.detailInputs[0].SetValue(wi.Fields.Title)
	m.detailInputs[1].SetValue(wi.Fields.State)
	if wi.Fields.AssignedTo != nil {
		m.detailInputs[2].SetValue(wi.Fields.AssignedTo.UniqueName)
	} else {
		m.detailInputs[2].SetValue("")
	}
	m.detailInputs[3].SetValue(wi.Fields.Tags)
	m.detailInputs[4].SetValue("")
	m.comments = nil
	m.parentItem = nil
	m.childItems = nil
	m.relatedExpanded = false
	m.relatedCursor = 0
	m.commentsExpanded = false
	m.commentScroll = 0
	m.iterationExpanded = false
	m.iterationCursor = 0
	m.detailFocus = 0
	m.err = nil
	m.message = ""

	// Fetch comments and related items for the new work item
	return m, tea.Batch(m.fetchComments(wi.ID), m.fetchRelatedItems(wi.ID))
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

	// Iteration section
	iterationHeaderStyle := labelStyle.Copy()
	if m.iterationExpanded {
		iterationHeaderStyle = iterationHeaderStyle.Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229"))
	}

	if m.iterationExpanded {
		b.WriteString(iterationHeaderStyle.Render("▼ Iteration"))
		b.WriteString(" ")
		b.WriteString(hintStyle.Render("(ctrl+t: collapse, ↑↓: select, enter: set)"))
	} else {
		b.WriteString(labelStyle.Render("▶ Iteration"))
		b.WriteString(" ")
		b.WriteString(hintStyle.Render("(ctrl+t: change)"))
	}
	b.WriteString("\n")

	if !m.iterationExpanded {
		// Show current iteration
		iterPath := wi.Fields.IterationPath
		if iterPath == "" {
			iterPath = "(none)"
		}
		b.WriteString(detailStyle.Render(iterPath))
		b.WriteString("\n")
	} else {
		// Show iteration dropdown
		iterItemStyle := lipgloss.NewStyle().Padding(0, 1)
		selectedIterStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

		if len(m.iterations) == 0 {
			b.WriteString(detailStyle.Render("Loading iterations..."))
			b.WriteString("\n")
		} else {
			displayOrder := m.getIterationDisplayOrder()
			for displayIdx, iter := range displayOrder {
				style := iterItemStyle
				if m.iterationCursor == displayIdx {
					style = selectedIterStyle
				}
				// Mark current iteration
				marker := "  "
				if iter.Path == wi.Fields.IterationPath {
					marker = "✓ "
				}
				// Show timeframe if available
				timeFrame := ""
				if iter.Attributes != nil && iter.Attributes.TimeFrame != "" {
					timeFrame = fmt.Sprintf(" [%s]", iter.Attributes.TimeFrame)
				}
				b.WriteString(style.Render(fmt.Sprintf("%s%s%s", marker, iter.Name, timeFrame)))
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")

	// Related items section (parent/children)
	relatedHeaderStyle := labelStyle.Copy()
	if m.relatedExpanded {
		relatedHeaderStyle = relatedHeaderStyle.Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229"))
	}

	relatedCount := len(m.childItems)
	if m.parentItem != nil {
		relatedCount++
	}

	if m.relatedExpanded {
		b.WriteString(relatedHeaderStyle.Render(fmt.Sprintf("▼ Related Items (%d)", relatedCount)))
		b.WriteString(" ")
		b.WriteString(hintStyle.Render("(ctrl+r: collapse, ↑↓: select, enter: open)"))
	} else {
		b.WriteString(labelStyle.Render(fmt.Sprintf("▶ Related Items (%d)", relatedCount)))
		b.WriteString(" ")
		b.WriteString(hintStyle.Render("(ctrl+r: expand)"))
	}
	b.WriteString("\n")

	if relatedCount == 0 && !m.relatedExpanded {
		b.WriteString(detailStyle.Render("No parent or child items"))
		b.WriteString("\n")
	} else if !m.relatedExpanded {
		// Collapsed: show summary
		var summaryParts []string
		if m.parentItem != nil {
			summaryParts = append(summaryParts, fmt.Sprintf("Parent: %s #%d", m.parentItem.Fields.WorkItemType, m.parentItem.ID))
		}
		if len(m.childItems) > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("%d children", len(m.childItems)))
		}
		b.WriteString(detailStyle.Render(strings.Join(summaryParts, " • ")))
		b.WriteString("\n")
	} else {
		// Expanded: show all related items
		relatedItemStyle := lipgloss.NewStyle().
			Padding(0, 1)
		selectedRelatedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

		cursorIdx := 0
		if m.parentItem != nil {
			style := relatedItemStyle
			if m.relatedCursor == cursorIdx {
				style = selectedRelatedStyle
			}
			parentInfo := fmt.Sprintf("⬆ Parent: %s #%d - %s [%s]",
				m.parentItem.Fields.WorkItemType,
				m.parentItem.ID,
				truncateString(m.parentItem.Fields.Title, 40),
				m.parentItem.Fields.State)
			b.WriteString(style.Render(parentInfo))
			b.WriteString("\n")
			cursorIdx++
		}

		for i, child := range m.childItems {
			style := relatedItemStyle
			if m.relatedCursor == cursorIdx+i {
				style = selectedRelatedStyle
			}
			childInfo := fmt.Sprintf("⬇ Child: %s #%d - %s [%s]",
				child.Fields.WorkItemType,
				child.ID,
				truncateString(child.Fields.Title, 40),
				child.Fields.State)
			b.WriteString(style.Render(childInfo))
			b.WriteString("\n")
		}

		// Show message when no related items exist (but section is expanded)
		if relatedCount == 0 && !m.creatingRelated {
			b.WriteString(detailStyle.Render("No parent or child items - use ctrl+n to add child or ctrl+p to add parent"))
			b.WriteString("\n")
		}

		// Show create related form if active
		if m.creatingRelated {
			b.WriteString("\n")
			createFormStyle := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("39")).
				Padding(0, 1)
			relationType := "Child"
			if !m.createRelatedAsChild {
				relationType = "Parent"
			}
			wiType := "Task"
			if m.createRelatedType < len(m.workItemTypes) {
				wiType = m.workItemTypes[m.createRelatedType]
			}
			// Show cursor indicator on focused field
			titleCursor := ""
			assigneeCursor := ""
			if m.createRelatedFocus == 0 {
				titleCursor = "_"
			} else {
				assigneeCursor = "_"
			}
			formContent := fmt.Sprintf("Create New %s (%s)\nTitle: %s%s\nAssigned To: %s%s\n\n←/→: change type • tab: switch field",
				relationType, wiType, m.createRelatedTitle, titleCursor, m.createRelatedAssignee, assigneeCursor)
			b.WriteString(createFormStyle.Render(formContent))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	// Comments section
	commentHeaderStyle := labelStyle.Copy()
	if m.commentsExpanded {
		commentHeaderStyle = commentHeaderStyle.Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229"))
	}

	if m.commentsExpanded {
		b.WriteString(commentHeaderStyle.Render(fmt.Sprintf("▼ Comments (%d)", len(m.comments))))
		b.WriteString(" ")
		b.WriteString(hintStyle.Render("(ctrl+e: collapse, ctrl+n/p: scroll)"))
	} else {
		b.WriteString(labelStyle.Render(fmt.Sprintf("▶ Comments (%d)", len(m.comments))))
		b.WriteString(" ")
		b.WriteString(hintStyle.Render("(ctrl+e: expand)"))
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

		// Get the organization URL for mention links
		orgURL := ""
		if m.client != nil {
			orgURL = fmt.Sprintf("https://dev.azure.com/%s", m.client.Organization)
		}

		for i := start; i < end; i++ {
			c := m.comments[i]
			dateStr := ""
			if t, err := time.Parse(time.RFC3339, c.CreatedDate); err == nil {
				dateStr = t.Format("Jan 02, 15:04")
			}
			header := fmt.Sprintf("%s - %s", c.CreatedBy.DisplayName, dateStr)
			// Process mentions and strip HTML tags
			text := stripHTMLTags(c.Text, orgURL)
			if len(text) > 200 {
				text = text[:197] + "..."
			}
			b.WriteString(commentStyle.Render(fmt.Sprintf("%s\n%s", header, text)))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	// Planning section
	planningHeaderStyle := labelStyle.Copy()
	if m.planningExpanded {
		planningHeaderStyle = planningHeaderStyle.Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229"))
	}

	if m.planningExpanded {
		b.WriteString(planningHeaderStyle.Render("▼ Planning"))
		b.WriteString(" ")
		b.WriteString(hintStyle.Render("(ctrl+g: collapse, ↑↓: navigate, enter: save)"))
	} else {
		b.WriteString(labelStyle.Render("▶ Planning"))
		b.WriteString(" ")
		b.WriteString(hintStyle.Render("(ctrl+g: edit)"))
	}
	b.WriteString("\n")

	if !m.planningExpanded {
		// Collapsed: show summary of planning values
		var planningParts []string
		if wi.Fields.StoryPoints != nil {
			planningParts = append(planningParts, fmt.Sprintf("Story Points: %.1f", *wi.Fields.StoryPoints))
		}
		if wi.Fields.OriginalEstimate != nil {
			planningParts = append(planningParts, fmt.Sprintf("Original: %.1fh", *wi.Fields.OriginalEstimate))
		}
		if wi.Fields.RemainingWork != nil {
			planningParts = append(planningParts, fmt.Sprintf("Remaining: %.1fh", *wi.Fields.RemainingWork))
		}
		if wi.Fields.CompletedWork != nil {
			planningParts = append(planningParts, fmt.Sprintf("Completed: %.1fh", *wi.Fields.CompletedWork))
		}
		if len(planningParts) == 0 {
			b.WriteString(detailStyle.Render("No planning data"))
		} else {
			b.WriteString(detailStyle.Render(strings.Join(planningParts, " • ")))
		}
		b.WriteString("\n")
	} else {
		// Expanded: show editable planning fields dynamically
		if len(m.planningFields) == 0 {
			b.WriteString(detailStyle.Render("Loading available planning fields..."))
			b.WriteString("\n")
		} else {
			for i, field := range m.planningFields {
				if i >= len(m.planningInputs) {
					break
				}
				style := detailStyle
				if i == m.planningFocus {
					style = lipgloss.NewStyle().Foreground(lipgloss.Color("229"))
				}
				b.WriteString(style.Render(field.DisplayName + ": "))
				b.WriteString(m.planningInputs[i].View())
				b.WriteString("\n")
			}
		}
		if len(m.planningFields) == 0 {
			b.WriteString(hintStyle.Render("(no planning fields available for this work item type)"))
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
		b.WriteString(helpStyle.Render("ctrl+e: collapse comments • ctrl+n/p: scroll • esc: back"))
	} else if m.iterationExpanded {
		b.WriteString(helpStyle.Render("ctrl+t: collapse • ↑↓: select • enter: set iteration • esc: back"))
	} else if m.creatingRelated {
		b.WriteString(helpStyle.Render("type title • ←/→: change type • enter: create • esc: cancel"))
	} else if m.confirmingDelete {
		confirmStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		b.WriteString(confirmStyle.Render(fmt.Sprintf("Remove link to #%d? (y/n)", m.confirmDeleteTargetID)))
	} else if m.relatedExpanded {
		b.WriteString(helpStyle.Render("ctrl+r: collapse • ctrl+n: new child • ctrl+p: new parent • d: remove link • ↑↓: select • enter: open • esc: back"))
	} else if m.planningExpanded {
		b.WriteString(helpStyle.Render("ctrl+g: collapse • ↑↓: navigate • enter: save • esc: back"))
	} else {
		b.WriteString(helpStyle.Render("tab/↑↓: navigate • ctrl+s: save • ctrl+t: iteration • ctrl+e: comments • ctrl+r: related • ctrl+g: planning • esc: back"))
	}

	return boxStyle.Render(b.String())
}

// truncateString truncates a string to the specified length, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// updatePlanningFocus updates which planning input has focus
func (m *Model) updatePlanningFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.planningInputs))
	for i := range m.planningInputs {
		if i == m.planningFocus {
			cmds[i] = m.planningInputs[i].Focus()
		} else {
			m.planningInputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

// updatePlanningInputs updates all planning inputs with the given message
func (m *Model) updatePlanningInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.planningInputs))
	for i := range m.planningInputs {
		m.planningInputs[i], cmds[i] = m.planningInputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

// savePlanningFieldsDynamic parses and saves the planning fields dynamically based on available fields
func (m *Model) savePlanningFieldsDynamic() tea.Cmd {
	if len(m.planningFields) == 0 {
		return nil
	}

	fields := make(map[string]float64)

	// Parse each field based on the dynamic field definitions
	for i, field := range m.planningFields {
		if i >= len(m.planningInputs) {
			break
		}
		if v := m.planningInputs[i].Value(); v != "" {
			var f float64
			if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
				fields[field.ReferenceName] = f
			}
		}
	}

	// Only update if at least one field has a value
	if len(fields) == 0 {
		return nil
	}

	m.loading = true
	return m.updatePlanningDynamic(m.selectedItem.ID, fields)
}

// savePlanningFields parses and saves the planning fields
func (m *Model) savePlanningFields() tea.Cmd {
	var storyPoints, originalEstimate, remainingWork, completedWork *float64

	// Parse Story Points
	if v := m.planningInputs[0].Value(); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			storyPoints = &f
		}
	}

	// Parse Original Estimate
	if v := m.planningInputs[1].Value(); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			originalEstimate = &f
		}
	}

	// Parse Remaining Work
	if v := m.planningInputs[2].Value(); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			remainingWork = &f
		}
	}

	// Parse Completed Work
	if v := m.planningInputs[3].Value(); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			completedWork = &f
		}
	}

	// Only update if at least one field has a value
	if storyPoints == nil && originalEstimate == nil && remainingWork == nil && completedWork == nil {
		return nil
	}

	m.loading = true
	return m.updatePlanning(m.selectedItem.ID, storyPoints, originalEstimate, remainingWork, completedWork)
}

// getIterationDisplayOrder returns iterations with current iteration first
func (m Model) getIterationDisplayOrder() []azdo.Iteration {
	if len(m.iterations) == 0 || m.selectedItem == nil {
		return m.iterations
	}

	currentPath := m.selectedItem.Fields.IterationPath
	var currentIdx = -1

	// Find current iteration index
	for i, iter := range m.iterations {
		if iter.Path == currentPath {
			currentIdx = i
			break
		}
	}

	// If current iteration not found, return original order
	if currentIdx < 0 {
		return m.iterations
	}

	// Build reordered list with current first
	result := make([]azdo.Iteration, 0, len(m.iterations))
	result = append(result, m.iterations[currentIdx])
	for i, iter := range m.iterations {
		if i != currentIdx {
			result = append(result, iter)
		}
	}

	return result
}
