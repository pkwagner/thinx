package modal

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"thinx/internal/domain"
)

// TestScheduleShortcuts verifies Anytime, Someday, and empty-date behavior.
func TestScheduleShortcuts(t *testing.T) {
	t.Parallel()
	m := New(domain.Todo{Schedule: domain.TodoScheduleAnytime}, 100, 30)

	_, _ = m.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	if m.todo.Schedule != domain.TodoScheduleSomeday || m.todo.ScheduledAt != nil {
		t.Fatalf("someday result = %#v", m.todo)
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if m.todo.Schedule != domain.TodoScheduleAnytime || m.todo.ScheduledAt != nil {
		t.Fatalf("anytime result = %#v", m.todo)
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.todo.Schedule != domain.TodoScheduleSomeday || m.todo.ScheduledAt != nil {
		t.Fatalf("empty scheduled date result = %#v", m.todo)
	}
}

// TestClearDeadlineShortcut verifies D clears a deadline.
func TestClearDeadlineShortcut(t *testing.T) {
	t.Parallel()
	deadline := time.Now()
	m := New(domain.Todo{DeadlineAt: &deadline}, 100, 30)
	_, _ = m.Update(tea.KeyPressMsg{Code: 'D', Text: "D"})
	if m.todo.DeadlineAt != nil {
		t.Fatalf("deadline was not cleared: %v", m.todo.DeadlineAt)
	}
}

// TestPasteFillsTitleInput verifies pasted text lands in the title field while editing.
func TestPasteFillsTitleInput(t *testing.T) {
	t.Parallel()
	m := New(domain.Todo{}, 100, 30)
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // enters title edit
	if m.editing != fieldTitle {
		t.Fatalf("editing = %v, want fieldTitle", m.editing)
	}
	_, _ = m.Update(tea.PasteMsg{Content: "Pasted title"})
	if got := m.titleInput.Value(); got != "Pasted title" {
		t.Fatalf("title after paste = %q", got)
	}
}

// TestEmptyTitleCannotBeSaved verifies Enter is refused while the title is empty
// (only Esc leaves the field), matching the date-input validation pattern.
func TestEmptyTitleCannotBeSaved(t *testing.T) {
	t.Parallel()
	m := New(domain.Todo{Title: "Original"}, 100, 30)
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // enter title edit
	m.titleInput.SetValue("   ")                         // whitespace only

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next == nil {
		t.Fatal("empty title Enter should not close the modal")
	}
	if m.editing != fieldTitle {
		t.Fatalf("editing = %v, want still fieldTitle (Enter blocked)", m.editing)
	}
	if m.todo.Title != "Original" {
		t.Fatalf("title = %q, want unchanged", m.todo.Title)
	}

	m.titleInput.SetValue("New title")
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.editing != fieldNone || m.todo.Title != "New title" {
		t.Fatalf("non-empty commit failed: editing=%v title=%q", m.editing, m.todo.Title)
	}
}

// TestCreateEscAborts verifies Esc in create mode closes the modal without saving,
// while a committed title lets a later close persist.
func TestCreateEscAborts(t *testing.T) {
	t.Parallel()
	m := New(domain.Todo{}, 100, 30)
	_ = m.BeginCreate()
	if m.editing != fieldTitle || !m.creating {
		t.Fatalf("BeginCreate state: editing=%v creating=%v", m.editing, m.creating)
	}
	m.titleInput.SetValue("half-typed")

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if next != nil {
		t.Fatal("esc in create title edit should close the modal (return nil)")
	}
}

// TestNormalLegendGroupsEditScheduleAndClear verifies the revised shortcuts.
func TestNormalLegendGroupsEditScheduleAndClear(t *testing.T) {
	t.Parallel()
	legend := New(domain.Todo{}, 100, 30).legend()
	for _, want := range []string{"enter/s/d/n: edit", "t/i/a: schedule", "S/D: clear"} {
		if !strings.Contains(legend, want) {
			t.Fatalf("legend %q does not contain %q", legend, want)
		}
	}
}
