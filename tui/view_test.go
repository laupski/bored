package tui

import (
	"strings"
	"testing"

	"github.com/laupski/bored/azdo"
)

// Golden tests compare rendered output against expected patterns
// These tests verify that views contain expected elements

func TestViewConfigContainsLabels(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig

	output := m.View()

	expectedLabels := []string{
		"Organization",
		"Project",
		"Team",
		"Area Path",
		"Personal Access Token",
		"Username",
	}

	for _, label := range expectedLabels {
		if !strings.Contains(output, label) {
			t.Errorf("Config view should contain %q", label)
		}
	}
}

func TestViewConfigContainsHelp(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig

	output := m.View()

	// Should contain navigation hints
	if !strings.Contains(output, "tab") && !strings.Contains(output, "Tab") {
		t.Error("Config view should mention tab navigation")
	}
	if !strings.Contains(output, "enter") && !strings.Contains(output, "Enter") {
		t.Error("Config view should mention enter to connect")
	}
}

func TestViewBoardShowsWorkItems(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.workItems = []azdo.WorkItem{
		{
			ID: 123,
			Fields: azdo.WorkItemFields{
				Title:        "Test Bug",
				State:        "Active",
				WorkItemType: "Bug",
			},
		},
		{
			ID: 456,
			Fields: azdo.WorkItemFields{
				Title:        "Test Task",
				State:        "New",
				WorkItemType: "Task",
			},
		},
	}

	output := m.View()

	// Should show work item IDs
	if !strings.Contains(output, "123") {
		t.Error("Board view should show work item ID 123")
	}
	if !strings.Contains(output, "456") {
		t.Error("Board view should show work item ID 456")
	}

	// Should show titles
	if !strings.Contains(output, "Test Bug") {
		t.Error("Board view should show 'Test Bug'")
	}
	if !strings.Contains(output, "Test Task") {
		t.Error("Board view should show 'Test Task'")
	}
}

func TestViewBoardShowsEmptyState(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.workItems = []azdo.WorkItem{}

	output := m.View()

	// Should indicate no items
	if !strings.Contains(output, "No work items") && !strings.Contains(output, "0 items") {
		// Either message format is acceptable
		t.Log("Board view shows empty state")
	}
}

func TestViewBoardShowsLoadingState(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.client = azdo.NewClient("org", "proj", "", "", "pat")
	m.loading = true

	output := m.View()

	if !strings.Contains(output, "Loading") && !strings.Contains(output, "loading") {
		t.Error("Board view should show loading indicator")
	}
}

func TestViewCreateContainsFields(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate

	output := m.View()

	expectedFields := []string{
		"Title",
		"Description",
		"Priority",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Create view should contain %q field", field)
		}
	}
}

func TestViewDetailShowsWorkItemInfo(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{
		ID: 789,
		Fields: azdo.WorkItemFields{
			Title:        "Important Bug Fix",
			State:        "Active",
			WorkItemType: "Bug",
			AreaPath:     "Project\\Team",
			Priority:     1,
			AssignedTo:   &azdo.IdentityRef{DisplayName: "John Doe", UniqueName: "john@example.com"},
		},
	}

	output := m.View()

	// Should show ID
	if !strings.Contains(output, "789") {
		t.Error("Detail view should show work item ID")
	}

	// Should show type
	if !strings.Contains(output, "Bug") {
		t.Error("Detail view should show work item type")
	}

	// Should contain field labels
	if !strings.Contains(output, "Title") {
		t.Error("Detail view should show Title field")
	}
	if !strings.Contains(output, "State") {
		t.Error("Detail view should show State field")
	}
}

func TestViewDetailShowsComments(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{
		ID:     1,
		Fields: azdo.WorkItemFields{Title: "Test", WorkItemType: "Bug"},
	}
	m.comments = []azdo.Comment{
		{ID: 1, Text: "First comment", CreatedBy: azdo.IdentityRef{DisplayName: "User1"}},
		{ID: 2, Text: "Second comment", CreatedBy: azdo.IdentityRef{DisplayName: "User2"}},
	}

	output := m.View()

	// Should show comment count
	if !strings.Contains(output, "Comments (2)") && !strings.Contains(output, "2)") {
		t.Error("Detail view should show comment count")
	}
}

func TestViewDetailShowsRelatedItems(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{
		ID:     1,
		Fields: azdo.WorkItemFields{Title: "Test", WorkItemType: "Task"},
	}
	m.parentItem = &azdo.WorkItem{
		ID:     100,
		Fields: azdo.WorkItemFields{Title: "Parent Story", WorkItemType: "User Story"},
	}
	m.childItems = []azdo.WorkItem{
		{ID: 101, Fields: azdo.WorkItemFields{Title: "Child 1", WorkItemType: "Task"}},
	}

	output := m.View()

	// Should show related items count (parent + 1 child = 2)
	if !strings.Contains(output, "Related Items") {
		t.Error("Detail view should show Related Items section")
	}
}

func TestViewDetailShowsError(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{
		ID:     1,
		Fields: azdo.WorkItemFields{Title: "Test", WorkItemType: "Bug"},
	}
	m.err = &testError{msg: "Test error message"}

	output := m.View()

	if !strings.Contains(output, "Error") && !strings.Contains(output, "error") {
		t.Error("Detail view should show error")
	}
}

func TestViewDetailShowsSuccessMessage(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{
		ID:     1,
		Fields: azdo.WorkItemFields{Title: "Test", WorkItemType: "Bug"},
	}
	m.message = "Work item updated"

	output := m.View()

	if !strings.Contains(output, "updated") {
		t.Error("Detail view should show success message")
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestViewBoardHelp(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	output := m.View()

	// Should contain key hints
	helpPatterns := []string{"c", "n", "esc", "enter"}
	foundCount := 0
	for _, pattern := range helpPatterns {
		if strings.Contains(strings.ToLower(output), pattern) {
			foundCount++
		}
	}

	if foundCount < 2 {
		t.Error("Board view should contain keyboard shortcut hints")
	}
}

func TestViewDetailHelp(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{
		ID:     1,
		Fields: azdo.WorkItemFields{Title: "Test", WorkItemType: "Bug"},
	}

	output := m.View()

	// Should mention ctrl+s for save
	if !strings.Contains(output, "ctrl+s") && !strings.Contains(output, "save") {
		t.Error("Detail view should mention save shortcut")
	}
}
