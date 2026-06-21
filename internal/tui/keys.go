package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"thinx/internal/tui/uihelp"
)

type keyMap struct {
	previousList key.Binding
	nextList     key.Binding
	previousTodo key.Binding
	nextTodo     key.Binding
	openTask     key.Binding
	quit         key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		previousList: key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "prev list")),
		nextList:     key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "next list")),
		previousTodo: key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "prev todo")),
		nextTodo:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "next todo")),
		openTask:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open todo")),
		quit:         key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k keyMap) legend() string {
	parts := []string{
		uihelp.MultiBindingHelp("select list", k.previousList, k.nextList),
		uihelp.MultiBindingHelp("select todo", k.previousTodo, k.nextTodo),
		uihelp.BindingHelp(k.openTask),
		uihelp.BindingHelp(k.quit),
	}
	return strings.Join(parts, "; ")
}
