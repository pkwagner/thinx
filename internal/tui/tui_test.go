package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/pkwagner/thinx/internal/domain"
)

type statusRepository struct {
	id           string
	status       domain.TodoStatus
	deleted      string
	updateBefore domain.Todo
	updateAfter  domain.Todo
	created      domain.Todo
	err          error
}

// Create records the requested new todo.
func (r *statusRepository) Create(_ context.Context, todo domain.Todo) (domain.Todo, error) {
	r.created = todo
	if todo.ID == "" {
		todo.ID = "created-id"
	}
	return todo, r.err
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

// loadedModel returns an idle, focused model with one todo, as if a load finished.
func loadedModel(repo domain.TodoRepository) Model {
	m := NewModel(repo)
	m.cloudOps = 0
	m.list.SetItems(todoItems([]domain.Todo{{ID: "one", Title: "One"}}))
	return m
}

// TestShouldSync covers the auto-refresh gate.
func TestShouldSync(t *testing.T) {
	t.Parallel()
	stale := time.Now().Add(-2 * autoRefreshInterval)
	base := loadedModel(&statusRepository{})
	base.focused = true
	base.lastSync = stale

	if !base.shouldSync() {
		t.Fatal("focused idle stale model should sync")
	}

	fresh := base
	fresh.lastSync = time.Now()
	if fresh.shouldSync() {
		t.Fatal("recently synced model should not sync")
	}

	blurred := base
	blurred.focused = false
	if blurred.shouldSync() {
		t.Fatal("blurred model should not sync")
	}

	busy := base
	busy.cloudOps = 1
	if busy.shouldSync() {
		t.Fatal("busy model should not sync")
	}

	withModal := base
	withModal = withModal.openModal()
	if withModal.shouldSync() {
		t.Fatal("model with open modal should not sync")
	}
}

// TestLoadTodosSetsSyncedFlag verifies forceSync is reported back on the message.
func TestLoadTodosSetsSyncedFlag(t *testing.T) {
	t.Parallel()
	repo := &statusRepository{}
	if msg := loadTodos(repo, domain.ListToday, true)().(todosLoadedMsg); !msg.synced {
		t.Fatal("forceSync load should report synced=true")
	}
	if msg := loadTodos(repo, domain.ListToday, false)().(todosLoadedMsg); msg.synced {
		t.Fatal("local load should report synced=false")
	}
}

// TestFocusTriggersRefreshWhenStale verifies regaining focus refreshes a stale list.
func TestFocusTriggersRefreshWhenStale(t *testing.T) {
	t.Parallel()
	m := loadedModel(&statusRepository{})
	m.focused = false // lastSync stays zero → stale

	next, _ := m.Update(tea.FocusMsg{})
	m = next.(Model)
	if !m.focused || m.cloudOps != 1 {
		t.Fatalf("focus refresh: focused=%v cloudOps=%d", m.focused, m.cloudOps)
	}
}

// TestFocusDoesNotRefreshWhenFresh verifies the rate-limit window blocks refresh.
func TestFocusDoesNotRefreshWhenFresh(t *testing.T) {
	t.Parallel()
	m := loadedModel(&statusRepository{})
	m.lastSync = time.Now()

	next, _ := m.Update(tea.FocusMsg{})
	m = next.(Model)
	if m.cloudOps != 0 {
		t.Fatalf("fresh focus should not refresh: cloudOps=%d", m.cloudOps)
	}
}

// TestBlurBlocksHeartbeatRefresh verifies a blurred terminal never syncs, but the
// heartbeat keeps ticking.
func TestBlurBlocksHeartbeatRefresh(t *testing.T) {
	t.Parallel()
	m := loadedModel(&statusRepository{})

	next, _ := m.Update(tea.BlurMsg{})
	m = next.(Model)
	if m.focused {
		t.Fatal("blur did not clear focus")
	}

	next, cmd := m.Update(heartbeatMsg{})
	m = next.(Model)
	if m.cloudOps != 0 {
		t.Fatalf("blurred heartbeat should not refresh: cloudOps=%d", m.cloudOps)
	}
	if cmd == nil {
		t.Fatal("heartbeat should re-arm itself even when blurred")
	}
}

// TestHeartbeatRefreshesAndRearms verifies the periodic tick refreshes when idle
// and stale, and reschedules itself.
func TestHeartbeatRefreshesAndRearms(t *testing.T) {
	t.Parallel()
	m := loadedModel(&statusRepository{}) // focused, lastSync zero → stale

	next, cmd := m.Update(heartbeatMsg{})
	m = next.(Model)
	if m.cloudOps != 1 {
		t.Fatalf("idle heartbeat should refresh: cloudOps=%d", m.cloudOps)
	}
	if cmd == nil {
		t.Fatal("heartbeat should re-arm itself")
	}
}

// TestHeartbeatBlockedByModal verifies an open modal suppresses auto-refresh.
func TestHeartbeatBlockedByModal(t *testing.T) {
	t.Parallel()
	m := loadedModel(&statusRepository{})
	m.list.Select(0)
	m = m.openModal()

	next, _ := m.Update(heartbeatMsg{})
	m = next.(Model)
	if m.cloudOps != 0 {
		t.Fatalf("heartbeat with modal open should not refresh: cloudOps=%d", m.cloudOps)
	}
}

// TestRefreshKeyBypassesWindowButRespectsCloudOps verifies manual `r` ignores the
// rate-limit window yet is still blocked while a cloud op is in flight.
func TestRefreshKeyBypassesWindowButRespectsCloudOps(t *testing.T) {
	t.Parallel()
	m := loadedModel(&statusRepository{})
	m.lastSync = time.Now() // fresh: auto-refresh would be blocked

	next, _ := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m = next.(Model)
	if m.cloudOps != 1 {
		t.Fatalf("manual refresh should bypass the window: cloudOps=%d", m.cloudOps)
	}

	next, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m = next.(Model)
	if m.cloudOps != 1 {
		t.Fatalf("manual refresh during a cloud op should be blocked: cloudOps=%d", m.cloudOps)
	}
}

// TestSyncedLoadUpdatesLastSyncAndPreservesCursor verifies a server sync records
// its time and a quiet refresh keeps the selected row.
func TestSyncedLoadUpdatesLastSyncAndPreservesCursor(t *testing.T) {
	t.Parallel()
	m := NewModel(&statusRepository{})
	m.cloudOps = 1
	m.list.SetItems(todoItems([]domain.Todo{
		{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"},
	}))
	m.list.Select(2)

	next, _ := m.Update(todosLoadedMsg{
		list:   domain.ListToday,
		todos:  []domain.Todo{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
		synced: true,
	})
	m = next.(Model)

	if m.lastSync.IsZero() {
		t.Fatal("synced load did not record lastSync")
	}
	if m.list.Index() != 2 {
		t.Fatalf("cursor = %d, want preserved at 2", m.list.Index())
	}
}

// TestUnsyncedLoadLeavesLastSync verifies a local reload does not move the window.
func TestUnsyncedLoadLeavesLastSync(t *testing.T) {
	t.Parallel()
	m := NewModel(&statusRepository{})
	m.cloudOps = 1
	was := time.Now().Add(-time.Minute)
	m.lastSync = was

	next, _ := m.Update(todosLoadedMsg{list: domain.ListToday, synced: false})
	m = next.(Model)
	if !m.lastSync.Equal(was) {
		t.Fatalf("lastSync changed on unsynced load: %v != %v", m.lastSync, was)
	}
}

// TestViewEnablesFocusReporting verifies focus events are requested from the terminal.
func TestViewEnablesFocusReporting(t *testing.T) {
	t.Parallel()
	if !NewModel(&statusRepository{}).View().ReportFocus {
		t.Fatal("View should enable ReportFocus")
	}
}

// TestNewTodoForListPrefills verifies tab-based prefill of a new todo.
func TestNewTodoForListPrefills(t *testing.T) {
	t.Parallel()
	m := NewModel(&statusRepository{})
	cases := []struct {
		list     domain.TodoList
		schedule domain.TodoSchedule
		hasDate  bool
	}{
		{domain.ListInbox, domain.TodoScheduleInbox, false},
		{domain.ListToday, domain.TodoScheduleAnytime, true},
		{domain.ListAnytime, domain.TodoScheduleAnytime, false},
		{domain.ListSomeday, domain.TodoScheduleSomeday, false},
		{domain.ListScheduled, domain.TodoScheduleSomeday, true}, // empty list -> tomorrow
	}
	for _, tc := range cases {
		todo := m.newTodoForList(tc.list)
		if todo.Schedule != tc.schedule {
			t.Errorf("list %v: schedule=%v want %v", tc.list, todo.Schedule, tc.schedule)
		}
		if (todo.ScheduledAt != nil) != tc.hasDate {
			t.Errorf("list %v: hasDate=%v want %v", tc.list, todo.ScheduledAt != nil, tc.hasDate)
		}
	}
}

// TestNewTodoForScheduledCopiesHoveredDate verifies the Scheduled tab copies the
// hovered todo's scheduled date.
func TestNewTodoForScheduledCopiesHoveredDate(t *testing.T) {
	t.Parallel()
	m := NewModel(&statusRepository{})
	when := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)
	m.list.SetItems(todoItems([]domain.Todo{{ID: "a", Title: "A", ScheduledAt: &when}}))
	m.list.Select(0)

	todo := m.newTodoForList(domain.ListScheduled)
	if todo.ScheduledAt == nil || !todo.ScheduledAt.Equal(when) {
		t.Fatalf("scheduled date = %v, want copied %v", todo.ScheduledAt, when)
	}
}

// TestCreateModalDisabledInArchive verifies "n" does nothing in the Logbook.
func TestCreateModalDisabledInArchive(t *testing.T) {
	t.Parallel()
	m := NewModel(&statusRepository{})
	m.width, m.height = 100, 30
	for i, tab := range m.tabs {
		if tab.list == domain.ListLogbook {
			m.active = i
		}
	}
	next, _ := m.openCreateModal()
	if next.(Model).modal != nil {
		t.Fatal("create modal must not open in the Archive")
	}
}

// TestCreateModalSavesNonEmptyOnClose verifies a titled new todo persists on close.
func TestCreateModalSavesNonEmptyOnClose(t *testing.T) {
	t.Parallel()
	repo := &statusRepository{}
	m := NewModel(repo)
	m.width, m.height = 100, 30

	next, _ := m.openCreateModal()
	m = next.(Model)
	if m.modal == nil || !m.modalCreating {
		t.Fatal("create modal not opened")
	}
	tm, _ := m.Update(tea.PasteMsg{Content: "Water plants"})
	m = tm.(Model)
	tm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // commit title
	m = tm.(Model)
	tm, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"}) // close
	m = tm.(Model)

	if m.modal != nil || m.modalCreating {
		t.Fatalf("modal should be closed: modal=%v creating=%v", m.modal, m.modalCreating)
	}
	if cmd == nil {
		t.Fatal("closing a titled new todo should schedule a create")
	}
}

// TestCreateModalDiscardsEmptyOnClose verifies esc on an untitled new todo aborts.
func TestCreateModalDiscardsEmptyOnClose(t *testing.T) {
	t.Parallel()
	repo := &statusRepository{}
	m := NewModel(repo)
	m.width, m.height = 100, 30

	next, _ := m.openCreateModal()
	m = next.(Model)
	tm, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape}) // abort empty
	m = tm.(Model)

	if m.modal != nil || m.modalCreating {
		t.Fatal("modal should be closed after esc")
	}
	if cmd != nil {
		cmd() // drain any batched work
	}
	if repo.created.Title != "" {
		t.Fatalf("empty new todo must not persist, got %q", repo.created.Title)
	}
}

// TestCreateAndReloadPersistsAndReloads verifies the command creates then reloads.
func TestCreateAndReloadPersistsAndReloads(t *testing.T) {
	t.Parallel()
	repo := &statusRepository{}
	cmd := createAndReload(repo, domain.Todo{Title: "X", Schedule: domain.TodoScheduleInbox}, domain.ListInbox)
	msg, ok := cmd().(todosLoadedMsg)
	if !ok {
		t.Fatal("createAndReload should return todosLoadedMsg")
	}
	if repo.created.Title != "X" {
		t.Fatalf("repo.Create not called with the new todo, got %q", repo.created.Title)
	}
	if msg.list != domain.ListInbox {
		t.Fatalf("reload list = %v, want Inbox", msg.list)
	}
}

// TestTodayHidesScheduledDate verifies the Today list omits the (redundant)
// scheduled date while Scheduled still shows it.
func TestTodayHidesScheduledDate(t *testing.T) {
	t.Parallel()
	when := time.Now()
	todo := domain.Todo{Title: "X", ScheduledAt: &when}
	if strings.Contains(todoDetails(todo, domain.ListToday), "@") {
		t.Fatalf("Today should not show the scheduled date: %q", todoDetails(todo, domain.ListToday))
	}
	if !strings.Contains(todoDetails(todo, domain.ListScheduled), "@") {
		t.Fatal("Scheduled should still show the scheduled date")
	}
}

// TestCreatedTodoIsHighlighted verifies the reload selects a newly created todo
// via todosLoadedMsg.selectID, rather than preserving the prior cursor.
func TestCreatedTodoIsHighlighted(t *testing.T) {
	t.Parallel()
	m := NewModel(&statusRepository{})
	m.cloudOps = 1
	m.list.SetItems(todoItems([]domain.Todo{{ID: "a", Title: "A"}, {ID: "b", Title: "B"}}))
	m.list.Select(0)

	next, _ := m.Update(todosLoadedMsg{
		list:     domain.ListToday,
		todos:    []domain.Todo{{ID: "a", Title: "A"}, {ID: "b", Title: "B"}, {ID: "c", Title: "C"}},
		selectID: "c",
	})
	m = next.(Model)
	if got := m.list.SelectedItem().(domain.Todo).ID; got != "c" {
		t.Fatalf("selected todo = %q, want the newly created %q", got, "c")
	}
}

// TestTodoDetailsRendersProjectHierarchy verifies each level of the enclosing
// hierarchy gets its own tag, outermost first, and that Inbox omits them (a
// todo in the Inbox has no project by definition).
func TestTodoDetailsRendersProjectHierarchy(t *testing.T) {
	t.Parallel()
	todo := domain.Todo{Title: "Schlauch Schreibtisch", Project: []string{"Wohnung einrichten", "🏗️"}}

	if got, want := todoDetails(todo, domain.ListAnytime), "#Wohnung einrichten #🏗️"; got != want {
		t.Fatalf("details = %q, want %q", got, want)
	}
	if got := todoDetails(todo, domain.ListInbox); strings.Contains(got, "#") {
		t.Fatalf("Inbox should not show the hierarchy, got %q", got)
	}
	if got := todoDetails(domain.Todo{Title: "Loose"}, domain.ListAnytime); got != "" {
		t.Fatalf("a todo without a project should have no tags, got %q", got)
	}
}
