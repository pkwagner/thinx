package domain

import "context"

type TodoList int

const (
	ListToday TodoList = iota
	ListInbox
	ListScheduled
	ListAnytime
	ListSomeday
	ListLogbook
)

type TodoFilter struct {
	List TodoList
}

type TodoRepository interface {
	List(ctx context.Context, filter TodoFilter, forceSync bool) ([]Todo, error)
	Create(ctx context.Context, todo Todo) (Todo, error)
	Update(ctx context.Context, before, after Todo) error
	SetStatus(ctx context.Context, id string, status TodoStatus) error
	Delete(ctx context.Context, id string) error
}
