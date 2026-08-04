package domain

import (
	"testing"
	"time"
)

// TestTodoSameEditableFields verifies modal field comparison.
func TestTodoSameEditableFields(t *testing.T) {
	t.Parallel()
	date := time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC)
	todo := Todo{Title: "Title", Note: "Note", Schedule: TodoScheduleAnytime, ScheduledAt: &date}
	copy := todo
	copy.Project = []string{"Ignored"}
	if !todo.SameEditableFields(copy) {
		t.Fatal("non-editable project changed comparison")
	}
	copy.Note = "Changed"
	if todo.SameEditableFields(copy) {
		t.Fatal("changed note considered equal")
	}
}
