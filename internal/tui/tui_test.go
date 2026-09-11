package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
)

// enterKey builds a real enter press for driving wizard Update directly.
func enterKey() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEnter}
}

// TestWizardEngineWalk drives the real wizardModel.Update through a site-like
// wizard: input steps, a body step, review, and finish. This is the flow that
// silently died before (unfocused inputs, missing stepBody case).
func TestWizardEngineWalk(t *testing.T) {
	ti := newTextInput("", "My Site", normalize(Options{}))
	area := textarea.New()
	area.SetStyles(textareaStyles())
	steps := []step{
		{kind: stepInput, label: "title", input: &ti, setter: func(v string) error {
			if v == "" {
				return fmt.Errorf("a title is required")
			}
			title := v
			_ = title
			return nil
		}},
		{kind: stepBody, label: "body", area: &area, setter: func(v string) error {
			return nil
		}},
	}
	w := &wizardModel{
		opts:  normalize(Options{}),
		title: "test",
		steps: steps,
		finish: func(w *wizardModel) error {
			return nil
		},
	}
	w.slide = NewSlide(8)
	w.screen = "prompt"
	w.focusStep()

	// enter on title (has default value) -> advances to body step
	m, _ := w.Update(enterKey())
	w = m.(*wizardModel)
	if w.current() != &w.steps[1] {
		t.Fatalf("expected to advance to body step, screen=%q stepIdx=%d", w.screen, w.stepIdx)
	}
	// enter on body -> review
	m, _ = w.Update(enterKey())
	w = m.(*wizardModel)
	if w.screen != "review" {
		t.Fatalf("expected review after body step, screen=%q", w.screen)
	}
	// enter on review (Yes default) starts finish; pump the result to done
	m, _ = w.Update(enterKey())
	w = m.(*wizardModel)
	m, _ = w.Update(wizardQuitMsg{})
	if w.screen != "done" {
		t.Fatalf("expected done screen, screen=%q", w.screen)
	}
}

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

func TestRunInitRefusesExistingSite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tofu.toml"), []byte("title = 'x'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := RunInit(dir, Options{})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("want already-exists refusal, got %v", err)
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
