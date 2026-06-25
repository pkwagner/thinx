package tui

import (
	"context"
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"
	"thinx/internal/domain"
)

// mutationDisplay is the minimum time an optimistic status/delete change stays
// on screen before the reconciling reload replaces it.
var mutationDisplay = time.Second

// loadTodos fetches a list's todos in the background.
func loadTodos(repo domain.TodoRepository, list domain.TodoList, forceSync bool) tea.Cmd {
	return func() tea.Msg {
		todos, err := repo.List(context.Background(), domain.TodoFilter{List: list}, forceSync)
		return todosLoadedMsg{list: list, todos: todos, err: err}
	}
}

// saveStatusAndReload persists a status change, then reloads the list.
func saveStatusAndReload(repo domain.TodoRepository, id string, status domain.TodoStatus, list domain.TodoList) tea.Cmd {
	return mutateAndReload(repo, id, list, func(ctx context.Context) error {
		return repo.SetStatus(ctx, id, status)
	})
}

// deleteAndReload trashes a todo, then reloads the list.
func deleteAndReload(repo domain.TodoRepository, id string, list domain.TodoList) tea.Cmd {
	return mutateAndReload(repo, id, list, func(ctx context.Context) error {
		return repo.Delete(ctx, id)
	})
}

// mutateAndReload applies a write, holds the optimistic row on screen for at
// least mutationDisplay, then reloads the list so its contents reflect the
// result. Chaining the reload after the write keeps the two ordered, and the
// reload doubles as rollback when the write fails.
func mutateAndReload(repo domain.TodoRepository, id string, list domain.TodoList, write func(context.Context) error) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		start := time.Now()
		writeErr := write(ctx)
		if d := mutationDisplay - time.Since(start); d > 0 {
			time.Sleep(d)
		}
		todos, listErr := repo.List(ctx, domain.TodoFilter{List: list}, false)
		return mutationDoneMsg{id: id, list: list, todos: todos, err: errors.Join(writeErr, listErr)}
	}
}

// saveAndReload persists modal edits and then reloads the list, so its
// membership reflects the new schedule and status without reimplementing
// Things' placement rules in the TUI.
func saveAndReload(repo domain.TodoRepository, before, after domain.Todo, list domain.TodoList) tea.Cmd {
	return func() tea.Msg {
		updateErr := repo.Update(context.Background(), before, after)
		todos, listErr := repo.List(context.Background(), domain.TodoFilter{List: list}, false)
		return todosLoadedMsg{list: list, todos: todos, err: errors.Join(updateErr, listErr)}
	}
}
