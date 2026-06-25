package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"thinx/internal/domain"
	"thinx/internal/tui/modal"
)

type tab struct {
	label string
	color string
	list  domain.TodoList
}

type todosLoadedMsg struct {
	list  domain.TodoList
	todos []domain.Todo
	err   error
}

// mutationDoneMsg carries the reloaded list after a status or delete write that
// has also been shown for its minimum display time.
type mutationDoneMsg struct {
	id    string
	list  domain.TodoList
	todos []domain.Todo
	err   error
}

type Model struct {
	repo          domain.TodoRepository
	keys          keyMap
	tabs          []tab
	list          list.Model
	spinner       spinner.Model
	modal         *modal.Model
	modalOriginal domain.Todo
	active        int
	cloudOps      int
	err           error
	width         int
	height        int
	// pending holds todos with an in-flight status/delete write, mapped to
	// whether that write is a deletion (rendered struck through until reload).
	pending map[string]bool
}

// NewModel creates the initial TUI state.
func NewModel(repo domain.TodoRepository) Model {
	pending := map[string]bool{}
	todos := list.New(nil, todoDelegate{list: domain.ListToday, pending: pending}, 0, 0)
	todos.SetShowTitle(false)
	todos.SetShowStatusBar(false)
	todos.SetShowPagination(false)
	todos.SetShowHelp(false)
	todos.SetFilteringEnabled(false)
	todos.SetStatusBarItemName("todo", "todos")
	todos.Styles.NoItems = statusStyle
	todos.Styles.Spinner = statusStyle

	s := spinner.New()
	s.Spinner = spinner.Monkey

	return Model{
		repo:    repo,
		keys:    newKeyMap(),
		list:    todos,
		spinner: s,
		tabs: []tab{
			{label: "Inbox", color: "#5f87d7", list: domain.ListInbox},
			{label: "Today", color: "#d8a100", list: domain.ListToday},
			{label: "Scheduled", color: "#d75f5f", list: domain.ListScheduled},
			{label: "Anytime", color: "#008b8b", list: domain.ListAnytime},
			{label: "Someday", color: "#af5fd7", list: domain.ListSomeday},
			{label: "Archive", color: "#666666", list: domain.ListLogbook},
		},
		active:   1,
		cloudOps: 1, // initial load kicked off by Init
		pending:  pending,
	}
}

// Init returns the initial Bubble Tea command.
func (m Model) Init() tea.Cmd {
	return tea.Batch(loadTodos(m.repo, m.tabs[m.active].list, true), m.spinner.Tick)
}

// Update applies terminal events to the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg.Width, msg.Height), nil
	case todosLoadedMsg:
		m.cloudOps--
		if msg.list != m.tabs[m.active].list {
			return m, nil
		}
		m.err = msg.err
		m.list.SetDelegate(todoDelegate{list: msg.list, pending: m.pending})
		cmd := m.list.SetItems(todoItems(msg.todos))
		m.list.Select(0)
		return m, cmd
	case spinner.TickMsg:
		if m.cloudOps == 0 {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case mutationDoneMsg:
		delete(m.pending, msg.id)
		m.cloudOps--
		if msg.err != nil {
			m.err = msg.err
		}
		if msg.list != m.tabs[m.active].list {
			return m, nil
		}
		// Reload reconciles membership (the row leaves the list on success) and
		// rolls back optimistic edits on failure. Keep the cursor in place.
		index := m.list.Index()
		cmd := m.list.SetItems(todoItems(msg.todos))
		if len(msg.todos) > 0 {
			m.list.Select(min(index, len(msg.todos)-1))
		}
		return m, cmd
	case tea.KeyPressMsg:
		if m.modal != nil {
			return m.updateModal(msg)
		}

		switch {
		case key.Matches(msg, m.keys.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.openTask):
			return m.openModal(), nil
		case key.Matches(msg, m.keys.completeTodo):
			status := domain.TodoStatusCompleted
			if m.tabs[m.active].list == domain.ListLogbook {
				status = domain.TodoStatusOpen
			}
			return m.setTodoStatus(status)
		case key.Matches(msg, m.keys.cancelTodo):
			return m.setTodoStatus(domain.TodoStatusCanceled)
		case key.Matches(msg, m.keys.deleteTodo):
			return m.deleteTodo()
		case key.Matches(msg, m.keys.refresh):
			return m.startRefresh()
		case key.Matches(msg, m.keys.previousList):
			return m.switchTab(-1)
		case key.Matches(msg, m.keys.nextList):
			return m.switchTab(1)
		case key.Matches(msg, m.keys.previousTodo):
			m.list.CursorUp()
			return m, nil
		case key.Matches(msg, m.keys.nextTodo):
			m.list.CursorDown()
			return m, nil
		}
	}

	return m, nil
}

// setTodoStatus optimistically restyles the selected todo, then persists the
// change and reloads the list once it has been shown long enough.
func (m Model) setTodoStatus(status domain.TodoStatus) (tea.Model, tea.Cmd) {
	index := m.list.GlobalIndex()
	todo, ok := m.list.SelectedItem().(domain.Todo)
	if _, busy := m.pending[todo.ID]; !ok || busy {
		return m, nil
	}
	switch status {
	case domain.TodoStatusOpen:
		if todo.Status == domain.TodoStatusOpen {
			return m, nil
		}
	case domain.TodoStatusCompleted, domain.TodoStatusCanceled:
		if todo.Status != domain.TodoStatusOpen {
			return m, nil
		}
	default:
		return m, nil
	}

	todo.Status = status
	m.pending[todo.ID] = false
	spinnerCmd := m.beginCloudOperation()
	return m, tea.Batch(
		m.list.SetItem(index, todo),
		saveStatusAndReload(m.repo, todo.ID, status, m.tabs[m.active].list),
		spinnerCmd,
	)
}

// deleteTodo strikes through the selected todo, then trashes it and reloads.
func (m Model) deleteTodo() (tea.Model, tea.Cmd) {
	todo, ok := m.list.SelectedItem().(domain.Todo)
	if _, busy := m.pending[todo.ID]; !ok || busy {
		return m, nil
	}
	m.pending[todo.ID] = true
	spinnerCmd := m.beginCloudOperation()
	return m, tea.Batch(
		deleteAndReload(m.repo, todo.ID, m.tabs[m.active].list),
		spinnerCmd,
	)
}

// beginCloudOperation tracks a unit of cloud work, starting the spinner when it
// is the first one in flight.
func (m *Model) beginCloudOperation() tea.Cmd {
	m.cloudOps++
	if m.cloudOps == 1 {
		return m.spinner.Tick
	}
	return nil
}

// View renders the complete terminal screen.
func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true

	if m.width < 70 || m.height < 5 {
		v.Content = errorStyle.Render("That's a pretty small terminal. Try resizing it?")
		return v
	}

	header := m.renderHeader()
	footer := legendStyle.Render(m.keys.legend(m.tabs[m.active].list))

	listView := m.list.View()
	if m.cloudOps > 0 && len(m.list.Items()) == 0 {
		listView = statusStyle.Height(m.list.Height()).Render("Loading...")
	}
	v.Content = lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		listView,
		"",
		footer,
	)
	if m.modal != nil {
		v.Content = m.modal.Overlay(v.Content)
	}
	return v
}

func (m Model) resize(width, height int) tea.Model {
	m.width = width
	m.height = height
	m.list.SetSize(width, max(0, height-4))
	if m.modal != nil {
		m.modal.SetSize(width, height)
	}
	return m
}

func (m Model) openModal() Model {
	todo, ok := m.list.SelectedItem().(domain.Todo)
	if !ok {
		return m
	}
	m.modal = modal.New(todo, m.width, m.height)
	m.modalOriginal = todo
	return m
}

func (m Model) updateModal(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	openModal := m.modal
	nextModal, cmd := openModal.Update(msg)
	m.modal = nextModal
	if m.modal != nil {
		return m, cmd
	}

	// The modal closed: if anything changed, write the edited row into the list
	// and persist, then reload so the list's membership reflects the new
	// schedule/status per Things' own rules.
	before := m.modalOriginal
	current := openModal.Todo()
	m.modalOriginal = domain.Todo{}
	if before.SameEditableFields(current) {
		return m, cmd
	}
	listCmd := m.list.SetItem(m.list.GlobalIndex(), current)
	spinnerCmd := m.beginCloudOperation()
	return m, tea.Batch(
		cmd,
		listCmd,
		saveAndReload(m.repo, before, current, m.tabs[m.active].list),
		spinnerCmd,
	)
}

func (m Model) startRefresh() (tea.Model, tea.Cmd) {
	m.err = nil
	cmd := m.list.SetItems(nil)
	spinnerCmd := m.beginCloudOperation()
	return m, tea.Batch(cmd, loadTodos(m.repo, m.tabs[m.active].list, true), spinnerCmd)
}

func (m Model) switchTab(delta int) (tea.Model, tea.Cmd) {
	m.active = (m.active + delta + len(m.tabs)) % len(m.tabs)
	m.err = nil
	cmd := m.list.SetItems(nil)
	spinnerCmd := m.beginCloudOperation()
	return m, tea.Batch(cmd, loadTodos(m.repo, m.tabs[m.active].list, false), spinnerCmd)
}

// renderHeader renders tabs and sync status.
func (m Model) renderHeader() string {
	tabs := make([]string, len(m.tabs))
	for i, t := range m.tabs {
		style := inactiveTabStyle
		if i == m.active {
			style = activeTabStyle.Background(lipgloss.Color(t.color))
		}
		tabs[i] = style.Render(t.label)
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	if m.cloudOps > 0 {
		spinner := m.spinner.View()
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.PlaceHorizontal(m.width-lipgloss.Width(spinner), lipgloss.Left, tabBar),
			spinner,
		)
	}
	return tabBar
}
