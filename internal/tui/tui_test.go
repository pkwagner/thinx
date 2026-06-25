package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"thinx/internal/domain"
)

type statusRepository struct {
	id           string
	status       domain.TodoStatus
	deleted      string
	updateBefore domain.Todo
	updateAfter  domain.Todo
	err          error
}

// Update records the requested task edit.
func (r *statusRepository) Update(_ context.Context, before, after domain.Todo) error {
	r.updateBefore = before
	r.updateAfter = after
	return r.err
}

// Delete records the requested deletion.
func (r *statusRepository) Delete(_ context.Context, id string) error {
	r.deleted = id
	return r.err
}

// List returns no todos for status command tests.
func (r *statusRepository) List(context.Context, domain.TodoFilter, bool) ([]domain.Todo, error) {
	return nil, nil
}

// SetStatus records the requested status.
func (r *statusRepository) SetStatus(_ context.Context, id string, status domain.TodoStatus) error {
	r.id = id
	r.status = status
	return r.err
}

// TestSetTodoStatusShowsOptimisticallyThenReloads verifies the row restyles
// immediately and leaves the list once the reconciling reload arrives.
func TestSetTodoStatusShowsOptimisticallyThenReloads(t *testing.T) {
	t.Parallel()
	repo := &statusRepository{}
	m := NewModel(repo)
	m.cloudOps = 0
	m.list.SetItems(todoItems([]domain.Todo{{ID: "one", Title: "One"}, {ID: "two", Title: "Two"}}))

	next, _ := m.setTodoStatus(domain.TodoStatusCompleted)
	m = next.(Model)
	marked := m.list.Items()[0].(domain.Todo)
	if marked.Status != domain.TodoStatusCompleted || m.cloudOps != 1 {
		t.Fatalf("optimistic state: status=%v cloudOps=%d", marked.Status, m.cloudOps)
	}

	// The reload drops the completed todo (no longer part of the list).
	next, _ = m.Update(mutationDoneMsg{id: "one", list: domain.ListToday, todos: []domain.Todo{{ID: "two", Title: "Two"}}})
	m = next.(Model)
	if len(m.list.Items()) != 1 || m.list.Items()[0].(domain.Todo).ID != "two" || m.cloudOps != 0 {
		t.Fatalf("after reload: items=%#v cloudOps=%d", m.list.Items(), m.cloudOps)
	}
}

// TestSetTodoStatusRestoresTodoAfterFailure verifies a failed write is rolled
// back by the reload, which still returns the unchanged todo.
func TestSetTodoStatusRestoresTodoAfterFailure(t *testing.T) {
	t.Parallel()
	repo := &statusRepository{}
	m := NewModel(repo)
	m.cloudOps = 0
	m.list.SetItems(todoItems([]domain.Todo{{ID: "one", Title: "One"}}))

	next, _ := m.setTodoStatus(domain.TodoStatusCanceled)
	m = next.(Model)
	wantErr := errors.New("write failed")
	next, _ = m.Update(mutationDoneMsg{
		id:    "one",
		list:  domain.ListToday,
		todos: []domain.Todo{{ID: "one", Title: "One", Status: domain.TodoStatusOpen}},
		err:   wantErr,
	})
	m = next.(Model)

	todo := m.list.Items()[0].(domain.Todo)
	if todo.Status != domain.TodoStatusOpen {
		t.Fatalf("status = %v, want open", todo.Status)
	}
	if !errors.Is(m.err, wantErr) || m.cloudOps != 0 {
		t.Fatalf("error = %v, cloudOps = %d", m.err, m.cloudOps)
	}
}

// TestCompleteInLogbookReopensTodo verifies c undoes terminal statuses in Logbook.
func TestCompleteInLogbookReopensTodo(t *testing.T) {
	t.Parallel()
	repo := &statusRepository{}
	m := NewModel(repo)
	m.cloudOps = 0
	m.active = 5
	m.list.SetItems(todoItems([]domain.Todo{{ID: "one", Title: "One", Status: domain.TodoStatusCanceled}}))

	next, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m = next.(Model)
	if got := m.list.Items()[0].(domain.Todo).Status; got != domain.TodoStatusOpen {
		t.Fatalf("status = %v, want open", got)
	}
	if m.cloudOps != 1 {
		t.Fatalf("cloudOps = %d, want 1", m.cloudOps)
	}
}

// TestDeleteTodoStrikesThroughThenReloads verifies backspace marks the row as
// deleting (struck through) and removes it once the reload returns.
func TestDeleteTodoStrikesThroughThenReloads(t *testing.T) {
	t.Parallel()
	repo := &statusRepository{}
	m := NewModel(repo)
	m.cloudOps = 0
	m.list.SetItems(todoItems([]domain.Todo{{ID: "one", Title: "One"}}))

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = next.(Model)
	if len(m.list.Items()) != 1 || m.cloudOps != 1 || !m.pending["one"] {
		t.Fatalf("after delete: items=%d cloudOps=%d pending=%v", len(m.list.Items()), m.cloudOps, m.pending["one"])
	}
	if !todoStatusStyle(domain.TodoStatusOpen, true).GetStrikethrough() {
		t.Fatal("pending deletion is not struck through")
	}
	if todoStatusStyle(domain.TodoStatusOpen, false).GetStrikethrough() {
		t.Fatal("non-deleting style is struck through")
	}

	next, _ = m.Update(mutationDoneMsg{id: "one", list: domain.ListToday, todos: nil})
	m = next.(Model)
	if len(m.list.Items()) != 0 || m.cloudOps != 0 {
		t.Fatalf("delete result: items=%d cloudOps=%d", len(m.list.Items()), m.cloudOps)
	}
	if _, ok := m.pending["one"]; ok {
		t.Fatal("pending entry not cleared after reload")
	}
}

// TestLoadingViewReplacesEmptyPlaceholder verifies loading is shown in the list body.
func TestLoadingViewReplacesEmptyPlaceholder(t *testing.T) {
	t.Parallel()
	m := NewModel(&statusRepository{})
	m = m.resize(100, 20).(Model)
	content := m.View().Content
	if !strings.Contains(content, "Loading") || strings.Contains(content, "No todos.") {
		t.Fatalf("loading view = %q", content)
	}
}

// TestLegendCombinesCompletionBindings verifies c and C share one help entry.
func TestLegendCombinesCompletionBindings(t *testing.T) {
	t.Parallel()
	legend := newKeyMap().legend(domain.ListToday)
	if !strings.Contains(legend, "c/C: check") {
		t.Fatalf("legend = %q", legend)
	}
}

// TestLogbookLegendShowsUncheck verifies Logbook's completion action label.
func TestLogbookLegendShowsUncheck(t *testing.T) {
	t.Parallel()
	legend := newKeyMap().legend(domain.ListLogbook)
	if !strings.Contains(legend, "c: uncheck") || strings.Contains(legend, "c/C: check") {
		t.Fatalf("legend = %q", legend)
	}
}

// TestModalCloseSyncsChangedFields verifies one cloud update starts on close.
func TestModalCloseSyncsChangedFields(t *testing.T) {
	t.Parallel()
	repo := &statusRepository{}
	original := domain.Todo{ID: "one", Title: "One", Schedule: domain.TodoScheduleInbox}
	m := NewModel(repo)
	m.cloudOps = 0
	m.list.SetItems(todoItems([]domain.Todo{original}))
	m = m.openModal()

	_, _ = m.modal.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	next, _ := m.updateModal(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = next.(Model)

	updated := m.list.Items()[0].(domain.Todo)
	if m.modal != nil || updated.Schedule != domain.TodoScheduleAnytime || updated.ScheduledAt == nil {
		t.Fatalf("modal close result: modal=%v todo=%#v", m.modal, updated)
	}
	if m.cloudOps != 1 {
		t.Fatalf("cloudOps = %d, want 1", m.cloudOps)
	}

	// The reload triggered by the close clears the in-flight operation.
	next, _ = m.Update(todosLoadedMsg{list: domain.ListToday})
	m = next.(Model)
	if m.cloudOps != 0 {
		t.Fatalf("cloudOps after reload = %d, want 0", m.cloudOps)
	}
}

// TestSaveAndReload persists the edit and reloads the list in one operation.
func TestSaveAndReload(t *testing.T) {
	t.Parallel()
	repo := &statusRepository{}
	before := domain.Todo{ID: "one", Title: "One", Schedule: domain.TodoScheduleInbox}
	after := domain.Todo{ID: "one", Title: "One", Schedule: domain.TodoScheduleAnytime}

	msg := saveAndReload(repo, before, after, domain.ListToday)().(todosLoadedMsg)
	if msg.list != domain.ListToday || msg.err != nil {
		t.Fatalf("reload message = %#v", msg)
	}
	if !repo.updateBefore.SameEditableFields(before) || !repo.updateAfter.SameEditableFields(after) {
		t.Fatalf("persisted (%#v, %#v)", repo.updateBefore, repo.updateAfter)
	}
}

// TestModalCloseSkipsUnchangedTodo verifies closing alone does not communicate.
func TestModalCloseSkipsUnchangedTodo(t *testing.T) {
	t.Parallel()
	repo := &statusRepository{}
	m := NewModel(repo)
	m.cloudOps = 0
	m.list.SetItems(todoItems([]domain.Todo{{ID: "one", Title: "One"}}))
	m = m.openModal()

	next, cmd := m.updateModal(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = next.(Model)
	if cmd != nil || m.cloudOps != 0 {
		t.Fatalf("unchanged close: cmd=%v cloudOps=%d", cmd != nil, m.cloudOps)
	}
}

// TestLoadedTodosRemainVisibleWithSyncError verifies cached results survive sync errors.
func TestLoadedTodosRemainVisibleWithSyncError(t *testing.T) {
	t.Parallel()
	m := NewModel(&statusRepository{})
	wantErr := errors.New("sync failed")
	next, _ := m.Update(todosLoadedMsg{
		list:  domain.ListToday,
		todos: []domain.Todo{{ID: "one", Title: "Cached"}},
		err:   wantErr,
	})
	m = next.(Model)

	if len(m.list.Items()) != 1 || m.list.Items()[0].(domain.Todo).ID != "one" {
		t.Fatalf("items = %#v", m.list.Items())
	}
	if m.cloudOps != 0 || !errors.Is(m.err, wantErr) {
		t.Fatalf("cloudOps = %d, err = %v", m.cloudOps, m.err)
	}
}

// TestSaveStatusAndReload writes the status, then reloads in one command.
func TestSaveStatusAndReload(t *testing.T) {
	defer func(d time.Duration) { mutationDisplay = d }(mutationDisplay)
	mutationDisplay = 0

	repo := &statusRepository{}
	msg := saveStatusAndReload(repo, "one", domain.TodoStatusCanceled, domain.ListToday)().(mutationDoneMsg)

	if msg.id != "one" || msg.list != domain.ListToday || msg.err != nil {
		t.Fatalf("message = %#v", msg)
	}
	if repo.id != "one" || repo.status != domain.TodoStatusCanceled {
		t.Fatalf("saved (%q, %v), want (one, canceled)", repo.id, repo.status)
	}
}
