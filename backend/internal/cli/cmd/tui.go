package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/codetasker/backend/internal/cli/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive CodeTasker terminal user interface",
	RunE:  runTUI,
}

func runTUI(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(tui.InitialModel(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}
	return nil
}
