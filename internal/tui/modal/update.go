package modal

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"thinx/internal/domain"
	"thinx/internal/tui/uihelp"
)

// Update applies a key press and returns the updated modal, or nil when it should close.
func (m *Model) Update(msg tea.KeyPressMsg) (*Model, tea.Cmd) {
	switch m.editing {
	case fieldNone:
		return m.updateNormal(msg)
	case fieldTitle:
		return m.updateTitleEdit(msg)
	case fieldScheduled:
		return m.updateDateEdit(msg, &m.scheduledInput)
	case fieldDeadline:
		return m.updateDateEdit(msg, &m.deadlineInput)
	case fieldNote:
		return m.updateNoteEdit(msg)
	}
	return m, nil
}

func (m *Model) updateNormal(msg tea.KeyPressMsg) (*Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.close):
		return nil, nil
	case key.Matches(msg, m.keys.editTitle):
		m.editing = fieldTitle
		m.titleInput.SetValue(m.todo.Title)
		m.titleInput.CursorEnd()
		return m, m.titleInput.Focus()
	case key.Matches(msg, m.keys.editNote):
		m.editing = fieldNote
		m.noteInput.SetValue(m.todo.Note)
		return m, m.noteInput.Focus()
	}
	if m.todo.Status == domain.TodoStatusOpen {
		switch {
		case key.Matches(msg, m.keys.scheduleToday):
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
			m.todo.Schedule = domain.TodoScheduleAnytime
			m.todo.ScheduledAt = &today
			m.saved = true
			return m, nil
		case key.Matches(msg, m.keys.scheduleInbox):
			m.todo.Schedule = domain.TodoScheduleInbox
			m.todo.ScheduledAt = nil
			m.saved = true
			return m, nil
		case key.Matches(msg, m.keys.scheduleAnytime):
			m.todo.Schedule = domain.TodoScheduleAnytime
			m.todo.ScheduledAt = nil
			m.saved = true
			return m, nil
		case key.Matches(msg, m.keys.scheduleSomeday):
			m.todo.Schedule = domain.TodoScheduleSomeday
			m.todo.ScheduledAt = nil
			m.saved = true
			return m, nil
		case key.Matches(msg, m.keys.clearDeadline):
			m.todo.DeadlineAt = nil
			m.saved = true
			return m, nil
		case key.Matches(msg, m.keys.editScheduled):
			m.editing = fieldScheduled
			if m.todo.ScheduledAt != nil {
				m.scheduledInput.SetValue(uihelp.FormatDate(*m.todo.ScheduledAt))
			} else {
				m.scheduledInput.SetValue("")
			}
			m.scheduledInput.CursorEnd()
			return m, m.scheduledInput.Focus()
		case key.Matches(msg, m.keys.editDeadline):
			m.editing = fieldDeadline
			if m.todo.DeadlineAt != nil {
				m.deadlineInput.SetValue(uihelp.FormatDate(*m.todo.DeadlineAt))
			} else {
				m.deadlineInput.SetValue("")
			}
			m.deadlineInput.CursorEnd()
			return m, m.deadlineInput.Focus()
		}
	}
	return m, nil
}

func (m *Model) updateTitleEdit(msg tea.KeyPressMsg) (*Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.save):
		m.todo.Title = strings.TrimSpace(m.titleInput.Value())
		m.editing = fieldNone
		m.titleInput.Blur()
		m.saved = true
		return m, nil
	case key.Matches(msg, m.keys.cancel):
		m.editing = fieldNone
		m.titleInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.titleInput, cmd = m.titleInput.Update(msg)
	return m, cmd
}

func (m *Model) updateDateEdit(msg tea.KeyPressMsg, input *textinput.Model) (*Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.cancel):
		m.editing = fieldNone
		input.Blur()
		return m, nil
	case key.Matches(msg, m.keys.save):
		var t *time.Time
		if val := strings.TrimSpace(input.Value()); val != "" {
			parsed, err := time.ParseInLocation("2006-01-02", val, time.Local)
			if err != nil {
				return m, nil // stay editing; red text signals error
			}
			t = &parsed
		}
		switch m.editing {
		case fieldScheduled:
			m.todo.ScheduledAt = t
			if t == nil {
				m.todo.Schedule = domain.TodoScheduleSomeday
			} else if m.todo.Schedule == domain.TodoScheduleInbox {
				m.todo.Schedule = domain.TodoScheduleAnytime
			}
		case fieldDeadline:
			m.todo.DeadlineAt = t
		}
		m.editing = fieldNone
		input.Blur()
		m.saved = true
		return m, nil
	}
	var cmd tea.Cmd
	*input, cmd = input.Update(msg)
	return m, cmd
}

func (m *Model) updateNoteEdit(msg tea.KeyPressMsg) (*Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.cancel):
		m.editing = fieldNone
		m.noteInput.Blur()
		return m, nil
	case key.Matches(msg, m.keys.noteNewline):
		m.noteInput.InsertString("\n")
		return m, nil
	case key.Matches(msg, m.keys.save):
		m.todo.Note = strings.TrimRight(m.noteInput.Value(), "\n")
		m.editing = fieldNone
		m.noteInput.Blur()
		m.saved = true
		return m, nil
	}
	var cmd tea.Cmd
	m.noteInput, cmd = m.noteInput.Update(msg)
	return m, cmd
}
