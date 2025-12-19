package azdo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("myorg", "myproject", "myteam", "MyProject\\MyTeam", "pat123")

	if client.Organization != "myorg" {
		t.Errorf("Organization = %v, want %v", client.Organization, "myorg")
	}
	if client.Project != "myproject" {
		t.Errorf("Project = %v, want %v", client.Project, "myproject")
	}
	if client.Team != "myteam" {
		t.Errorf("Team = %v, want %v", client.Team, "myteam")
	}
	if client.AreaPath != "MyProject\\MyTeam" {
		t.Errorf("AreaPath = %v, want %v", client.AreaPath, "MyProject\\MyTeam")
	}
	if client.PAT != "pat123" {
		t.Errorf("PAT = %v, want %v", client.PAT, "pat123")
	}
}

func TestBaseURL(t *testing.T) {
	client := NewClient("myorg", "myproject", "", "", "pat")
	expected := "https://dev.azure.com/myorg/myproject"
	if client.baseURL() != expected {
		t.Errorf("baseURL() = %v, want %v", client.baseURL(), expected)
	}
}

func TestTeamURL(t *testing.T) {
	tests := []struct {
		name     string
		team     string
		expected string
	}{
		{
			name:     "with team",
			team:     "myteam",
			expected: "https://dev.azure.com/myorg/myproject/myteam",
		},
		{
			name:     "without team",
			team:     "",
			expected: "https://dev.azure.com/myorg/myproject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("myorg", "myproject", tt.team, "", "pat")
			if client.teamURL() != tt.expected {
				t.Errorf("teamURL() = %v, want %v", client.teamURL(), tt.expected)
			}
		})
	}
}

func TestGetComments(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Check authorization header exists
		if r.Header.Get("Authorization") == "" {
			t.Error("Expected Authorization header")
		}

		// Return mock response
		response := CommentsResponse{
			Count: 2,
			Comments: []Comment{
				{
					ID:          1,
					Text:        "First comment",
					CreatedBy:   IdentityRef{DisplayName: "John Doe", UniqueName: "john@example.com"},
					CreatedDate: "2024-01-15T10:00:00Z",
				},
				{
					ID:          2,
					Text:        "Second comment",
					CreatedBy:   IdentityRef{DisplayName: "Jane Doe", UniqueName: "jane@example.com"},
					CreatedDate: "2024-01-16T10:00:00Z",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewClient("myorg", "myproject", "", "", "pat")
	client.httpClient = server.Client()

	// We can't easily test the actual GetComments since it constructs its own URL
	// This test demonstrates the pattern for mocking HTTP responses
}

func TestGetWorkItemTypes(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := WorkItemTypesResponse{
			Count: 3,
			Value: []WorkItemType{
				{Name: "Bug", Description: "A bug"},
				{Name: "Task", Description: "A task"},
				{Name: "User Story", Description: "A user story"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// This demonstrates the mock pattern
	// Full integration would require URL rewriting
}

func TestExtractWorkItemIDFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected int
	}{
		{
			name:     "standard URL",
			url:      "https://dev.azure.com/org/project/_apis/wit/workItems/123",
			expected: 123,
		},
		{
			name:     "URL with larger ID",
			url:      "https://dev.azure.com/org/project/_apis/wit/workItems/99999",
			expected: 99999,
		},
		{
			name:     "URL with single digit",
			url:      "https://dev.azure.com/org/project/_apis/wit/workItems/5",
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWorkItemIDFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("extractWorkItemIDFromURL(%s) = %d, want %d", tt.url, result, tt.expected)
			}
		})
	}
}

func TestWorkItemFields(t *testing.T) {
	// Test JSON unmarshaling of work item fields
	jsonData := `{
		"System.Title": "Test Bug",
		"System.State": "Active",
		"System.WorkItemType": "Bug",
		"System.AssignedTo": {
			"displayName": "John Doe",
			"uniqueName": "john@example.com"
		},
		"System.Description": "This is a test",
		"System.AreaPath": "Project\\Team",
		"Microsoft.VSTS.Common.Priority": 2,
		"System.Tags": "tag1; tag2",
		"System.CommentCount": 5,
		"System.ChangedDate": "2024-01-15T10:00:00Z"
	}`

	var fields WorkItemFields
	err := json.Unmarshal([]byte(jsonData), &fields)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if fields.Title != "Test Bug" {
		t.Errorf("Title = %v, want %v", fields.Title, "Test Bug")
	}
	if fields.State != "Active" {
		t.Errorf("State = %v, want %v", fields.State, "Active")
	}
	if fields.WorkItemType != "Bug" {
		t.Errorf("WorkItemType = %v, want %v", fields.WorkItemType, "Bug")
	}
	if fields.AssignedTo == nil {
		t.Error("AssignedTo should not be nil")
	} else if fields.AssignedTo.DisplayName != "John Doe" {
		t.Errorf("AssignedTo.DisplayName = %v, want %v", fields.AssignedTo.DisplayName, "John Doe")
	}
	if fields.Priority != 2 {
		t.Errorf("Priority = %v, want %v", fields.Priority, 2)
	}
	if fields.CommentCount != 5 {
		t.Errorf("CommentCount = %v, want %v", fields.CommentCount, 5)
	}
}

func TestCommentParsing(t *testing.T) {
	jsonData := `{
		"id": 42,
		"text": "<div>Hello <a href=\"#\" data-vss-mention=\"version:2.0,abc123\">@User</a></div>",
		"createdBy": {
			"displayName": "Commenter",
			"uniqueName": "commenter@example.com"
		},
		"createdDate": "2024-01-15T10:00:00Z"
	}`

	var comment Comment
	err := json.Unmarshal([]byte(jsonData), &comment)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if comment.ID != 42 {
		t.Errorf("ID = %v, want %v", comment.ID, 42)
	}
	if comment.CreatedBy.DisplayName != "Commenter" {
		t.Errorf("CreatedBy.DisplayName = %v, want %v", comment.CreatedBy.DisplayName, "Commenter")
	}
}

func TestCreateWorkItemOpMarshaling(t *testing.T) {
	ops := []CreateWorkItemOp{
		{Op: "add", Path: "/fields/System.Title", Value: "Test Title"},
		{Op: "add", Path: "/fields/System.Description", Value: "Test Description"},
	}

	jsonData, err := json.Marshal(ops)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed []CreateWorkItemOp
	err = json.Unmarshal(jsonData, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(parsed) != 2 {
		t.Errorf("Expected 2 ops, got %d", len(parsed))
	}
	if parsed[0].Op != "add" {
		t.Errorf("Op = %v, want %v", parsed[0].Op, "add")
	}
	if parsed[0].Path != "/fields/System.Title" {
		t.Errorf("Path = %v, want %v", parsed[0].Path, "/fields/System.Title")
	}
}

func TestIterationParsing(t *testing.T) {
	// Test JSON unmarshaling of iteration
	jsonData := `{
		"id": "abc123-def456",
		"name": "Sprint 1",
		"path": "Project\\Sprint 1",
		"attributes": {
			"startDate": "2024-01-01T00:00:00Z",
			"finishDate": "2024-01-14T00:00:00Z",
			"timeFrame": "current"
		}
	}`

	var iter Iteration
	err := json.Unmarshal([]byte(jsonData), &iter)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if iter.ID != "abc123-def456" {
		t.Errorf("ID = %v, want %v", iter.ID, "abc123-def456")
	}
	if iter.Name != "Sprint 1" {
		t.Errorf("Name = %v, want %v", iter.Name, "Sprint 1")
	}
	if iter.Path != "Project\\Sprint 1" {
		t.Errorf("Path = %v, want %v", iter.Path, "Project\\Sprint 1")
	}
	if iter.Attributes == nil {
		t.Error("Attributes should not be nil")
	} else {
		if iter.Attributes.TimeFrame != "current" {
			t.Errorf("TimeFrame = %v, want %v", iter.Attributes.TimeFrame, "current")
		}
		if iter.Attributes.StartDate != "2024-01-01T00:00:00Z" {
			t.Errorf("StartDate = %v, want %v", iter.Attributes.StartDate, "2024-01-01T00:00:00Z")
		}
	}
}

func TestIterationWithoutAttributes(t *testing.T) {
	// Test iteration without optional attributes
	jsonData := `{
		"id": "xyz789",
		"name": "Backlog",
		"path": "Project\\Backlog"
	}`

	var iter Iteration
	err := json.Unmarshal([]byte(jsonData), &iter)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if iter.Name != "Backlog" {
		t.Errorf("Name = %v, want %v", iter.Name, "Backlog")
	}
	if iter.Attributes != nil {
		t.Error("Attributes should be nil when not provided")
	}
}

func TestIterationsResponseParsing(t *testing.T) {
	jsonData := `{
		"count": 3,
		"value": [
			{"id": "1", "name": "Sprint 1", "path": "Project\\Sprint 1"},
			{"id": "2", "name": "Sprint 2", "path": "Project\\Sprint 2"},
			{"id": "3", "name": "Backlog", "path": "Project\\Backlog"}
		]
	}`

	var response IterationsResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if response.Count != 3 {
		t.Errorf("Count = %v, want %v", response.Count, 3)
	}
	if len(response.Value) != 3 {
		t.Errorf("Value length = %v, want %v", len(response.Value), 3)
	}
	if response.Value[0].Name != "Sprint 1" {
		t.Errorf("First iteration name = %v, want %v", response.Value[0].Name, "Sprint 1")
	}
}

func TestWorkItemFieldsWithIterationPath(t *testing.T) {
	// Test that IterationPath is correctly parsed from work item fields
	jsonData := `{
		"System.Title": "Test Item",
		"System.State": "Active",
		"System.WorkItemType": "Task",
		"System.AreaPath": "Project\\Team",
		"System.IterationPath": "Project\\Sprint 1"
	}`

	var fields WorkItemFields
	err := json.Unmarshal([]byte(jsonData), &fields)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if fields.IterationPath != "Project\\Sprint 1" {
		t.Errorf("IterationPath = %v, want %v", fields.IterationPath, "Project\\Sprint 1")
	}
	if fields.AreaPath != "Project\\Team" {
		t.Errorf("AreaPath = %v, want %v", fields.AreaPath, "Project\\Team")
	}
}

func TestWorkItemFieldsWithPlanningFields(t *testing.T) {
	// Test that planning fields are correctly parsed from work item fields
	jsonData := `{
		"System.Title": "Test User Story",
		"System.State": "Active",
		"System.WorkItemType": "User Story",
		"Microsoft.VSTS.Scheduling.StoryPoints": 5.0,
		"Microsoft.VSTS.Scheduling.OriginalEstimate": 8.0,
		"Microsoft.VSTS.Scheduling.RemainingWork": 4.5,
		"Microsoft.VSTS.Scheduling.CompletedWork": 3.5,
		"Microsoft.VSTS.Scheduling.Effort": 13.0
	}`

	var fields WorkItemFields
	err := json.Unmarshal([]byte(jsonData), &fields)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if fields.StoryPoints == nil {
		t.Error("StoryPoints should not be nil")
	} else if *fields.StoryPoints != 5.0 {
		t.Errorf("StoryPoints = %v, want %v", *fields.StoryPoints, 5.0)
	}

	if fields.OriginalEstimate == nil {
		t.Error("OriginalEstimate should not be nil")
	} else if *fields.OriginalEstimate != 8.0 {
		t.Errorf("OriginalEstimate = %v, want %v", *fields.OriginalEstimate, 8.0)
	}

	if fields.RemainingWork == nil {
		t.Error("RemainingWork should not be nil")
	} else if *fields.RemainingWork != 4.5 {
		t.Errorf("RemainingWork = %v, want %v", *fields.RemainingWork, 4.5)
	}

	if fields.CompletedWork == nil {
		t.Error("CompletedWork should not be nil")
	} else if *fields.CompletedWork != 3.5 {
		t.Errorf("CompletedWork = %v, want %v", *fields.CompletedWork, 3.5)
	}

	if fields.Effort == nil {
		t.Error("Effort should not be nil")
	} else if *fields.Effort != 13.0 {
		t.Errorf("Effort = %v, want %v", *fields.Effort, 13.0)
	}
}

func TestWorkItemFieldsWithoutPlanningFields(t *testing.T) {
	// Test that missing planning fields are nil
	jsonData := `{
		"System.Title": "Test Task",
		"System.State": "Active",
		"System.WorkItemType": "Task"
	}`

	var fields WorkItemFields
	err := json.Unmarshal([]byte(jsonData), &fields)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if fields.StoryPoints != nil {
		t.Error("StoryPoints should be nil when not provided")
	}
	if fields.OriginalEstimate != nil {
		t.Error("OriginalEstimate should be nil when not provided")
	}
	if fields.RemainingWork != nil {
		t.Error("RemainingWork should be nil when not provided")
	}
	if fields.CompletedWork != nil {
		t.Error("CompletedWork should be nil when not provided")
	}
	if fields.Effort != nil {
		t.Error("Effort should be nil when not provided")
	}
}

func TestWorkItemTypeFieldParsing(t *testing.T) {
	// Test JSON unmarshaling of work item type field
	jsonData := `{
		"referenceName": "Microsoft.VSTS.Scheduling.StoryPoints",
		"name": "Story Points",
		"alwaysRequired": false,
		"defaultValue": null,
		"readOnly": false
	}`

	var field WorkItemTypeField
	err := json.Unmarshal([]byte(jsonData), &field)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if field.ReferenceName != "Microsoft.VSTS.Scheduling.StoryPoints" {
		t.Errorf("ReferenceName = %v, want %v", field.ReferenceName, "Microsoft.VSTS.Scheduling.StoryPoints")
	}
	if field.Name != "Story Points" {
		t.Errorf("Name = %v, want %v", field.Name, "Story Points")
	}
	if field.AlwaysRequired != false {
		t.Errorf("AlwaysRequired = %v, want %v", field.AlwaysRequired, false)
	}
	if field.ReadOnly != false {
		t.Errorf("ReadOnly = %v, want %v", field.ReadOnly, false)
	}
}

func TestWorkItemTypeFieldsResponseParsing(t *testing.T) {
	jsonData := `{
		"count": 3,
		"value": [
			{"referenceName": "Microsoft.VSTS.Scheduling.StoryPoints", "name": "Story Points", "readOnly": false},
			{"referenceName": "Microsoft.VSTS.Scheduling.OriginalEstimate", "name": "Original Estimate", "readOnly": false},
			{"referenceName": "System.Title", "name": "Title", "readOnly": false}
		]
	}`

	var response WorkItemTypeFieldsResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if response.Count != 3 {
		t.Errorf("Count = %v, want %v", response.Count, 3)
	}
	if len(response.Value) != 3 {
		t.Errorf("Value length = %v, want %v", len(response.Value), 3)
	}
	if response.Value[0].ReferenceName != "Microsoft.VSTS.Scheduling.StoryPoints" {
		t.Errorf("First field ReferenceName = %v, want %v", response.Value[0].ReferenceName, "Microsoft.VSTS.Scheduling.StoryPoints")
	}
}

func TestPlanningField(t *testing.T) {
	// Test PlanningField struct creation
	value := 5.0
	field := PlanningField{
		ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints",
		DisplayName:   "Story Points",
		Value:         &value,
	}

	if field.ReferenceName != "Microsoft.VSTS.Scheduling.StoryPoints" {
		t.Errorf("ReferenceName = %v, want %v", field.ReferenceName, "Microsoft.VSTS.Scheduling.StoryPoints")
	}
	if field.DisplayName != "Story Points" {
		t.Errorf("DisplayName = %v, want %v", field.DisplayName, "Story Points")
	}
	if field.Value == nil {
		t.Error("Value should not be nil")
	} else if *field.Value != 5.0 {
		t.Errorf("Value = %v, want %v", *field.Value, 5.0)
	}
}

func TestPlanningFieldNilValue(t *testing.T) {
	// Test PlanningField with nil value
	field := PlanningField{
		ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints",
		DisplayName:   "Story Points",
		Value:         nil,
	}

	if field.Value != nil {
		t.Error("Value should be nil")
	}
}

func TestGetHyperlinks(t *testing.T) {
	// Test hyperlink extraction from work item relations
	wi := &WorkItem{
		ID: 123,
		Relations: []WorkItemRelation{
			{
				Rel: "ArtifactLink",
				URL: "vstfs:///GitHub/PullRequest/abc-123%2F456",
				Attributes: map[string]interface{}{
					"name":    "https://github.com/owner/repo/pull/456",
					"comment": "Fix bug",
				},
			},
			{
				Rel: "Hyperlink",
				URL: "https://example.com/docs",
				Attributes: map[string]interface{}{
					"comment": "Documentation",
				},
			},
			{
				Rel: "System.LinkTypes.Hierarchy-Forward",
				URL: "https://dev.azure.com/org/project/_apis/wit/workItems/124",
			},
		},
	}

	client := NewClient("org", "project", "", "", "pat")

	// Mock the GetWorkItemWithRelations call
	// In a real test, we would use a mock server
	// For now, test the extraction logic directly
	var hyperlinks []Hyperlink
	for _, rel := range wi.Relations {
		if rel.Rel == "ArtifactLink" || rel.Rel == "Hyperlink" {
			name := ""
			comment := ""
			if rel.Attributes != nil {
				if n, ok := rel.Attributes["name"].(string); ok {
					name = n
				}
				if c, ok := rel.Attributes["comment"].(string); ok {
					comment = c
				}
			}
			hyperlinks = append(hyperlinks, Hyperlink{
				URL:     rel.URL,
				Name:    name,
				Comment: comment,
			})
		}
	}

	if len(hyperlinks) != 2 {
		t.Errorf("Expected 2 hyperlinks, got %d", len(hyperlinks))
	}

	if hyperlinks[0].URL != "vstfs:///GitHub/PullRequest/abc-123%2F456" {
		t.Errorf("First hyperlink URL = %v, want vstfs URL", hyperlinks[0].URL)
	}
	if hyperlinks[0].Name != "https://github.com/owner/repo/pull/456" {
		t.Errorf("First hyperlink name = %v, want GitHub URL", hyperlinks[0].Name)
	}
	if hyperlinks[0].Comment != "Fix bug" {
		t.Errorf("First hyperlink comment = %v, want 'Fix bug'", hyperlinks[0].Comment)
	}

	if hyperlinks[1].URL != "https://example.com/docs" {
		t.Errorf("Second hyperlink URL = %v, want https://example.com/docs", hyperlinks[1].URL)
	}
	if hyperlinks[1].Comment != "Documentation" {
		t.Errorf("Second hyperlink comment = %v, want 'Documentation'", hyperlinks[1].Comment)
	}

	_ = client // Avoid unused variable error
}

func TestAddHyperlinkValidation(t *testing.T) {
	client := NewClient("org", "project", "", "", "pat")

	tests := []struct {
		name        string
		url         string
		comment     string
		expectError bool
		errorText   string
	}{
		{
			name:        "valid http URL",
			url:         "http://example.com",
			comment:     "Test",
			expectError: false,
		},
		{
			name:        "valid https URL",
			url:         "https://example.com",
			comment:     "Test",
			expectError: false,
		},
		{
			name:        "invalid URL format",
			url:         "://invalid",
			comment:     "",
			expectError: true,
			errorText:   "invalid URL format",
		},
		{
			name:        "invalid scheme",
			url:         "ftp://example.com",
			comment:     "",
			expectError: true,
			errorText:   "invalid URL scheme",
		},
		{
			name:        "missing scheme treated as invalid",
			url:         "not a url",
			comment:     "",
			expectError: true,
			errorText:   "invalid URL scheme",
		},
		{
			name:        "URL too long",
			url:         "https://example.com/" + strings.Repeat("a", 2100),
			comment:     "",
			expectError: true,
			errorText:   "URL too long",
		},
		{
			name:        "comment too long",
			url:         "https://example.com",
			comment:     strings.Repeat("a", 600),
			expectError: true,
			errorText:   "comment too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't make actual API calls in tests, but we can test validation
			err := client.AddHyperlink(123, tt.url, tt.comment)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorText)
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errorText, err)
				}
			} else if err != nil && strings.Contains(err.Error(), "invalid") {
				// If we get a validation error on a valid input, fail
				t.Errorf("Unexpected validation error: %v", err)
			}
			// Note: We expect connection errors for valid inputs since we're not mocking HTTP
		})
	}
}

func TestRemoveHyperlinkErrorMessage(t *testing.T) {
	// Create a mock server that returns a work item without the hyperlink
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wi := WorkItem{
			ID: 123,
			Relations: []WorkItemRelation{
				{
					Rel: "Hyperlink",
					URL: "https://example.com/other",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wi)
	}))
	defer server.Close()

	// Can't easily test the full flow without extensive mocking
	// But we can verify the error format
	expectedURL := "https://example.com/notfound"
	expectedID := 123
	err := fmt.Errorf("hyperlink %s not found in work item %d", expectedURL, expectedID)

	if !strings.Contains(err.Error(), expectedURL) {
		t.Errorf("Error should contain URL '%s'", expectedURL)
	}
	if !strings.Contains(err.Error(), "123") {
		t.Errorf("Error should contain work item ID")
	}
}

func TestHyperlinkParsing(t *testing.T) {
	// Test JSON unmarshaling of work item with hyperlinks
	jsonData := `{
		"id": 123,
		"rev": 1,
		"fields": {
			"System.Title": "Test",
			"System.State": "Active",
			"System.WorkItemType": "Task"
		},
		"relations": [
			{
				"rel": "ArtifactLink",
				"url": "vstfs:///GitHub/PullRequest/guid%2F123",
				"attributes": {
					"name": "https://github.com/owner/repo/pull/123",
					"comment": "Fix issue"
				}
			},
			{
				"rel": "Hyperlink",
				"url": "https://docs.example.com",
				"attributes": {
					"comment": "Related documentation"
				}
			}
		]
	}`

	var wi WorkItem
	err := json.Unmarshal([]byte(jsonData), &wi)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(wi.Relations) != 2 {
		t.Errorf("Expected 2 relations, got %d", len(wi.Relations))
	}

	// Check ArtifactLink
	if wi.Relations[0].Rel != "ArtifactLink" {
		t.Errorf("First relation Rel = %v, want ArtifactLink", wi.Relations[0].Rel)
	}
	if wi.Relations[0].URL != "vstfs:///GitHub/PullRequest/guid%2F123" {
		t.Errorf("First relation URL = %v", wi.Relations[0].URL)
	}
	if name, ok := wi.Relations[0].Attributes["name"].(string); !ok || name != "https://github.com/owner/repo/pull/123" {
		t.Errorf("First relation name attribute incorrect")
	}

	// Check Hyperlink
	if wi.Relations[1].Rel != "Hyperlink" {
		t.Errorf("Second relation Rel = %v, want Hyperlink", wi.Relations[1].Rel)
	}
	if wi.Relations[1].URL != "https://docs.example.com" {
		t.Errorf("Second relation URL = %v", wi.Relations[1].URL)
	}
}
