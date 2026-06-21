package modal

import "charm.land/bubbles/v2/key"

type keyMap struct {
	close           key.Binding
	editTitle       key.Binding
	editScheduled   key.Binding
	scheduleToday   key.Binding
	scheduleInbox   key.Binding
	scheduleAnytime key.Binding
	scheduleSomeday key.Binding
	editDeadline    key.Binding
	clearDeadline   key.Binding
	editNote        key.Binding
	noteNewline     key.Binding
	save            key.Binding
	cancel          key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		close:           key.NewBinding(key.WithKeys("esc", "q", "ctrl+c"), key.WithHelp("esc", "close")),
		editTitle:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "edit title")),
		editScheduled:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "edit scheduled")),
		scheduleToday:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "today")),
		scheduleInbox:   key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "inbox")),
		scheduleAnytime: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "anytime")),
		scheduleSomeday: key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "someday")),
		editDeadline:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "edit deadline")),
		clearDeadline:   key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "clear deadline")),
		editNote:        key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "edit note")),
		noteNewline:     key.NewBinding(key.WithKeys("shift+enter")),
		save:            key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
		cancel:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}
