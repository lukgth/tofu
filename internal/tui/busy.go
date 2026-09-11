package tui

import (
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// spinnerModel wraps the bubbles spinner for the kit.
type spinnerModel struct {
	m spinner.Model
}

// Message aliases so screens can switch without importing bubbles types.
type (
	spinnerTickMsg   = spinner.TickMsg
	progressFrameMsg = progress.FrameMsg
)

func newSpinnerModel() spinnerModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = Accent
	return spinnerModel{m: sp}
}

func (s spinnerModel) tick() func() tea.Msg { return s.m.Tick }

func (s *spinnerModel) update(msg tea.Msg) tea.Cmd {
	m, cmd := s.m.Update(msg)
	s.m = m
	return cmd
}

func (s spinnerModel) view() string { return s.m.View() }

// progressModel wraps the bubbles progress bar (default blend, springy).
type progressModel struct {
	m progress.Model
}

func newProgressModel(width int) progressModel {
	return progressModel{m: progress.New(
		progress.WithWidth(width),
		progress.WithDefaultBlend(),
		progress.WithSpringOptions(12, 0.8),
	)}
}

func (p progressModel) setPercent(v float64) tea.Cmd { return p.m.SetPercent(v) }

func (p *progressModel) updateFrame(msg progressFrameMsg) tea.Cmd {
	m, cmd := p.m.Update(msg)
	p.m = m
	return cmd
}

func (p progressModel) view() string { return p.m.View() }
