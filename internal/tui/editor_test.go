package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
)

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
	msg := openInEditor("x")()
	done, ok := msg.(editorDoneMsg)
	if !ok {
		t.Fatalf("expected editorDoneMsg, got %T", msg)
	}
	if done.runErr == nil || !strings.Contains(done.runErr.Error(), "EDITOR") {
		t.Fatalf("runErr = %v, want EDITOR error", done.runErr)
	}
	if done.tmp == "" {
		t.Fatal("tmp should be populated so Update can clean it up")
	}
	os.Remove(done.tmp)
}
