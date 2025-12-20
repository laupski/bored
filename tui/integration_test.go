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

// ============ Additional Coverage Tests ============

func TestBoardNavigation(t *testing.T) {
	m := setupBoardModel()

	// Test down with 'j'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)
	if m.cursor != 1 {
		t.Errorf("After 'j', cursor = %v, want 1", m.cursor)
	}

	// Test up with 'k'
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)
	if m.cursor != 0 {
		t.Errorf("After 'k', cursor = %v, want 0", m.cursor)
	}

	// Test cursor doesn't go below 0
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)
	if m.cursor != 0 {
		t.Errorf("Cursor should not go below 0, got %v", m.cursor)
	}

	// Test cursor doesn't exceed items
	for i := 0; i < 10; i++ {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = newModel.(Model)
	}
	if m.cursor >= len(m.workItems) {
		t.Errorf("Cursor should not exceed workItems, got %v", m.cursor)
	}
}

func TestBoardPagination(t *testing.T) {
	m := setupBoardModel()

	// Test left/right pagination keys (requires hasMoreData or apiPage)
	m.hasMoreData = true
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = newModel.(Model)
	if !m.loading {
		t.Error("Right key with hasMoreData should trigger loading")
	}

	// Reset and test left with no previous page
	m = setupBoardModel()
	m.apiPage = 0
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = newModel.(Model)
	// Should not change page when already at first page

	// Test left with previous page available
	m.apiPage = 1
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = newModel.(Model)
	if !m.loading {
		t.Error("Left key with previous page should trigger loading")
	}
}

func TestBoardHomeEndKeys(t *testing.T) {
	m := setupBoardModel()
	m.cursor = 1

	// Test home key (at first page)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyHome})
	m = newModel.(Model)
	if m.cursor != 0 {
		t.Errorf("Home key should move cursor to 0, got %v", m.cursor)
	}

	// Test end key
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = newModel.(Model)
	if m.cursor != len(m.workItems)-1 {
		t.Errorf("End key should move cursor to last item, got %v", m.cursor)
	}
}

func TestBoardRefresh(t *testing.T) {
	m := setupBoardModel()
	m.err = &testError{msg: "Old error"}

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = newModel.(Model)

	if !m.loading {
		t.Error("Refresh should set loading to true")
	}
	if m.err != nil {
		t.Error("Refresh should clear error")
	}
}

func TestBoardCreateKeys(t *testing.T) {
	m := setupBoardModel()

	// Test 'n' key for create
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newModel.(Model)

	if m.view != ViewCreate {
		t.Errorf("'n' should switch to create view, got %v", m.view)
	}
	if m.createFocus != 0 {
		t.Error("Create focus should start at 0")
	}
}

func TestBoardDeleteFlow(t *testing.T) {
	m := setupBoardModel()

	// Start delete
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newModel.(Model)

	if !m.deletingWorkItem {
		t.Error("'d' should start delete confirmation")
	}
	if m.deleteWorkItemID != m.workItems[0].ID {
		t.Error("Delete should target selected item")
	}

	// Cancel with escape
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)

	if m.deletingWorkItem {
		t.Error("ESC should cancel delete")
	}

	// Start delete again
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newModel.(Model)

	// Type wrong title
	m.deleteConfirmInput = "wrong title"
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if m.err == nil {
		t.Error("Wrong title should set error")
	}
	if m.deletingWorkItem {
		t.Error("Wrong title should exit delete mode")
	}

	// Test backspace in delete mode
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newModel.(Model)
	m.deleteConfirmInput = "test"

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(Model)
	if m.deleteConfirmInput != "tes" {
		t.Errorf("Backspace should remove last char, got %s", m.deleteConfirmInput)
	}

	// Test typing in delete mode
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = newModel.(Model)
	if m.deleteConfirmInput != "tesa" {
		t.Errorf("Should add char, got %s", m.deleteConfirmInput)
	}

	// Test space in delete mode
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = newModel.(Model)
	if m.deleteConfirmInput != "tesa " {
		t.Errorf("Space should add space, got %s", m.deleteConfirmInput)
	}
}

func TestBoardQuit(t *testing.T) {
	m := setupBoardModel()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if cmd == nil {
		t.Error("'q' should return quit command")
	}
}

func TestBoardNotificationClear(t *testing.T) {
	m := setupBoardModel()
	m.notifyMessage = "Test notification"

	// Any key should clear notification
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)

	if m.notifyMessage != "" {
		t.Error("Key press should clear notification")
	}
}

func TestConfigNavigation(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig

	// Test shift+tab navigation
	m.configFocus = 1
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = newModel.(Model)
	if m.configFocus != 0 {
		t.Errorf("Shift+tab should decrease focus, got %v", m.configFocus)
	}

	// Test wrap around
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = newModel.(Model)
	if m.configFocus != len(m.configInputs)-1 {
		t.Errorf("Shift+tab should wrap to last, got %v", m.configFocus)
	}
}

func TestConfigConnect(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig

	// Set required values
	m.configInputs[0].SetValue("org")
	m.configInputs[1].SetValue("proj")
	m.configInputs[2].SetValue("team")
	m.configInputs[3].SetValue("area")
	m.configInputs[4].SetValue("pat")
	m.configInputs[5].SetValue("user")

	// Press enter to connect
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if m.client == nil {
		t.Error("Enter should create client")
	}
	if !m.loading {
		t.Error("Enter should set loading")
	}
	if m.username != "user" {
		t.Errorf("Username should be set, got %s", m.username)
	}
}

func TestConfigOpenSettings(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig

	// Press ctrl+f to open settings
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	m = newModel.(Model)

	if m.view != ViewConfigFile {
		t.Errorf("Ctrl+F should switch to config file view, got %v", m.view)
	}
}

func TestCreateNavigation(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate

	// Test tab
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(Model)
	if m.createFocus != 1 {
		t.Errorf("Tab should advance focus, got %v", m.createFocus)
	}

	// Test shift+tab
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = newModel.(Model)
	if m.createFocus != 0 {
		t.Errorf("Shift+tab should go back, got %v", m.createFocus)
	}

	// Wrap around backwards
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = newModel.(Model)
	if m.createFocus != len(m.createInputs) {
		t.Errorf("Shift+tab should wrap, got %v", m.createFocus)
	}
}

func TestCreateTypeSelector(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate
	m.createFocus = len(m.createInputs) // Focus on type selector

	// Test right arrow
	startType := m.createType
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = newModel.(Model)
	expectedType := (startType + 1) % len(m.workItemTypes)
	if m.createType != expectedType {
		t.Errorf("Right should advance type, expected %d, got %d", expectedType, m.createType)
	}

	// Test left arrow
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = newModel.(Model)
	if m.createType != startType {
		t.Errorf("Left should go back to start type %d, got %d", startType, m.createType)
	}

	// Test left wrap around
	m.createType = 0
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = newModel.(Model)
	if m.createType != len(m.workItemTypes)-1 {
		t.Errorf("Left should wrap, got %d", m.createType)
	}
}

func TestCreateSubmit(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	// Empty title should not submit
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if m.loading {
		t.Error("Empty title should not trigger submit")
	}

	// Set title and submit
	m.createInputs[0].SetValue("New Item")
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	if !m.loading {
		t.Error("Valid title should trigger submit")
	}
}

func TestDetailPlanningToggle(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	// Test Ctrl+P key - exercises the detail update handler
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	// The planning toggle behavior depends on internal state
}

func TestDetailIterationsToggle(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	// Test Ctrl+I key - exercises the detail update handler
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlI})
	// The iterations toggle behavior depends on internal state
}

func TestDetailHyperlinksToggle(t *testing.T) {
	m := setupDetailModel()

	// Toggle hyperlinks with Ctrl+L
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	m = newModel.(Model)

	if !m.hyperlinksExpanded {
		t.Error("Ctrl+L should expand hyperlinks")
	}
}

func TestDetailAddCommentMode(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.commentsExpanded = true

	// Test 'n' key to start new comment in comments expanded mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newModel.(Model)

	// The model should handle the comment creation flow
	// This exercises the comments expanded keyboard handling
}

func TestDetailSave(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	// Save with Ctrl+S
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = newModel.(Model)

	if !m.loading {
		t.Error("Ctrl+S should trigger save")
	}
}

func TestUpdateConfigInputs(t *testing.T) {
	m := NewModel()

	// Call updateConfigInputs
	cmd := m.updateConfigInputs(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if cmd == nil {
		t.Error("updateConfigInputs should return a batch command")
	}
}

func TestUpdateCreateInputs(t *testing.T) {
	m := NewModel()

	// Call updateCreateInputs - exercises the function regardless of return value
	_ = m.updateCreateInputs(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
}

func TestUpdateDetailInputs(t *testing.T) {
	m := NewModel()

	// Call updateDetailInputs - exercises the function regardless of return value
	_ = m.updateDetailInputs(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
}

func TestUpdateConfigFileFocus(t *testing.T) {
	m := NewModel()
	m.configFileFocus = 0

	// Call updateConfigFileFocus - exercises the function
	_ = m.updateConfigFileFocus()
}

func TestUpdatePlanningFocus(t *testing.T) {
	m := NewModel()
	m.planningFocus = 0

	// Call updatePlanningFocus - exercises the function
	_ = m.updatePlanningFocus()
}

func TestUpdatePlanningInputs(t *testing.T) {
	m := NewModel()

	// Call updatePlanningInputs - exercises the function
	_ = m.updatePlanningInputs(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
}

func TestViewBoardWithNotification(t *testing.T) {
	m := setupBoardModel()
	m.notifyMessage = "Test notification"

	output := m.View()

	if output == "" {
		t.Error("View should produce output")
	}
}

func TestViewBoardWithDeleteConfirmation(t *testing.T) {
	m := setupBoardModel()
	m.deletingWorkItem = true
	m.deleteWorkItemID = 123
	m.deleteWorkItemTitle = "Test Title"
	m.deleteConfirmInput = "Test"

	output := m.View()

	if output == "" {
		t.Error("View should produce output")
	}
}

func TestViewBoardWithError(t *testing.T) {
	m := setupBoardModel()
	m.err = &testError{msg: "Test error"}

	output := m.View()

	if output == "" {
		t.Error("View should produce output")
	}
}

func TestViewBoardWithMessage(t *testing.T) {
	m := setupBoardModel()
	m.message = "Success message"

	output := m.View()

	if output == "" {
		t.Error("View should produce output")
	}
}

func TestViewCreateWithError(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate
	m.err = &testError{msg: "Test error"}

	output := m.View()

	if output == "" {
		t.Error("View should produce output")
	}
}

func TestViewCreateWithLoading(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate
	m.loading = true

	output := m.View()

	if output == "" {
		t.Error("View should produce output")
	}
}

func TestViewCreateWithAreaPath(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate
	m.client = azdo.NewClient("org", "proj", "", "Project\\Team", "pat")

	output := m.View()

	if output == "" {
		t.Error("View should produce output")
	}
}

func TestViewConfigWithLoading(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig
	m.loading = true

	output := m.View()

	if output == "" {
		t.Error("View should produce output")
	}
}

func TestViewConfigWithError(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig
	m.err = &testError{msg: "Test error"}

	output := m.View()

	if output == "" {
		t.Error("View should produce output")
	}
}

func TestStartNotificationTicker(t *testing.T) {
	m := NewModel()

	// Should return a command
	cmd := m.startNotificationTicker()

	if cmd == nil {
		t.Error("startNotificationTicker should return a command")
	}
}

func TestRelatedCursorNavigation(t *testing.T) {
	m := setupDetailModel()
	m.relatedExpanded = true
	m.parentItem = &azdo.WorkItem{ID: 100}
	m.childItems = []azdo.WorkItem{{ID: 101}}

	// Navigate down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)

	if m.relatedCursor != 1 {
		t.Errorf("Down should move related cursor, got %d", m.relatedCursor)
	}

	// Navigate up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)

	if m.relatedCursor != 0 {
		t.Errorf("Up should move related cursor back, got %d", m.relatedCursor)
	}
}

func TestIterationsCursorNavigation(t *testing.T) {
	m := setupDetailModel()
	m.iterationExpanded = true
	m.iterations = []azdo.Iteration{
		{ID: "1", Name: "Sprint 1"},
		{ID: "2", Name: "Sprint 2"},
	}

	// Navigate down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)

	if m.iterationCursor != 1 {
		t.Errorf("Down should move iteration cursor, got %d", m.iterationCursor)
	}
}

func TestHyperlinksCursorNavigation(t *testing.T) {
	m := setupDetailModel()
	m.hyperlinksExpanded = true
	m.hyperlinks = []azdo.Hyperlink{
		{URL: "https://example1.com"},
		{URL: "https://example2.com"},
	}

	// Navigate down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)

	if m.hyperlinkCursor != 1 {
		t.Errorf("Down should move hyperlink cursor, got %d", m.hyperlinkCursor)
	}
}

func TestCommentScrolling(t *testing.T) {
	m := setupDetailModel()
	m.commentsExpanded = true
	m.comments = make([]azdo.Comment, 20)
	for i := range m.comments {
		m.comments[i] = azdo.Comment{ID: i, Text: "Comment"}
	}

	// Test navigation in comments mode - exercises the code path
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	// The scroll behavior depends on display constraints
}

func TestFetchWorkItemsCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	cmd := m.fetchWorkItems()
	if cmd == nil {
		t.Error("fetchWorkItems should return a command")
	}
}

func TestFetchWorkItemsPageCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	cmd := m.fetchWorkItemsPage(0)
	if cmd == nil {
		t.Error("fetchWorkItemsPage should return a command")
	}
}

func TestFetchWorkItemTypesCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	cmd := m.fetchWorkItemTypes()
	if cmd == nil {
		t.Error("fetchWorkItemTypes should return a command")
	}
}

func TestAddCommentCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	cmd := m.addComment(123, "Test comment")
	if cmd == nil {
		t.Error("addComment should return a command")
	}
}

func TestUpdateWorkItemCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.selectedItem = &azdo.WorkItem{ID: 123, Fields: azdo.WorkItemFields{Title: "Test"}}

	cmd := m.updateWorkItem(123, "Title", "Active", "user@example.com", "tag1")
	if cmd == nil {
		t.Error("updateWorkItem should return a command")
	}
}

func TestCreateWorkItemCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.createInputs[0].SetValue("Title")
	m.createInputs[1].SetValue("Desc")
	m.createInputs[2].SetValue("2")

	cmd := m.createWorkItem()
	if cmd == nil {
		t.Error("createWorkItem should return a command")
	}
}

func TestRemoveLinkCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.selectedItem = &azdo.WorkItem{ID: 123}

	cmd := m.removeLink(123, 456, true)
	if cmd == nil {
		t.Error("removeLink should return a command")
	}
}

func TestDeleteWorkItemCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	cmd := m.deleteWorkItem(123)
	if cmd == nil {
		t.Error("deleteWorkItem should return a command")
	}
}

func TestFetchIterationsCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	cmd := m.fetchIterations()
	if cmd == nil {
		t.Error("fetchIterations should return a command")
	}
}

func TestUpdateIterationCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.selectedItem = &azdo.WorkItem{ID: 123}

	cmd := m.updateIteration(123, "Project\\Sprint 1")
	if cmd == nil {
		t.Error("updateIteration should return a command")
	}
}

func TestFetchPlanningFieldsCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.selectedItem = &azdo.WorkItem{ID: 123, Fields: azdo.WorkItemFields{WorkItemType: "Bug"}}

	cmd := m.fetchPlanningFields("Bug")
	if cmd == nil {
		t.Error("fetchPlanningFields should return a command")
	}
}

func TestUpdatePlanningDynamicCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.selectedItem = &azdo.WorkItem{ID: 123}
	m.planningFields = []azdo.PlanningField{
		{ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints", DisplayName: "Story Points"},
	}
	m.planningInputs[0].SetValue("5")

	fields := map[string]float64{"Microsoft.VSTS.Scheduling.StoryPoints": 5.0}
	cmd := m.updatePlanningDynamic(123, fields)
	if cmd == nil {
		t.Error("updatePlanningDynamic should return a command")
	}
}

func TestAddHyperlinkCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.selectedItem = &azdo.WorkItem{ID: 123}

	cmd := m.addHyperlink(123, "https://example.com", "comment")
	if cmd == nil {
		t.Error("addHyperlink should return a command")
	}
}

func TestRemoveHyperlinkCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.selectedItem = &azdo.WorkItem{ID: 123}

	cmd := m.removeHyperlink(123, "https://example.com")
	if cmd == nil {
		t.Error("removeHyperlink should return a command")
	}
}

func TestCheckForChangesCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.username = "user@example.com"

	cmd := m.checkForChanges()
	if cmd == nil {
		t.Error("checkForChanges should return a command")
	}
}

func TestCreateRelatedItemCommand(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.selectedItem = &azdo.WorkItem{ID: 123}

	cmd := m.createRelatedItem(123, true, "Child Title", "Task", "user@example.com")
	if cmd == nil {
		t.Error("createRelatedItem should return a command")
	}
}

// ============ Additional Detail View Tests ============

func TestDetailViewNavigation(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	// Test all detail navigation keys
	keys := []tea.KeyMsg{
		{Type: tea.KeyTab},
		{Type: tea.KeyShiftTab},
		{Type: tea.KeyUp},
		{Type: tea.KeyDown},
		{Type: tea.KeyLeft},
		{Type: tea.KeyRight},
		{Type: tea.KeyHome},
		{Type: tea.KeyEnd},
		{Type: tea.KeyPgUp},
		{Type: tea.KeyPgDown},
	}

	for _, key := range keys {
		newModel, _ := m.Update(key)
		_ = newModel.(Model)
	}
}

func TestDetailViewModeToggles(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	// Test all toggle keys
	toggleKeys := []tea.KeyMsg{
		{Type: tea.KeyCtrlE}, // Comments
		{Type: tea.KeyCtrlR}, // Related
		{Type: tea.KeyCtrlP}, // Planning
		{Type: tea.KeyCtrlI}, // Iterations
		{Type: tea.KeyCtrlL}, // Hyperlinks
	}

	for _, key := range toggleKeys {
		newModel, _ := m.Update(key)
		_ = newModel.(Model)
	}
}

func TestDetailRelatedItemActions(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.relatedExpanded = true
	m.parentItem = &azdo.WorkItem{ID: 100, Fields: azdo.WorkItemFields{Title: "Parent"}}
	m.childItems = []azdo.WorkItem{
		{ID: 101, Fields: azdo.WorkItemFields{Title: "Child"}},
	}

	// Navigate and select
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
}

func TestDetailHyperlinkActions(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.hyperlinksExpanded = true
	m.hyperlinks = []azdo.Hyperlink{
		{URL: "https://example.com", Comment: "Test"},
	}

	// Navigate and select
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
}

func TestDetailIterationActions(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.iterationExpanded = true
	m.iterations = []azdo.Iteration{
		{ID: "1", Name: "Sprint 1", Path: "Project\\Sprint 1"},
	}

	// Select iteration
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
}

func TestDetailPlanningNavigation(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.planningExpanded = true
	m.planningFields = []azdo.PlanningField{
		{ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints", DisplayName: "Story Points"},
	}

	// Navigate planning fields
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
}

// ============ ConfigFile View Tests ============

func TestViewConfigFile(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile

	output := m.View()
	if output == "" {
		t.Error("ViewConfigFile should produce output")
	}
}

func TestUpdateConfigFileNavigation(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile

	// Test navigation
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
}

func TestUpdateConfigFileSave(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile

	// Set max work items value and try to save
	m.configFileInputs[0].SetValue("100")
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
}

// ============ Message Handler Tests ============

func TestIterationsMsgHandler(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 123}

	iterations := []azdo.Iteration{
		{ID: "1", Name: "Sprint 1", Path: "Project\\Sprint 1"},
	}
	msg := iterationsMsg{iterations: iterations, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if len(updated.iterations) != 1 {
		t.Errorf("Expected 1 iteration, got %d", len(updated.iterations))
	}
}

func TestPlanningFieldsMsgHandler(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 123}

	fields := []azdo.PlanningField{
		{ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints", DisplayName: "Story Points"},
	}
	msg := planningFieldsMsg{fields: fields, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if len(updated.planningFields) != 1 {
		t.Errorf("Expected 1 planning field, got %d", len(updated.planningFields))
	}
}

func TestUpdatePlanningMsgHandler(t *testing.T) {
	sp := 5.0
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 123}

	wi := &azdo.WorkItem{ID: 123, Fields: azdo.WorkItemFields{StoryPoints: &sp}}
	msg := updatePlanningMsg{item: wi, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.selectedItem.Fields.StoryPoints == nil {
		t.Error("StoryPoints should be set")
	}
}

func TestHyperlinksMsgHandler(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 123}

	links := []azdo.Hyperlink{
		{URL: "https://example.com"},
	}
	msg := hyperlinksMsg{hyperlinks: links, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if len(updated.hyperlinks) != 1 {
		t.Errorf("Expected 1 hyperlink, got %d", len(updated.hyperlinks))
	}
}

func TestAddHyperlinkMsgHandler(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 123}
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	msg := addHyperlinkMsg{err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.message == "" {
		t.Error("Should set success message")
	}
}

func TestRemoveHyperlinkMsgHandler(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 123}
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	msg := removeHyperlinkMsg{err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.message == "" {
		t.Error("Should set success message")
	}
}

func TestUpdateIterationMsgHandler(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 123}

	wi := &azdo.WorkItem{ID: 123, Fields: azdo.WorkItemFields{IterationPath: "Project\\Sprint 2"}}
	msg := updateIterationMsg{item: wi, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.selectedItem.Fields.IterationPath != "Project\\Sprint 2" {
		t.Errorf("Expected 'Project\\Sprint 2', got %s", updated.selectedItem.Fields.IterationPath)
	}
}

func TestWorkItemTypesMsgHandler(t *testing.T) {
	m := NewModel()

	types := []string{"Bug", "Task"}
	msg := workItemTypesMsg{types: types, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if len(updated.workItemTypes) != 2 {
		t.Errorf("Expected 2 types, got %d", len(updated.workItemTypes))
	}
}

func TestCreateResultMsgHandler(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	wi := &azdo.WorkItem{ID: 123, Fields: azdo.WorkItemFields{Title: "New Item"}}
	msg := createResultMsg{item: wi, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.view != ViewBoard {
		t.Errorf("Should return to board view, got %v", updated.view)
	}
}

func TestUpdateWorkItemMsgHandler(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 123}

	wi := &azdo.WorkItem{ID: 123, Fields: azdo.WorkItemFields{Title: "Updated Title"}}
	msg := updateWorkItemMsg{item: wi, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.message == "" {
		t.Error("Should set success message")
	}
}

func TestDeleteWorkItemMsgHandler(t *testing.T) {
	m := setupBoardModel()
	m.deleteWorkItemID = 1

	msg := deleteWorkItemMsg{err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.message == "" {
		t.Error("Should set success message")
	}
}

func TestAddCommentMsgHandler(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 123}
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	msg := addCommentMsg{err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.message == "" {
		t.Error("Should set success message")
	}
}

func TestRemoveLinkMsgHandler(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 123}
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	msg := removeLinkMsg{err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.message == "" {
		t.Error("Should set success message")
	}
}

func TestCreateRelatedMsgHandler(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 123}
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	wi := &azdo.WorkItem{ID: 456, Fields: azdo.WorkItemFields{Title: "Child Item"}}
	msg := createRelatedMsg{item: wi, asChild: true, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.message == "" {
		t.Error("Should set success message")
	}
}

// ============ Error Handling Tests ============

func TestMsgErrorHandling(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 123}
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	// These messages set m.err on error
	testCases := []tea.Msg{
		updatePlanningMsg{err: &testError{msg: "Test error"}},
		addHyperlinkMsg{err: &testError{msg: "Test error"}},
		removeHyperlinkMsg{err: &testError{msg: "Test error"}},
		updateIterationMsg{err: &testError{msg: "Test error"}},
		updateWorkItemMsg{err: &testError{msg: "Test error"}},
		addCommentMsg{err: &testError{msg: "Test error"}},
		removeLinkMsg{err: &testError{msg: "Test error"}},
		createRelatedMsg{err: &testError{msg: "Test error"}},
	}

	for _, msg := range testCases {
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)
		if updated.err == nil {
			t.Errorf("Expected error to be set for %T", msg)
		}
	}

	// These messages don't set m.err (they silently ignore errors)
	silentErrorMsgs := []tea.Msg{
		iterationsMsg{err: &testError{msg: "Test error"}},
		planningFieldsMsg{err: &testError{msg: "Test error"}},
		hyperlinksMsg{err: &testError{msg: "Test error"}},
	}

	for _, msg := range silentErrorMsgs {
		_, _ = m.Update(msg) // Just ensure they don't panic
	}
}

func TestTickMsgHandler(t *testing.T) {
	m := NewModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.username = "user@example.com"
	m.view = ViewBoard

	msg := tickMsg(time.Now())
	_, cmd := m.Update(msg)

	// Should return a command (checkForChanges or next tick)
	_ = cmd
}

// ============ View Edge Cases ============

func TestViewDetailWithAllSections(t *testing.T) {
	m := setupDetailModel()
	m.commentsExpanded = true
	m.relatedExpanded = true
	m.planningExpanded = true
	m.iterationExpanded = true
	m.hyperlinksExpanded = true
	m.comments = []azdo.Comment{{ID: 1, Text: "Comment"}}
	m.parentItem = &azdo.WorkItem{ID: 100}
	m.childItems = []azdo.WorkItem{{ID: 101}}
	m.iterations = []azdo.Iteration{{ID: "1", Name: "Sprint 1"}}
	m.hyperlinks = []azdo.Hyperlink{{URL: "https://example.com"}}
	m.planningFields = []azdo.PlanningField{
		{ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints", DisplayName: "Story Points"},
	}

	output := m.View()
	if output == "" {
		t.Error("View should produce output")
	}
}

func TestViewDetailWithWorkItemTypeBadges(t *testing.T) {
	types := []string{"Bug", "Task", "User Story", "Feature", "Epic", "Issue"}

	for _, wiType := range types {
		m := setupDetailModel()
		m.selectedItem.Fields.WorkItemType = wiType
		output := m.View()
		if output == "" {
			t.Errorf("View should produce output for type %s", wiType)
		}
	}
}

// ============ Additional updateDetail Coverage Tests ============

func TestDetailCreateRelatedTypeSelection(t *testing.T) {
	m := setupDetailModel()
	m.relatedExpanded = true
	m.creatingRelated = true
	m.workItemTypes = []string{"Bug", "Task", "User Story"}
	m.createRelatedType = 0

	// Test right arrow to change type
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("right")})
	updated := newModel.(Model)
	if updated.createRelatedType != 1 {
		t.Errorf("Right arrow should increment type, got %d", updated.createRelatedType)
	}

	// Test left arrow to change type
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("left")})
	updated = newModel.(Model)
	if updated.createRelatedType != 0 {
		t.Errorf("Left arrow should decrement type, got %d", updated.createRelatedType)
	}

	// Test left arrow wrapping
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("left")})
	updated = newModel.(Model)
	if updated.createRelatedType != 2 {
		t.Errorf("Left arrow should wrap to end, got %d", updated.createRelatedType)
	}
}

func TestDetailCreateRelatedInputFields(t *testing.T) {
	m := setupDetailModel()
	m.relatedExpanded = true
	m.creatingRelated = true
	m.createRelatedFocus = 0
	m.createRelatedTitle = "Test"
	m.createRelatedAssignee = "user@test.com"
	m.workItemTypes = []string{"Bug", "Task"}

	// Test backspace on title
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updated := newModel.(Model)
	if updated.createRelatedTitle != "Tes" {
		t.Errorf("Backspace should remove last char from title, got %s", updated.createRelatedTitle)
	}

	// Test space on title
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	updated = newModel.(Model)

	// Test tab to switch to assignee field
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated = newModel.(Model)
	if updated.createRelatedFocus != 1 {
		t.Errorf("Tab should switch to assignee field, got focus %d", updated.createRelatedFocus)
	}

	// Test backspace on assignee
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updated = newModel.(Model)
	if updated.createRelatedAssignee != "user@test.co" {
		t.Errorf("Backspace should remove last char from assignee, got %s", updated.createRelatedAssignee)
	}

	// Test typing a character on assignee
	updated.createRelatedFocus = 1
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	updated = newModel.(Model)
	if updated.createRelatedAssignee != "user@test.com" {
		t.Errorf("Typing should add char to assignee")
	}

	// Test space on assignee field
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	updated = newModel.(Model)
}

func TestDetailCreateRelatedSubmit(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.relatedExpanded = true
	m.creatingRelated = true
	m.createRelatedTitle = "New Child Item"
	m.createRelatedAsChild = true
	m.workItemTypes = []string{"Bug", "Task"}
	m.createRelatedType = 0

	// Test enter to submit
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := newModel.(Model)
	if updated.creatingRelated {
		t.Error("Creating related should be false after submit")
	}
	if !updated.loading {
		t.Error("Should be loading after submit")
	}
}

func TestDetailAddHyperlinkInputFields(t *testing.T) {
	m := setupDetailModel()
	m.hyperlinksExpanded = true
	m.addingHyperlink = true
	m.hyperlinkFocus = 0
	m.hyperlinkURL = "https://example.com"
	m.hyperlinkComment = "Test comment"

	// Test backspace on URL
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updated := newModel.(Model)
	if updated.hyperlinkURL != "https://example.co" {
		t.Errorf("Backspace should remove last char from URL, got %s", updated.hyperlinkURL)
	}

	// Test tab to switch to comment field
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated = newModel.(Model)
	if updated.hyperlinkFocus != 1 {
		t.Errorf("Tab should switch to comment field, got focus %d", updated.hyperlinkFocus)
	}

	// Test backspace on comment
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updated = newModel.(Model)
	if updated.hyperlinkComment != "Test commen" {
		t.Errorf("Backspace should remove last char from comment, got %s", updated.hyperlinkComment)
	}

	// Test typing a character on comment
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	updated = newModel.(Model)

	// Test space on comment field
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	updated = newModel.(Model)

	// Test space on URL field
	updated.hyperlinkFocus = 0
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	updated = newModel.(Model)
}

func TestDetailAddHyperlinkSubmit(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.hyperlinksExpanded = true
	m.addingHyperlink = true
	m.hyperlinkURL = "https://github.com/example/repo/pull/1"
	m.hyperlinkComment = "PR Link"

	// Test enter to submit
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := newModel.(Model)
	if updated.addingHyperlink {
		t.Error("Adding hyperlink should be false after submit")
	}
	if !updated.loading {
		t.Error("Should be loading after submit")
	}
}

func TestDetailIterationSelection(t *testing.T) {
	m := setupDetailModel()
	m.iterationExpanded = true
	m.iterations = []azdo.Iteration{
		{ID: "1", Name: "Sprint 1", Path: "Project\\Sprint 1"},
		{ID: "2", Name: "Sprint 2", Path: "Project\\Sprint 2"},
		{ID: "3", Name: "Sprint 3", Path: "Project\\Sprint 3"},
	}
	m.iterationCursor = 0

	// Test down arrow
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := newModel.(Model)
	if updated.iterationCursor != 1 {
		t.Errorf("Down should increment cursor, got %d", updated.iterationCursor)
	}

	// Test up arrow
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated = newModel.(Model)
	if updated.iterationCursor != 0 {
		t.Errorf("Up should decrement cursor, got %d", updated.iterationCursor)
	}

	// Test up at top (shouldn't go negative)
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated = newModel.(Model)
	if updated.iterationCursor != 0 {
		t.Errorf("Up at top should stay at 0, got %d", updated.iterationCursor)
	}

	// Test down at bottom (shouldn't exceed length)
	updated.iterationCursor = 2
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated = newModel.(Model)
	if updated.iterationCursor != 2 {
		t.Errorf("Down at bottom should stay at max, got %d", updated.iterationCursor)
	}
}

func TestDetailIterationSubmit(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.iterationExpanded = true
	m.iterations = []azdo.Iteration{
		{ID: "1", Name: "Sprint 1", Path: "Project\\Sprint 1"},
		{ID: "2", Name: "Sprint 2", Path: "Project\\Sprint 2"},
	}
	m.iterationCursor = 1

	// Test enter to select iteration
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := newModel.(Model)
	if !updated.loading {
		t.Error("Should be loading after iteration selection")
	}
}

func TestDetailCommentScroll(t *testing.T) {
	m := setupDetailModel()
	m.commentsExpanded = true
	m.comments = []azdo.Comment{
		{ID: 1, Text: "Comment 1"},
		{ID: 2, Text: "Comment 2"},
		{ID: 3, Text: "Comment 3"},
	}
	m.commentScroll = 0

	// Test ctrl+n to scroll down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	updated := newModel.(Model)
	if updated.commentScroll != 1 {
		t.Errorf("Ctrl+N should scroll down, got %d", updated.commentScroll)
	}

	// Test ctrl+p to scroll up
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	updated = newModel.(Model)
	if updated.commentScroll != 0 {
		t.Errorf("Ctrl+P should scroll up, got %d", updated.commentScroll)
	}

	// Test ctrl+p at top (shouldn't go negative)
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	updated = newModel.(Model)
	if updated.commentScroll != 0 {
		t.Errorf("Ctrl+P at top should stay at 0, got %d", updated.commentScroll)
	}
}

func TestDetailCreateChildFromRelated(t *testing.T) {
	m := setupDetailModel()
	m.relatedExpanded = true
	m.creatingRelated = false
	m.username = "user@test.com"

	// Test ctrl+n to create child
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	updated := newModel.(Model)
	if !updated.creatingRelated {
		t.Error("Ctrl+N in related mode should start creating")
	}
	if !updated.createRelatedAsChild {
		t.Error("Should be creating as child")
	}
}

func TestDetailCreateParentFromRelated(t *testing.T) {
	m := setupDetailModel()
	m.relatedExpanded = true
	m.creatingRelated = false
	m.username = "user@test.com"

	// Test ctrl+p to create parent
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	updated := newModel.(Model)
	if !updated.creatingRelated {
		t.Error("Ctrl+P in related mode should start creating")
	}
	if updated.createRelatedAsChild {
		t.Error("Should be creating as parent")
	}
}

func TestDetailDeleteLinkConfirmation(t *testing.T) {
	m := setupDetailModel()
	m.relatedExpanded = true
	m.parentItem = &azdo.WorkItem{ID: 100, Fields: azdo.WorkItemFields{Title: "Parent"}}
	m.childItems = []azdo.WorkItem{
		{ID: 101, Fields: azdo.WorkItemFields{Title: "Child 1"}},
	}
	m.relatedCursor = 0 // On parent

	// Test d to start delete confirmation
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated := newModel.(Model)
	if !updated.confirmingDelete {
		t.Error("D should start delete confirmation")
	}
	if updated.confirmDeleteTargetID != 100 {
		t.Errorf("Should target parent ID 100, got %d", updated.confirmDeleteTargetID)
	}

	// Test n to cancel
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	updated = newModel.(Model)
	if updated.confirmingDelete {
		t.Error("N should cancel confirmation")
	}

	// Start again and confirm
	updated.confirmingDelete = true
	updated.confirmDeleteTargetID = 100
	updated.confirmDeleteIsParent = true
	updated.client = azdo.NewClient("org", "proj", "", "", "pat")

	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	updated = newModel.(Model)
	if updated.confirmingDelete {
		t.Error("Y should confirm and clear confirmation")
	}
	if !updated.loading {
		t.Error("Should be loading after confirmation")
	}
}

func TestDetailDeleteChildLink(t *testing.T) {
	m := setupDetailModel()
	m.relatedExpanded = true
	m.parentItem = nil
	m.childItems = []azdo.WorkItem{
		{ID: 101, Fields: azdo.WorkItemFields{Title: "Child 1"}},
		{ID: 102, Fields: azdo.WorkItemFields{Title: "Child 2"}},
	}
	m.relatedCursor = 0 // On first child

	// Test d to start delete confirmation
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated := newModel.(Model)
	if !updated.confirmingDelete {
		t.Error("D should start delete confirmation")
	}
	if updated.confirmDeleteTargetID != 101 {
		t.Errorf("Should target child ID 101, got %d", updated.confirmDeleteTargetID)
	}
	if updated.confirmDeleteIsParent {
		t.Error("Should not be parent")
	}
}

func TestDetailDeleteHyperlink(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.hyperlinksExpanded = true
	m.hyperlinks = []azdo.Hyperlink{
		{URL: "https://example.com/1", Comment: "Link 1"},
		{URL: "https://example.com/2", Comment: "Link 2"},
	}
	m.hyperlinkCursor = 0

	// Test d to delete hyperlink
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated := newModel.(Model)
	if !updated.loading {
		t.Error("Should be loading after delete hyperlink")
	}
}

func TestDetailPlanningModeNavigation(t *testing.T) {
	m := setupDetailModel()
	m.planningExpanded = true
	m.planningFields = []azdo.PlanningField{
		{ReferenceName: "Microsoft.VSTS.Scheduling.Effort", DisplayName: "Effort"},
		{ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints", DisplayName: "Story Points"},
	}
	m.planningFocus = 0

	// Test tab to move down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated := newModel.(Model)
	if updated.planningFocus != 1 {
		t.Errorf("Tab should move to next field, got %d", updated.planningFocus)
	}

	// Test shift+tab to move up
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	updated = newModel.(Model)
	if updated.planningFocus != 0 {
		t.Errorf("Shift+Tab should move to prev field, got %d", updated.planningFocus)
	}

	// Test shift+tab wrap around
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	updated = newModel.(Model)
	if updated.planningFocus != 1 {
		t.Errorf("Shift+Tab should wrap to end, got %d", updated.planningFocus)
	}
}

func TestDetailPlanningSubmit(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.planningExpanded = true
	m.planningFields = []azdo.PlanningField{
		{ReferenceName: "Microsoft.VSTS.Scheduling.Effort", DisplayName: "Effort"},
	}

	// Test enter to save
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := newModel.(Model)
	_ = updated // Just ensure no panic
}

func TestDetailNavigateToRelatedParent(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.relatedExpanded = true
	m.parentItem = &azdo.WorkItem{ID: 100, Fields: azdo.WorkItemFields{Title: "Parent"}}
	m.childItems = nil
	m.relatedCursor = 0

	// Test enter to navigate to parent
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := newModel.(Model)
	// navigateToWorkItem should update selectedItem
	if updated.selectedItem.ID != 100 {
		t.Errorf("Should navigate to parent with ID 100, got %d", updated.selectedItem.ID)
	}
}

func TestDetailNavigateToRelatedChild(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.relatedExpanded = true
	m.parentItem = nil
	m.childItems = []azdo.WorkItem{
		{ID: 201, Fields: azdo.WorkItemFields{Title: "Child 1"}},
		{ID: 202, Fields: azdo.WorkItemFields{Title: "Child 2"}},
	}
	m.relatedCursor = 1

	// Test enter to navigate to child
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := newModel.(Model)
	if updated.selectedItem.ID != 202 {
		t.Errorf("Should navigate to child with ID 202, got %d", updated.selectedItem.ID)
	}
}

func TestDetailRelatedNavigationWithParent(t *testing.T) {
	m := setupDetailModel()
	m.relatedExpanded = true
	m.parentItem = &azdo.WorkItem{ID: 100, Fields: azdo.WorkItemFields{Title: "Parent"}}
	m.childItems = []azdo.WorkItem{
		{ID: 101, Fields: azdo.WorkItemFields{Title: "Child 1"}},
	}
	m.relatedCursor = 0 // On parent

	// Test down to move to child
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := newModel.(Model)
	if updated.relatedCursor != 1 {
		t.Errorf("Down should move to child, got %d", updated.relatedCursor)
	}

	// Test up to wrap around
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated = newModel.(Model)
	if updated.relatedCursor != 0 {
		t.Errorf("Up should move to parent, got %d", updated.relatedCursor)
	}

	// Test shift+tab for navigation up (wrapping)
	updated.relatedCursor = 0
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	updated = newModel.(Model)
	if updated.relatedCursor != 1 {
		t.Errorf("Shift+Tab should wrap around, got %d", updated.relatedCursor)
	}
}

func TestDetailHyperlinkNavigation(t *testing.T) {
	m := setupDetailModel()
	m.hyperlinksExpanded = true
	m.hyperlinks = []azdo.Hyperlink{
		{URL: "https://example.com/1", Comment: "Link 1"},
		{URL: "https://example.com/2", Comment: "Link 2"},
		{URL: "https://example.com/3", Comment: "Link 3"},
	}
	m.hyperlinkCursor = 0

	// Test down/tab to navigate
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := newModel.(Model)
	if updated.hyperlinkCursor != 1 {
		t.Errorf("Down should move cursor, got %d", updated.hyperlinkCursor)
	}

	// Test shift+tab to navigate up with wrap
	updated.hyperlinkCursor = 0
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	updated = newModel.(Model)
	if updated.hyperlinkCursor != 2 {
		t.Errorf("Shift+Tab should wrap to end, got %d", updated.hyperlinkCursor)
	}
}

func TestDetailAddHyperlinkMode(t *testing.T) {
	m := setupDetailModel()
	m.hyperlinksExpanded = true
	m.addingHyperlink = false
	m.creatingRelated = false
	m.confirmingDelete = false

	// Test 'a' to start adding hyperlink
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	updated := newModel.(Model)
	if !updated.addingHyperlink {
		t.Error("'a' should start adding hyperlink mode")
	}
}

func TestDetailCtrlGExitPlanning(t *testing.T) {
	m := setupDetailModel()
	m.planningExpanded = true

	// Test ctrl+g to exit planning
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	updated := newModel.(Model)
	if updated.planningExpanded {
		t.Error("Ctrl+G should exit planning mode")
	}
}

func TestDetailIterationToggle(t *testing.T) {
	m := setupDetailModel()
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.iterationExpanded = false
	m.iterations = nil // Empty iterations to trigger fetch

	// Test ctrl+t to toggle iteration
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	updated := newModel.(Model)
	if !updated.iterationExpanded {
		t.Error("Ctrl+T should expand iterations")
	}

	// Test ctrl+t again to close
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	updated = newModel.(Model)
	if updated.iterationExpanded {
		t.Error("Ctrl+T again should collapse iterations")
	}
}

func TestDetailPlanningModeEmptyFields(t *testing.T) {
	m := setupDetailModel()
	m.planningExpanded = true
	m.planningFields = nil // Empty fields

	// Navigation should handle empty fields without panic
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated := newModel.(Model)
	if updated.planningFocus != 0 {
		t.Errorf("Focus should be 0 with empty fields, got %d", updated.planningFocus)
	}
}

// ============ Config File Update Tests ============

func TestConfigFileToggleBooleans(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile
	m.configFileFocus = 0
	m.appConfig.DefaultShowAll = false

	// Test enter to toggle DefaultShowAll
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := newModel.(Model)
	if !updated.appConfig.DefaultShowAll {
		t.Error("Enter should toggle DefaultShowAll to true")
	}

	// Test space to toggle back
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	updated = newModel.(Model)
	if updated.appConfig.DefaultShowAll {
		t.Error("Space should toggle DefaultShowAll to false")
	}

	// Test EnableNotifications toggle
	updated.configFileFocus = 1
	updated.appConfig.EnableNotifications = false
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = newModel.(Model)
	if !updated.appConfig.EnableNotifications {
		t.Error("Enter should toggle EnableNotifications to true")
	}
}

func TestConfigFileNavigation(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile
	m.configFileFocus = 0

	// Test tab navigation
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated := newModel.(Model)
	if updated.configFileFocus != 1 {
		t.Errorf("Tab should move to next field, got %d", updated.configFileFocus)
	}

	// Test down navigation
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated = newModel.(Model)
	if updated.configFileFocus != 2 {
		t.Errorf("Down should move to MaxWorkItems, got %d", updated.configFileFocus)
	}

	// Test shift+tab navigation
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	updated = newModel.(Model)
	if updated.configFileFocus != 1 {
		t.Errorf("Shift+Tab should go back, got %d", updated.configFileFocus)
	}

	// Test up navigation
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated = newModel.(Model)
	if updated.configFileFocus != 0 {
		t.Errorf("Up should go to first field, got %d", updated.configFileFocus)
	}

	// Test shift+tab wrap around
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	updated = newModel.(Model)
	if updated.configFileFocus != 2 {
		t.Errorf("Shift+Tab from 0 should wrap to 2, got %d", updated.configFileFocus)
	}
}

func TestConfigFileTextInput(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile
	m.configFileFocus = 2 // MaxWorkItems field

	// Type a character - should update the input
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")})
	updated := newModel.(Model)
	_ = updated // Just ensure no panic
}

func TestConfigFileEscape(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile
	m.appConfigMessage = "Some message"

	// Test esc to return to config
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := newModel.(Model)
	if updated.view != ViewConfig {
		t.Errorf("Esc should return to config view, got %v", updated.view)
	}
	if updated.appConfigMessage != "" {
		t.Error("Message should be cleared")
	}
}

// ============ Board View Edge Cases ============

func TestBoardViewEmpty(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.workItems = nil
	m.err = nil

	output := m.View()
	if output == "" {
		t.Error("View should produce output even with empty work items")
	}
}

func TestBoardViewWithError(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.err = &testError{msg: "Test error"}

	output := m.View()
	if output == "" {
		t.Error("View should produce output with error")
	}
}

func TestBoardViewWithMessage(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.message = "Success!"

	output := m.View()
	if output == "" {
		t.Error("View should produce output with message")
	}
}

func TestBoardViewWithNotification(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.notifyMessage = "Work item changed!"

	output := m.View()
	if output == "" {
		t.Error("View should produce output with notification")
	}
}

func TestBoardViewDeleteConfirmation(t *testing.T) {
	m := setupBoardModel()
	m.deletingWorkItem = true
	m.deleteWorkItemID = 123
	m.deleteWorkItemTitle = "Test Item"
	m.deleteConfirmInput = "Test"

	output := m.View()
	if output == "" {
		t.Error("View should produce output with delete confirmation")
	}
}

// ============ Detail View Edge Cases ============

func TestDetailViewCommentsExpanded(t *testing.T) {
	m := setupDetailModel()
	m.commentsExpanded = true
	m.comments = []azdo.Comment{
		{ID: 1, Text: "<p>HTML Comment</p>", CreatedBy: azdo.IdentityRef{DisplayName: "User 1"}},
		{ID: 2, Text: "Plain comment", CreatedBy: azdo.IdentityRef{DisplayName: "User 2"}},
	}

	output := m.View()
	if output == "" {
		t.Error("View should produce output with expanded comments")
	}
}

func TestDetailViewIterationsExpanded(t *testing.T) {
	m := setupDetailModel()
	m.iterationExpanded = true
	m.iterations = []azdo.Iteration{
		{ID: "1", Name: "Sprint 1", Path: "Project\\Sprint 1"},
		{ID: "2", Name: "Sprint 2", Path: "Project\\Sprint 2"},
	}

	output := m.View()
	if output == "" {
		t.Error("View should produce output with expanded iterations")
	}
}

func TestDetailViewPlanningExpanded(t *testing.T) {
	m := setupDetailModel()
	m.planningExpanded = true
	val := 5.0
	m.planningFields = []azdo.PlanningField{
		{ReferenceName: "Microsoft.VSTS.Scheduling.Effort", DisplayName: "Effort", Value: &val},
	}

	output := m.View()
	if output == "" {
		t.Error("View should produce output with expanded planning")
	}
}

func TestDetailViewCreatingRelated(t *testing.T) {
	m := setupDetailModel()
	m.relatedExpanded = true
	m.creatingRelated = true
	m.createRelatedTitle = "New Item"
	m.createRelatedAssignee = "user@test.com"
	m.workItemTypes = []string{"Bug", "Task"}

	output := m.View()
	if output == "" {
		t.Error("View should produce output with create related dialog")
	}
}

func TestDetailViewAddingHyperlink(t *testing.T) {
	m := setupDetailModel()
	m.hyperlinksExpanded = true
	m.addingHyperlink = true
	m.hyperlinkURL = "https://github.com/example"
	m.hyperlinkComment = "PR Link"

	output := m.View()
	if output == "" {
		t.Error("View should produce output with add hyperlink dialog")
	}
}

func TestDetailViewConfirmingDelete(t *testing.T) {
	m := setupDetailModel()
	m.relatedExpanded = true
	m.confirmingDelete = true
	m.confirmDeleteTargetID = 456

	output := m.View()
	if output == "" {
		t.Error("View should produce output with delete confirmation")
	}
}

func TestDetailViewHyperlinksList(t *testing.T) {
	m := setupDetailModel()
	m.hyperlinksExpanded = true
	m.hyperlinks = []azdo.Hyperlink{
		{URL: "https://github.com/example/repo/pull/1", Comment: "PR 1"},
		{URL: "vstfs:///Git/PullRequestId/12345", Comment: "Azure PR"},
	}

	output := m.View()
	if output == "" {
		t.Error("View should produce output with hyperlinks list")
	}
}

// ============ Config View Edge Cases ============

func TestConfigViewWithError(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig
	m.err = &testError{msg: "Connection failed"}

	output := m.View()
	if output == "" {
		t.Error("View should produce output with error")
	}
}

func TestConfigViewWithAppConfigMessage(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig
	m.appConfigMessage = "Config saved!"

	output := m.View()
	if output == "" {
		t.Error("View should produce output with message")
	}
}

// ============ Create View Edge Cases ============

func TestCreateViewWithAreaPath(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate
	m.client = azdo.NewClient("org", "proj", "", "Project\\Team", "pat")
	m.workItemTypes = []string{"Bug", "Task"}

	output := m.View()
	if output == "" {
		t.Error("View should produce output with area path")
	}
}

func TestCreateViewWithError(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate
	m.err = &testError{msg: "Create failed"}
	m.workItemTypes = []string{"Bug", "Task"}

	output := m.View()
	if output == "" {
		t.Error("View should produce output with error")
	}
}

func TestCreateViewLoading(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate
	m.loading = true
	m.workItemTypes = []string{"Bug", "Task"}

	output := m.View()
	if output == "" {
		t.Error("View should produce output while loading")
	}
}

// ============ Additional Model Tests ============

func TestWindowSizeMsg(t *testing.T) {
	m := NewModel()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.width != 120 {
		t.Errorf("Width should be 120, got %d", updated.width)
	}
	if updated.height != 40 {
		t.Errorf("Height should be 40, got %d", updated.height)
	}
}

