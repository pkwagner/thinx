package domain

import "time"

type Todo struct {
	ID          string
	Title       string
	ScheduledAt *time.Time
	CheckedAt   *time.Time
}

func (t Todo) Checked() bool {
	return t.CheckedAt != nil
}
