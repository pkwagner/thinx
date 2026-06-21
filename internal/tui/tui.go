package tui

import (
	"context"

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

type Model struct {
	repo    domain.TodoRepository
	keys    keyMap
	tabs    []tab
	list    list.Model
	spinner spinner.Model
	modal   *modal.Model
	active  int
	loading bool
	err     error
	width   int
	height  int
}

// NewModel creates the initial TUI state.
func NewModel(repo domain.TodoRepository) Model {
	todos := list.New(nil, todoDelegate{list: domain.ListToday}, 0, 0)
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
			{label: "Upcoming", color: "#d75f5f", list: domain.ListScheduled},
			{label: "Anytime", color: "#008b8b", list: domain.ListAnytime},
			{label: "Logbook", color: "#666666", list: domain.ListLogbook},
		},
		active:  1,
		loading: true,
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
		if msg.list != m.tabs[m.active].list {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		m.list.SetDelegate(todoDelegate{list: msg.list})
		cmd := m.list.SetItems(todoItems(msg.todos))
		m.list.Select(0)
		return m, cmd
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyPressMsg:
		if m.modal != nil {
			return m.updateModal(msg)
		}

		// TODO undefined behavior if switching tabs while loading
		switch {
		case key.Matches(msg, m.keys.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.openTask):
			return m.openModal(), nil
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

// View renders the complete terminal screen.
func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true

	if m.width < 70 || m.height < 5 {
		v.Content = errorStyle.Render("That's a pretty small terminal. Try resizing it?")
		return v
	}

	header := m.renderHeader()
	footer := legendStyle.Render(m.keys.legend())

	v.Content = lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.list.View(),
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
	return m
}

func (m Model) updateModal(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.modal, cmd = m.modal.Update(msg)
	if m.modal == nil {
		return m, cmd
	}
	if m.modal.TakeSaved() {
		cmd = tea.Batch(cmd, m.list.SetItem(m.list.GlobalIndex(), m.modal.Todo()))
	}
	return m, cmd
}

func (m Model) switchTab(delta int) (tea.Model, tea.Cmd) {
	m.active = (m.active + delta + len(m.tabs)) % len(m.tabs)
	m.err = nil
	m.loading = true
	cmd := m.list.SetItems(nil)
	return m, tea.Batch(cmd, loadTodos(m.repo, m.tabs[m.active].list, false), m.spinner.Tick)
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
	if m.loading {
		spinner := m.spinner.View()
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.PlaceHorizontal(m.width-lipgloss.Width(spinner), lipgloss.Left, tabBar),
			spinner,
		)
	}
	return tabBar
}

func loadTodos(repo domain.TodoRepository, list domain.TodoList, forceSync bool) tea.Cmd {
	return func() tea.Msg {
		todos, err := repo.List(context.Background(), domain.TodoFilter{List: list}, forceSync)
		return todosLoadedMsg{list: list, todos: todos, err: err}
	}
}
