package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/pkwagner/thinx/internal/domain"
)

// Run starts the Bubble Tea terminal interface.
func Run(repo domain.TodoRepository) error {
	_, err := tea.NewProgram(NewModel(repo)).Run()
	return err
}
