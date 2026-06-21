package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
)

type keyMap struct {
	previousList key.Binding
	nextList     key.Binding
	previousTodo key.Binding
	nextTodo     key.Binding
	quit         key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		previousList: key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "prev list")),
		nextList:     key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "next list")),
		previousTodo: key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "prev todo")),
		nextTodo:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "next todo")),
		quit:         key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k keyMap) legend() string {
	parts := []string{
		multiBindingHelp("select list", k.previousList, k.nextList),
		multiBindingHelp("select todo", k.previousTodo, k.nextTodo),
		bindingHelp(k.quit),
	}
	return strings.Join(parts, "; ")
}

func bindingHelp(binding key.Binding) string {
	help := binding.Help()
	return help.Key + ": " + help.Desc
}

func multiBindingHelp(help string, bindings ...key.Binding) string {
	keys := make([]string, len(bindings))
	for i, b := range bindings {
		keys[i] = b.Help().Key
	}

	return strings.Join(keys, "/") + ": " + help
}
