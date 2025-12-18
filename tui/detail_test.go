package tui

import (
	"bored/azdo"
	"strings"
	"testing"
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
