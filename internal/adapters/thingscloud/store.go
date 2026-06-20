package thingscloud

import (
	"context"
	"fmt"

	things "github.com/arthursoares/things-cloud-sdk"
	thingssync "github.com/arthursoares/things-cloud-sdk/sync"
	"thinx/internal/domain"
)

type Store struct {
	syncer *thingssync.Syncer
}

func NewStore(dbPath, email, password string) (*Store, error) {
	client := things.New(things.APIEndpoint, email, password)
	syncer, err := thingssync.Open(dbPath, client)
	if err != nil {
		return nil, err
	}
	return &Store{syncer: syncer}, nil
}

func (s *Store) Close() error {
	return s.syncer.Close()
}

func (s *Store) List(ctx context.Context, filter domain.TodoFilter, forceSync bool) ([]domain.Todo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if forceSync {
		if _, err := s.syncer.Sync(); err != nil {
			return nil, err
		}
	}

	state := s.syncer.State()
	opts := thingssync.QueryOpts{}

	var tasks []*things.Task
	var err error
	switch filter.List {
	case domain.ListToday:
		tasks, err = state.TasksInToday(opts)
	case domain.ListInbox:
		tasks, err = state.TasksInInbox(opts)
	case domain.ListScheduled:
		tasks, err = state.TasksInUpcoming(opts)
	case domain.ListAnytime:
		tasks, err = state.TasksInAnytime(opts)
	case domain.ListLogbook:
		tasks, err = state.TasksInLogbook(opts)
	default:
		return nil, fmt.Errorf("unknown todo list %d", filter.List)
	}
	if err != nil {
		return nil, err
	}

	todos := make([]domain.Todo, 0, len(tasks))
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if task == nil {
			continue
		}
		todos = append(todos, domain.Todo{
			ID:          task.UUID,
			Title:       task.Title,
			ScheduledAt: task.ScheduledDate,
			CheckedAt:   task.CompletionDate,
		})
	}
	return todos, nil
}
