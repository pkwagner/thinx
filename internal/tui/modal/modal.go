// Package modal renders and edits a single task in a centered overlay dialog.
package modal

import (
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"thinx/internal/domain"
)

const (
	maxWidth   = 76 // maximum outer width of the bordered box
	margin     = 6  // columns kept free on each side at narrow widths
	labelWidth = 11 // width of the left label column
	noteHeight = 4  // lines shown in the note textarea
)

// field identifies which part of the modal is currently being edited.
type field int

const (
	fieldNone field = iota
	fieldTitle
	fieldScheduled
	fieldDeadline
	fieldNote
)

// Model is the task-detail overlay.
type Model struct {
	todo           domain.Todo
	keys           keyMap
	editing        field
	saved          bool
	titleInput     textinput.Model
	scheduledInput textinput.Model
	deadlineInput  textinput.Model
	noteInput      textarea.Model
	width          int
	height         int
}

// New creates a modal for the given todo, sized to the terminal.
func New(todo domain.Todo, width, height int) *Model {
	title := textinput.New()
	title.Prompt = ""
	title.CharLimit = 200

	note := textarea.New()
	note.Prompt = ""
	note.ShowLineNumbers = false
	note.SetHeight(noteHeight)

	m := &Model{
		todo:           todo,
		keys:           newKeyMap(),
		titleInput:     title,
		scheduledInput: newDateInput(),
		deadlineInput:  newDateInput(),
		noteInput:      note,
	}
	m.SetSize(width, height)
	return m
}

func newDateInput() textinput.Model {
	di := textinput.New()
	di.Prompt = ""
	di.CharLimit = 10
	di.Placeholder = "yyyy-mm-dd"
	return di
}

// Todo returns the (possibly edited) task.
func (m *Model) Todo() domain.Todo { return m.todo }

// TakeSaved reports whether the todo changed since the last call and resets the flag.
func (m *Model) TakeSaved() bool {
	saved := m.saved
	m.saved = false
	return saved
}

// SetSize stores the terminal size and reflows all inputs.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	// textinput renders at SetWidth+1 (cursor), so subtract 1 to fill exactly.
	m.titleInput.SetWidth(m.contentWidth() - 1)
	m.scheduledInput.SetWidth(m.dateInputWidth(m.scheduledHint()))
	m.deadlineInput.SetWidth(m.dateInputWidth(m.deadlineHint()))
	m.noteInput.SetWidth(m.contentWidth() - labelWidth)
}
