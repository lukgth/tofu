package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"tofu/internal/cli"
	"tofu/internal/config"
)

type spring interface {
	Update(pos, vel, equilibriumPos float64) (newPos, newVel float64)
}

// wizardModel drives the multi-step prompts of one wizard.
// Screens: prompt -> review -> working -> done.
type wizardModel struct {
	root     string
	opts     Options
	screen   string
	title    string
	isDark   bool
	slide    Slide
	spring   spring
	spinnerM spinnerModel
	prog     progressModel
	progMsg  string
	vp       viewportModel
	confirm  choiceModel
	errorMsg string
	doneMsg  string
	steps    []step
	stepIdx  int
	finish   func(w *wizardModel) error
	summary  func(w *wizardModel) string
}

type wizardQuitMsg struct{ err error }

func runWizardProgram(w *wizardModel) error {
	w.slide = NewSlide(8)
	w.screen = "prompt"
	w.focusStep()
	p := tea.NewProgram(w)
	_, err := p.Run()
	return err
}

func (w *wizardModel) Init() tea.Cmd {
	return SlideCmd(w.slide, w.opts.NoAnimations)
}

func (w *wizardModel) current() *step {
	if w.stepIdx < 0 || w.stepIdx >= len(w.steps) {
		return nil
	}
	return &w.steps[w.stepIdx]
}

func (w *wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.opts.Width = msg.Width
		return w, nil
	case editorDoneMsg:
		if msg.tmp != "" {
			defer os.Remove(msg.tmp)
		}
		if msg.runErr != nil {
			w.errorMsg = "editor failed: " + msg.runErr.Error()
			return w, nil
		}
		data, err := os.ReadFile(msg.tmp)
		if err != nil {
			w.errorMsg = "editor failed: " + err.Error()
			return w, nil
		}
		if s := w.current(); s != nil && s.kind == stepBody && s.area != nil {
			s.area.SetValue(string(data))
		}
		w.errorMsg = ""
		w.focusStep()
		return w, nil

	case frameMsg:
		var cmd tea.Cmd
		if w.slide.Update(w.spring) {
			cmd = FrameTick()
		}
		return w, cmd

	case wizardQuitMsg:
		if msg.err != nil {
			w.errorMsg = msg.err.Error()
			w.screen = "prompt"
			w.stepIdx = 0
			w.focusStep()
			return w, nil
		}
		w.screen = "done"
		return w, nil

	case tea.MouseClickMsg:
		if msg.Mouse().Button != tea.MouseLeft {
			return w, nil
		}
		switch w.screen {
		case "review":
			// Yes/No cards start after title + summary viewport; select
			// the clicked card and confirm.
			idx := w.confirm.indexAt(msg.Mouse().Y - 4)
			if idx >= 0 {
				w.confirm.cursor = idx
				return w, w.doChoiceAdvance()
			}
			return w, nil
		case "prompt":
			s := w.current()
			if s != nil && s.kind == stepChoice {
				// Choice cards start at body row (title + label above).
				if idx := s.choice.indexAt(msg.Mouse().Y - 2); idx >= 0 {
					s.choice.cursor = idx
					return w, w.doChoiceAdvance()
				}
			}
			return w, nil
		case "done":
			return w, tea.Quit
		}
		return w, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return w, tea.Quit
		}

		switch w.screen {
		case "review":
			switch msg.String() {
			case "enter":
				if w.confirm.selected() == "Yes" {
					w.screen = "working"
					w.progMsg = "writing"
					return w, tea.Batch(w.spinnerM.tick(), w.doFinish())
				}
				w.screen = "prompt"
				w.stepIdx = 0
				w.focusStep()
				return w, nil
			case "esc", "q":
				w.screen = "prompt"
				w.stepIdx = 0
				w.focusStep()
				return w, nil
			}
			w.confirm = w.confirm.update(msg)
			return w, nil

		case "working":
			return w, nil

		case "done":
			return w, tea.Quit

		case "prompt":
			if msg.String() == "esc" {
				if w.stepIdx == 0 {
					return w, tea.Quit
				}
				w.stepIdx--
				w.focusStep()
				return w, nil
			}
			s := w.current()
			if s == nil {
				return w, w.gotoReview()
			}
			switch s.kind {
			case stepInput:
				if msg.String() == "enter" {
					val := strings.TrimSpace(s.input.Value())
					if s.setter != nil {
						if err := s.setter(val); err != nil {
							w.errorMsg = err.Error()
							return w, nil
						}
					}
					w.errorMsg = ""
					w.stepIdx++
					if w.stepIdx >= len(w.steps) {
						return w, w.gotoReview()
					}
					w.focusStep()
					return w, nil
				}
				var cmd tea.Cmd
				*s.input, cmd = s.input.Update(msg)
				return w, cmd

			case stepChoice:
				if msg.String() == "enter" {
					return w, w.doChoiceAdvance()
				}
				*s.choice = s.choice.update(msg)
				return w, nil
			case stepBody:
				if msg.String() == "ctrl+o" {
					return w, openInEditor(s.area.Value())
				}
				if msg.String() == "enter" {
					val := strings.TrimSpace(s.area.Value())
					if s.setter != nil {
						if err := s.setter(val); err != nil {
							w.errorMsg = err.Error()
							return w, nil
						}
					}
					w.errorMsg = ""
					w.stepIdx++
					if w.stepIdx >= len(w.steps) {
						return w, w.gotoReview()
					}
					w.focusStep()
					return w, nil
				}
				var cmd tea.Cmd
				*s.area, cmd = s.area.Update(msg)
				return w, cmd

			case stepFile:
				if msg.String() == "enter" {
					pick := *s.pick
					ok, val := pick.DidSelectFile(nil)
					if ok && s.setter != nil {
						if err := s.setter(val); err != nil {
							w.errorMsg = err.Error()
							return w, nil
						}
					}
					w.stepIdx++
					if w.stepIdx >= len(w.steps) {
						return w, w.gotoReview()
					}

					w.focusStep()
					return w, nil
				}
				if msg.String() == "esc" {
					w.stepIdx++
					if w.stepIdx >= len(w.steps) {
						return w, w.gotoReview()
					}
					w.focusStep()
					return w, nil
				}
				var cmd tea.Cmd
				*s.pick, cmd = s.pick.Update(msg)
				return w, cmd
			}
		}
	}
	return w, nil
}

// focusStep focuses the interactive control of the current step. Inputs and
// textareas are inert while unfocused in bubbles v2.
func (w *wizardModel) focusStep() {
	if s := w.current(); s != nil {
		switch s.kind {
		case stepInput:
			s.input.Focus()
		case stepBody:
			s.area.Focus()
		case stepChoice, stepFile:
		}
	}
}

func (w *wizardModel) gotoReview() tea.Cmd {
	w.screen = "review"
	w.confirm = newChoice([]string{"Yes", "No"}, w.isDark, w.opts.Width)
	if w.summary != nil {
		w.vp.setContent(w.summary(w))
	}
	return nil
}

// doChoiceAdvance applies the current choice selection and moves to the
// next step (or review). Used by both the enter key and mouse clicks.
func (w *wizardModel) doChoiceAdvance() tea.Cmd {
	s := w.current()
	if s == nil || s.kind != stepChoice {
		return nil
	}
	val := s.choice.selected()
	if s.setter != nil {
		if err := s.setter(val); err != nil {
			w.errorMsg = err.Error()
			return nil
		}
	}
	w.errorMsg = ""
	w.stepIdx++
	if w.stepIdx >= len(w.steps) {
		return w.gotoReview()
	}
	w.focusStep()
	return nil
}

func (w *wizardModel) doFinish() tea.Cmd {
	return func() tea.Msg {
		err := w.finish(w)
		return wizardQuitMsg{err: err}
	}
}

func (w *wizardModel) View() tea.View {
	v := w.viewContent()
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (w *wizardModel) viewContent() tea.View {
	switch w.screen {
	case "working":
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			AnimatedTitle(w.slide.X, "tofu"),
			w.spinnerM.view()+" "+w.progMsg,
			w.prog.view(),
		))
	case "review":
		footer := HelpFooter(w.opts)
		if w.errorMsg != "" {
			footer = ErrorStyle.Render(w.errorMsg) + "\n" + footer
		}
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			AnimatedTitle(w.slide.X, "tofu "+w.title),
			w.vp.view(),
			"",
			w.confirm.view(w.isDark),
			footer,
		))
	case "done":
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			AnimatedTitle(w.slide.X, "tofu"),
			Accent.Render("✓ Done! "+w.doneMsg),
			"",
			HelpStyle.Render("press enter to go back to the menu"),
		))
	}
	footer := HelpFooter(w.opts)
	if s := w.current(); s != nil && s.kind == stepBody && w.opts.ShowHelp {
		footer = HelpStyle.Render("enter next • esc back • ctrl+o $EDITOR • ctrl+c quit")
	}
	if w.errorMsg != "" {
		footer = ErrorStyle.Render(w.errorMsg) + "\n" + footer
	}
	body := ""
	label := ""
	if s := w.current(); s != nil {
		label = s.label
		switch s.kind {
		case stepInput:
			body = s.input.View()
		case stepChoice:
			body = s.choice.view(w.isDark)
		case stepBody:
			body = s.area.View()
		case stepFile:
			body = s.pick.View()
		}
	}
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		AnimatedTitle(w.slide.X, "tofu "+w.title),
		Dim.Render(label),
		body,
		footer,
	))
}

var _ = key.NewBinding
var _ = fmt.Sprintf
var _ = textinput.New

// RunInit is the new-site wizard: asks for the basics, scaffolds the site.
func RunInit(root string, opts Options) error {
	if hasSite(root) {
		return fmt.Errorf("a site already exists in %s (tofu.toml); edit it directly instead", root)
	}
	o := normalize(opts)
	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	titleIn := newTextInput("", "My Tofu Site", o)
	authorIn := newTextInput("", "Jane Doe", o)
	descIn := newTextInput("", "A cute little blog", o)
	urlIn := newTextInput("", "https://example.com", o)

	var cfg = cli.DefaultConfig()
	steps := []step{
		{kind: stepInput, label: "site title", input: &titleIn, setter: func(v string) error {
			if v != "" {
				cfg.Title = v
			}
			return nil
		}},
		{kind: stepInput, label: "author", input: &authorIn, setter: func(v string) error {
			cfg.Author = v
			return nil
		}},
		{kind: stepInput, label: "description", input: &descIn, setter: func(v string) error {
			cfg.Description = v
			return nil
		}},
		{kind: stepInput, label: "base url", input: &urlIn, setter: func(v string) error {
			cfg.BaseURL = v
			return nil
		}},
	}

	w := &wizardModel{
		root:     root,
		opts:     o,
		title:    "new site",
		isDark:   isDark,
		spring:   NewSlideSpring(),
		steps:    steps,
		spinnerM: newSpinnerModel(),
		prog:     newProgressModel(o.Width - 8),
		vp:       newViewport(o.Width, 14),
		finish: func(w *wizardModel) error {
			if err := config.Save(filepath.Join(w.root, "tofu.toml"), cfg); err != nil {
				return err
			}
			if _, err := cli.InitScaffold(w.root, true); err != nil {
				return err
			}
			w.doneMsg = "created " + w.root
			return nil
		},
		summary: func(w *wizardModel) string {
			return strings.Join([]string{
				Title.Render("review"),
				"title: " + cfg.Title,
				"author: " + cfg.Author,
				"description: " + cfg.Description,
				"base url: " + cfg.BaseURL,
				"dir: " + w.root,
			}, "\n")
		},
	}
	return runWizardProgram(w)
}
