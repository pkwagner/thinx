package domain

import "context"

type TodoList int

const (
	ListToday TodoList = iota
	ListInbox
	ListScheduled
	ListAnytime
	ListLogbook
)

type TodoFilter struct {
	List TodoList
}

type TodoRepository interface {
	List(ctx context.Context, filter TodoFilter, forceSync bool) ([]Todo, error)
}
