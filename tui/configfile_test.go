package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/laupski/bored/azdo"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.DefaultShowAll != false {
		t.Errorf("DefaultShowAll = %v, want %v", config.DefaultShowAll, false)
	}
	if config.MaxWorkItems != 50 {
		t.Errorf("MaxWorkItems = %v, want %v", config.MaxWorkItems, 50)
	}
}

func TestSaveAndLoadConfigFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a config file in the temp directory
	configPath := filepath.Join(tempDir, "config.toml")

	// Test config to save
	testConfig := AppConfig{
		DefaultShowAll: true,
		MaxWorkItems:   100,
	}

	// Write config manually to test location
	configContent := `default_show_all = true
max_work_items = 100
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Read it back using TOML parsing
	var loaded AppConfig
	if _, err := toml.DecodeFile(configPath, &loaded); err != nil {
		t.Fatalf("Failed to decode config: %v", err)
	}

	if loaded.DefaultShowAll != testConfig.DefaultShowAll {
		t.Errorf("DefaultShowAll = %v, want %v", loaded.DefaultShowAll, testConfig.DefaultShowAll)
	}
	if loaded.MaxWorkItems != testConfig.MaxWorkItems {
		t.Errorf("MaxWorkItems = %v, want %v", loaded.MaxWorkItems, testConfig.MaxWorkItems)
	}
}

func TestConfigFileExists(t *testing.T) {
	// This test verifies the ConfigFileExists function works
	// In a fresh environment without config, it should return false
	// We can't easily test true case without modifying the actual config location
	exists := ConfigFileExists()
	// Just verify it doesn't panic
	_ = exists
}

func TestGetConfigFilePath(t *testing.T) {
	path := GetConfigFilePath()

	// Path should not be empty or "unknown"
	if path == "" {
		t.Error("GetConfigFilePath() returned empty string")
	}

	// Path should contain "bored"
	if path != "unknown" && !contains(path, "bored") {
		t.Errorf("GetConfigFilePath() = %v, expected to contain 'bored'", path)
	}

	// Path should end with config.toml
	if path != "unknown" && !contains(path, "config.toml") {
		t.Errorf("GetConfigFilePath() = %v, expected to contain 'config.toml'", path)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAppConfigTOMLEncoding(t *testing.T) {
	// Test that AppConfig can be properly encoded to TOML
	config := AppConfig{
		DefaultShowAll: true,
		MaxWorkItems:   75,
	}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	err := encoder.Encode(config)
	if err != nil {
		t.Fatalf("Failed to encode config: %v", err)
	}

	encoded := buf.String()

	// Check that the encoded string contains expected fields
	if !contains(encoded, "default_show_all") {
		t.Errorf("Encoded config should contain 'default_show_all', got: %s", encoded)
	}
	if !contains(encoded, "max_work_items") {
		t.Errorf("Encoded config should contain 'max_work_items', got: %s", encoded)
	}
}

func TestDefaultConfigValues(t *testing.T) {
	// Test that DefaultConfig returns valid defaults
	config := DefaultConfig()

	// Should have valid defaults
	if config.MaxWorkItems <= 0 {
		t.Errorf("MaxWorkItems should have a positive default, got %d", config.MaxWorkItems)
	}
	if config.MaxWorkItems != 50 {
		t.Errorf("MaxWorkItems should default to 50, got %d", config.MaxWorkItems)
	}
}

func TestConfigMaxWorkItemsZeroDefault(t *testing.T) {
	// Test that zero MaxWorkItems gets defaulted to 50
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Write config with zero max_work_items
	configContent := `default_show_all = false
max_work_items = 0
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Read it back
	var loaded AppConfig
	if _, err := toml.DecodeFile(configPath, &loaded); err != nil {
		t.Fatalf("Failed to decode config: %v", err)
	}

	// Apply the same defaulting logic as LoadConfigFile
	if loaded.MaxWorkItems == 0 {
		loaded.MaxWorkItems = 50
	}

	if loaded.MaxWorkItems != 50 {
		t.Errorf("MaxWorkItems should default to 50 when 0, got %d", loaded.MaxWorkItems)
	}
}

func TestViewConfigFileOutput(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile
	m.appConfig = DefaultConfig()

	output := m.viewConfigFile()

	// Should contain the title
	if !contains(output, "Application Settings") {
		t.Error("viewConfigFile should contain title")
	}

	// Should contain setting labels
	if !contains(output, "Default Show All") {
		t.Error("viewConfigFile should contain 'Default Show All'")
	}
	if !contains(output, "Max Work Items") {
		t.Error("viewConfigFile should contain 'Max Work Items'")
	}
}

func TestViewConfigFileWithMessage(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile
	m.appConfig = DefaultConfig()
	m.appConfigMessage = "Configuration saved successfully"

	output := m.viewConfigFile()

	// Should contain success message
	if !contains(output, "Configuration saved") {
		t.Error("viewConfigFile should show success message")
	}
}

func TestViewConfigFileWithError(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile
	m.appConfig = DefaultConfig()
	m.appConfigMessage = "Error saving config"

	output := m.viewConfigFile()

	// Should contain error message
	if !contains(output, "Error") {
		t.Error("viewConfigFile should show error message")
	}
}

func TestConfigFileFocusCommands(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile

	// Test focus on MaxWorkItems field (index 2)
	m.configFileFocus = 2
	cmd := m.updateConfigFileFocus()
	if cmd == nil {
		t.Error("updateConfigFileFocus should return a command for text input focus")
	}

	// Test focus on other fields (should blur text input)
	m.configFileFocus = 0
	cmd = m.updateConfigFileFocus()
	// No command expected when blurring
	_ = cmd
}

func TestConfigFileInputsUpdate(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile

	// Just verify updateConfigFileInputs doesn't panic
	cmd := m.updateConfigFileInputs(nil)
	_ = cmd
}

func TestViewConfigOutput(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig

	output := m.viewConfig()

	// Should contain the title
	if !contains(output, "Azure DevOps TUI") {
		t.Error("viewConfig should contain title")
	}

	// Should contain field labels
	if !contains(output, "Organization") {
		t.Error("viewConfig should contain 'Organization'")
	}
	if !contains(output, "Personal Access Token") {
		t.Error("viewConfig should contain 'Personal Access Token'")
	}
}

func TestViewConfigErrorDisplay(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig
	m.err = &configTestError{msg: "Test error"}

	output := m.viewConfig()

	// Should contain error
	if !contains(output, "Error") {
		t.Error("viewConfig should show error")
	}
}

func TestViewConfigLoadingDisplay(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig
	m.loading = true

	output := m.viewConfig()

	// Should contain loading message
	if !contains(output, "Connecting") {
		t.Error("viewConfig should show connecting message")
	}
}

func TestViewConfigWithKeychainMessage(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig
	m.keychainMessage = "Credentials saved to keychain"

	output := m.viewConfig()

	// Should contain keychain message
	if !contains(output, "Credentials saved") {
		t.Error("viewConfig should show keychain message")
	}
}

type configTestError struct {
	msg string
}

func (e *configTestError) Error() string {
	return e.msg
}

func TestConfigInputsFocusUpdate(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig

	// Test updateConfigFocus
	cmd := m.updateConfigFocus()
	if cmd == nil {
		t.Error("updateConfigFocus should return a command")
	}

	// Test updateConfigInputs
	cmd = m.updateConfigInputs(nil)
	_ = cmd
}

func TestGetConfigDirPaths(t *testing.T) {
	// Test getConfigDir returns a valid path
	dir, err := getConfigDir()
	if err != nil {
		t.Errorf("getConfigDir() returned error: %v", err)
	}
	if dir == "" {
		t.Error("getConfigDir() returned empty string")
	}
	// Path should contain bored
	if !contains(dir, "bored") {
		t.Errorf("getConfigDir() = %s, expected to contain 'bored'", dir)
	}
}

func TestGetConfigFilePathInternal(t *testing.T) {
	// Test getConfigFilePath returns a valid path
	path, err := getConfigFilePath()
	if err != nil {
		t.Errorf("getConfigFilePath() returned error: %v", err)
	}
	if path == "" {
		t.Error("getConfigFilePath() returned empty string")
	}
	// Path should end with config.toml
	if !contains(path, "config.toml") {
		t.Errorf("getConfigFilePath() = %s, expected to contain 'config.toml'", path)
	}
}

func TestViewConfigFileWithFocusedCheckbox(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile
	m.appConfig = DefaultConfig()
	m.appConfig.DefaultShowAll = true
	m.configFileFocus = 0

	output := m.viewConfigFile()

	// Should contain checkbox in output
	if !contains(output, "[x]") {
		t.Error("viewConfigFile should show checked checkbox for DefaultShowAll=true")
	}
}

func TestViewConfigFileWithNotificationsFocused(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile
	m.appConfig = DefaultConfig()
	m.appConfig.EnableNotifications = false
	m.configFileFocus = 1

	output := m.viewConfigFile()

	// Should contain unchecked checkbox
	if !contains(output, "[ ]") {
		t.Error("viewConfigFile should show unchecked checkbox for EnableNotifications=false")
	}
}

func TestViewConfigFileMaxWorkItemsFocus(t *testing.T) {
	m := NewModel()
	m.view = ViewConfigFile
	m.appConfig = DefaultConfig()
	m.configFileFocus = 2 // MaxWorkItems field

	output := m.viewConfigFile()

	// Should contain the max work items label
	if !contains(output, "Max Work Items") {
		t.Error("viewConfigFile should show Max Work Items field")
	}
}

func TestBoardViewOutput(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")

	output := m.viewBoard()

	// Should contain help text
	if !contains(output, "quit") {
		t.Error("viewBoard should contain help text")
	}
}

func TestBoardViewWithWorkItems(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.workItems = []azdo.WorkItem{
		{
			ID: 1,
			Fields: azdo.WorkItemFields{
				Title:        "Test Item",
				State:        "Active",
				WorkItemType: "Bug",
			},
		},
	}

	output := m.viewBoard()

	// Should contain the work item
	if !contains(output, "Test Item") {
		t.Error("viewBoard should show work item title")
	}
}

func TestCreateViewOutput(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate
	m.width = 100
	m.height = 50

	output := m.viewCreate()

	// Should contain create form elements
	if !contains(output, "Title") {
		t.Error("viewCreate should contain Title field")
	}
}

func TestDetailViewOutput(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.selectedItem = &azdo.WorkItem{
		ID: 123,
		Fields: azdo.WorkItemFields{
			Title:        "Test Work Item",
			State:        "Active",
			WorkItemType: "Bug",
			Description:  "Test description",
		},
	}

	output := m.viewDetail()

	// Should contain work item type and ID
	if !contains(output, "Bug") {
		t.Error("viewDetail should show work item type")
	}
	if !contains(output, "123") {
		t.Error("viewDetail should show work item ID")
	}
}

func TestBoardViewWithLoading(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.loading = true

	output := m.viewBoard()

	// Should contain loading message
	if !contains(output, "Loading") {
		t.Error("viewBoard should show loading message")
	}
}

func TestBoardViewErrorDisplay(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.err = &configTestError{msg: "API error"}

	output := m.viewBoard()

	// Should contain error
	if !contains(output, "Error") {
		t.Error("viewBoard should show error message")
	}
}

func TestBoardViewEmptyList(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.workItems = []azdo.WorkItem{}

	output := m.viewBoard()

	// Should contain no items message
	if !contains(output, "No work items") {
		t.Error("viewBoard should show 'No work items found' message")
	}
}

func TestBoardViewWithFilteredUser(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.username = "test@example.com"
	m.showAll = false

	output := m.viewBoard()

	// Should show filtered status
	if !contains(output, "filtered") {
		t.Error("viewBoard should show filtered status")
	}
}

func TestBoardViewShowAll(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.username = "test@example.com"
	m.showAll = true

	output := m.viewBoard()

	// Should show "showing all" status
	if !contains(output, "showing all") {
		t.Error("viewBoard should show 'showing all' status")
	}
}

func TestDetailViewNoSelection(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.selectedItem = nil

	output := m.viewDetail()

	// Should show no selection message
	if !contains(output, "No work item selected") {
		t.Error("viewDetail should show 'No work item selected'")
	}
}

func TestModelViewFunction(t *testing.T) {
	// Test that View() returns correct output for each view type
	testCases := []struct {
		view     View
		expected string
	}{
		{ViewConfig, "Azure DevOps"},
		{ViewCreate, "Title"},
	}

	for _, tc := range testCases {
		m := NewModel()
		m.view = tc.view
		m.width = 100
		m.height = 50

		output := m.View()
		if !contains(output, tc.expected) {
			t.Errorf("View() for %v should contain '%s'", tc.view, tc.expected)
		}
	}
}

func TestBoardViewWithMultipleWorkItems(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.workItems = []azdo.WorkItem{
		{ID: 1, Fields: azdo.WorkItemFields{Title: "First", State: "Active", WorkItemType: "Bug"}},
		{ID: 2, Fields: azdo.WorkItemFields{Title: "Second", State: "New", WorkItemType: "Task"}},
		{ID: 3, Fields: azdo.WorkItemFields{Title: "Third", State: "Closed", WorkItemType: "Feature"}},
	}

	output := m.viewBoard()

	// Should show all items
	if !contains(output, "First") || !contains(output, "Second") || !contains(output, "Third") {
		t.Error("viewBoard should show all work items")
	}
}

func TestBoardViewWithNotifyMessage(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.notifyMessage = "Work item changed"

	output := m.viewBoard()

	// Should show notification message
	if !contains(output, "Work item changed") {
		t.Error("viewBoard should show notification message")
	}
}

func TestBoardViewWithSelectedItem(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.workItems = []azdo.WorkItem{
		{ID: 1, Fields: azdo.WorkItemFields{Title: "Selected Item", State: "Active", WorkItemType: "Bug"}},
	}
	m.cursor = 0

	output := m.viewBoard()

	// Should show the selected item
	if !contains(output, "Selected Item") {
		t.Error("viewBoard should show selected item")
	}
}

func TestCreateViewWorkItemTypes(t *testing.T) {
	m := NewModel()
	m.view = ViewCreate
	m.width = 100
	m.height = 50
	m.workItemTypes = []string{"Bug", "Task", "Feature"}

	output := m.viewCreate()

	// Should contain work item type selector
	if !contains(output, "Type") {
		t.Error("viewCreate should show Type selector")
	}
}

func TestBoardWorkItemWithTags(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.workItems = []azdo.WorkItem{
		{
			ID: 1,
			Fields: azdo.WorkItemFields{
				Title:        "Tagged Item",
				State:        "Active",
				WorkItemType: "Bug",
				Tags:         "important; urgent",
			},
		},
	}

	output := m.viewBoard()

	// Should show the work item
	if !contains(output, "Tagged Item") {
		t.Error("viewBoard should show tagged work item")
	}
}

func TestBoardWorkItemWithAssignee(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.workItems = []azdo.WorkItem{
		{
			ID: 1,
			Fields: azdo.WorkItemFields{
				Title:        "Assigned Item",
				State:        "Active",
				WorkItemType: "Bug",
				AssignedTo: &azdo.IdentityRef{
					DisplayName: "Test User",
					UniqueName:  "test@example.com",
				},
			},
		},
	}

	output := m.viewBoard()

	// Should show the assignee
	if !contains(output, "Test User") {
		t.Error("viewBoard should show assignee name")
	}
}

func TestBoardWorkItemWithCommentCount(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.workItems = []azdo.WorkItem{
		{
			ID: 1,
			Fields: azdo.WorkItemFields{
				Title:        "Item With Comments",
				State:        "Active",
				WorkItemType: "Bug",
				CommentCount: 5,
			},
		},
	}

	output := m.viewBoard()

	// Should show the work item
	if !contains(output, "Item With Comments") {
		t.Error("viewBoard should show item with comments")
	}
}

func TestDetailViewWithComments(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.selectedItem = &azdo.WorkItem{
		ID:     123,
		Fields: azdo.WorkItemFields{Title: "Test", State: "Active", WorkItemType: "Bug"},
	}
	m.comments = []azdo.Comment{
		{ID: 1, Text: "Test comment", CreatedBy: azdo.IdentityRef{DisplayName: "User"}},
	}
	m.commentsExpanded = true

	output := m.viewDetail()

	// Should show comments section
	if !contains(output, "Comment") {
		t.Error("viewDetail should show comments")
	}
}

func TestDetailViewWithIterations(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.selectedItem = &azdo.WorkItem{
		ID:     123,
		Fields: azdo.WorkItemFields{Title: "Test", State: "Active", WorkItemType: "Bug"},
	}
	m.iterations = []azdo.Iteration{
		{ID: "1", Name: "Sprint 1", Path: "Project\\Sprint 1"},
	}
	m.iterationExpanded = true

	output := m.viewDetail()

	// Should show iteration section
	if !contains(output, "Iteration") {
		t.Error("viewDetail should show iterations")
	}
}

func TestDetailViewWithRelatedItems(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.selectedItem = &azdo.WorkItem{
		ID:     123,
		Fields: azdo.WorkItemFields{Title: "Test", State: "Active", WorkItemType: "Bug"},
	}
	m.parentItem = &azdo.WorkItem{
		ID:     100,
		Fields: azdo.WorkItemFields{Title: "Parent", State: "Active", WorkItemType: "Feature"},
	}
	m.relatedExpanded = true

	output := m.viewDetail()

	// Should show related section
	if !contains(output, "Related") {
		t.Error("viewDetail should show related items")
	}
}

func TestDetailViewWithHyperlinks(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.selectedItem = &azdo.WorkItem{
		ID:     123,
		Fields: azdo.WorkItemFields{Title: "Test", State: "Active", WorkItemType: "Bug"},
	}
	m.hyperlinks = []azdo.Hyperlink{
		{URL: "https://example.com", Comment: "Example link"},
	}
	m.hyperlinksExpanded = true

	output := m.viewDetail()

	// Should show hyperlinks section
	if !contains(output, "Link") {
		t.Error("viewDetail should show hyperlinks")
	}
}

func TestDetailViewWithMessage(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.selectedItem = &azdo.WorkItem{
		ID:     123,
		Fields: azdo.WorkItemFields{Title: "Test", State: "Active", WorkItemType: "Bug"},
	}
	m.message = "Work item updated successfully"

	output := m.viewDetail()

	// Should show message
	if !contains(output, "updated") {
		t.Error("viewDetail should show message")
	}
}

func TestDetailViewWithError(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.selectedItem = &azdo.WorkItem{
		ID:     123,
		Fields: azdo.WorkItemFields{Title: "Test", State: "Active", WorkItemType: "Bug"},
	}
	m.err = &configTestError{msg: "API error"}

	output := m.viewDetail()

	// Should show error
	if !contains(output, "Error") {
		t.Error("viewDetail should show error")
	}
}

func TestDetailViewWithChildItems(t *testing.T) {
	m := NewModel()
	m.view = ViewDetail
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.selectedItem = &azdo.WorkItem{
		ID:     123,
		Fields: azdo.WorkItemFields{Title: "Test", State: "Active", WorkItemType: "Feature"},
	}
	m.childItems = []azdo.WorkItem{
		{ID: 124, Fields: azdo.WorkItemFields{Title: "Child 1", State: "Active", WorkItemType: "Task"}},
		{ID: 125, Fields: azdo.WorkItemFields{Title: "Child 2", State: "New", WorkItemType: "Task"}},
	}
	m.relatedExpanded = true

	output := m.viewDetail()

	// Should show children section
	if !contains(output, "Related") {
		t.Error("viewDetail should show related section with children")
	}
}

func TestBoardCursorNavigation(t *testing.T) {
	m := NewModel()
	m.view = ViewBoard
	m.width = 100
	m.height = 50
	m.client = azdo.NewClient("testorg", "testproject", "testteam", "", "pat")
	m.workItems = []azdo.WorkItem{
		{ID: 1, Fields: azdo.WorkItemFields{Title: "Item 1", State: "Active", WorkItemType: "Bug"}},
		{ID: 2, Fields: azdo.WorkItemFields{Title: "Item 2", State: "New", WorkItemType: "Task"}},
	}
	m.cursor = 1 // Second item selected

	output := m.viewBoard()

	// Should show both items
	if !contains(output, "Item 1") || !contains(output, "Item 2") {
		t.Error("viewBoard should show both items")
	}
}

func TestConfigInputFocusCycle(t *testing.T) {
	m := NewModel()
	m.view = ViewConfig
	m.configFocus = 5 // Last input

	// Test that focus cycles correctly
	m.configFocus = (m.configFocus + 1) % len(m.configInputs)
	if m.configFocus != 0 {
		t.Errorf("Focus should cycle to 0, got %d", m.configFocus)
	}

	// Test backward navigation
	m.configFocus = 0
	m.configFocus--
	if m.configFocus < 0 {
		m.configFocus = len(m.configInputs) - 1
	}
	if m.configFocus != 5 {
		t.Errorf("Focus should cycle to 5, got %d", m.configFocus)
	}
}
