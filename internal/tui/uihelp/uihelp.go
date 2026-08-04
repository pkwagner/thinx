// Package uihelp holds small formatting helpers shared across the TUI packages.
package uihelp

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
)

// FormatDate renders a date in the canonical yyyy-mm-dd form used throughout the UI.
func FormatDate(t time.Time) string {
	return DateOnly(t).Format("2006-01-02")
}

// TodayDate returns the current local calendar day encoded the way Things stores
// dates: midnight UTC of that day. Things writes every scheduled date as a bare
// calendar day at 00:00 UTC, so building one from the local clock keeps "today"
// meaning the user's day while staying on the wire format. Writing local midnight
// instead would land on the previous day in UTC and shift the task by a day.
func TodayDate() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// DateOnly strips a timestamp down to its calendar day at midnight UTC.
func DateOnly(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// BindingHelp renders a single binding as "key: description".
func BindingHelp(binding key.Binding) string {
	help := binding.Help()
	return help.Key + ": " + help.Desc
}

// MultiBindingHelp renders several bindings sharing one description as "k1/k2/k3: description".
func MultiBindingHelp(help string, bindings ...key.Binding) string {
	keys := make([]string, len(bindings))
	for i, b := range bindings {
		keys[i] = b.Help().Key
	}
	return strings.Join(keys, "/") + ": " + help
}
