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

// NewStore opens the Things Cloud-backed persistent store.
func NewStore(dbPath, email, password string) (*Store, error) {
	client := things.New(things.APIEndpoint, email, password)
	syncer, err := thingssync.Open(dbPath, client)
	if err != nil {
		return nil, err
	}
	return &Store{syncer: syncer}, nil
}

// Close releases the store database connection.
func (s *Store) Close() error {
	return s.syncer.Close()
}

// List returns todos for the requested Things view.
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
	case domain.ListSomeday:
		tasks, err = state.TasksInSomeday(opts)
	case domain.ListLogbook:
		tasks, err = state.TasksInLogbook(opts)
	default:
		return nil, fmt.Errorf("unknown todo list %d", filter.List)
	}
	if err != nil {
		return nil, err
	}

	projectTitlesCache := map[string]string{}
	todos := make([]domain.Todo, 0, len(tasks))
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if task == nil {
			continue
		}
		project, err := projectTitle(state, projectTitlesCache, task)
		if err != nil {
			return nil, err
		}

		checklist, err := checklistItems(state, task.UUID)
		if err != nil {
			return nil, err
		}

		todos = append(todos, domain.Todo{
			ID:          task.UUID,
			Title:       task.Title,
			Status:      mapStatus(task.Status),
			Schedule:    mapSchedule(task.Schedule),
			Project:     project,
			Note:        task.Note,
			Checklist:   checklist,
			ScheduledAt: task.ScheduledDate,
			DeadlineAt:  task.DeadlineDate,
			CheckedAt:   task.CompletionDate,
		})
	}
	return todos, nil
}

func checklistItems(state *thingssync.State, taskUUID string) ([]domain.ChecklistItem, error) {
	raw, err := state.ChecklistItems(taskUUID)
	if err != nil {
		return nil, err
	}
	items := make([]domain.ChecklistItem, 0, len(raw))
	for _, item := range raw {
		if item == nil {
			continue
		}
		items = append(items, domain.ChecklistItem{
			Title:     item.Title,
			Completed: item.Status == things.TaskStatusCompleted,
		})
	}
	return items, nil
}

func mapSchedule(s things.TaskSchedule) domain.TodoSchedule {
	switch s {
	case things.TaskScheduleAnytime:
		return domain.TodoScheduleAnytime
	case things.TaskScheduleSomeday:
		return domain.TodoScheduleSomeday
	default:
		return domain.TodoScheduleInbox
	}
}

func mapStatus(s things.TaskStatus) domain.TodoStatus {
	switch s {
	case things.TaskStatusCompleted:
		return domain.TodoStatusCompleted
	case things.TaskStatusCanceled:
		return domain.TodoStatusCanceled
	default:
		return domain.TodoStatusOpen
	}
}

func projectTitle(state *thingssync.State, cache map[string]string, task *things.Task) (string, error) {
	if len(task.ParentTaskIDs) == 0 || task.ParentTaskIDs[0] == "" {
		return "", nil
	}
	projectID := task.ParentTaskIDs[0]
	if title, ok := cache[projectID]; ok {
		return title, nil
	}
	project, err := state.Task(projectID)
	if err != nil || project == nil {
		return "", err
	}
	cache[projectID] = project.Title
	return project.Title, nil
}
