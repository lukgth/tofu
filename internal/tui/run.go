package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
)

// runProgram runs a Bubble Tea program honoring the shared Options:
// --no-color forces a colorless profile for the session, and --timeout quits
// the program after the given duration (0 = never).
func runProgram(m tea.Model, o Options) (tea.Model, error) {
	var opts []tea.ProgramOption
	if o.NoColor {
		// Bubble Tea renders through its own renderer, so the profile must be
		// forced at construction; lipgloss's Writer.Profile has no effect here.
		opts = append(opts, tea.WithColorProfile(colorprofile.NoTTY))
	}
	p := tea.NewProgram(m, opts...)
	if o.Timeout > 0 {
		stop := time.AfterFunc(o.Timeout, p.Quit)
		defer stop.Stop()
	}
	return p.Run()
}
