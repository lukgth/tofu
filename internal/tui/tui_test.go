package tui

import (
	"strings"
	"testing"
)

func TestMenuItemsOrderAndLabels(t *testing.T) {
	want := []string{
		"New site",
		"New blog post",
		"Edit existing posts",
		"Build site",
		"Preview (serve)",
		"List posts",
		"Quit",
	}
	actions := menuActions()
	if len(actions) != len(want) {
		t.Fatalf("got %d actions, want %d", len(actions), len(want))
	}
	for i, a := range actions {
		if a.label != want[i] {
			t.Errorf("action %d = %q, want %q", i, a.label, want[i])
		}
	}
	// without a site, the items render the need-site suffix
	items := menuItems(t.TempDir())
	got := items[1].(menuItem).label
	if !strings.Contains(got, "(need site") {
		t.Errorf("no-site item = %q, want need-site suffix", got)
	}
	// quit never needs a site
	if items[6].(menuItem).label != "Quit" {
		t.Errorf("quit label = %q", items[6].(menuItem).label)
	}
}

func TestAnimatedTitleFinalState(t *testing.T) {
	got := AnimatedTitle(0, "hello")
	want := Title.Render("hello")
	if got != want {
		t.Errorf("AnimatedTitle(0) = %q, want %q", got, want)
	}
	got = AnimatedTitle(3, "hi")
	if !strings.HasPrefix(got, "   ") {
		t.Errorf("AnimatedTitle(3) should pad 3 spaces: %q", got)
	}
}

func TestNormalizeDefaults(t *testing.T) {
	o := normalize(Options{})
	if o.CharLimit != 400 || o.Width != 80 {
		t.Errorf("normalize defaults = %+v", o)
	}
	o = normalize(Options{CharLimit: 0, Width: 0})
	if o.CharLimit != 400 {
		t.Errorf("0 char limit should mean 400")
	}
}

func TestSlideSettles(t *testing.T) {
	s := NewSlide(8)
	sp := NewSlideSpring()
	steps := 0
	for s.Update(sp) && steps < 200 {
		steps++
	}
	if !s.Done || s.X != 0 {
		t.Errorf("slide did not settle: done=%v x=%f", s.Done, s.X)
	}
}
