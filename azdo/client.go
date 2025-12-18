package azdo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	Organization string
	Project      string
	Team         string
	AreaPath     string
	PAT          string
	httpClient   *http.Client
}

type WorkItem struct {
	ID        int                `json:"id"`
	Rev       int                `json:"rev"`
	Fields    WorkItemFields     `json:"fields"`
	URL       string             `json:"url"`
	Relations []WorkItemRelation `json:"relations,omitempty"`
}

type WorkItemRelation struct {
	Rel        string                 `json:"rel"`
	URL        string                 `json:"url"`
	Attributes map[string]interface{} `json:"attributes"`
}

type WorkItemFields struct {
	Title         string       `json:"System.Title"`
	State         string       `json:"System.State"`
	WorkItemType  string       `json:"System.WorkItemType"`
	AssignedTo    *IdentityRef `json:"System.AssignedTo"`
	Description   string       `json:"System.Description"`
	AreaPath      string       `json:"System.AreaPath"`
	IterationPath string       `json:"System.IterationPath"`
	Priority      int          `json:"Microsoft.VSTS.Common.Priority"`
	Tags          string       `json:"System.Tags"`
	CommentCount  int          `json:"System.CommentCount"`
	ChangedDate   string       `json:"System.ChangedDate"`
	// Planning fields
	StoryPoints      *float64 `json:"Microsoft.VSTS.Scheduling.StoryPoints,omitempty"`
	OriginalEstimate *float64 `json:"Microsoft.VSTS.Scheduling.OriginalEstimate,omitempty"`
	RemainingWork    *float64 `json:"Microsoft.VSTS.Scheduling.RemainingWork,omitempty"`
	CompletedWork    *float64 `json:"Microsoft.VSTS.Scheduling.CompletedWork,omitempty"`
	Effort           *float64 `json:"Microsoft.VSTS.Scheduling.Effort,omitempty"`
}

type IdentityRef struct {
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
}

type Comment struct {
	ID          int         `json:"id"`
	Text        string      `json:"text"`
	CreatedBy   IdentityRef `json:"createdBy"`
	CreatedDate string      `json:"createdDate"`
}

type CommentsResponse struct {
	Count    int       `json:"count"`
	Comments []Comment `json:"comments"`
}

type WorkItemQueryResult struct {
	WorkItems []WorkItemRef `json:"workItems"`
}

type WorkItemRef struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type WorkItemListResponse struct {
	Count int        `json:"count"`
	Value []WorkItem `json:"value"`
}

type CreateWorkItemOp struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

type WorkItemType struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Icon        struct {
		URL string `json:"url"`
	} `json:"icon"`
}

type WorkItemTypesResponse struct {
	Count int            `json:"count"`
	Value []WorkItemType `json:"value"`
}

type Iteration struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	Path       string               `json:"path"`
	Attributes *IterationAttributes `json:"attributes,omitempty"`
}

type IterationAttributes struct {
	StartDate  string `json:"startDate,omitempty"`
	FinishDate string `json:"finishDate,omitempty"`
	TimeFrame  string `json:"timeFrame,omitempty"`
}

type IterationsResponse struct {
	Count int         `json:"count"`
	Value []Iteration `json:"value"`
}

// WorkItemTypeField represents a field definition for a work item type
type WorkItemTypeField struct {
	ReferenceName  string      `json:"referenceName"`
	Name           string      `json:"name"`
	AlwaysRequired bool        `json:"alwaysRequired"`
	DefaultValue   interface{} `json:"defaultValue"`
	ReadOnly       bool        `json:"readOnly"`
}

type WorkItemTypeFieldsResponse struct {
	Count int                 `json:"count"`
	Value []WorkItemTypeField `json:"value"`
}

// PlanningField represents a planning field that can be displayed/edited
type PlanningField struct {
	ReferenceName string   // Azure DevOps field reference name
	DisplayName   string   // User-friendly display name
	Value         *float64 // Current value
}

func NewClient(org, project, team, areaPath, pat string) *Client {
	return &Client{
		Organization: org,
		Project:      project,
		Team:         team,
		AreaPath:     areaPath,
		PAT:          pat,
		httpClient:   &http.Client{},
	}
}

func (c *Client) authHeader() string {
	auth := base64.StdEncoding.EncodeToString([]byte(":" + c.PAT))
	return "Basic " + auth
}

func (c *Client) baseURL() string {
	return fmt.Sprintf("https://dev.azure.com/%s/%s", c.Organization, c.Project)
}

func (c *Client) teamURL() string {
	if c.Team != "" {
		return fmt.Sprintf("https://dev.azure.com/%s/%s/%s", c.Organization, c.Project, c.Team)
	}
	return c.baseURL()
}

func (c *Client) GetWorkItems(workItemType string, top int) ([]WorkItem, error) {
	return c.GetWorkItemsFiltered(workItemType, "", top)
}

func (c *Client) GetWorkItemsFiltered(workItemType, assignedTo string, top int) ([]WorkItem, error) {
	return c.GetWorkItemsPaged(workItemType, assignedTo, top, 0)
}

// GetWorkItemsPaged fetches work items with pagination support
func (c *Client) GetWorkItemsPaged(workItemType, assignedTo string, top int, skip int) ([]WorkItem, error) {
	query := fmt.Sprintf("SELECT [System.Id] FROM WorkItems WHERE [System.TeamProject] = '%s'", c.Project)
	if workItemType != "" {
		query += fmt.Sprintf(" AND [System.WorkItemType] = '%s'", workItemType)
	}
	if assignedTo != "" {
		query += fmt.Sprintf(" AND [System.AssignedTo] = '%s'", assignedTo)
	}
	if c.AreaPath != "" {
		query += fmt.Sprintf(" AND [System.AreaPath] UNDER '%s'", c.AreaPath)
	}
	query += " ORDER BY [System.ChangedDate] DESC"

	// Use team URL for WIQL queries when team is specified - the team context
	// automatically scopes queries to the team's configured area paths
	// Note: WIQL doesn't support $skip directly, so we fetch top+skip and slice
	wiqlURL := fmt.Sprintf("%s/_apis/wit/wiql?api-version=7.0&$top=%d", c.teamURL(), top+skip)

	body := map[string]string{"query": query}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", wiqlURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var queryResult WorkItemQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&queryResult); err != nil {
		return nil, err
	}

	if len(queryResult.WorkItems) == 0 {
		return []WorkItem{}, nil
	}

	// Skip items for pagination
	workItemRefs := queryResult.WorkItems
	if skip > 0 && skip < len(workItemRefs) {
		workItemRefs = workItemRefs[skip:]
	} else if skip >= len(workItemRefs) {
		return []WorkItem{}, nil
	}

	// Limit to top items
	if len(workItemRefs) > top {
		workItemRefs = workItemRefs[:top]
	}

	ids := make([]string, len(workItemRefs))
	for i, wi := range workItemRefs {
		ids[i] = fmt.Sprintf("%d", wi.ID)
	}

	return c.getWorkItemsByIDs(ids)
}

func (c *Client) getWorkItemsByIDs(ids []string) ([]WorkItem, error) {
	if len(ids) == 0 {
		return []WorkItem{}, nil
	}

	idsParam := ""
	for i, id := range ids {
		if i > 0 {
			idsParam += ","
		}
		idsParam += id
	}

	getURL := fmt.Sprintf("%s/_apis/wit/workitems?ids=%s&$expand=relations&api-version=7.0", c.baseURL(), url.QueryEscape(idsParam))

	req, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result WorkItemListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Value, nil
}

func (c *Client) CreateWorkItem(workItemType, title, description string, priority int) (*WorkItem, error) {
	return c.CreateWorkItemWithAssignee(workItemType, title, description, priority, "")
}

func (c *Client) CreateWorkItemWithAssignee(workItemType, title, description string, priority int, assignedTo string) (*WorkItem, error) {
	createURL := fmt.Sprintf("%s/_apis/wit/workitems/$%s?api-version=7.0", c.baseURL(), url.PathEscape(workItemType))

	ops := []CreateWorkItemOp{
		{Op: "add", Path: "/fields/System.Title", Value: title},
	}
	if description != "" {
		ops = append(ops, CreateWorkItemOp{Op: "add", Path: "/fields/System.Description", Value: description})
	}
	if priority > 0 {
		ops = append(ops, CreateWorkItemOp{Op: "add", Path: "/fields/Microsoft.VSTS.Common.Priority", Value: priority})
	}
	if c.AreaPath != "" {
		ops = append(ops, CreateWorkItemOp{Op: "add", Path: "/fields/System.AreaPath", Value: c.AreaPath})
	}
	if assignedTo != "" {
		ops = append(ops, CreateWorkItemOp{Op: "add", Path: "/fields/System.AssignedTo", Value: assignedTo})
	}

	jsonBody, _ := json.Marshal(ops)

	req, err := http.NewRequest("POST", createURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json-patch+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var workItem WorkItem
	if err := json.NewDecoder(resp.Body).Decode(&workItem); err != nil {
		return nil, err
	}

	return &workItem, nil
}

// CreateWorkItemWithParent creates a work item with a parent link
func (c *Client) CreateWorkItemWithParent(workItemType, title, description string, priority int, parentID int) (*WorkItem, error) {
	return c.CreateWorkItemWithParentAndAssignee(workItemType, title, description, priority, parentID, "")
}

// CreateWorkItemWithParentAndAssignee creates a work item with a parent link and assignee
func (c *Client) CreateWorkItemWithParentAndAssignee(workItemType, title, description string, priority int, parentID int, assignedTo string) (*WorkItem, error) {
	createURL := fmt.Sprintf("%s/_apis/wit/workitems/$%s?api-version=7.0", c.baseURL(), url.PathEscape(workItemType))

	ops := []CreateWorkItemOp{
		{Op: "add", Path: "/fields/System.Title", Value: title},
	}
	if description != "" {
		ops = append(ops, CreateWorkItemOp{Op: "add", Path: "/fields/System.Description", Value: description})
	}
	if priority > 0 {
		ops = append(ops, CreateWorkItemOp{Op: "add", Path: "/fields/Microsoft.VSTS.Common.Priority", Value: priority})
	}
	if c.AreaPath != "" {
		ops = append(ops, CreateWorkItemOp{Op: "add", Path: "/fields/System.AreaPath", Value: c.AreaPath})
	}
	if assignedTo != "" {
		ops = append(ops, CreateWorkItemOp{Op: "add", Path: "/fields/System.AssignedTo", Value: assignedTo})
	}

	// Add parent link
	parentURL := fmt.Sprintf("%s/_apis/wit/workItems/%d", c.baseURL(), parentID)
	ops = append(ops, CreateWorkItemOp{
		Op:   "add",
		Path: "/relations/-",
		Value: map[string]interface{}{
			"rel": "System.LinkTypes.Hierarchy-Reverse",
			"url": parentURL,
		},
	})

	jsonBody, _ := json.Marshal(ops)

	req, err := http.NewRequest("POST", createURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json-patch+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var workItem WorkItem
	if err := json.NewDecoder(resp.Body).Decode(&workItem); err != nil {
		return nil, err
	}

	return &workItem, nil
}

// AddChildLink adds a child link from parentID to childID
func (c *Client) AddChildLink(parentID, childID int) error {
	updateURL := fmt.Sprintf("%s/_apis/wit/workitems/%d?api-version=7.0", c.baseURL(), parentID)

	childURL := fmt.Sprintf("%s/_apis/wit/workItems/%d", c.baseURL(), childID)
	ops := []CreateWorkItemOp{
		{
			Op:   "add",
			Path: "/relations/-",
			Value: map[string]interface{}{
				"rel": "System.LinkTypes.Hierarchy-Forward",
				"url": childURL,
			},
		},
	}

	jsonBody, _ := json.Marshal(ops)

	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json-patch+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// RemoveRelation removes a relation from a work item by relation index
func (c *Client) RemoveRelation(workItemID int, relationIndex int) error {
	updateURL := fmt.Sprintf("%s/_apis/wit/workitems/%d?api-version=7.0", c.baseURL(), workItemID)

	ops := []CreateWorkItemOp{
		{
			Op:   "remove",
			Path: fmt.Sprintf("/relations/%d", relationIndex),
		},
	}

	jsonBody, _ := json.Marshal(ops)

	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json-patch+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// RemoveHierarchyLink removes the parent-child link between two work items
// If isParent is true, removes the parent link from the current item
// If isParent is false, removes the child link (current item is parent of targetID)
func (c *Client) RemoveHierarchyLink(workItemID int, targetID int, isParent bool) error {
	// Get the work item with relations to find the index
	wi, err := c.GetWorkItemWithRelations(workItemID)
	if err != nil {
		return err
	}

	targetURL := fmt.Sprintf("workItems/%d", targetID)
	relType := "System.LinkTypes.Hierarchy-Reverse" // parent link
	if !isParent {
		relType = "System.LinkTypes.Hierarchy-Forward" // child link
	}

	// Find the relation index
	for i, rel := range wi.Relations {
		if rel.Rel == relType && strings.Contains(rel.URL, targetURL) {
			return c.RemoveRelation(workItemID, i)
		}
	}

	return fmt.Errorf("relation not found")
}

func (c *Client) GetWorkItemTypes() ([]string, error) {
	typesURL := fmt.Sprintf("%s/_apis/wit/workitemtypes?api-version=7.0", c.baseURL())

	req, err := http.NewRequest("GET", typesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result WorkItemTypesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Filter to common work item types (exclude hidden/system types)
	var types []string
	for _, wit := range result.Value {
		// Skip hidden types that start with certain prefixes
		if wit.Name != "" {
			types = append(types, wit.Name)
		}
	}

	return types, nil
}

func (c *Client) GetComments(workItemID int) ([]Comment, error) {
	commentsURL := fmt.Sprintf("%s/_apis/wit/workitems/%d/comments?api-version=7.0-preview.3", c.baseURL(), workItemID)

	req, err := http.NewRequest("GET", commentsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result CommentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Comments, nil
}

func (c *Client) AddComment(workItemID int, text string) error {
	commentURL := fmt.Sprintf("%s/_apis/wit/workitems/%d/comments?api-version=7.0-preview.3", c.baseURL(), workItemID)

	body := map[string]string{"text": text}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", commentURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *Client) UpdateWorkItem(workItemID int, title, state, assignedTo, tags string) (*WorkItem, error) {
	updateURL := fmt.Sprintf("%s/_apis/wit/workitems/%d?api-version=7.0", c.baseURL(), workItemID)

	var ops []CreateWorkItemOp
	if title != "" {
		ops = append(ops, CreateWorkItemOp{Op: "replace", Path: "/fields/System.Title", Value: title})
	}
	if state != "" {
		ops = append(ops, CreateWorkItemOp{Op: "replace", Path: "/fields/System.State", Value: state})
	}
	// AssignedTo can be empty string to unassign
	ops = append(ops, CreateWorkItemOp{Op: "replace", Path: "/fields/System.AssignedTo", Value: assignedTo})
	// Tags can be empty string to clear tags
	ops = append(ops, CreateWorkItemOp{Op: "replace", Path: "/fields/System.Tags", Value: tags})

	if len(ops) == 0 {
		return nil, fmt.Errorf("no updates specified")
	}

	jsonBody, _ := json.Marshal(ops)

	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json-patch+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var workItem WorkItem
	if err := json.NewDecoder(resp.Body).Decode(&workItem); err != nil {
		return nil, err
	}

	return &workItem, nil
}

// GetWorkItemWithRelations fetches a single work item with its relations expanded
func (c *Client) GetWorkItemWithRelations(workItemID int) (*WorkItem, error) {
	getURL := fmt.Sprintf("%s/_apis/wit/workitems/%d?$expand=relations&api-version=7.0", c.baseURL(), workItemID)

	req, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var workItem WorkItem
	if err := json.NewDecoder(resp.Body).Decode(&workItem); err != nil {
		return nil, err
	}

	return &workItem, nil
}

// GetRelatedWorkItems fetches parent and child work items for a given work item
func (c *Client) GetRelatedWorkItems(workItemID int) (parent *WorkItem, children []WorkItem, err error) {
	// First get the work item with relations
	wi, err := c.GetWorkItemWithRelations(workItemID)
	if err != nil {
		return nil, nil, err
	}

	var parentID int
	var childIDs []string

	// Parse relations to find parent and children
	// "System.LinkTypes.Hierarchy-Reverse" = parent (this item is a child of the target)
	// "System.LinkTypes.Hierarchy-Forward" = child (this item is a parent of the target)
	for _, rel := range wi.Relations {
		switch rel.Rel {
		case "System.LinkTypes.Hierarchy-Reverse":
			// Extract ID from URL: .../workitems/123
			parentID = extractWorkItemIDFromURL(rel.URL)
		case "System.LinkTypes.Hierarchy-Forward":
			childID := extractWorkItemIDFromURL(rel.URL)
			if childID > 0 {
				childIDs = append(childIDs, fmt.Sprintf("%d", childID))
			}
		}
	}

	// Fetch parent if exists
	if parentID > 0 {
		parent, err = c.GetWorkItemWithRelations(parentID)
		if err != nil {
			// Don't fail if we can't get parent, just log it
			parent = nil
		}
	}

	// Fetch children if exist
	if len(childIDs) > 0 {
		children, err = c.getWorkItemsByIDs(childIDs)
		if err != nil {
			// Don't fail if we can't get children
			children = nil
		}
	}

	return parent, children, nil
}

// extractWorkItemIDFromURL extracts the work item ID from a URL like
// https://dev.azure.com/org/project/_apis/wit/workItems/123
func extractWorkItemIDFromURL(urlStr string) int {
	// Parse the URL and get the last path segment
	var id int
	fmt.Sscanf(urlStr[len(urlStr)-10:], "%d", &id)
	if id > 0 {
		return id
	}
	// Try parsing from the end more carefully
	for i := len(urlStr) - 1; i >= 0; i-- {
		if urlStr[i] == '/' {
			fmt.Sscanf(urlStr[i+1:], "%d", &id)
			return id
		}
	}
	return 0
}

// DeleteWorkItem deletes a work item by ID
func (c *Client) DeleteWorkItem(workItemID int) error {
	deleteURL := fmt.Sprintf("%s/_apis/wit/workitems/%d?api-version=7.0", c.baseURL(), workItemID)

	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *Client) TestConnection() error {
	testURL := fmt.Sprintf("https://dev.azure.com/%s/_apis/projects/%s?api-version=7.0", c.Organization, c.Project)

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("connection failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetIterations fetches available iterations for the team
func (c *Client) GetIterations() ([]Iteration, error) {
	// Use team URL to get team iterations
	iterationsURL := fmt.Sprintf("%s/_apis/work/teamsettings/iterations?api-version=7.0", c.teamURL())

	req, err := http.NewRequest("GET", iterationsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result IterationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Value, nil
}

// UpdateWorkItemPlanning updates the planning fields of a work item
// Pass nil for any field you don't want to update
func (c *Client) UpdateWorkItemPlanning(workItemID int, storyPoints, originalEstimate, remainingWork, completedWork *float64) (*WorkItem, error) {
	updateURL := fmt.Sprintf("%s/_apis/wit/workitems/%d?api-version=7.0", c.baseURL(), workItemID)

	var ops []CreateWorkItemOp

	// Use "add" operation for fields that might not exist, "replace" might fail
	if storyPoints != nil {
		ops = append(ops, CreateWorkItemOp{Op: "add", Path: "/fields/Microsoft.VSTS.Scheduling.StoryPoints", Value: *storyPoints})
	}
	if originalEstimate != nil {
		ops = append(ops, CreateWorkItemOp{Op: "add", Path: "/fields/Microsoft.VSTS.Scheduling.OriginalEstimate", Value: *originalEstimate})
	}
	if remainingWork != nil {
		ops = append(ops, CreateWorkItemOp{Op: "add", Path: "/fields/Microsoft.VSTS.Scheduling.RemainingWork", Value: *remainingWork})
	}
	if completedWork != nil {
		ops = append(ops, CreateWorkItemOp{Op: "add", Path: "/fields/Microsoft.VSTS.Scheduling.CompletedWork", Value: *completedWork})
	}

	if len(ops) == 0 {
		return nil, fmt.Errorf("no planning updates specified")
	}

	jsonBody, _ := json.Marshal(ops)

	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json-patch+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var workItem WorkItem
	if err := json.NewDecoder(resp.Body).Decode(&workItem); err != nil {
		return nil, err
	}

	return &workItem, nil
}

// GetWorkItemTypeFields fetches the available fields for a work item type
func (c *Client) GetWorkItemTypeFields(workItemType string) ([]WorkItemTypeField, error) {
	fieldsURL := fmt.Sprintf("%s/_apis/wit/workitemtypes/%s/fields?api-version=7.0", c.baseURL(), url.PathEscape(workItemType))

	req, err := http.NewRequest("GET", fieldsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result WorkItemTypeFieldsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Value, nil
}

// GetPlanningFields returns the available planning fields for a work item type
// This filters to only scheduling/planning related fields
func (c *Client) GetPlanningFields(workItemType string) ([]PlanningField, error) {
	fields, err := c.GetWorkItemTypeFields(workItemType)
	if err != nil {
		return nil, err
	}

	// Planning field reference names and their display names
	planningFieldMap := map[string]string{
		"Microsoft.VSTS.Scheduling.StoryPoints":      "Story Points",
		"Microsoft.VSTS.Scheduling.OriginalEstimate": "Original Estimate (hours)",
		"Microsoft.VSTS.Scheduling.RemainingWork":    "Remaining Work (hours)",
		"Microsoft.VSTS.Scheduling.CompletedWork":    "Completed Work (hours)",
		"Microsoft.VSTS.Scheduling.Effort":           "Effort",
	}

	var planningFields []PlanningField
	for _, field := range fields {
		if displayName, ok := planningFieldMap[field.ReferenceName]; ok {
			// Only include if not read-only
			if !field.ReadOnly {
				planningFields = append(planningFields, PlanningField{
					ReferenceName: field.ReferenceName,
					DisplayName:   displayName,
				})
			}
		}
	}

	return planningFields, nil
}

// UpdateWorkItemPlanningDynamic updates planning fields dynamically based on the provided map
func (c *Client) UpdateWorkItemPlanningDynamic(workItemID int, fields map[string]float64) (*WorkItem, error) {
	updateURL := fmt.Sprintf("%s/_apis/wit/workitems/%d?api-version=7.0", c.baseURL(), workItemID)

	var ops []CreateWorkItemOp

	for referenceName, value := range fields {
		ops = append(ops, CreateWorkItemOp{
			Op:    "add",
			Path:  "/fields/" + referenceName,
			Value: value,
		})
	}

	if len(ops) == 0 {
		return nil, fmt.Errorf("no planning updates specified")
	}

	jsonBody, _ := json.Marshal(ops)

	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json-patch+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var workItem WorkItem
	if err := json.NewDecoder(resp.Body).Decode(&workItem); err != nil {
		return nil, err
	}

	return &workItem, nil
}

// GetRecentlyChangedWorkItems fetches work items assigned to a user that changed within the last N minutes
// Excludes changes made by the user themselves (only notifies on changes by others)
func (c *Client) GetRecentlyChangedWorkItems(assignedTo string, withinMinutes int) ([]WorkItem, error) {
	if assignedTo == "" {
		return []WorkItem{}, nil
	}

	// Build WIQL query for recently changed items assigned to user
	// Exclude items where the user themselves made the change
	query := fmt.Sprintf("SELECT [System.Id] FROM WorkItems WHERE [System.TeamProject] = '%s'", c.Project)
	query += fmt.Sprintf(" AND [System.AssignedTo] = '%s'", assignedTo)
	query += fmt.Sprintf(" AND [System.ChangedBy] <> '%s'", assignedTo)
	query += fmt.Sprintf(" AND [System.ChangedDate] >= @Today - %d", withinMinutes)
	if c.AreaPath != "" {
		query += fmt.Sprintf(" AND [System.AreaPath] UNDER '%s'", c.AreaPath)
	}
	query += " ORDER BY [System.ChangedDate] DESC"

	wiqlURL := fmt.Sprintf("%s/_apis/wit/wiql?api-version=7.0", c.teamURL())

	body := map[string]string{"query": query}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", wiqlURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var queryResult WorkItemQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&queryResult); err != nil {
		return nil, err
	}

	if len(queryResult.WorkItems) == 0 {
		return []WorkItem{}, nil
	}

	ids := make([]string, len(queryResult.WorkItems))
	for i, wi := range queryResult.WorkItems {
		ids[i] = fmt.Sprintf("%d", wi.ID)
	}

	return c.getWorkItemsByIDs(ids)
}

// UpdateWorkItemIteration updates the iteration path of a work item
func (c *Client) UpdateWorkItemIteration(workItemID int, iterationPath string) (*WorkItem, error) {
	updateURL := fmt.Sprintf("%s/_apis/wit/workitems/%d?api-version=7.0", c.baseURL(), workItemID)

	ops := []CreateWorkItemOp{
		{Op: "replace", Path: "/fields/System.IterationPath", Value: iterationPath},
	}

	jsonBody, _ := json.Marshal(ops)

	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json-patch+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var workItem WorkItem
	if err := json.NewDecoder(resp.Body).Decode(&workItem); err != nil {
		return nil, err
	}

	return &workItem, nil
}
