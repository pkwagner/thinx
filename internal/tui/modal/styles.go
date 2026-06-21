package modal

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(1, 2)

	titleStyle = lipgloss.NewStyle().Bold(true)

	keyHintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	completedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	canceledStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	inboxStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#4a9ef4"))
	anytimeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#008b8b"))
	todayStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#d8a100"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	labelStyle = lipgloss.NewStyle().
			Faint(true).
			Width(labelWidth)

	legendStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

var rainbowHex = []string{
	"#ff4444", // red
	"#ff8c00", // orange
	"#ffd700", // yellow
	"#44cc44", // green
	"#4499ff", // blue
	"#7744cc", // indigo
	"#cc44ff", // violet
}

// rainbowText renders each rune of s in the next rainbow color.
func rainbowText(s string) string {
	var sb strings.Builder
	for i, r := range []rune(s) {
		c := lipgloss.Color(rainbowHex[i%len(rainbowHex)])
		sb.WriteString(lipgloss.NewStyle().Foreground(c).Render(string(r)))
	}
	return sb.String()
}
