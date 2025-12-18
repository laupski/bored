package azdo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		json.NewEncoder(w).Encode(response)
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
		json.NewEncoder(w).Encode(response)
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
