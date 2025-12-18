package tui

import (
	"strings"
	"testing"

	"github.com/laupski/bored/azdo"
)

func TestParseMentions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		orgURL   string
		contains string
	}{
		{
			name:     "parses simple mention",
			input:    `<a href="#" data-vss-mention="version:2.0,abc123-def456">@John Doe</a>`,
			orgURL:   "https://dev.azure.com/myorg",
			contains: "@John Doe",
		},
		{
			name:     "preserves text without mentions",
			input:    "Hello world, no mentions here",
			orgURL:   "https://dev.azure.com/myorg",
			contains: "Hello world, no mentions here",
		},
		{
			name:     "handles multiple mentions",
			input:    `<a href="#" data-vss-mention="version:2.0,abc123">@John</a> and <a href="#" data-vss-mention="version:2.0,def456">@Jane</a>`,
			orgURL:   "https://dev.azure.com/myorg",
			contains: "@John",
		},
		{
			name:     "handles empty org URL",
			input:    `<a href="#" data-vss-mention="version:2.0,abc123">@User</a>`,
			orgURL:   "",
			contains: "@User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMentions(tt.input, tt.orgURL)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("parseMentions() = %v, want to contain %v", result, tt.contains)
			}
		})
	}
}

func TestParseURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "parses http URL",
			input:    "Check out http://example.com for more info",
			contains: "http://example.com",
		},
		{
			name:     "parses https URL",
			input:    "Visit https://secure.example.com/path",
			contains: "https://secure.example.com/path",
		},
		{
			name:     "handles URL with trailing punctuation",
			input:    "See https://example.com.",
			contains: "https://example.com",
		},
		{
			name:     "preserves text without URLs",
			input:    "No URLs here",
			contains: "No URLs here",
		},
		{
			name:     "handles multiple URLs",
			input:    "Visit https://one.com and https://two.com",
			contains: "https://one.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseURLs(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("parseURLs() = %v, want to contain %v", result, tt.contains)
			}
		})
	}
}

func TestParseHTMLLinks(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contains    string
		notContains string
	}{
		{
			name:     "parses anchor tag with URL",
			input:    `<a href="https://example.com">Click here</a>`,
			contains: "https://example.com",
		},
		{
			name:        "skips anchor-only links",
			input:       `<a href="#">Click here</a>`,
			contains:    "Click here",
			notContains: "#",
		},
		{
			name:     "shows text and URL when different",
			input:    `<a href="https://example.com/long/path">Short text</a>`,
			contains: "Short text",
		},
		{
			name:     "skips mention tags",
			input:    `<a href="#" data-vss-mention="version:2.0,abc">@User</a>`,
			contains: "data-vss-mention", // Should be unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHTMLLinks(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("parseHTMLLinks() = %v, want to contain %v", result, tt.contains)
			}
			if tt.notContains != "" && strings.Contains(result, tt.notContains) {
				t.Errorf("parseHTMLLinks() = %v, should not contain %v", result, tt.notContains)
			}
		})
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		orgURL      string
		contains    string
		notContains string
	}{
		{
			name:        "strips div tags",
			input:       "<div>Hello World</div>",
			orgURL:      "",
			contains:    "Hello World",
			notContains: "<div>",
		},
		{
			name:     "converts br to newline",
			input:    "Line 1<br>Line 2",
			orgURL:   "",
			contains: "Line 1\nLine 2",
		},
		{
			name:     "converts br/ to newline",
			input:    "Line 1<br/>Line 2",
			orgURL:   "",
			contains: "Line 1\nLine 2",
		},
		{
			name:        "strips p tags",
			input:       "<p>Paragraph</p>",
			orgURL:      "",
			contains:    "Paragraph",
			notContains: "<p>",
		},
		{
			name:     "decodes HTML entities",
			input:    "&amp; &nbsp;text",
			orgURL:   "",
			contains: "& ",
		},
		{
			name:     "preserves mentions while stripping",
			input:    `<div><a href="#" data-vss-mention="version:2.0,abc">@User</a></div>`,
			orgURL:   "https://dev.azure.com/org",
			contains: "@User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripHTMLTags(tt.input, tt.orgURL)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("stripHTMLTags() = %q, want to contain %q", result, tt.contains)
			}
			if tt.notContains != "" && strings.Contains(result, tt.notContains) {
				t.Errorf("stripHTMLTags() = %q, should not contain %q", result, tt.notContains)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "no truncation needed",
			input:  "Short",
			maxLen: 10,
			want:   "Short",
		},
		{
			name:   "exact length",
			input:  "Exact",
			maxLen: 5,
			want:   "Exact",
		},
		{
			name:   "truncates with ellipsis",
			input:  "This is a long string",
			maxLen: 10,
			want:   "This is...",
		},
		{
			name:   "very short maxLen",
			input:  "Hello",
			maxLen: 3,
			want:   "Hel",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.want)
			}
		})
	}
}

func TestGetIterationDisplayOrder(t *testing.T) {
	tests := []struct {
		name            string
		iterations      []azdo.Iteration
		currentPath     string
		wantFirstName   string
		wantOrderLength int
	}{
		{
			name:            "empty iterations",
			iterations:      []azdo.Iteration{},
			currentPath:     "Project\\Sprint 1",
			wantFirstName:   "",
			wantOrderLength: 0,
		},
		{
			name: "current iteration first",
			iterations: []azdo.Iteration{
				{ID: "1", Name: "Sprint 1", Path: "Project\\Sprint 1"},
				{ID: "2", Name: "Sprint 2", Path: "Project\\Sprint 2"},
				{ID: "3", Name: "Sprint 3", Path: "Project\\Sprint 3"},
			},
			currentPath:     "Project\\Sprint 2",
			wantFirstName:   "Sprint 2",
			wantOrderLength: 3,
		},
		{
			name: "current iteration already first",
			iterations: []azdo.Iteration{
				{ID: "1", Name: "Sprint 1", Path: "Project\\Sprint 1"},
				{ID: "2", Name: "Sprint 2", Path: "Project\\Sprint 2"},
			},
			currentPath:     "Project\\Sprint 1",
			wantFirstName:   "Sprint 1",
			wantOrderLength: 2,
		},
		{
			name: "current iteration not found",
			iterations: []azdo.Iteration{
				{ID: "1", Name: "Sprint 1", Path: "Project\\Sprint 1"},
				{ID: "2", Name: "Sprint 2", Path: "Project\\Sprint 2"},
			},
			currentPath:     "Project\\Sprint 99",
			wantFirstName:   "Sprint 1",
			wantOrderLength: 2,
		},
		{
			name: "single iteration",
			iterations: []azdo.Iteration{
				{ID: "1", Name: "Only Sprint", Path: "Project\\Only Sprint"},
			},
			currentPath:     "Project\\Only Sprint",
			wantFirstName:   "Only Sprint",
			wantOrderLength: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				iterations: tt.iterations,
				selectedItem: &azdo.WorkItem{
					Fields: azdo.WorkItemFields{
						IterationPath: tt.currentPath,
					},
				},
			}

			result := m.getIterationDisplayOrder()

			if len(result) != tt.wantOrderLength {
				t.Errorf("getIterationDisplayOrder() length = %d, want %d", len(result), tt.wantOrderLength)
			}

			if tt.wantOrderLength > 0 && result[0].Name != tt.wantFirstName {
				t.Errorf("getIterationDisplayOrder() first item = %q, want %q", result[0].Name, tt.wantFirstName)
			}
		})
	}
}

func TestGetIterationDisplayOrderNilSelectedItem(t *testing.T) {
	m := Model{
		iterations: []azdo.Iteration{
			{ID: "1", Name: "Sprint 1", Path: "Project\\Sprint 1"},
		},
		selectedItem: nil,
	}

	result := m.getIterationDisplayOrder()

	// Should return original iterations when selectedItem is nil
	if len(result) != 1 {
		t.Errorf("getIterationDisplayOrder() with nil selectedItem should return original iterations, got length %d", len(result))
	}
}

func TestUpdatePlanningInputsFromWorkItemDynamic(t *testing.T) {
	storyPoints := 5.0
	originalEstimate := 8.0
	remainingWork := 4.5

	m := NewModel()
	m.selectedItem = &azdo.WorkItem{
		ID: 123,
		Fields: azdo.WorkItemFields{
			Title:            "Test Item",
			WorkItemType:     "User Story",
			StoryPoints:      &storyPoints,
			OriginalEstimate: &originalEstimate,
			RemainingWork:    &remainingWork,
			CompletedWork:    nil, // Test nil handling
		},
	}
	m.planningFields = []azdo.PlanningField{
		{ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints", DisplayName: "Story Points"},
		{ReferenceName: "Microsoft.VSTS.Scheduling.OriginalEstimate", DisplayName: "Original Estimate"},
		{ReferenceName: "Microsoft.VSTS.Scheduling.RemainingWork", DisplayName: "Remaining Work"},
		{ReferenceName: "Microsoft.VSTS.Scheduling.CompletedWork", DisplayName: "Completed Work"},
	}

	m.updatePlanningInputsFromWorkItemDynamic()

	// Check that inputs are populated correctly
	if m.planningInputs[0].Value() != "5.0" {
		t.Errorf("planningInputs[0] = %v, want %v", m.planningInputs[0].Value(), "5.0")
	}
	if m.planningInputs[1].Value() != "8.0" {
		t.Errorf("planningInputs[1] = %v, want %v", m.planningInputs[1].Value(), "8.0")
	}
	if m.planningInputs[2].Value() != "4.5" {
		t.Errorf("planningInputs[2] = %v, want %v", m.planningInputs[2].Value(), "4.5")
	}
	if m.planningInputs[3].Value() != "" {
		t.Errorf("planningInputs[3] should be empty for nil value, got %v", m.planningInputs[3].Value())
	}
}

func TestUpdatePlanningInputsFromWorkItemDynamicNilSelectedItem(t *testing.T) {
	m := NewModel()
	m.selectedItem = nil
	m.planningFields = []azdo.PlanningField{
		{ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints", DisplayName: "Story Points"},
	}

	// Should not panic when selectedItem is nil
	m.updatePlanningInputsFromWorkItemDynamic()
}

func TestUpdatePlanningInputsFromWorkItem(t *testing.T) {
	storyPoints := 3.0
	originalEstimate := 10.0
	remainingWork := 6.0
	completedWork := 4.0

	m := NewModel()
	m.selectedItem = &azdo.WorkItem{
		ID: 456,
		Fields: azdo.WorkItemFields{
			Title:            "Test Task",
			WorkItemType:     "Task",
			StoryPoints:      &storyPoints,
			OriginalEstimate: &originalEstimate,
			RemainingWork:    &remainingWork,
			CompletedWork:    &completedWork,
		},
	}

	m.updatePlanningInputsFromWorkItem()

	// Check static input population
	if m.planningInputs[0].Value() != "3.0" {
		t.Errorf("planningInputs[0] = %v, want %v", m.planningInputs[0].Value(), "3.0")
	}
	if m.planningInputs[1].Value() != "10.0" {
		t.Errorf("planningInputs[1] = %v, want %v", m.planningInputs[1].Value(), "10.0")
	}
	if m.planningInputs[2].Value() != "6.0" {
		t.Errorf("planningInputs[2] = %v, want %v", m.planningInputs[2].Value(), "6.0")
	}
	if m.planningInputs[3].Value() != "4.0" {
		t.Errorf("planningInputs[3] = %v, want %v", m.planningInputs[3].Value(), "4.0")
	}
}

func TestUpdatePlanningInputsFromWorkItemNilValues(t *testing.T) {
	m := NewModel()
	m.selectedItem = &azdo.WorkItem{
		ID: 789,
		Fields: azdo.WorkItemFields{
			Title:        "Test Item",
			WorkItemType: "Bug",
			// All planning fields are nil
		},
	}

	m.updatePlanningInputsFromWorkItem()

	// All inputs should be empty
	for i := 0; i < 4; i++ {
		if m.planningInputs[i].Value() != "" {
			t.Errorf("planningInputs[%d] should be empty, got %v", i, m.planningInputs[i].Value())
		}
	}
}

func TestPlanningStateInitialization(t *testing.T) {
	m := NewModel()

	// Check that planning state is properly initialized
	if m.planningExpanded {
		t.Error("planningExpanded should be false initially")
	}
	if m.planningFocus != 0 {
		t.Errorf("planningFocus = %v, want %v", m.planningFocus, 0)
	}
	if len(m.planningInputs) != 4 {
		t.Errorf("planningInputs length = %v, want %v", len(m.planningInputs), 4)
	}
	if m.planningFields != nil {
		t.Error("planningFields should be nil initially")
	}
}

func TestPlanningFieldsCount(t *testing.T) {
	tests := []struct {
		name       string
		fields     []azdo.PlanningField
		wantLength int
	}{
		{
			name:       "empty fields",
			fields:     []azdo.PlanningField{},
			wantLength: 0,
		},
		{
			name: "single field",
			fields: []azdo.PlanningField{
				{ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints", DisplayName: "Story Points"},
			},
			wantLength: 1,
		},
		{
			name: "multiple fields",
			fields: []azdo.PlanningField{
				{ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints", DisplayName: "Story Points"},
				{ReferenceName: "Microsoft.VSTS.Scheduling.OriginalEstimate", DisplayName: "Original Estimate"},
				{ReferenceName: "Microsoft.VSTS.Scheduling.RemainingWork", DisplayName: "Remaining Work"},
			},
			wantLength: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				planningFields: tt.fields,
			}
			if len(m.planningFields) != tt.wantLength {
				t.Errorf("planningFields length = %v, want %v", len(m.planningFields), tt.wantLength)
			}
		})
	}
}
