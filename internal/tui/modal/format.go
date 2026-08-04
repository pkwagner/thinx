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

// todayDate and dateOnly re-export the shared date convention (see uihelp) so the
// modal's comparisons and the dates it writes stay on Things' wire format.
func todayDate() time.Time           { return uihelp.TodayDate() }
func dateOnly(t time.Time) time.Time { return uihelp.DateOnly(t) }

// scheduleForDate mirrors how Things files a task by its date: a day that has not
// arrived defers the task (Someday, shown under Upcoming), while today or an
// earlier day makes it started (Anytime, shown under Today until it is done).
func scheduleForDate(t time.Time) domain.TodoSchedule {
	if dateOnly(t).After(todayDate()) {
		return domain.TodoScheduleSomeday
	}
	return domain.TodoScheduleAnytime
}

// effectiveScheduled returns the date the Scheduled field stands for. A started
// task whose day has passed carries over into Today, so it reads as today rather
// than as its original day — and editing it has to start from that same date,
// otherwise "+1 day" would move the task into the past.
func effectiveScheduled(schedule domain.TodoSchedule, scheduledAt *time.Time) *time.Time {
	if scheduledAt == nil || schedule != domain.TodoScheduleAnytime {
		return scheduledAt
	}
	today := todayDate()
	scheduled := dateOnly(*scheduledAt)
	if scheduled.Before(today) {
		return &today
	}
	return &scheduled
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
		scheduled := *effectiveScheduled(schedule, scheduledAt)
		if scheduled.Equal(todayDate()) {
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
