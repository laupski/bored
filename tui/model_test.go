package tui

import (
	"os"
	"testing"

	"github.com/laupski/bored/azdo"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMain(m *testing.M) {
	// Set test mode to skip keychain/config file access
	os.Setenv("GO_TEST_MODE", "1")
	os.Exit(m.Run())
}

func TestNewModel(t *testing.T) {
	m := NewModel()

	if m.view != ViewConfig {
		t.Errorf("Initial view = %v, want %v", m.view, ViewConfig)
	}
	if len(m.configInputs) != 6 {
		t.Errorf("configInputs length = %v, want %v", len(m.configInputs), 6)
	}
	if len(m.createInputs) != 4 {
		t.Errorf("createInputs length = %v, want %v", len(m.createInputs), 4)
	}
	if len(m.detailInputs) != 5 {
		t.Errorf("detailInputs length = %v, want %v", len(m.detailInputs), 5)
	}
	if len(m.workItemTypes) != 5 {
		t.Errorf("workItemTypes length = %v, want %v", len(m.workItemTypes), 5)
	}
}

func TestModelInit(t *testing.T) {
	m := NewModel()
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init() should return a command for text input blinking")
	}
}

func TestModelUpdateWindowSize(t *testing.T) {
	m := NewModel()
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.width != 100 {
		t.Errorf("width = %v, want %v", updated.width, 100)
	}
	if updated.height != 50 {
		t.Errorf("height = %v, want %v", updated.height, 50)
	}
}

func TestModelUpdateCtrlC(t *testing.T) {
	m := NewModel()
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}

	_, cmd := m.Update(msg)

	// cmd should be tea.Quit
	if cmd == nil {
		t.Error("Ctrl+C should return a quit command")
	}
}

func TestModelUpdateEscFromCreate(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.view != ViewBoard {
		t.Errorf("ESC from Create should go to Board, got %v", updated.view)
	}
}

func TestModelUpdateEscFromDetail(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.view != ViewBoard {
		t.Errorf("ESC from Detail should go to Board, got %v", updated.view)
	}
}

func TestConfigViewTabNavigation(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig

	// Initial focus should be 0
	if m.configFocus != 0 {
		t.Errorf("Initial configFocus = %v, want 0", m.configFocus)
	}

	// Tab should advance focus
	msg := tea.KeyMsg{Type: tea.KeyTab}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.configFocus != 1 {
		t.Errorf("After Tab, configFocus = %v, want 1", updated.configFocus)
	}
}

func TestBoardViewToggleShowAll(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.client = azdo.NewClient("org", "proj", "", "", "pat")

	// Initial showAll should be false
	if m.showAll != false {
		t.Errorf("Initial showAll = %v, want false", m.showAll)
	}

	// 'a' key should toggle showAll
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.showAll != true {
		t.Errorf("After 'a', showAll = %v, want true", updated.showAll)
	}
}

func TestDetailViewToggleComments(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 1}

	// Initial commentsExpanded should be false
	if m.commentsExpanded != false {
		t.Errorf("Initial commentsExpanded = %v, want false", m.commentsExpanded)
	}

	// Ctrl+E should toggle comments
	msg := tea.KeyMsg{Type: tea.KeyCtrlE}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.commentsExpanded != true {
		t.Errorf("After Ctrl+E, commentsExpanded = %v, want true", updated.commentsExpanded)
	}
}

func TestDetailViewToggleRelated(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = &azdo.WorkItem{ID: 1}

	// Initial relatedExpanded should be false
	if m.relatedExpanded != false {
		t.Errorf("Initial relatedExpanded = %v, want false", m.relatedExpanded)
	}

	// Ctrl+R should toggle related
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.relatedExpanded != true {
		t.Errorf("After Ctrl+R, relatedExpanded = %v, want true", updated.relatedExpanded)
	}
}

func TestConnectMsg(t *testing.T) {
	m := NewModel()
	m.loading = true

	// Success case
	msg := connectMsg{err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.loading != false {
		t.Errorf("After connect success, loading = %v, want false", updated.loading)
	}
	if updated.view != ViewBoard {
		t.Errorf("After connect success, view = %v, want ViewBoard", updated.view)
	}
}

func TestWorkItemsMsg(t *testing.T) {
	m := NewModel()
	m.loading = true

	items := []azdo.WorkItem{
		{ID: 1, Fields: azdo.WorkItemFields{Title: "Test 1"}},
		{ID: 2, Fields: azdo.WorkItemFields{Title: "Test 2"}},
	}
	msg := workItemsMsg{items: items, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.loading != false {
		t.Errorf("After workItems success, loading = %v, want false", updated.loading)
	}
	if len(updated.workItems) != 2 {
		t.Errorf("workItems length = %v, want 2", len(updated.workItems))
	}
}

func TestCommentsMsg(t *testing.T) {
	m := NewModel()
	m.loading = true

	comments := []azdo.Comment{
		{ID: 1, Text: "Comment 1"},
		{ID: 2, Text: "Comment 2"},
	}
	msg := commentsMsg{comments: comments, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.loading != false {
		t.Errorf("After comments success, loading = %v, want false", updated.loading)
	}
	if len(updated.comments) != 2 {
		t.Errorf("comments length = %v, want 2", len(updated.comments))
	}
}

func TestRelatedItemsMsg(t *testing.T) {
	m := NewModel()

	parent := &azdo.WorkItem{ID: 100, Fields: azdo.WorkItemFields{Title: "Parent"}}
	children := []azdo.WorkItem{
		{ID: 101, Fields: azdo.WorkItemFields{Title: "Child 1"}},
		{ID: 102, Fields: azdo.WorkItemFields{Title: "Child 2"}},
	}
	msg := relatedItemsMsg{parent: parent, children: children, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.parentItem == nil {
		t.Error("parentItem should not be nil")
	} else if updated.parentItem.ID != 100 {
		t.Errorf("parentItem.ID = %v, want 100", updated.parentItem.ID)
	}
	if len(updated.childItems) != 2 {
		t.Errorf("childItems length = %v, want 2", len(updated.childItems))
	}
}

func TestViewReturnsString(t *testing.T) {
	views := []View{ViewConfig, ViewBoard, ViewCreate, ViewDetail}

	for _, v := range views {
		m := NewModel()
		m.view = v

		// Board view requires a client
		if v == ViewBoard {
			m.client = azdo.NewClient("org", "proj", "", "", "pat")
		}

		// For detail view, we need a selected item
		if v == ViewDetail {
			m.selectedItem = &azdo.WorkItem{
				ID: 1,
				Fields: azdo.WorkItemFields{
					Title:        "Test",
					State:        "Active",
					WorkItemType: "Bug",
				},
			}
		}

		output := m.View()
		if output == "" {
			t.Errorf("View() for %v returned empty string", v)
		}
	}
}

func TestIsRunningInDocker(t *testing.T) {
	// This test verifies the function runs without error
	// The actual result depends on the environment
	result := isRunningInDocker()
	// In normal test environment, should return false
	if result {
		t.Log("Running in Docker environment")
	} else {
		t.Log("Not running in Docker environment")
	}
}

func TestConnectMsgWithErrorState(t *testing.T) {
	m := NewModel()
	m.loading = true

	// Test failed connection
	testErr := &modelTestError{msg: "connection failed"}
	msg := connectMsg{err: testErr}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.loading {
		t.Error("loading should be false after connect error")
	}
	if updated.err == nil {
		t.Error("err should be set after connect error")
	}
	if updated.view != ViewConfig {
		t.Errorf("view should remain ViewConfig after error, got %v", updated.view)
	}
}

type modelTestError struct {
	msg string
}

func (e *modelTestError) Error() string {
	return e.msg
}
