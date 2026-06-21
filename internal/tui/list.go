package tui

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"thinx/internal/domain"
	"thinx/internal/tui/uihelp"
)

type todoDelegate struct {
	list domain.TodoList
}

func (todoDelegate) Height() int { return 1 }

func (todoDelegate) Spacing() int { return 0 }

func (todoDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d todoDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	todo, ok := item.(domain.Todo)
	if !ok {
		return
	}

	// Show cursor for active item
	prefix := "  "
	style := todoStyle
	if index == m.Index() {
		prefix = "│ "
		style = selectedTodoStyle
	}

	title := style.Render(prefix + todo.Title)

	details := todoDetails(todo, d.list)
	detailsStyle := todoDetailsStyle
	if index == m.Index() {
		detailsStyle = selectedTodoDetailsStyle
	}
	details = detailsStyle.Render(details)

	fmt.Fprint(w, lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.PlaceHorizontal(m.Width()-lipgloss.Width(details), lipgloss.Left, title),
		details,
	))
}

func todoItems(todos []domain.Todo) []list.Item {
	items := make([]list.Item, len(todos))
	for i, todo := range todos {
		items[i] = todo
	}
	return items
}

func todoDetails(todo domain.Todo, list domain.TodoList) string {
	details := []string{}
	if list != domain.ListInbox && todo.Project != "" {
		details = append(details, "#"+todo.Project)
	}
	if todo.DeadlineAt != nil {
		details = append(details, "!"+uihelp.FormatDate(*todo.DeadlineAt))
	}
	switch list {
	case domain.ListScheduled:
		if todo.ScheduledAt != nil {
			details = append(details, "@"+uihelp.FormatDate(*todo.ScheduledAt))
		}
	case domain.ListLogbook:
		if todo.CheckedAt != nil {
			details = append(details, "@"+uihelp.FormatDate(*todo.CheckedAt))
		}
	}
	return strings.Join(details, " ")
}
