package domain

import "time"

// TodoSchedule represents which Things list a task lives in.
type TodoSchedule int

const (
	TodoScheduleInbox   TodoSchedule = iota // st=0: unprocessed
	TodoScheduleAnytime TodoSchedule = iota // st=1: started, no pinned date
	TodoScheduleSomeday TodoSchedule = iota // st=2: deferred
)

// TodoStatus represents the completion state of a task.
type TodoStatus int

const (
	TodoStatusOpen      TodoStatus = iota
	TodoStatusCompleted TodoStatus = iota
	TodoStatusCanceled  TodoStatus = iota
)

// ChecklistItem is a single item within a task's checklist.
type ChecklistItem struct {
	Title     string
	Completed bool
}

type Todo struct {
	ID          string
	Title       string
	Status      TodoStatus
	Schedule    TodoSchedule
	Project     string
	Note        string
	Checklist   []ChecklistItem
	ScheduledAt *time.Time
	DeadlineAt  *time.Time
	CheckedAt   *time.Time
}

// FilterValue returns the todo text used by TUI list filtering.
func (t Todo) FilterValue() string {
	return t.Title
}
