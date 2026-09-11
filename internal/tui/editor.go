package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// editorDoneMsg carries the result of one external-editor round trip.
type editorDoneMsg struct {
	tmp    string
	runErr error
}

type editorSelection struct {
	cmd           *exec.Cmd
	needSelection bool
	fromSource    string
	err           error
}

// sessionEditor is the editor explicitly chosen from the review screen's
// change-editor action for this run; "" when none.
var sessionEditor string

func setSessionEditor(v string) { sessionEditor = v }
func clearSessionEditor()       { sessionEditor = "" }

// errEditorUnavailable is returned when no editor can be resolved at all:
// no usable $EDITOR, no ~/.selected_editor, and no select-editor binary.
var errEditorUnavailable = errors.New("EDITOR is not set and select-editor is not available")

// selectedEditorPath returns the editor command recorded by select-editor.
func selectedEditorPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(home + "/.selected_editor")
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`^[[:space:]]*SELECTED_EDITOR="([^"]+)"`)
	selected := ""
	for _, line := range strings.Split(string(data), "\n") {
		if m := re.FindStringSubmatch(line); len(m) == 2 {
			selected = m[1]
		}
	}
	return selected
}

// tryEditor builds a command from an editor value ("vim -x" style allowed)
// when its binary resolves in PATH; nil otherwise.
func tryEditor(value, path string) *exec.Cmd {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return nil
	}
	if _, err := exec.LookPath(fields[0]); err != nil {
		return nil
	}
	return exec.Command(fields[0], append(fields[1:], path)...)
}

func resolveEditor(path string) (editorSelection, error) {
	if sessionEditor != "" {
		if cmd := tryEditor(sessionEditor, path); cmd != nil {
			return editorSelection{cmd: cmd, fromSource: "select-editor"}, nil
		}
	}
	if ed := os.Getenv("EDITOR"); ed != "" {
		if cmd := tryEditor(ed, path); cmd != nil {
			return editorSelection{cmd: cmd, fromSource: "$EDITOR"}, nil
		}
	}
	if selected := selectedEditorPath(); selected != "" {
		if cmd := tryEditor(selected, path); cmd != nil {
			return editorSelection{cmd: cmd, fromSource: "select-editor"}, nil
		}
	}
	if _, err := exec.LookPath("select-editor"); err == nil {
		for _, candidate := range []string{"sensible-editor", "/usr/bin/vi", "/usr/bin/editor", "nano", "vim"} {
			if editor, err := exec.LookPath(candidate); err == nil {
				return editorSelection{cmd: exec.Command(editor, path), needSelection: true}, nil
			}
		}
		return editorSelection{needSelection: true}, nil
	}
	return editorSelection{}, errEditorUnavailable
}

// resolveEditorCmd retains the legacy command-only resolver API.
func resolveEditorCmd(path string) (*exec.Cmd, error) {
	sel, err := resolveEditor(path)
	if err != nil {
		return nil, err
	}
	if sel.needSelection {
		return nil, errEditorUnavailable
	}
	return sel.cmd, nil
}

// runSelectEditorCmd suspends into select-editor and hands the selection
// outcome to fn. With force=false it is a no-op when an editor is already
// resolvable (used by the ctrl+o fallback). The review screen's change-
// editor action passes force=true: the user explicitly asked to re-pick,
// so suspend even when $EDITOR or a previous selection exists.
func runSelectEditorCmd(path string, fn func(editorSelection) tea.Msg, force bool) tea.Cmd {
	if !force {
		sel, _ := resolveEditor(path)
		if !sel.needSelection {
			return func() tea.Msg { return fn(sel) }
		}
	}
	selectEditor, err := exec.LookPath("select-editor")
	if err != nil {
		return func() tea.Msg {
			return fn(editorSelection{err: errEditorUnavailable})
		}
	}
	return tea.ExecProcess(exec.Command(selectEditor), func(runErr error) tea.Msg {
		if runErr != nil {
			return fn(editorSelection{err: fmt.Errorf("select-editor failed: %w", runErr)})
		}
		// The user just picked explicitly: drop any previous override,
		// then prefer the value select-editor just wrote over $EDITOR.
		sessionEditor = ""
		if selected := selectedEditorPath(); selected != "" {
			if cmd := tryEditor(selected, path); cmd != nil {
				return fn(editorSelection{cmd: cmd, fromSource: "select-editor"})
			}
		}
		next, err := resolveEditor(path)
		if err != nil || next.needSelection {
			if err == nil {
				err = errEditorUnavailable
			}
			return fn(editorSelection{err: err})
		}
		return fn(next)
	})
}

func editorDisplayName() string {
	if sessionEditor != "" {
		return sessionEditor
	}
	if ed := os.Getenv("EDITOR"); ed != "" {
		return ed
	}
	if selected := selectedEditorPath(); selected != "" {
		return selected
	}
	if _, err := exec.LookPath("select-editor"); err == nil {
		return "select-editor will ask"
	}
	return ""
}

func writeBodyTemp(content string) (string, error) {
	f, err := os.CreateTemp("", "tofu-body-*.md")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	return name, f.Close()
}

func openInEditor(content string) tea.Cmd {
	tmp, err := writeBodyTemp(content)
	if err != nil {
		e := err
		return func() tea.Msg { return editorDoneMsg{runErr: e} }
	}
	sel, err := resolveEditor(tmp)
	if err != nil {
		e := err
		return func() tea.Msg { return editorDoneMsg{tmp: tmp, runErr: e} }
	}
	if sel.needSelection {
		return runSelectEditorCmd(tmp, func(sel2 editorSelection) tea.Msg {
			if sel2.err != nil {
				return editorDoneMsg{tmp: tmp, runErr: sel2.err}
			}
			return editorDoneMsg{tmp: tmp}
		}, false)
	}
	return tea.ExecProcess(sel.cmd, func(runErr error) tea.Msg {
		return editorDoneMsg{tmp: tmp, runErr: runErr}
	})
}
