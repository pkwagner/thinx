package tui

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pkwagner/thinx/internal/domain"
	"github.com/pkwagner/thinx/internal/tui/uihelp"
)

type todoDelegate struct {
	list    domain.TodoList
	pending map[string]bool // todo ID -> deletion pending (render struck through)
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
	deleting := d.pending[todo.ID]
	prefixStyle := todoStyle
	titleStyle := todoStatusStyle(todo.Status, deleting).UnsetPaddingLeft()
	selected := index == m.Index()
	if selected {
		prefix = "│ "
		prefixStyle = prefixStyle.Bold(true)
		titleStyle = titleStyle.Bold(true)
	}

	title := prefixStyle.Render(prefix) + titleStyle.Render(todo.Title)

	details := todoDetails(todo, d.list)
	details = todoDetailsRenderStyle(selected, deleting).Render(details)

	fmt.Fprint(w, lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.PlaceHorizontal(m.Width()-lipgloss.Width(details), lipgloss.Left, title),
		details,
	))
}

// todoDetailsRenderStyle returns the status-neutral details style.
func todoDetailsRenderStyle(selected, deleting bool) lipgloss.Style {
	style := todoDetailsStyle
	if selected {
		style = selectedTodoDetailsStyle
	}
	if deleting {
		style = style.Strikethrough(true)
	}
	return style
}

// todoStatusStyle returns the title style for a todo's current UI state.
func todoStatusStyle(status domain.TodoStatus, deleting bool) lipgloss.Style {
	var style lipgloss.Style
	switch status {
	case domain.TodoStatusCompleted:
		style = completedTodoStyle
	case domain.TodoStatusCanceled:
		style = canceledTodoStyle
	default:
		style = todoStyle
	}
	if deleting {
		style = style.Strikethrough(true)
	}
	return style
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
	if list != domain.ListInbox {
		for _, name := range todo.Project {
			details = append(details, "#"+name)
		}
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
