package tui

import (
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/harmonica"

	"charm.land/bubbletea/v2"
)

// NewSlideSpring is the shared slide spring (cute but fast).
func NewSlideSpring() harmonica.Spring {
	return harmonica.NewSpring(harmonica.FPS(60), 7.0, 0.3)
}

// NewBounceSpring is the success-bounce spring (one visible overshoot).
func NewBounceSpring() harmonica.Spring {
	return harmonica.NewSpring(harmonica.FPS(60), 7.0, 0.15)
}

// frameMsg drives slide animations at 60fps.
type frameMsg struct{ t time.Time }

// FrameTick fires a frameMsg every 1/60s while a model animates.
func FrameTick() tea.Cmd {
	return tea.Tick(time.Second/60, func(t time.Time) tea.Msg {
		return frameMsg{t: t}
	})
}

// Slide is the shared per-model slide state.
type Slide struct {
	X, V float64
	Done bool
}

// NewSlide starts a slide from offset to 0.
func NewSlide(offset float64) Slide {
	return Slide{X: offset}
}

// Update advances the slide; returns true when the tick should continue.
func (s *Slide) Update(sp spring) bool {
	if s.Done {
		return false
	}
	s.X, s.V = sp.Update(s.X, s.V, 0)
	if abs(s.X) < 0.1 && abs(s.V) < 0.1 {
		s.X = 0
		s.V = 0
		s.Done = true
		return false
	}
	return true
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// SlideCmd starts the 60fps tick loop if the slide isn't done.
func SlideCmd(s Slide, noAnimations bool) tea.Cmd {
	if s.Done || noAnimations {
		return nil
	}
	return FrameTick()
}

// AnimatedTitle renders the title offset by x cells to the right.
func AnimatedTitle(x float64, text string) string {
	pad := int(math.Round(math.Max(0, x)))
	return strings.Repeat(" ", pad) + Title.Render(text)
}
