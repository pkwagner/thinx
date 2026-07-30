package tui

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pkwagner/thinx/internal/domain"
	"github.com/pkwagner/thinx/internal/tui/modal"
	"github.com/pkwagner/thinx/internal/tui/uihelp"
)

// autoRefreshInterval is the minimum time between server syncs and the period of
// the background heartbeat that keeps a focused, idle list current.
const autoRefreshInterval = 30 * time.Second

type tab struct {
	label string
	color string
	list  domain.TodoList
}

type todosLoadedMsg struct {
	list     domain.TodoList
	todos    []domain.Todo
	synced   bool   // the load performed a server sync (forceSync)
	selectID string // if set, highlight this todo after loading (e.g. a new one)
	err      error
}

// heartbeatMsg fires on the periodic timer that drives idle auto-refresh.
type heartbeatMsg struct{}

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
	modalCreating bool // the open modal is composing a new todo, not editing one
	active        int
	cloudOps      int
	err           error
	width         int
	height        int
	focused       bool      // terminal focus; gates auto-refresh
	lastSync      time.Time // when the last server sync completed
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
		cloudOps: 1,    // initial load kicked off by Init
		focused:  true, // assume focused until told otherwise
		pending:  pending,
	}
}

// Init returns the initial Bubble Tea command.
func (m Model) Init() tea.Cmd {
	return tea.Batch(loadTodos(m.repo, m.tabs[m.active].list, true), m.spinner.Tick, heartbeat())
}

// Update applies terminal events to the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg.Width, msg.Height), nil
	case todosLoadedMsg:
		m.cloudOps--
		if msg.synced {
			// Record the attempt regardless of error so a failing sync doesn't
			// re-trigger every 30s; the manual `r` key overrides the window.
			m.lastSync = time.Now()
		}
		if msg.list != m.tabs[m.active].list {
			return m, nil
		}
		m.err = msg.err
		m.list.SetDelegate(todoDelegate{list: msg.list, pending: m.pending})
		// Preserve the cursor: tab-switch/initial/`r` clear the list first (so
		// index is 0 → top), a quiet background refresh keeps the real position.
		index := m.list.Index()
		cmd := m.list.SetItems(todoItems(msg.todos))
		if len(msg.todos) > 0 {
			target := min(index, len(msg.todos)-1)
			if msg.selectID != "" {
				// Highlight a specific todo (e.g. one just created) if present.
				for i, todo := range msg.todos {
					if todo.ID == msg.selectID {
						target = i
						break
					}
				}
			}
			m.list.Select(target)
		}
		return m, cmd
	case tea.FocusMsg:
		m.focused = true
		return m.maybeAutoRefresh()
	case tea.BlurMsg:
		m.focused = false
		return m, nil
	case heartbeatMsg:
		next, cmd := m.maybeAutoRefresh()
		return next, tea.Batch(cmd, heartbeat())
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
	case tea.PasteMsg:
		if m.modal != nil {
			return m.updateModal(msg)
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.modal != nil {
			return m.updateModal(msg)
		}

		switch {
		case key.Matches(msg, m.keys.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.openTask):
			return m.openModal(), nil
		case key.Matches(msg, m.keys.newTodo):
			return m.openCreateModal()
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
			if m.cloudOps > 0 {
				return m, nil
			}
			return m.backgroundRefresh()
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
	v.ReportFocus = true // deliver tea.FocusMsg/tea.BlurMsg for auto-refresh gating

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

// openCreateModal opens the shared modal on a blank todo prefilled for the active
// tab, with the title field focused. Creation is disabled in the Archive.
func (m Model) openCreateModal() (tea.Model, tea.Cmd) {
	list := m.tabs[m.active].list
	if list == domain.ListLogbook {
		return m, nil
	}
	m.modal = modal.New(m.newTodoForList(list), m.width, m.height)
	m.modalCreating = true
	return m, m.modal.BeginCreate()
}

// newTodoForList builds a blank todo with schedule/date prefilled from the tab:
// Inbox/Today/Anytime/Someday map to their schedule; Scheduled copies the hovered
// todo's date (or tomorrow when the list is empty).
func (m Model) newTodoForList(list domain.TodoList) domain.Todo {
	todo := domain.Todo{Status: domain.TodoStatusOpen}
	switch list {
	case domain.ListInbox:
		todo.Schedule = domain.TodoScheduleInbox
	case domain.ListToday:
		today := uihelp.TodayDate()
		todo.Schedule = domain.TodoScheduleAnytime
		todo.ScheduledAt = &today
	case domain.ListSomeday:
		todo.Schedule = domain.TodoScheduleSomeday
	case domain.ListScheduled:
		todo.Schedule = domain.TodoScheduleSomeday
		when := uihelp.TodayDate().AddDate(0, 0, 1)
		if hovered, ok := m.list.SelectedItem().(domain.Todo); ok && hovered.ScheduledAt != nil {
			when = *hovered.ScheduledAt
		}
		todo.ScheduledAt = &when
	default: // ListAnytime
		todo.Schedule = domain.TodoScheduleAnytime
	}
	return todo
}

func (m Model) updateModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	openModal := m.modal
	nextModal, cmd := openModal.Update(msg)
	m.modal = nextModal
	if m.modal != nil {
		return m, cmd
	}

	current := openModal.Todo()

	// The modal closed while creating: persist the new todo unless it was left
	// without a title (an aborted create), then reload so it lands in place.
	if m.modalCreating {
		m.modalCreating = false
		if strings.TrimSpace(current.Title) == "" {
			return m, cmd
		}
		spinnerCmd := m.beginCloudOperation()
		return m, tea.Batch(cmd, createAndReload(m.repo, current, m.tabs[m.active].list), spinnerCmd)
	}

	// The modal closed after an edit: if anything changed, write the edited row
	// into the list and persist, then reload so the list's membership reflects the
	// new schedule/status per Things' own rules.
	before := m.modalOriginal
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

// shouldSync reports whether a server sync is currently warranted: the terminal
// is focused, no modal is open, nothing else is in flight, and the rate-limit
// window has elapsed.
func (m Model) shouldSync() bool {
	return m.focused && m.modal == nil && m.cloudOps == 0 &&
		time.Since(m.lastSync) >= (autoRefreshInterval-time.Second)
}

// backgroundRefresh quietly pulls fresh data for the current tab. It does not
// clear the list, so the current items and cursor stay until fresh data swaps
// in. Shared by the `r` key, focus-gain, and the heartbeat.
func (m Model) backgroundRefresh() (tea.Model, tea.Cmd) {
	m.err = nil
	spinnerCmd := m.beginCloudOperation()
	return m, tea.Batch(loadTodos(m.repo, m.tabs[m.active].list, true), spinnerCmd)
}

// maybeAutoRefresh refreshes only when the gate allows it.
func (m Model) maybeAutoRefresh() (tea.Model, tea.Cmd) {
	if !m.shouldSync() {
		return m, nil
	}
	return m.backgroundRefresh()
}

func (m Model) switchTab(delta int) (tea.Model, tea.Cmd) {
	m.active = (m.active + delta + len(m.tabs)) % len(m.tabs)
	m.err = nil
	// A stale switch doubles as the server sync; a fresh one is an instant local read.
	forceSync := m.shouldSync()
	cmd := m.list.SetItems(nil)
	spinnerCmd := m.beginCloudOperation()
	// TODO: Make this two calls, one for showing the cache and the other for syncing in the background
	return m, tea.Batch(cmd, loadTodos(m.repo, m.tabs[m.active].list, forceSync), spinnerCmd)
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
