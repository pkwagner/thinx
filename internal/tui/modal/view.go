package modal

import (
	"strings"

	"charm.land/lipgloss/v2"
	"thinx/internal/domain"
	"thinx/internal/tui/uihelp"
)

// Overlay renders the modal centered over base.
func (m *Model) Overlay(base string) string {
	box := m.view()
	x := (m.width - lipgloss.Width(box)) / 2
	y := (m.height - lipgloss.Height(box)) / 2

	canvas := lipgloss.NewCanvas(m.width, m.height)
	canvas.Compose(lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(box).X(x).Y(y).Z(1),
	))
	return canvas.Render()
}

// view renders the bordered modal box.
func (m *Model) view() string {
	return style.Width(m.outerWidth()).Render(lipgloss.JoinVertical(
		lipgloss.Left,
		m.header(),
		"",
		m.fields(),
		"",
		legendStyle.Render(m.legend()),
	))
}

// header renders the title row.
func (m *Model) header() string {
	if m.editing == fieldTitle {
		return m.titleInput.View()
	}
	hint := keyHintStyle.Render(m.keys.editTitle.Help().Key)
	title := titleStyle.Render(m.todo.Title)
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.PlaceHorizontal(m.contentWidth()-lipgloss.Width(hint), lipgloss.Left, title),
		hint,
	)
}

// fields renders the labelled task details, varying by status.
func (m *Model) fields() string {
	rows := []string{
		m.row("Project", orDash(m.todo.Project), ""),
	}

	switch m.todo.Status {
	case domain.TodoStatusCompleted:
		date := dateOrDash(m.todo.CheckedAt)
		if m.todo.CheckedAt != nil {
			date = completedStyle.Render(date)
		}
		rows = append(rows, m.row("Completed", date, ""))
	case domain.TodoStatusCanceled:
		date := dateOrDash(m.todo.CheckedAt)
		if m.todo.CheckedAt != nil {
			date = canceledStyle.Render(date)
		}
		rows = append(rows, m.row("Canceled", date, ""))
	default:
		var scheduledVal string
		if m.editing == fieldScheduled {
			scheduledVal = dateInputView(m.scheduledInput)
		} else {
			scheduledVal = scheduledValue(m.todo.Schedule, m.todo.ScheduledAt)
		}
		var deadlineVal string
		if m.editing == fieldDeadline {
			deadlineVal = dateInputView(m.deadlineInput)
		} else {
			deadlineVal = dateOrDash(m.todo.DeadlineAt)
		}
		rows = append(rows,
			m.row("Scheduled", scheduledVal, m.scheduledHint()),
			m.row("Deadline", deadlineVal, m.deadlineHint()),
		)
	}

	rows = append(rows, m.noteSection(), m.checklistSection())

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// row renders a label–value pair with an optional right-aligned key hint.
func (m *Model) row(label, value, hint string) string {
	labelStr := labelStyle.Render(label)
	if hint == "" {
		return lipgloss.JoinHorizontal(lipgloss.Top, labelStr, value)
	}
	hintStr := keyHintStyle.Render(hint)
	valueWidth := m.contentWidth() - labelWidth - lipgloss.Width(hintStr)
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		labelStr,
		lipgloss.PlaceHorizontal(valueWidth, lipgloss.Left, value),
		hintStr,
	)
}

// noteSection renders the Notes label and content (or textarea when editing).
// The label is left-aligned with the first line; subsequent lines are indented.
func (m *Model) noteSection() string {
	labelStr := labelStyle.Render("Notes")
	hint := keyHintStyle.Render(m.keys.editNote.Help().Key)

	if m.editing == fieldNote {
		return lipgloss.JoinHorizontal(lipgloss.Top, labelStr, m.noteInput.View())
	}

	valueWidth := m.contentWidth() - labelWidth - lipgloss.Width(hint)
	indent := strings.Repeat(" ", labelWidth)

	var lines []string
	if m.todo.Note == "" {
		lines = []string{legendStyle.Render("—")}
	} else {
		text := strings.TrimRight(strings.ReplaceAll(m.todo.Note, "\r\n", "\n"), "\n")
		lines = strings.Split(lipgloss.Wrap(text, valueWidth, ""), "\n")
	}

	var sb strings.Builder
	for i, line := range lines {
		if i == 0 {
			sb.WriteString(lipgloss.JoinHorizontal(
				lipgloss.Top,
				labelStr,
				lipgloss.PlaceHorizontal(valueWidth, lipgloss.Left, line),
				hint,
			))
			continue
		}
		sb.WriteByte('\n')
		sb.WriteString(indent)
		sb.WriteString(line)
	}
	return sb.String()
}

// checklistSection renders checklist items below a "Checklist" label.
// Completed items are prefixed with ✓ in green; open items with —.
// Always rendered; shows — when the checklist is empty.
func (m *Model) checklistSection() string {
	labelStr := labelStyle.Render("Checklist")
	indent := strings.Repeat(" ", labelWidth)

	if len(m.todo.Checklist) == 0 {
		return lipgloss.JoinHorizontal(lipgloss.Top, labelStr, legendStyle.Render("—"))
	}

	var sb strings.Builder
	for i, item := range m.todo.Checklist {
		if i > 0 {
			sb.WriteByte('\n')
			sb.WriteString(indent)
		}
		var line string
		if item.Completed {
			line = completedStyle.Render("✓ " + item.Title)
		} else {
			line = "— " + item.Title
		}
		if i == 0 {
			sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, labelStr, line))
		} else {
			sb.WriteString(line)
		}
	}
	return sb.String()
}

// legend returns the key hint line at the bottom of the modal.
func (m *Model) legend() string {
	if m.editing != fieldNone {
		return strings.Join([]string{uihelp.BindingHelp(m.keys.save), uihelp.BindingHelp(m.keys.cancel)}, "; ")
	}
	if m.todo.Status == domain.TodoStatusOpen {
		return strings.Join([]string{
			uihelp.MultiBindingHelp("edit field", m.keys.editTitle, m.keys.editScheduled, m.keys.editDeadline, m.keys.editNote),
			uihelp.MultiBindingHelp("schedule", m.keys.scheduleToday, m.keys.scheduleInbox, m.keys.scheduleSomeday),
			uihelp.BindingHelp(m.keys.close),
		}, "; ")
	}
	return strings.Join([]string{
		uihelp.MultiBindingHelp("edit field", m.keys.editTitle, m.keys.editNote),
		uihelp.BindingHelp(m.keys.close),
	}, "; ")
}

func (m *Model) scheduledHint() string {
	return strings.Join([]string{
		m.keys.editScheduled.Help().Key,
		m.keys.scheduleToday.Help().Key,
		m.keys.scheduleInbox.Help().Key,
		m.keys.scheduleSomeday.Help().Key,
	}, "/")
}

func (m *Model) deadlineHint() string {
	return m.keys.editDeadline.Help().Key
}

// outerWidth is the total width of the bordered box.
func (m *Model) outerWidth() int {
	return min(m.width-2*margin, maxWidth)
}

// contentWidth is the text width available inside the border and padding.
func (m *Model) contentWidth() int {
	return m.outerWidth() - style.GetHorizontalBorderSize() - style.GetHorizontalPadding()
}

// dateInputWidth computes SetWidth for a date input in a row with the given hint.
func (m *Model) dateInputWidth(hint string) int {
	hintWidth := lipgloss.Width(keyHintStyle.Render(hint))
	return m.contentWidth() - labelWidth - hintWidth - 1
}
