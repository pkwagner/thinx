package tui

import (
	"strings"

	"thinx/internal/domain"
	"thinx/internal/tui/uihelp"

	"charm.land/bubbles/v2/key"
)

type keyMap struct {
	previousList key.Binding
	nextList     key.Binding
	previousTodo key.Binding
	nextTodo     key.Binding
	openTask     key.Binding
	completeTodo key.Binding
	cancelTodo   key.Binding
	deleteTodo   key.Binding
	refresh      key.Binding
	quit         key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		previousList: key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "prev list")),
		nextList:     key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "next list")),
		previousTodo: key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "prev todo")),
		nextTodo:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "next todo")),
		openTask:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		completeTodo: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "complete")),
		cancelTodo:   key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "cancel")),
		deleteTodo:   key.NewBinding(key.WithKeys("backspace"), key.WithHelp("⌫", "delete")),
		refresh:      key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		quit:         key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k keyMap) legend(list domain.TodoList) string {
	checkHelp := uihelp.MultiBindingHelp("check", k.completeTodo, k.cancelTodo)
	if list == domain.ListLogbook {
		checkHelp = "c: uncheck"
	}
	parts := []string{
		uihelp.MultiBindingHelp("select", k.previousTodo, k.nextTodo, k.previousList, k.nextList),
		uihelp.BindingHelp(k.openTask),
		checkHelp,
		uihelp.BindingHelp(k.deleteTodo),
		uihelp.BindingHelp(k.refresh),
		uihelp.BindingHelp(k.quit),
	}
	return strings.Join(parts, "; ")
}
