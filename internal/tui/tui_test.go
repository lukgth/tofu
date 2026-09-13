package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
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

func TestChoiceCursorNavigation(t *testing.T) {
	c := newChoice([]string{"Yes", "No"}, true, 80)
	if c.selected() != "Yes" {
		t.Fatalf("default selection = %q", c.selected())
	}
	c = c.update(keyPress("down"))
	if c.selected() != "No" {
		t.Fatalf("after down = %q", c.selected())
	}
	c = c.update(keyPress("down"))
	if c.selected() != "No" {
		t.Fatalf("down past end should clamp, got %q", c.selected())
	}
	c = c.update(keyPress("up"))
	c = c.update(keyPress("up"))
	if c.selected() != "Yes" {
		t.Fatalf("after up up = %q", c.selected())
	}
	// every option is rendered - nothing hidden behind pagination
	v := c.view(true)
	if !strings.Contains(v, "Yes") || !strings.Contains(v, "No") {
		t.Errorf("view must show all options, got %q", v)
	}
	if strings.Contains(v, "••") {
		t.Errorf("view must not render pagination dots: %q", v)
	}
}

func keyPress(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	}
	return tea.KeyPressMsg{Code: ' ', Text: s}
}

func TestBodyStepRendersAndAdvances(t *testing.T) {
	area := textarea.New()
	area.SetStyles(textareaStyles())
	area.SetValue("hello body")
	steps := []step{
		{kind: stepBody, label: "body", area: &area},
	}
	w := &wizardModel{
		opts:  normalize(Options{}),
		title: "t",
		steps: steps,
	}
	w.screen = "prompt"
	w.focusStep()

	v := w.View()
	// the body textarea must be visible, not rendered as empty string
	if !strings.Contains(v.Content, "hello body") {
		t.Errorf("body step view hides textarea content")
	}
	m, _ := w.Update(keyPress("enter"))
	w = m.(*wizardModel)
	if w.screen != "review" {
		t.Fatalf("enter on body step should reach review, screen=%q", w.screen)
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

// A post whose frontmatter slug differs from its filename must resolve to the
// file that was actually parsed, not content/posts/<slug>.md.
func TestPostPathForSlugUsesRealPath(t *testing.T) {
	root := t.TempDir()
	posts := filepath.Join(root, "content", "posts")
	if err := os.MkdirAll(posts, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(posts, "file-name.md"),
		[]byte("---\ntitle: T\ndate: 2026-01-01\nslug: custom-name\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := postPathForSlug(root, "custom-name")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(posts, "file-name.md"); got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if _, err := os.Stat(got); err != nil {
		t.Errorf("returned path does not exist: %v", err)
	}
}

// The edit wizard prefills the date step from frontmatter, which ParseFile
// accepts as RFC 3339; submitting that prefilled value must not error.
func TestEditWizardDateAcceptsPrefilledRFC3339(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.md")
	if err := os.WriteFile(path, []byte("---\ntitle: T\ndate: 2026-01-01T10:00:00Z\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := newEditWizard(dir, path, normalize(Options{}), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range w.steps {
		if s.label != "date" {
			continue
		}
		if err := s.setter(s.input.Value()); err != nil {
			t.Fatalf("prefilled RFC 3339 date rejected: %v", err)
		}
		return
	}
	t.Fatal("no date step found")
}

func TestAddWizardFlagsBindsOptions(t *testing.T) {
	o := &Options{}
	cmd := &cobra.Command{Use: "x", RunE: func(*cobra.Command, []string) error { return nil }}
	AddWizardFlags(cmd, o)
	cmd.SetArgs([]string{"--no-color", "--timeout", "5s"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !o.NoColor || o.Timeout != 5*time.Second {
		t.Errorf("flags not bound: %+v", o)
	}
}

// Flags must be registered when the command is constructed, not from RunE, or
// cobra rejects them as unknown.
func TestWizardCommandsExposeFlags(t *testing.T) {
	for _, c := range WizardCommands() {
		if c.Flags().Lookup("no-color") == nil || c.Flags().Lookup("timeout") == nil {
			t.Errorf("%s does not expose the wizard flags", c.Name())
		}
	}
}

// RunEdit must use the slug it was given: an unknown slug errors before any
// TUI starts, rather than silently falling through to the picker.
func TestRunEditHonorsSlugArgument(t *testing.T) {
	root := t.TempDir()
	posts := filepath.Join(root, "content", "posts")
	if err := os.MkdirAll(posts, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(posts, "p.md"),
		[]byte("---\ntitle: T\ndate: 2026-01-01\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RunEdit(root, "no-such-slug", normalize(Options{})); err == nil {
		t.Fatal("RunEdit with an unknown slug should error, not open the picker")
	}
}
