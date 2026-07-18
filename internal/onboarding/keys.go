package onboarding

import "charm.land/bubbles/v2/key"

type keyMap struct {
	next   key.Binding
	prev   key.Binding
	submit key.Binding
	quit   key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		next:   key.NewBinding(key.WithKeys("tab", "down"), key.WithHelp("tab", "select")),
		prev:   key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("shift+tab", "select")),
		submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		quit:   key.NewBinding(key.WithKeys("ctrl+c", "esc"), key.WithHelp("ctrl+c", "quit")),
	}
}
