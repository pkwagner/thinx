package onboarding

import "charm.land/lipgloss/v2"

var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1, 2)
	titleStyle    = lipgloss.NewStyle().Bold(true).PaddingBottom(1)
	labelStyle    = lipgloss.NewStyle().Faint(true).Width(labelWidth)
	providerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#4a9ef4"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#d8a100"))
	legendStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)
