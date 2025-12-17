package azdo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	Organization string
	Project      string
	PAT          string
	httpClient   *http.Client
}

type WorkItem struct {
	ID     int            `json:"id"`
	Rev    int            `json:"rev"`
	Fields WorkItemFields `json:"fields"`
	URL    string         `json:"url"`
}

type WorkItemFields struct {
	Title        string       `json:"System.Title"`
	State        string       `json:"System.State"`
	WorkItemType string       `json:"System.WorkItemType"`
	AssignedTo   *IdentityRef `json:"System.AssignedTo"`
	Description  string       `json:"System.Description"`
	AreaPath     string       `json:"System.AreaPath"`
	Priority     int          `json:"Microsoft.VSTS.Common.Priority"`
	Tags         string       `json:"System.Tags"`
	CommentCount int          `json:"System.CommentCount"`
	ChangedDate  string       `json:"System.ChangedDate"`
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

func NewClient(org, project, pat string) *Client {
	return &Client{
		Organization: org,
		Project:      project,
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

func (c *Client) GetWorkItems(workItemType string, top int) ([]WorkItem, error) {
	return c.GetWorkItemsFiltered(workItemType, "", top)
}

func (c *Client) GetWorkItemsFiltered(workItemType, assignedTo string, top int) ([]WorkItem, error) {
	query := fmt.Sprintf("SELECT [System.Id] FROM WorkItems WHERE [System.TeamProject] = '%s'", c.Project)
	if workItemType != "" {
		query += fmt.Sprintf(" AND [System.WorkItemType] = '%s'", workItemType)
	}
	if assignedTo != "" {
		query += fmt.Sprintf(" AND [System.AssignedTo] = '%s'", assignedTo)
	}
	query += " ORDER BY [System.ChangedDate] DESC"

	wiqlURL := fmt.Sprintf("%s/_apis/wit/wiql?api-version=7.0&$top=%d", c.baseURL(), top)

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

	getURL := fmt.Sprintf("%s/_apis/wit/workitems?ids=%s&api-version=7.0", c.baseURL(), url.QueryEscape(idsParam))

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
