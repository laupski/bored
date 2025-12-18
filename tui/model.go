package tui

import (
	"bored/azdo"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type View int

const (
	ViewConfig View = iota
	ViewBoard
	ViewCreate
	ViewDetail
	ViewConfigFile
)

type Model struct {
	view            View
	client          *azdo.Client
	workItems       []azdo.WorkItem
	cursor          int
	configInputs    []textinput.Model
	configFocus     int
	createInputs    []textinput.Model
	createFocus     int
	createType      int
	workItemTypes   []string
	err             error
	message         string
	width           int
	height          int
	loading         bool
	keychainLoaded  bool
	keychainMessage string
	username        string
	showAll         bool
	// App config (from config file)
	appConfig        AppConfig
	appConfigMessage string
	configFileFocus  int // 0=DefaultShowAll, 1=MaxWorkItems
	configFileInputs []textinput.Model
	// Detail view fields
	selectedItem     *azdo.WorkItem
	detailInputs     []textinput.Model
	detailFocus      int
	comments         []azdo.Comment
	commentsExpanded bool
	commentScroll    int
	// Related work items
	parentItem      *azdo.WorkItem
	childItems      []azdo.WorkItem
	relatedExpanded bool
	relatedCursor   int // 0 = parent, 1+ = children
	// Create related item state
	creatingRelated       bool   // true when in create related item mode
	createRelatedAsChild  bool   // true = create child, false = create parent
	createRelatedTitle    string // title for the new related item
	createRelatedType     int    // index into workItemTypes
	createRelatedAssignee string // assignee for the new related item
	createRelatedFocus    int    // 0 = title, 1 = assignee
	// Delete confirmation state
	confirmingDelete      bool // true when waiting for delete confirmation
	confirmDeleteTargetID int  // ID of the item to unlink
	confirmDeleteIsParent bool // true if removing parent link
	// Delete work item state (on board screen)
	deletingWorkItem    bool   // true when in delete confirmation mode
	deleteWorkItemID    int    // ID of work item to delete
	deleteWorkItemTitle string // Title of work item to delete (for confirmation)
	deleteConfirmInput  string // User's typed confirmation
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)
)

func NewModel() Model {
	configInputs := make([]textinput.Model, 6)

	configInputs[0] = textinput.New()
	configInputs[0].Placeholder = "myorg"
	configInputs[0].Focus()
	configInputs[0].Width = 40
	configInputs[0].Prompt = ""

	configInputs[1] = textinput.New()
	configInputs[1].Placeholder = "MyProject"
	configInputs[1].Width = 40
	configInputs[1].Prompt = ""

	configInputs[2] = textinput.New()
	configInputs[2].Placeholder = "MyTeam"
	configInputs[2].Width = 40
	configInputs[2].Prompt = ""

	configInputs[3] = textinput.New()
	configInputs[3].Placeholder = "Project\\Team"
	configInputs[3].Width = 40
	configInputs[3].Prompt = ""

	configInputs[4] = textinput.New()
	configInputs[4].Placeholder = "your-personal-access-token"
	configInputs[4].Width = 40
	configInputs[4].EchoMode = textinput.EchoPassword
	configInputs[4].Prompt = ""

	configInputs[5] = textinput.New()
	configInputs[5].Placeholder = "user@email.com"
	configInputs[5].Width = 40
	configInputs[5].Prompt = ""

	createInputs := make([]textinput.Model, 4)

	createInputs[0] = textinput.New()
	createInputs[0].Placeholder = "Work item title"
	createInputs[0].Width = 50
	createInputs[0].Prompt = ""

	createInputs[1] = textinput.New()
	createInputs[1].Placeholder = "Description (optional)"
	createInputs[1].Width = 50
	createInputs[1].Prompt = ""

	createInputs[2] = textinput.New()
	createInputs[2].Placeholder = "1-4"
	createInputs[2].Width = 10
	createInputs[2].Prompt = ""

	createInputs[3] = textinput.New()
	createInputs[3].Placeholder = "user@email.com"
	createInputs[3].Width = 40
	createInputs[3].Prompt = ""

	// Detail view inputs: Title, State, Assigned To, Tags, Comment
	detailInputs := make([]textinput.Model, 5)

	detailInputs[0] = textinput.New()
	detailInputs[0].Placeholder = "Title"
	detailInputs[0].Width = 60
	detailInputs[0].Prompt = ""

	detailInputs[1] = textinput.New()
	detailInputs[1].Placeholder = "State"
	detailInputs[1].Width = 20
	detailInputs[1].Prompt = ""

	detailInputs[2] = textinput.New()
	detailInputs[2].Placeholder = "user@email.com"
	detailInputs[2].Width = 40
	detailInputs[2].Prompt = ""

	detailInputs[3] = textinput.New()
	detailInputs[3].Placeholder = "tag1; tag2; tag3"
	detailInputs[3].Width = 40
	detailInputs[3].Prompt = ""

	detailInputs[4] = textinput.New()
	detailInputs[4].Placeholder = "Add a comment..."
	detailInputs[4].Width = 60
	detailInputs[4].Prompt = ""

	// Config file inputs: MaxWorkItems (only text input needed for number)
	configFileInputs := make([]textinput.Model, 1)

	configFileInputs[0] = textinput.New()
	configFileInputs[0].Placeholder = "50"
	configFileInputs[0].Width = 10
	configFileInputs[0].Prompt = ""

	// Load app config from file
	appConfig, _ := LoadConfigFile()

	m := Model{
		view:             ViewConfig,
		configInputs:     configInputs,
		createInputs:     createInputs,
		detailInputs:     detailInputs,
		configFileInputs: configFileInputs,
		appConfig:        appConfig,
		showAll:          appConfig.DefaultShowAll,
		workItemTypes:    []string{"Bug", "Task", "User Story", "Feature", "Epic"},
	}

	// Set initial value for max work items input
	m.configFileInputs[0].SetValue(fmt.Sprintf("%d", appConfig.MaxWorkItems))

	// Try to load credentials from keychain
	if org, project, team, areaPath, pat, username, err := LoadCredentials(); err == nil {
		m.configInputs[0].SetValue(org)
		m.configInputs[1].SetValue(project)
		m.configInputs[2].SetValue(team)
		m.configInputs[3].SetValue(areaPath)
		m.configInputs[4].SetValue(pat)
		m.configInputs[5].SetValue(username)
		m.username = username
		m.keychainLoaded = true
		m.keychainMessage = "Credentials loaded from keychain"
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

type workItemsMsg struct {
	items []azdo.WorkItem
	err   error
}

type createResultMsg struct {
	item *azdo.WorkItem
	err  error
}

type connectMsg struct {
	err error
}

type workItemTypesMsg struct {
	types []string
	err   error
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.view == ViewCreate || m.view == ViewDetail {
				m.view = ViewBoard
				m.err = nil
				m.message = ""
				return m, nil
			}
		}

	case connectMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.view = ViewBoard
		// Fetch work items and work item types in parallel
		return m, tea.Batch(m.fetchWorkItems(), m.fetchWorkItemTypes())

	case workItemTypesMsg:
		if msg.err == nil && len(msg.types) > 0 {
			m.workItemTypes = msg.types
			m.createType = 0 // Reset selection
		}
		return m, nil

	case workItemsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.workItems = msg.items
		m.err = nil
		m.message = ""
		return m, nil

	case createResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = fmt.Sprintf("Created work item #%d", msg.item.ID)
		m.view = ViewBoard
		for i := range m.createInputs {
			m.createInputs[i].SetValue("")
		}
		return m, m.fetchWorkItems()

	case commentsMsg:
		m.loading = false
		if msg.err == nil {
			m.comments = msg.comments
		}
		return m, nil

	case addCommentMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = "Comment added"
		m.detailInputs[4].SetValue("")
		return m, m.fetchComments(m.selectedItem.ID)

	case updateWorkItemMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = "Work item updated"
		m.selectedItem = msg.item
		return m, nil

	case relatedItemsMsg:
		if msg.err == nil {
			m.parentItem = msg.parent
			m.childItems = msg.children
		}
		return m, nil

	case createRelatedMsg:
		m.loading = false
		m.creatingRelated = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		relType := "parent"
		if msg.asChild {
			relType = "child"
		}
		m.message = fmt.Sprintf("Created %s #%d", relType, msg.item.ID)
		// Refresh related items
		return m, m.fetchRelatedItems(m.selectedItem.ID)

	case removeLinkMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = "Link removed"
		m.relatedCursor = 0
		// Refresh related items
		return m, m.fetchRelatedItems(m.selectedItem.ID)

	case deleteWorkItemMsg:
		m.loading = false
		m.deletingWorkItem = false
		m.deleteConfirmInput = ""
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = fmt.Sprintf("Deleted work item #%d", m.deleteWorkItemID)
		m.cursor = 0
		return m, m.fetchWorkItems()
	}

	switch m.view {
	case ViewConfig:
		return m.updateConfig(msg)
	case ViewBoard:
		return m.updateBoard(msg)
	case ViewCreate:
		return m.updateCreate(msg)
	case ViewDetail:
		return m.updateDetail(msg)
	case ViewConfigFile:
		return m.updateConfigFile(msg)
	}

	return m, nil
}

func (m Model) View() string {
	switch m.view {
	case ViewConfig:
		return m.viewConfig()
	case ViewBoard:
		return m.viewBoard()
	case ViewCreate:
		return m.viewCreate()
	case ViewDetail:
		return m.viewDetail()
	case ViewConfigFile:
		return m.viewConfigFile()
	}
	return ""
}

func (m Model) fetchWorkItems() tea.Cmd {
	return func() tea.Msg {
		assignedTo := ""
		if !m.showAll && m.username != "" {
			assignedTo = m.username
		}
		items, err := m.client.GetWorkItemsFiltered("", assignedTo, 50)
		return workItemsMsg{items: items, err: err}
	}
}

func (m Model) connect() tea.Cmd {
	return func() tea.Msg {
		err := m.client.TestConnection()
		return connectMsg{err: err}
	}
}

func (m Model) fetchWorkItemTypes() tea.Cmd {
	return func() tea.Msg {
		types, err := m.client.GetWorkItemTypes()
		return workItemTypesMsg{types: types, err: err}
	}
}

func (m Model) createWorkItem() tea.Cmd {
	return func() tea.Msg {
		title := m.createInputs[0].Value()
		desc := m.createInputs[1].Value()
		priority := 2
		if p := m.createInputs[2].Value(); p != "" {
			if p[0] >= '1' && p[0] <= '4' {
				priority = int(p[0] - '0')
			}
		}
		assignedTo := m.createInputs[3].Value()
		wiType := m.workItemTypes[m.createType]

		item, err := m.client.CreateWorkItemWithAssignee(wiType, title, desc, priority, assignedTo)
		return createResultMsg{item: item, err: err}
	}
}

type commentsMsg struct {
	comments []azdo.Comment
	err      error
}

type addCommentMsg struct {
	err error
}

type updateWorkItemMsg struct {
	item *azdo.WorkItem
	err  error
}

type relatedItemsMsg struct {
	parent   *azdo.WorkItem
	children []azdo.WorkItem
	err      error
}

type createRelatedMsg struct {
	item    *azdo.WorkItem
	asChild bool
	err     error
}

type removeLinkMsg struct {
	err error
}

type deleteWorkItemMsg struct {
	err error
}

func (m Model) fetchComments(workItemID int) tea.Cmd {
	return func() tea.Msg {
		comments, err := m.client.GetComments(workItemID)
		return commentsMsg{comments: comments, err: err}
	}
}

func (m Model) addComment(workItemID int, text string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.AddComment(workItemID, text)
		return addCommentMsg{err: err}
	}
}

func (m Model) updateWorkItem(workItemID int, title, state, assignedTo, tags string) tea.Cmd {
	return func() tea.Msg {
		item, err := m.client.UpdateWorkItem(workItemID, title, state, assignedTo, tags)
		return updateWorkItemMsg{item: item, err: err}
	}
}

func (m Model) fetchRelatedItems(workItemID int) tea.Cmd {
	return func() tea.Msg {
		parent, children, err := m.client.GetRelatedWorkItems(workItemID)
		return relatedItemsMsg{parent: parent, children: children, err: err}
	}
}

func (m Model) createRelatedItem(parentID int, asChild bool, title, wiType, assignee string) tea.Cmd {
	return func() tea.Msg {
		var item *azdo.WorkItem
		var err error

		if asChild {
			// Create a new work item as a child of the current item
			item, err = m.client.CreateWorkItemWithParentAndAssignee(wiType, title, "", 2, parentID, assignee)
		} else {
			// Create a new work item and make the current item its child
			item, err = m.client.CreateWorkItemWithAssignee(wiType, title, "", 2, assignee)
			if err == nil && item != nil {
				// Link the current item as a child of the new item
				err = m.client.AddChildLink(item.ID, parentID)
			}
		}

		return createRelatedMsg{item: item, asChild: asChild, err: err}
	}
}

func (m Model) removeLink(workItemID, targetID int, isParent bool) tea.Cmd {
	return func() tea.Msg {
		err := m.client.RemoveHierarchyLink(workItemID, targetID, isParent)
		return removeLinkMsg{err: err}
	}
}

func (m Model) deleteWorkItem(workItemID int) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DeleteWorkItem(workItemID)
		return deleteWorkItemMsg{err: err}
	}
}
