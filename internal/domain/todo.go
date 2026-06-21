package domain

import "time"

type Todo struct {
	ID          string
	Title       string
	Project     string
	ScheduledAt *time.Time
	DeadlineAt  *time.Time
	CheckedAt   *time.Time
}

// Checked reports whether the todo is completed.
func (t Todo) Checked() bool {
	return t.CheckedAt != nil
}

// FilterValue returns the todo text used by TUI list filtering.
func (t Todo) FilterValue() string {
	return t.Title
}
