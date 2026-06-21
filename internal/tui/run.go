package tui

import tea "github.com/charmbracelet/bubbletea"

import "thinx/internal/domain"

// Run starts the Bubble Tea terminal interface.
func Run(repo domain.TodoRepository) error {
	_, err := tea.NewProgram(NewModel(repo), tea.WithAltScreen()).Run()
	return err
}
