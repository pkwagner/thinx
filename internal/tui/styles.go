package tui

import "charm.land/lipgloss/v2"

var (
	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				Padding(0, 2).
				Margin(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 2).
			Margin(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			PaddingLeft(1)

	statusStyle = lipgloss.NewStyle().
			PaddingLeft(3)

	todoStyle = lipgloss.NewStyle().PaddingLeft(1)

	selectedTodoStyle = todoStyle.Bold(true)

	todoDetailsStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	selectedTodoDetailsStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	legendStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)
