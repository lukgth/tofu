package tui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
)

func truePath(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("true")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func writeSelectedEditor(t *testing.T, home, value string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, ".selected_editor"), []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSelectedEditorPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeSelectedEditor(t, home, "# comment\nSELECTED_EDITOR=\"/usr/bin/vim.tiny\"\nSELECTED_EDITOR=\"/bin/ed\"\n")
	if got := selectedEditorPath(); got != "/bin/ed" {
		t.Fatalf("selectedEditorPath() = %q, want last assignment", got)
	}
	writeSelectedEditor(t, home, "garbage\nSELECTED_EDITOR='/bin/ed'\n")
	if got := selectedEditorPath(); got != "" {
		t.Fatalf("single-quoted assignment = %q, want empty", got)
	}
	if err := os.Remove(filepath.Join(home, ".selected_editor")); err != nil {
		t.Fatal(err)
	}
	if got := selectedEditorPath(); got != "" {
		t.Fatalf("missing file = %q, want empty", got)
	}
}

func TestResolveEditorPrecedence(t *testing.T) {
	trueBin := truePath(t)
	home := t.TempDir()
	t.Setenv("HOME", filepath.Join(t.TempDir(), "poisoned"))
	t.Setenv("EDITOR", trueBin)
	sel, err := resolveEditor("file")
	if err != nil || sel.needSelection || sel.cmd == nil || sel.cmd.Path != trueBin {
		t.Fatalf("EDITOR selection = %#v, err %v", sel, err)
	}
	t.Setenv("EDITOR", "")
	writeSelectedEditor(t, home, "SELECTED_EDITOR=\""+trueBin+"\"\n")
	t.Setenv("HOME", home)
	sel, err = resolveEditor("file")
	if err != nil || sel.needSelection || sel.cmd == nil || sel.cmd.Path != trueBin {
		t.Fatalf("selected editor = %#v, err %v", sel, err)
	}
	t.Setenv("EDITOR", "nonexistent-binary-xyz")
	t.Setenv("PATH", filepath.Dir(trueBin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	sel, err = resolveEditor("file")
	if err != nil || sel.needSelection || sel.cmd == nil || sel.cmd.Path != trueBin {
		t.Fatalf("fallback selection = %#v, err %v", sel, err)
	}
	t.Setenv("EDITOR", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	sel, err = resolveEditor("file")
	if err == nil || !strings.Contains(err.Error(), "EDITOR") || sel.needSelection {
		t.Fatalf("unavailable selection = %#v, err %v", sel, err)
	}
	t.Setenv("PATH", filepath.Dir(trueBin)+string(os.PathListSeparator)+"/usr/bin")
	sel, err = resolveEditor("file")
	if err != nil || !sel.needSelection {
		t.Fatalf("select-editor selection = %#v, err %v", sel, err)
	}
}

func TestEditorDisplayName(t *testing.T) {
	trueBin := truePath(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EDITOR", trueBin)
	if got := editorDisplayName(); got != trueBin {
		t.Fatalf("EDITOR display = %q", got)
	}
	t.Setenv("EDITOR", "")
	writeSelectedEditor(t, home, "SELECTED_EDITOR=\""+trueBin+"\"\n")
	if got := editorDisplayName(); got != trueBin {
		t.Fatalf("selected display = %q", got)
	}
	os.Remove(filepath.Join(home, ".selected_editor"))
	t.Setenv("PATH", filepath.Dir(trueBin)+string(os.PathListSeparator)+"/usr/bin")
	if got := editorDisplayName(); got != "select-editor will ask" {
		t.Fatalf("select-editor display = %q", got)
	}
	t.Setenv("PATH", t.TempDir())
	if got := editorDisplayName(); got != "" {
		t.Fatalf("unavailable display = %q", got)
	}
}

// ctrlOKey builds a real ctrl+o press for driving wizard Update directly.
func ctrlOKey() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl}
}

func newBodyWizard(t *testing.T, body string) *wizardModel {
	t.Helper()
	area := textarea.New()
	area.SetStyles(textareaStyles())
	area.SetValue(body)
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
	return w
}

func TestCtrlOStartsExternalEdit(t *testing.T) {
	w := newBodyWizard(t, "draft body")
	m, cmd := w.Update(ctrlOKey())
	w = m.(*wizardModel)
	if cmd == nil {
		t.Fatalf("ctrl+o on body step should return a suspend cmd")
	}
	if w.screen != "prompt" || w.stepIdx != 0 {
		t.Fatalf("ctrl+o must not advance state, screen=%q stepIdx=%d", w.screen, w.stepIdx)
	}
	if w.current().area.Value() != "draft body" {
		t.Fatalf("textarea value changed by ctrl+o: %q", w.current().area.Value())
	}
}

func TestEditorDoneAppliesBody(t *testing.T) {
	w := newBodyWizard(t, "draft body")
	tmp := filepath.Join(t.TempDir(), "edited.md")
	if err := os.WriteFile(tmp, []byte("edited body"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, _ := w.Update(editorDoneMsg{tmp: tmp})
	w = m.(*wizardModel)
	if got := w.current().area.Value(); got != "edited body" {
		t.Fatalf("area value = %q, want edited body", got)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("temp file not removed: %v", err)
	}
	if w.screen != "prompt" || w.stepIdx != 0 {
		t.Fatalf("state should stay on body step, screen=%q stepIdx=%d", w.screen, w.stepIdx)
	}
	if w.errorMsg != "" {
		t.Fatalf("errorMsg = %q, want empty", w.errorMsg)
	}
}

func TestEditorDoneErrorKeepsState(t *testing.T) {
	w := newBodyWizard(t, "draft body")
	tmp := filepath.Join(t.TempDir(), "missing.md")
	m, _ := w.Update(editorDoneMsg{tmp: tmp, runErr: errors.New("boom")})
	w = m.(*wizardModel)
	if !strings.Contains(w.errorMsg, "boom") {
		t.Fatalf("errorMsg = %q, want it to mention boom", w.errorMsg)
	}
	if w.current().area.Value() != "draft body" {
		t.Fatalf("textarea changed on error: %q", w.current().area.Value())
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("temp file not removed: %v", err)
	}
}

func TestOpenInEditorWithoutEditorEnv(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	msg := openInEditor("x")()
	if msg == nil {
		t.Fatal("selection command returned nil message")
	}
	if done, ok := msg.(editorDoneMsg); ok {
		if done.runErr != nil {
			t.Fatalf("runErr = %v, want nil after selection", done.runErr)
		}
		if done.tmp == "" {
			t.Fatal("tmp should be populated so Update can clean it up")
		}
		os.Remove(done.tmp)
	}
}

func TestOpenInEditorUnavailable(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	done, ok := openInEditor("x")().(editorDoneMsg)
	if !ok || done.runErr == nil || !strings.Contains(done.runErr.Error(), "EDITOR") {
		t.Fatalf("result = %#v, want EDITOR error", done)
	}
	os.Remove(done.tmp)
}
func TestEditWizardEditorChoiceOpensEditor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.md")
	src := "---\ntitle: T\ndate: 2026-01-01\n---\nbody\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := newEditWizard("example", path, normalize(Options{}), false)
	if err != nil {
		t.Fatalf("newEditWizard: %v", err)
	}
	w.screen = "prompt"
	w.focusStep()
	w.stepIdx = 4
	w.focusStep()

	// 1) Choosing "Open $EDITOR" skips the textarea and lands on review;
	// the summary shows the editor intent.
	w.steps[4].choice.cursor = 1
	m, _ := w.Update(enterKey())
	w = m.(*wizardModel)
	if w.screen != "review" {
		t.Fatalf("screen = %q, want review after Open $EDITOR choice", w.screen)
	}
	if got := w.summary(w); !strings.Contains(got, "editor: true") || !strings.Contains(got, "opens after confirm") {
		t.Fatalf("summary = %q, want editor line", got)
	}

	// 2) With EDITOR set, doFinish returns the suspend cmd. The editor leg
	// must not run the quick-edit finish (which would SetBody("") and wipe
	// the file body).
	t.Setenv("EDITOR", "true")
	cmd := w.doFinish()
	if cmd == nil {
		t.Fatalf("doFinish returned nil cmd with EDITOR set; editor would not open")
	}
	msg := cmd()
	// execMsg is unexported in bubbletea v2; discriminate behaviorally: the
	// editor leg returns an exec message (anything but a quit), while every
	// failure path returns wizardQuitMsg.
	if _, isQuit := msg.(wizardQuitMsg); isQuit {
		t.Fatalf("cmd produced %T; editor suspension did not start", msg)
	}
	if raw, _ := os.ReadFile(path); !strings.HasSuffix(string(raw), "body\n") {
		t.Fatalf("editor path modified the body: %q", raw)
	}

	// 3) Without EDITOR and with no selected_editor, the wizard falls back
	// to select-editor: doFinish suspends into it (exec message, not a
	// quit). Simulate total unavailability by hiding select-editor via PATH.
	t.Setenv("EDITOR", "")
	t.Setenv("PATH", t.TempDir())
	cmd = w.doFinish()
	if cmd == nil {
		t.Fatalf("doFinish returned nil cmd with EDITOR unset")
	}
	quit, ok := cmd().(wizardQuitMsg)
	if !ok {
		t.Fatalf("expected wizardQuitMsg, got %T", msg)
	}
	if quit.err == nil || !strings.Contains(quit.err.Error(), "EDITOR") {
		t.Fatalf("err = %v, want EDITOR error", quit.err)
	}
}

// TestEditWizardQuickEditPreservesBody covers the textarea leg: confirm the
// body step, then the quick-edit finish must rewrite the body it collected
// instead of wiping it.
func TestEditWizardQuickEditPreservesBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.md")
	src := "---\ntitle: T\ndate: 2026-01-01\n---\nbody\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	w, err := newEditWizard("example", path, normalize(Options{}), false)
	if err != nil {
		t.Fatalf("newEditWizard: %v", err)
	}
	w.screen = "prompt"
	w.focusStep()

	// Quick edit keeps cursor 0 -> textarea step (idx 4) -> enter collects
	// the textarea value and lands on review.
	w.stepIdx = 4 // body choice step
	w.focusStep()
	m, _ := w.Update(enterKey())
	w = m.(*wizardModel)
	if w.stepIdx != 5 {
		t.Fatalf("stepIdx = %d, want 5 on textarea step after Quick edit choice", w.stepIdx)
	}
	m, _ = w.Update(enterKey())
	w = m.(*wizardModel)
	if w.screen != "review" {
		t.Fatalf("screen = %q, want review after body confirm", w.screen)
	}

	// The quick-edit finish runs SetBody with the collected textarea value.
	cmd := w.doFinish()
	quit, ok := cmd().(wizardQuitMsg)
	if !ok {
		t.Fatalf("expected wizardQuitMsg, got %T", cmd())
	}
	if quit.err != nil {
		t.Fatalf("quick-edit finish errored: %v", quit.err)
	}
	// The textarea was seeded with the parsed body ("body"); the finish must
	// write that value back. Before the fix the editor path could wipe it
	// with post.SetBody(path, "").
	if raw, _ := os.ReadFile(path); !strings.HasSuffix(string(raw), "body") || strings.HasSuffix(string(raw), "---\n") {
		t.Fatalf("body = %q, want textarea value preserved", raw)
	}
}

func TestSessionEditorOverride(t *testing.T) {
	defer clearSessionEditor()
	setSessionEditor("/bin/true")
	t.Setenv("EDITOR", "")
	sel, err := resolveEditor("f")
	if err != nil || sel.needSelection || sel.cmd == nil || sel.cmd.Path != "/bin/true" || sel.fromSource != "select-editor" {
		t.Fatalf("override selection = %#v, err %v", sel, err)
	}
	if got := editorDisplayName(); got != "/bin/true" {
		t.Fatalf("override display = %q", got)
	}
	t.Setenv("EDITOR", "nonexistent-editor-for-test")
	sel, err = resolveEditor("f")
	if err != nil || sel.needSelection || sel.cmd == nil || sel.cmd.Path != "/bin/true" || sel.fromSource != "select-editor" {
		t.Fatalf("override should beat unusable EDITOR: %#v, err %v", sel, err)
	}
}

func TestFromSourceAnnotation(t *testing.T) {
	defer clearSessionEditor()
	trueBin := truePath(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EDITOR", trueBin)
	sel, err := resolveEditor("f")
	if err != nil || sel.fromSource != "$EDITOR" {
		t.Fatalf("EDITOR source = %#v, err %v", sel, err)
	}
	t.Setenv("EDITOR", "")
	writeSelectedEditor(t, home, "SELECTED_EDITOR=\""+trueBin+"\"\n")
	sel, err = resolveEditor("f")
	if err != nil || sel.fromSource != "select-editor" {
		t.Fatalf("selected_editor source = %#v, err %v", sel, err)
	}
	os.Remove(filepath.Join(home, ".selected_editor"))
	selectDir := t.TempDir()
	selectEditor := filepath.Join(selectDir, "select-editor")
	if err := os.WriteFile(selectEditor, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", selectDir)
	sel, err = resolveEditor("f")
	if err != nil || !sel.needSelection || sel.fromSource != "" {
		t.Fatalf("need-selection source = %#v, err %v", sel, err)
	}
}

func newEditorChoiceTestWizard(t *testing.T) *wizardModel {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.md")
	if err := os.WriteFile(path, []byte("---\ntitle: T\ndate: 2026-01-01\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := newEditWizard("example", path, normalize(Options{}), false)
	if err != nil {
		t.Fatal(err)
	}
	w.screen = "prompt"
	w.focusStep()
	w.stepIdx = 4
	w.focusStep()
	w.steps[4].choice.cursor = 1
	m, _ := w.Update(enterKey())
	w = m.(*wizardModel)
	if w.screen != "review" {
		t.Fatalf("screen = %q, want review", w.screen)
	}
	return w
}

func TestReviewRepickKey(t *testing.T) {
	defer clearSessionEditor()
	t.Setenv("EDITOR", "")
	w := newEditorChoiceTestWizard(t)
	w.repickEditor = func() tea.Cmd {
		setSessionEditor("/bin/true")
		return func() tea.Msg { return editorRepickedMsg{} }
	}
	m, cmd := w.Update(tea.KeyPressMsg{Code: 'e'})
	w = m.(*wizardModel)
	if w.screen != "working" || cmd == nil {
		t.Fatalf("repick state screen=%q cmd=%v", w.screen, cmd)
	}
	msg := cmd()
	// doFinish-style wiring returns tea.Batch(spinner, repick); unwrap it.
	if batch, ok := msg.(tea.BatchMsg); ok {
		var found editorRepickedMsg
		for _, c := range batch {
			if c == nil {
				continue
			}
			if em, ok := c().(editorRepickedMsg); ok {
				found = em
				break
			}
		}
		msg = found
	}
	if _, ok := msg.(editorRepickedMsg); !ok {
		t.Fatalf("repick cmd message = %T, want editorRepickedMsg", msg)
	}
	m, _ = w.Update(msg)
	w = m.(*wizardModel)
	if w.screen != "review" || !strings.Contains(w.summary(w), "from select-editor") {
		t.Fatalf("after repick screen=%q summary=%q", w.screen, w.summary(w))
	}
}

func TestSummaryShowsSource(t *testing.T) {
	defer clearSessionEditor()
	trueBin := truePath(t)
	t.Setenv("EDITOR", trueBin)
	w := newEditorChoiceTestWizard(t)
	summary := w.summary(w)
	if !strings.Contains(summary, "editor: "+trueBin+" (from $EDITOR, opens after confirm)") {
		t.Fatalf("EDITOR summary = %q", summary)
	}
	setSessionEditor(trueBin)
	summary = w.summary(w)
	if !strings.Contains(summary, "editor: "+trueBin+" (from select-editor, opens after confirm)") {
		t.Fatalf("override summary = %q", summary)
	}
}
