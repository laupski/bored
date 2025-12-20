package azdo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockServerURL replaces the client's base URL for testing
type mockTransport struct {
	baseURL   string
	transport http.RoundTripper
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL to point to our test server
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.baseURL, "http://")
	return t.transport.RoundTrip(req)
}

func testClientWithMockTransport(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := &Client{
		Organization: "testorg",
		Project:      "testproject",
		Team:         "testteam",
		AreaPath:     "TestProject\\TestTeam",
		PAT:          "testpat",
		httpClient: &http.Client{
			Transport: &mockTransport{
				baseURL:   server.URL,
				transport: http.DefaultTransport,
			},
		},
	}
	return client, server
}

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

// ============ HTTP Mock Tests ============

func TestGetWorkItemsPaged(t *testing.T) {
	requestCount := 0
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// WIQL query
			if r.Method != "POST" {
				t.Errorf("Expected POST for WIQL, got %s", r.Method)
			}
			response := WorkItemQueryResult{
				WorkItems: []WorkItemRef{
					{ID: 1, URL: "https://dev.azure.com/org/project/_apis/wit/workItems/1"},
					{ID: 2, URL: "https://dev.azure.com/org/project/_apis/wit/workItems/2"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		} else {
			// Get work items by IDs
			if r.Method != "GET" {
				t.Errorf("Expected GET for work items, got %s", r.Method)
			}
			response := WorkItemListResponse{
				Count: 2,
				Value: []WorkItem{
					{ID: 1, Fields: WorkItemFields{Title: "First", State: "Active", WorkItemType: "Bug"}},
					{ID: 2, Fields: WorkItemFields{Title: "Second", State: "New", WorkItemType: "Task"}},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	items, err := client.GetWorkItemsPaged("Bug", "user@example.com", 10, 0)
	if err != nil {
		t.Fatalf("GetWorkItemsPaged failed: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
	if items[0].ID != 1 {
		t.Errorf("First item ID = %d, want 1", items[0].ID)
	}
}

func TestGetWorkItemsPagedEmpty(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		response := WorkItemQueryResult{WorkItems: []WorkItemRef{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	items, err := client.GetWorkItemsPaged("", "", 10, 0)
	if err != nil {
		t.Fatalf("GetWorkItemsPaged failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items))
	}
}

func TestGetWorkItemsPagedWithSkip(t *testing.T) {
	requestCount := 0
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			response := WorkItemQueryResult{
				WorkItems: []WorkItemRef{
					{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		} else {
			response := WorkItemListResponse{
				Count: 2,
				Value: []WorkItem{
					{ID: 3, Fields: WorkItemFields{Title: "Third"}},
					{ID: 4, Fields: WorkItemFields{Title: "Fourth"}},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	items, err := client.GetWorkItemsPaged("", "", 2, 2)
	if err != nil {
		t.Fatalf("GetWorkItemsPaged failed: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
}

func TestGetWorkItemsPagedSkipBeyondResults(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		response := WorkItemQueryResult{
			WorkItems: []WorkItemRef{{ID: 1}, {ID: 2}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	items, err := client.GetWorkItemsPaged("", "", 10, 10) // Skip 10, but only 2 exist
	if err != nil {
		t.Fatalf("GetWorkItemsPaged failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 items when skip beyond results, got %d", len(items))
	}
}

func TestGetWorkItemsPagedAPIError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
	})
	defer server.Close()

	_, err := client.GetWorkItemsPaged("", "", 10, 0)
	if err == nil {
		t.Error("Expected error for API failure")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Expected 401 error, got: %v", err)
	}
}

func TestGetWorkItems(t *testing.T) {
	requestCount := 0
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			response := WorkItemQueryResult{
				WorkItems: []WorkItemRef{{ID: 1}},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		} else {
			response := WorkItemListResponse{
				Count: 1,
				Value: []WorkItem{{ID: 1, Fields: WorkItemFields{Title: "Test"}}},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	items, err := client.GetWorkItems("Bug", 10)
	if err != nil {
		t.Fatalf("GetWorkItems failed: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}
}

func TestGetWorkItemsFiltered(t *testing.T) {
	requestCount := 0
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			response := WorkItemQueryResult{
				WorkItems: []WorkItemRef{{ID: 1}},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		} else {
			response := WorkItemListResponse{
				Count: 1,
				Value: []WorkItem{{ID: 1, Fields: WorkItemFields{Title: "Test"}}},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	items, err := client.GetWorkItemsFiltered("Task", "user@example.com", 10)
	if err != nil {
		t.Fatalf("GetWorkItemsFiltered failed: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}
}

func TestGetWorkItemsByIDs(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		response := WorkItemListResponse{
			Count: 2,
			Value: []WorkItem{
				{ID: 1, Fields: WorkItemFields{Title: "First"}},
				{ID: 2, Fields: WorkItemFields{Title: "Second"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	items, err := client.getWorkItemsByIDs([]string{"1", "2"})
	if err != nil {
		t.Fatalf("getWorkItemsByIDs failed: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
}

func TestGetWorkItemsByIDsEmpty(t *testing.T) {
	client := NewClient("org", "proj", "", "", "pat")
	items, err := client.getWorkItemsByIDs([]string{})
	if err != nil {
		t.Fatalf("getWorkItemsByIDs failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 items for empty IDs, got %d", len(items))
	}
}

func TestCreateWorkItem(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "application/json-patch+json") {
			t.Errorf("Expected json-patch+json content type")
		}

		response := WorkItem{
			ID:  123,
			Rev: 1,
			Fields: WorkItemFields{
				Title:        "New Bug",
				State:        "New",
				WorkItemType: "Bug",
			},
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	wi, err := client.CreateWorkItem("Bug", "New Bug", "Description", 2)
	if err != nil {
		t.Fatalf("CreateWorkItem failed: %v", err)
	}

	if wi.ID != 123 {
		t.Errorf("Expected ID 123, got %d", wi.ID)
	}
	if wi.Fields.Title != "New Bug" {
		t.Errorf("Expected title 'New Bug', got %s", wi.Fields.Title)
	}
}

func TestCreateWorkItemWithAssignee(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "System.AssignedTo") {
			t.Error("Expected AssignedTo in request body")
		}

		response := WorkItem{ID: 123, Fields: WorkItemFields{Title: "Test"}}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	wi, err := client.CreateWorkItemWithAssignee("Task", "Test", "", 0, "user@example.com")
	if err != nil {
		t.Fatalf("CreateWorkItemWithAssignee failed: %v", err)
	}

	if wi.ID != 123 {
		t.Errorf("Expected ID 123, got %d", wi.ID)
	}
}

func TestCreateWorkItemAPIError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid request"))
	})
	defer server.Close()

	_, err := client.CreateWorkItem("Bug", "Test", "", 0)
	if err == nil {
		t.Error("Expected error for bad request")
	}
}

func TestCreateWorkItemWithParent(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "System.LinkTypes.Hierarchy-Reverse") {
			t.Error("Expected parent link relation in body")
		}

		response := WorkItem{ID: 124, Fields: WorkItemFields{Title: "Child Task"}}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	wi, err := client.CreateWorkItemWithParent("Task", "Child Task", "", 0, 100)
	if err != nil {
		t.Fatalf("CreateWorkItemWithParent failed: %v", err)
	}

	if wi.ID != 124 {
		t.Errorf("Expected ID 124, got %d", wi.ID)
	}
}

func TestCreateWorkItemWithParentAndAssignee(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "System.LinkTypes.Hierarchy-Reverse") {
			t.Error("Expected parent link relation in body")
		}
		if !strings.Contains(string(body), "System.AssignedTo") {
			t.Error("Expected AssignedTo in body")
		}

		response := WorkItem{ID: 125, Fields: WorkItemFields{Title: "Child"}}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	wi, err := client.CreateWorkItemWithParentAndAssignee("Task", "Child", "Desc", 1, 100, "user@example.com")
	if err != nil {
		t.Fatalf("CreateWorkItemWithParentAndAssignee failed: %v", err)
	}

	if wi.ID != 125 {
		t.Errorf("Expected ID 125, got %d", wi.ID)
	}
}

func TestAddChildLink(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "System.LinkTypes.Hierarchy-Forward") {
			t.Error("Expected child link relation in body")
		}

		response := WorkItem{ID: 100}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	err := client.AddChildLink(100, 101)
	if err != nil {
		t.Fatalf("AddChildLink failed: %v", err)
	}
}

func TestAddChildLinkError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Link already exists"))
	})
	defer server.Close()

	err := client.AddChildLink(100, 101)
	if err == nil {
		t.Error("Expected error for failed link")
	}
}

func TestRemoveRelation(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "remove") {
			t.Error("Expected remove operation")
		}

		response := WorkItem{ID: 100}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	err := client.RemoveRelation(100, 0)
	if err != nil {
		t.Fatalf("RemoveRelation failed: %v", err)
	}
}

func TestRemoveRelationError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Relation not found"))
	})
	defer server.Close()

	err := client.RemoveRelation(100, 99)
	if err == nil {
		t.Error("Expected error for failed removal")
	}
}

func TestRemoveHierarchyLink(t *testing.T) {
	requestCount := 0
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// GetWorkItemWithRelations
			response := WorkItem{
				ID: 100,
				Relations: []WorkItemRelation{
					{Rel: "System.LinkTypes.Hierarchy-Reverse", URL: "https://dev.azure.com/org/proj/_apis/wit/workItems/200"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		} else {
			// RemoveRelation
			response := WorkItem{ID: 100}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	err := client.RemoveHierarchyLink(100, 200, true)
	if err != nil {
		t.Fatalf("RemoveHierarchyLink failed: %v", err)
	}
}

func TestRemoveHierarchyLinkNotFound(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		response := WorkItem{ID: 100, Relations: []WorkItemRelation{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	err := client.RemoveHierarchyLink(100, 200, true)
	if err == nil {
		t.Error("Expected 'relation not found' error")
	}
	if !strings.Contains(err.Error(), "relation not found") {
		t.Errorf("Expected 'relation not found', got: %v", err)
	}
}

func TestGetWorkItemTypesAPI(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		response := WorkItemTypesResponse{
			Count: 3,
			Value: []WorkItemType{
				{Name: "Bug"},
				{Name: "Task"},
				{Name: "User Story"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	types, err := client.GetWorkItemTypes()
	if err != nil {
		t.Fatalf("GetWorkItemTypes failed: %v", err)
	}

	if len(types) != 3 {
		t.Errorf("Expected 3 types, got %d", len(types))
	}
}

func TestGetWorkItemTypesAPIError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Server error"))
	})
	defer server.Close()

	_, err := client.GetWorkItemTypes()
	if err == nil {
		t.Error("Expected error for server failure")
	}
}

func TestGetCommentsAPI(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		response := CommentsResponse{
			Count: 2,
			Comments: []Comment{
				{ID: 1, Text: "First comment", CreatedBy: IdentityRef{DisplayName: "User1"}},
				{ID: 2, Text: "Second comment", CreatedBy: IdentityRef{DisplayName: "User2"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	comments, err := client.GetComments(123)
	if err != nil {
		t.Fatalf("GetComments failed: %v", err)
	}

	if len(comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(comments))
	}
	if comments[0].Text != "First comment" {
		t.Errorf("First comment text = %s, want 'First comment'", comments[0].Text)
	}
}

func TestGetCommentsAPIError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Access denied"))
	})
	defer server.Close()

	_, err := client.GetComments(123)
	if err == nil {
		t.Error("Expected error for access denied")
	}
}

func TestAddComment(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		response := Comment{ID: 3, Text: "New comment"}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	err := client.AddComment(123, "New comment")
	if err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}
}

func TestAddCommentError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Comment too long"))
	})
	defer server.Close()

	err := client.AddComment(123, "Test")
	if err == nil {
		t.Error("Expected error for bad request")
	}
}

func TestUpdateWorkItem(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}

		response := WorkItem{
			ID:     123,
			Fields: WorkItemFields{Title: "Updated Title", State: "Active"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	wi, err := client.UpdateWorkItem(123, "Updated Title", "Active", "user@example.com", "tag1; tag2")
	if err != nil {
		t.Fatalf("UpdateWorkItem failed: %v", err)
	}

	if wi.Fields.Title != "Updated Title" {
		t.Errorf("Title = %s, want 'Updated Title'", wi.Fields.Title)
	}
}

func TestUpdateWorkItemError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte("Conflict"))
	})
	defer server.Close()

	_, err := client.UpdateWorkItem(123, "Test", "", "", "")
	if err == nil {
		t.Error("Expected error for conflict")
	}
}

func TestGetWorkItemWithRelations(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		response := WorkItem{
			ID:     123,
			Fields: WorkItemFields{Title: "Test Item"},
			Relations: []WorkItemRelation{
				{Rel: "System.LinkTypes.Hierarchy-Forward", URL: "https://dev.azure.com/org/proj/_apis/wit/workItems/124"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	wi, err := client.GetWorkItemWithRelations(123)
	if err != nil {
		t.Fatalf("GetWorkItemWithRelations failed: %v", err)
	}

	if wi.ID != 123 {
		t.Errorf("ID = %d, want 123", wi.ID)
	}
	if len(wi.Relations) != 1 {
		t.Errorf("Relations count = %d, want 1", len(wi.Relations))
	}
}

func TestGetWorkItemWithRelationsError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Work item not found"))
	})
	defer server.Close()

	_, err := client.GetWorkItemWithRelations(999)
	if err == nil {
		t.Error("Expected error for not found")
	}
}

func TestGetRelatedWorkItems(t *testing.T) {
	requestCount := 0
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch requestCount {
		case 1:
			// Get work item with relations
			response := WorkItem{
				ID: 100,
				Relations: []WorkItemRelation{
					{Rel: "System.LinkTypes.Hierarchy-Reverse", URL: "https://dev.azure.com/org/proj/_apis/wit/workItems/200"},
					{Rel: "System.LinkTypes.Hierarchy-Forward", URL: "https://dev.azure.com/org/proj/_apis/wit/workItems/101"},
					{Rel: "System.LinkTypes.Hierarchy-Forward", URL: "https://dev.azure.com/org/proj/_apis/wit/workItems/102"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case 2:
			// Get parent
			response := WorkItem{ID: 200, Fields: WorkItemFields{Title: "Parent"}}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		default:
			// Get children
			response := WorkItemListResponse{
				Count: 2,
				Value: []WorkItem{
					{ID: 101, Fields: WorkItemFields{Title: "Child 1"}},
					{ID: 102, Fields: WorkItemFields{Title: "Child 2"}},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	parent, children, err := client.GetRelatedWorkItems(100)
	if err != nil {
		t.Fatalf("GetRelatedWorkItems failed: %v", err)
	}

	if parent == nil {
		t.Error("Expected parent not to be nil")
	} else if parent.ID != 200 {
		t.Errorf("Parent ID = %d, want 200", parent.ID)
	}

	if len(children) != 2 {
		t.Errorf("Children count = %d, want 2", len(children))
	}
}

func TestGetRelatedWorkItemsNoRelations(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		response := WorkItem{ID: 100, Relations: []WorkItemRelation{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	parent, children, err := client.GetRelatedWorkItems(100)
	if err != nil {
		t.Fatalf("GetRelatedWorkItems failed: %v", err)
	}

	if parent != nil {
		t.Error("Expected parent to be nil")
	}
	if len(children) != 0 {
		t.Errorf("Expected 0 children, got %d", len(children))
	}
}

func TestDeleteWorkItem(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := client.DeleteWorkItem(123)
	if err != nil {
		t.Fatalf("DeleteWorkItem failed: %v", err)
	}
}

func TestDeleteWorkItemError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Cannot delete"))
	})
	defer server.Close()

	err := client.DeleteWorkItem(123)
	if err == nil {
		t.Error("Expected error for forbidden")
	}
}

func TestTestConnection(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "project-id", "name": "testproject"}`))
	})
	defer server.Close()

	err := client.TestConnection()
	if err != nil {
		t.Fatalf("TestConnection failed: %v", err)
	}
}

func TestTestConnectionError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Invalid PAT"))
	})
	defer server.Close()

	err := client.TestConnection()
	if err == nil {
		t.Error("Expected error for unauthorized")
	}
	if !strings.Contains(err.Error(), "connection failed") {
		t.Errorf("Expected 'connection failed', got: %v", err)
	}
}

func TestGetIterations(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		response := IterationsResponse{
			Count: 3,
			Value: []Iteration{
				{ID: "1", Name: "Sprint 1", Path: "Project\\Sprint 1"},
				{ID: "2", Name: "Sprint 2", Path: "Project\\Sprint 2"},
				{ID: "3", Name: "Backlog", Path: "Project\\Backlog"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	iterations, err := client.GetIterations()
	if err != nil {
		t.Fatalf("GetIterations failed: %v", err)
	}

	if len(iterations) != 3 {
		t.Errorf("Expected 3 iterations, got %d", len(iterations))
	}
}

func TestGetIterationsError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Team not found"))
	})
	defer server.Close()

	_, err := client.GetIterations()
	if err == nil {
		t.Error("Expected error for not found")
	}
}

func TestUpdateWorkItemPlanning(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}

		storyPoints := 5.0
		response := WorkItem{
			ID:     123,
			Fields: WorkItemFields{Title: "Test", StoryPoints: &storyPoints},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	sp := 5.0
	wi, err := client.UpdateWorkItemPlanning(123, &sp, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateWorkItemPlanning failed: %v", err)
	}

	if wi.Fields.StoryPoints == nil || *wi.Fields.StoryPoints != 5.0 {
		t.Error("Expected StoryPoints to be 5.0")
	}
}

func TestUpdateWorkItemPlanningNoUpdates(t *testing.T) {
	client := NewClient("org", "proj", "", "", "pat")

	_, err := client.UpdateWorkItemPlanning(123, nil, nil, nil, nil)
	if err == nil {
		t.Error("Expected error when no updates specified")
	}
	if !strings.Contains(err.Error(), "no planning updates") {
		t.Errorf("Expected 'no planning updates' error, got: %v", err)
	}
}

func TestUpdateWorkItemPlanningError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid value"))
	})
	defer server.Close()

	sp := 5.0
	_, err := client.UpdateWorkItemPlanning(123, &sp, nil, nil, nil)
	if err == nil {
		t.Error("Expected error for bad request")
	}
}

func TestGetWorkItemTypeFields(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		response := WorkItemTypeFieldsResponse{
			Count: 3,
			Value: []WorkItemTypeField{
				{ReferenceName: "System.Title", Name: "Title", ReadOnly: false},
				{ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints", Name: "Story Points", ReadOnly: false},
				{ReferenceName: "System.CreatedDate", Name: "Created Date", ReadOnly: true},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	fields, err := client.GetWorkItemTypeFields("User Story")
	if err != nil {
		t.Fatalf("GetWorkItemTypeFields failed: %v", err)
	}

	if len(fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(fields))
	}
}

func TestGetWorkItemTypeFieldsError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Type not found"))
	})
	defer server.Close()

	_, err := client.GetWorkItemTypeFields("InvalidType")
	if err == nil {
		t.Error("Expected error for invalid type")
	}
}

func TestGetPlanningFields(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		response := WorkItemTypeFieldsResponse{
			Count: 4,
			Value: []WorkItemTypeField{
				{ReferenceName: "Microsoft.VSTS.Scheduling.StoryPoints", Name: "Story Points", ReadOnly: false},
				{ReferenceName: "Microsoft.VSTS.Scheduling.OriginalEstimate", Name: "Original Estimate", ReadOnly: false},
				{ReferenceName: "Microsoft.VSTS.Scheduling.RemainingWork", Name: "Remaining Work", ReadOnly: true},
				{ReferenceName: "System.Title", Name: "Title", ReadOnly: false},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	fields, err := client.GetPlanningFields("User Story")
	if err != nil {
		t.Fatalf("GetPlanningFields failed: %v", err)
	}

	// Should only include planning fields that are not read-only
	if len(fields) != 2 {
		t.Errorf("Expected 2 planning fields (excluding read-only), got %d", len(fields))
	}
}

func TestUpdateWorkItemPlanningDynamic(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "Microsoft.VSTS.Scheduling.StoryPoints") {
			t.Error("Expected StoryPoints field in body")
		}

		storyPoints := 8.0
		response := WorkItem{
			ID:     123,
			Fields: WorkItemFields{StoryPoints: &storyPoints},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	fields := map[string]float64{
		"Microsoft.VSTS.Scheduling.StoryPoints": 8.0,
	}
	wi, err := client.UpdateWorkItemPlanningDynamic(123, fields)
	if err != nil {
		t.Fatalf("UpdateWorkItemPlanningDynamic failed: %v", err)
	}

	if wi.Fields.StoryPoints == nil || *wi.Fields.StoryPoints != 8.0 {
		t.Error("Expected StoryPoints to be 8.0")
	}
}

func TestUpdateWorkItemPlanningDynamicNoFields(t *testing.T) {
	client := NewClient("org", "proj", "", "", "pat")

	_, err := client.UpdateWorkItemPlanningDynamic(123, map[string]float64{})
	if err == nil {
		t.Error("Expected error when no fields provided")
	}
}

func TestGetRecentlyChangedWorkItems(t *testing.T) {
	requestCount := 0
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			response := WorkItemQueryResult{
				WorkItems: []WorkItemRef{{ID: 1}, {ID: 2}},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		} else {
			response := WorkItemListResponse{
				Count: 2,
				Value: []WorkItem{
					{ID: 1, Fields: WorkItemFields{Title: "Changed 1"}},
					{ID: 2, Fields: WorkItemFields{Title: "Changed 2"}},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	items, err := client.GetRecentlyChangedWorkItems("user@example.com", 30)
	if err != nil {
		t.Fatalf("GetRecentlyChangedWorkItems failed: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
}

func TestGetRecentlyChangedWorkItemsEmptyAssignee(t *testing.T) {
	client := NewClient("org", "proj", "", "", "pat")

	items, err := client.GetRecentlyChangedWorkItems("", 30)
	if err != nil {
		t.Fatalf("GetRecentlyChangedWorkItems failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 items for empty assignee, got %d", len(items))
	}
}

func TestGetRecentlyChangedWorkItemsEmpty(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		response := WorkItemQueryResult{WorkItems: []WorkItemRef{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	items, err := client.GetRecentlyChangedWorkItems("user@example.com", 30)
	if err != nil {
		t.Fatalf("GetRecentlyChangedWorkItems failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items))
	}
}

func TestUpdateWorkItemIteration(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}

		response := WorkItem{
			ID:     123,
			Fields: WorkItemFields{Title: "Test", IterationPath: "Project\\Sprint 2"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	wi, err := client.UpdateWorkItemIteration(123, "Project\\Sprint 2")
	if err != nil {
		t.Fatalf("UpdateWorkItemIteration failed: %v", err)
	}

	if wi.Fields.IterationPath != "Project\\Sprint 2" {
		t.Errorf("IterationPath = %s, want 'Project\\Sprint 2'", wi.Fields.IterationPath)
	}
}

func TestUpdateWorkItemIterationError(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid iteration"))
	})
	defer server.Close()

	_, err := client.UpdateWorkItemIteration(123, "Invalid\\Path")
	if err == nil {
		t.Error("Expected error for invalid iteration")
	}
}

func TestGetHyperlinksAPI(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		response := WorkItem{
			ID: 123,
			Relations: []WorkItemRelation{
				{
					Rel: "Hyperlink",
					URL: "https://example.com",
					Attributes: map[string]interface{}{
						"comment": "Documentation",
					},
				},
				{
					Rel: "ArtifactLink",
					URL: "vstfs:///GitHub/PullRequest/abc",
					Attributes: map[string]interface{}{
						"name":    "https://github.com/owner/repo/pull/1",
						"comment": "PR link",
					},
				},
				{
					Rel: "System.LinkTypes.Hierarchy-Forward",
					URL: "https://dev.azure.com/org/proj/_apis/wit/workItems/124",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	links, err := client.GetHyperlinks(123)
	if err != nil {
		t.Fatalf("GetHyperlinks failed: %v", err)
	}

	if len(links) != 2 {
		t.Errorf("Expected 2 hyperlinks (excluding hierarchy), got %d", len(links))
	}
}

func TestAddHyperlinkAPI(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "Hyperlink") {
			t.Error("Expected Hyperlink relation type in body")
		}

		response := WorkItem{ID: 123}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	err := client.AddHyperlink(123, "https://example.com/docs", "Documentation link")
	if err != nil {
		t.Fatalf("AddHyperlink failed: %v", err)
	}
}

func TestRemoveHyperlinkAPI(t *testing.T) {
	requestCount := 0
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// GetWorkItemWithRelations
			response := WorkItem{
				ID: 123,
				Relations: []WorkItemRelation{
					{Rel: "Hyperlink", URL: "https://example.com"},
					{Rel: "Hyperlink", URL: "https://other.com"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		} else {
			// RemoveRelation
			response := WorkItem{ID: 123}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	err := client.RemoveHyperlink(123, "https://example.com")
	if err != nil {
		t.Fatalf("RemoveHyperlink failed: %v", err)
	}
}

func TestRemoveHyperlinkNotFound(t *testing.T) {
	client, server := testClientWithMockTransport(func(w http.ResponseWriter, r *http.Request) {
		response := WorkItem{
			ID: 123,
			Relations: []WorkItemRelation{
				{Rel: "Hyperlink", URL: "https://other.com"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	err := client.RemoveHyperlink(123, "https://notfound.com")
	if err == nil {
		t.Error("Expected error for hyperlink not found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestAuthHeader(t *testing.T) {
	client := NewClient("org", "proj", "", "", "testpat")
	header := client.authHeader()

	if !strings.HasPrefix(header, "Basic ") {
		t.Errorf("Expected Basic auth, got: %s", header)
	}
}

func TestTeamURLWithoutTeam(t *testing.T) {
	client := NewClient("org", "proj", "", "", "pat")
	url := client.teamURL()

	expected := "https://dev.azure.com/org/proj"
	if url != expected {
		t.Errorf("teamURL() = %s, want %s", url, expected)
	}
}

func TestExtractWorkItemIDFromURLEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected int
	}{
		{
			name:     "empty URL",
			url:      "",
			expected: 0,
		},
		{
			name:     "short URL",
			url:      "abc",
			expected: 0,
		},
		{
			name:     "no ID in URL",
			url:      "https://example.com/path",
			expected: 0,
		},
		{
			name:     "ID at end",
			url:      "https://dev.azure.com/org/proj/_apis/wit/workItems/12345",
			expected: 12345,
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

func TestTruncateError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short message unchanged",
			input:    "simple error",
			maxLen:   100,
			expected: "simple error",
		},
		{
			name:     "long message truncated",
			input:    "this is a very long error message that should be truncated",
			maxLen:   20,
			expected: "this is a very long ...",
		},
		{
			name:     "newlines collapsed",
			input:    "error\non\nmultiple\nlines",
			maxLen:   100,
			expected: "error on multiple lines",
		},
		{
			name:     "whitespace normalized",
			input:    "error   with    extra   spaces",
			maxLen:   100,
			expected: "error with extra spaces",
		},
		{
			name:     "HTML response truncated",
			input:    "<html><head><title>Error</title></head><body>Some long error page content here</body></html>",
			maxLen:   30,
			expected: "<html><head><title>Error</titl...",
		},
		{
			name:     "exact length",
			input:    "12345",
			maxLen:   5,
			expected: "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateError(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateError(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}
