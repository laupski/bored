package tui

import (
	"bytes"
	"testing"
	"time"

	"github.com/laupski/bored/azdo"

	tea "github.com/charmbracelet/bubbletea"
)

// Integration tests simulate user interactions through the full TUI flow
// These tests verify end-to-end behavior without mocking

func TestFullConfigToBoard(t *testing.T) {
	// This test simulates filling in config and connecting
	m := NewModel()

	// Verify starting in config view
	if m.view != ViewConfig {
		t.Fatalf("Should start in config view, got %v", m.view)
	}

	// Set config values (simulating user input)
	m.configInputs[0].SetValue("testorg")
	m.configInputs[1].SetValue("testproject")
	m.configInputs[2].SetValue("testteam")
	m.configInputs[4].SetValue("testpat")
	m.configInputs[5].SetValue("test@example.com")

	// Verify values are set
	if m.configInputs[0].Value() != "testorg" {
		t.Error("Organization should be set")
	}

	// Simulate successful connection message
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "testpat")
	newModel, _ := m.Update(connectMsg{err: nil})
	m = newModel.(Model)

	// Should now be in board view
	if m.view != ViewBoard {
		t.Errorf("After connect, should be in board view, got %v", m.view)
	}
}

func TestBoardNavigationFlow(t *testing.T) {
	m := setupBoardModel()

	// Initial cursor should be 0
	if m.cursor != 0 {
		t.Errorf("Initial cursor = %v, want 0", m.cursor)
	}

	// Move cursor down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)
	if m.cursor != 1 {
		t.Errorf("After down, cursor = %v, want 1", m.cursor)
	}

	// Move cursor up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)
	if m.cursor != 0 {
		t.Errorf("After up, cursor = %v, want 0", m.cursor)
	}
}

func TestBoardToDetailFlow(t *testing.T) {
	m := setupBoardModel()

	// Select first item with Enter
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	// Should now be in detail view
	if m.view != ViewDetail {
		t.Errorf("After Enter, should be in detail view, got %v", m.view)
	}

	// Selected item should be set
	if m.selectedItem == nil {
		t.Error("selectedItem should be set after selecting from board")
	} else if m.selectedItem.ID != 1 {
		t.Errorf("selectedItem.ID = %v, want 1", m.selectedItem.ID)
	}
}

func TestDetailEscapeFlow(t *testing.T) {
	m := setupDetailModel()

	// Should be in detail view
	if m.view != ViewDetail {
		t.Fatalf("Should start in detail view, got %v", m.view)
	}

	// Press ESC
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)

	// Should return to board
	if m.view != ViewBoard {
		t.Errorf("After ESC, should be in board view, got %v", m.view)
	}
}

func TestCreateWorkItemFlow(t *testing.T) {
	m := setupBoardModel()

	// Press 'c' to create
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = newModel.(Model)

	// Should be in create view
	if m.view != ViewCreate {
		t.Errorf("After 'c', should be in create view, got %v", m.view)
	}

	// Fill in fields
	m.createInputs[0].SetValue("New Bug Title")
	m.createInputs[1].SetValue("Bug description")
	m.createInputs[2].SetValue("2")

	// Verify fields are set
	if m.createInputs[0].Value() != "New Bug Title" {
		t.Error("Title should be set")
	}

	// ESC should return to board
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.view != ViewBoard {
		t.Errorf("After ESC from create, should be in board view, got %v", m.view)
	}
}

func TestDetailCommentsToggleFlow(t *testing.T) {
	m := setupDetailModel()
	m.comments = []azdo.Comment{
		{ID: 1, Text: "Test comment"},
	}

	// Initially comments not expanded
	if m.commentsExpanded {
		t.Error("Comments should not be expanded initially")
	}

	// Toggle with Ctrl+E
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = newModel.(Model)

	if !m.commentsExpanded {
		t.Error("Comments should be expanded after Ctrl+E")
	}

	// Toggle again
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = newModel.(Model)

	if m.commentsExpanded {
		t.Error("Comments should be collapsed after second Ctrl+E")
	}
}

func TestDetailRelatedItemsFlow(t *testing.T) {
	m := setupDetailModel()
	m.parentItem = &azdo.WorkItem{
		ID:     100,
		Fields: azdo.WorkItemFields{Title: "Parent"},
	}
	m.childItems = []azdo.WorkItem{
		{ID: 101, Fields: azdo.WorkItemFields{Title: "Child"}},
	}

	// Initially related not expanded
	if m.relatedExpanded {
		t.Error("Related items should not be expanded initially")
	}

	// Toggle with Ctrl+R
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = newModel.(Model)

	if !m.relatedExpanded {
		t.Error("Related items should be expanded after Ctrl+R")
	}

	// Cursor should be at 0 (parent)
	if m.relatedCursor != 0 {
		t.Errorf("relatedCursor = %v, want 0", m.relatedCursor)
	}

	// Navigate down
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)

	if m.relatedCursor != 1 {
		t.Errorf("After down, relatedCursor = %v, want 1", m.relatedCursor)
	}
}

func TestDetailFieldNavigation(t *testing.T) {
	m := setupDetailModel()

	// Initially focus on field 0
	if m.detailFocus != 0 {
		t.Errorf("Initial detailFocus = %v, want 0", m.detailFocus)
	}

	// Tab to next field
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(Model)

	if m.detailFocus != 1 {
		t.Errorf("After Tab, detailFocus = %v, want 1", m.detailFocus)
	}

	// Tab through all fields (should wrap around)
	for i := 0; i < 5; i++ {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = newModel.(Model)
	}

	if m.detailFocus != 1 {
		t.Errorf("After wrapping, detailFocus = %v, want 1", m.detailFocus)
	}
}

func TestAsyncMessageHandling(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.loading = true

	// Simulate receiving work items
	items := []azdo.WorkItem{
		{ID: 1, Fields: azdo.WorkItemFields{Title: "Item 1"}},
		{ID: 2, Fields: azdo.WorkItemFields{Title: "Item 2"}},
	}
	newModel, _ := m.Update(workItemsMsg{items: items, err: nil})
	m = newModel.(Model)

	if m.loading {
		t.Error("loading should be false after receiving items")
	}
	if len(m.workItems) != 2 {
		t.Errorf("workItems length = %v, want 2", len(m.workItems))
	}
}

func TestErrorHandling(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.loading = true

	// Simulate error
	err := &testError{msg: "API error"}
	newModel, _ := m.Update(workItemsMsg{items: nil, err: err})
	m = newModel.(Model)

	if m.loading {
		t.Error("loading should be false after error")
	}
	if m.err == nil {
		t.Error("err should be set after error message")
	}
}

// Helper functions

func setupBoardModel() Model {
	m := NewModel()
	m.view = ViewBoard
	m.client = azdo.NewClient("testorg", "testproject", "", "", "testpat")
	m.workItems = []azdo.WorkItem{
		{
			ID: 1,
			Fields: azdo.WorkItemFields{
				Title:        "First Item",
				State:        "Active",
				WorkItemType: "Bug",
			},
		},
		{
			ID: 2,
			Fields: azdo.WorkItemFields{
				Title:        "Second Item",
				State:        "New",
				WorkItemType: "Task",
			},
		},
	}
	return m
}

func setupDetailModel() Model {
	m := setupBoardModel()
	m.view = ViewDetail
	m.selectedItem = &m.workItems[0]
	m.detailInputs[0].SetValue(m.selectedItem.Fields.Title)
	m.detailInputs[1].SetValue(m.selectedItem.Fields.State)
	return m
}

// TestModelProgram tests running the model in a simulated program
func TestModelProgram(t *testing.T) {
	m := NewModel()

	// Create a buffer to capture output
	var buf bytes.Buffer

	// Run initial render
	output := m.View()
	buf.WriteString(output)

	// Verify we got output
	if buf.Len() == 0 {
		t.Error("Should produce output on View()")
	}
}

// Benchmark tests
func BenchmarkModelView(b *testing.B) {
	m := setupDetailModel()
	m.comments = make([]azdo.Comment, 10)
	for i := range m.comments {
		m.comments[i] = azdo.Comment{
			ID:        i,
			Text:      "Comment text here",
			CreatedBy: azdo.IdentityRef{DisplayName: "User"},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkModelUpdate(b *testing.B) {
	m := setupBoardModel()
	msg := tea.KeyMsg{Type: tea.KeyDown}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newModel, _ := m.Update(msg)
		m = newModel.(Model)
	}
}

// Timeout test to ensure operations complete quickly
func TestOperationsComplete(t *testing.T) {
	m := setupBoardModel()

	done := make(chan bool)
	go func() {
		// Run several operations
		for i := 0; i < 100; i++ {
			newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
			m = newModel.(Model)
			newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
			m = newModel.(Model)
			_ = m.View()
		}
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Error("Operations took too long")
	}
}
