package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
)

// runProgram runs a Bubble Tea program honoring the shared Options:
// --no-color forces a colorless profile for the session, and --timeout quits
// the program after the given duration (0 = never). Callers that only need
// the update loop (tests) can pass extra program options; a real screen runs
// with the renderer.
func runProgram(m tea.Model, o Options, programOpts ...tea.ProgramOption) (tea.Model, error) {
	var opts []tea.ProgramOption
	if o.NoColor {
		// Bubble Tea renders through its own renderer, so the profile must be
		// forced at construction; lipgloss's Writer.Profile has no effect here.
		opts = append(opts, tea.WithColorProfile(colorprofile.NoTTY))
	}
	opts = append(opts, programOpts...)
	p := tea.NewProgram(m, opts...)
	if o.Timeout > 0 {
		stop := time.AfterFunc(o.Timeout, p.Quit)
		defer stop.Stop()
	}
	// However the program ended — quit, ctrl+c, or the --timeout abort that
	// kills it mid-suspend — whatever it still holds is released now.
	// Neither release has another path: an aborted program delivers no final
	// message, so the round trip that normally deletes a scratch file and
	// the keypress that normally stops the preview server never happen.
	defer finishProgram(m)
	return p.Run()
}

// finishProgram releases the program's still-held resources: a menu preview
// server runs past the menu by design (Quit and ctrl+c stop it explicitly),
// but a program that died another way must not leave the port bound.
func finishProgram(m tea.Model) {
	removeTempFiles()
	if menu, ok := m.(MenuModel); ok {
		menu.stopServe()
	}
}
