package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// runProgram runs a Bubble Tea program honoring the shared Options:
// --no-color forces a colorless profile for the session, and --timeout quits
// the program after the given duration (0 = never).
func runProgram(p *tea.Program, o Options) (tea.Model, error) {
	if o.NoColor {
		lipgloss.Writer.Profile = colorprofile.NoTTY
	}
	if o.Timeout > 0 {
		stop := time.AfterFunc(o.Timeout, p.Quit)
		defer stop.Stop()
	}
	return p.Run()
}
