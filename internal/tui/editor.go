package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// editorDoneMsg carries the result of one external-editor round trip.
type editorDoneMsg struct {
	tmp    string
	runErr error
}

// resolveEditorCmd builds the editor command from $EDITOR. Flags are
// supported ("code --wait"); values are not shell-quoted.
func resolveEditorCmd(path string) (*exec.Cmd, error) {
	ed := os.Getenv("EDITOR")
	if ed == "" {
		return nil, fmt.Errorf("EDITOR is not set")
	}
	parts := strings.Fields(ed)
	return exec.Command(parts[0], append(parts[1:], path)...), nil
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

// openInEditor returns the tea.Cmd that suspends the program in $EDITOR on
// the current draft; the resume is delivered as an editorDoneMsg.
func openInEditor(content string) tea.Cmd {
	tmp, err := writeBodyTemp(content)
	if err != nil {
		e := err
		return func() tea.Msg { return editorDoneMsg{runErr: e} }
	}
	cmd, err := resolveEditorCmd(tmp)
	if err != nil {
		e := err
		return func() tea.Msg {
			return editorDoneMsg{tmp: tmp, runErr: e}
		}
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorDoneMsg{tmp: tmp, runErr: err}
	})
}
