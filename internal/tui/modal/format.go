package modal

import (
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"thinx/internal/domain"
	"thinx/internal/tui/uihelp"
)

// dateInputView renders a date input, styling the text red when the value is not a valid date.
func dateInputView(input textinput.Model) string {
	val := input.Value()
	if val != "" {
		if _, err := time.Parse("2006-01-02", val); err != nil {
			styles := input.Styles()
			styles.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
			input.SetStyles(styles)
		}
	}
	return input.View()
}

// scheduledValue returns the styled display value for the Scheduled field.
func scheduledValue(schedule domain.TodoSchedule, scheduledAt *time.Time) string {
	switch schedule {
	case domain.TodoScheduleInbox:
		return inboxStyle.Render("Inbox")
	case domain.TodoScheduleSomeday:
		if scheduledAt != nil {
			return uihelp.FormatDate(*scheduledAt)
		}
		return rainbowText("Someday")
	case domain.TodoScheduleAnytime:
		if scheduledAt == nil {
			return anytimeStyle.Render("Anytime")
		}
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		scheduled := time.Date(scheduledAt.Year(), scheduledAt.Month(), scheduledAt.Day(), 0, 0, 0, 0, time.Local)
		if scheduled.Before(today) {
			scheduled = today
		}
		if scheduled.Equal(today) {
			return todayStyle.Render("Today")
		}
		return uihelp.FormatDate(scheduled)
	default:
		return errorStyle.Render("Unknown schedule")
	}
}

func orDash(s string) string {
	if s == "" {
		return legendStyle.Render("—")
	}
	return s
}

func dateOrDash(t *time.Time) string {
	if t == nil {
		return legendStyle.Render("—")
	}
	return uihelp.FormatDate(*t)
}
