package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"tofu/internal/cli"
	"tofu/internal/post"
)

// RunEdit picks a post and edits it interactively.
// slug empty -> table picker first.
func RunEdit(root, slug string, opts Options) error {
	o := normalize(opts)
	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)

	posts, err := post.List(filepath.Join(root, "content"))
	if err != nil {
		return err
	}
	if slug == "" {
		if len(posts) == 0 {
			return fmt.Errorf("no posts to edit; try `tofu new`")
		}
		m := newEditPicker(posts, isDark, o)
		p := tea.NewProgram(m)
		model, err := p.Run()
		if err != nil {
			return err
		}
		picked := model.(editPickerModel).picked
		if picked == "" {
			return nil
		}
		slug = picked
	}

	path, err := postPathForSlug(root, slug)
	if err != nil {
		return err
	}
	return runEditForm(root, path, o, isDark)
}

func postPathForSlug(root, slug string) (string, error) {
	posts, err := post.List(filepath.Join(root, "content"))
	if err != nil {
		return "", err
	}
	for _, p := range posts {
		if p.Slug == slug {
			return filepath.Join(root, "content", "posts", slug+".md"), nil
		}
	}
	return "", fmt.Errorf("no post with slug %q", slug)
}

// editPickerModel is the date|slug|title table picker.
type editPickerModel struct {
	table  table.Model
	opts   Options
	picked string
	done   bool
	slide  Slide
	spring spring
}

func newEditPicker(posts []post.Post, isDark bool, o Options) editPickerModel {
	cols := []table.Column{
		{Title: "date", Width: 12},
		{Title: "slug", Width: 24},
		{Title: "title", Width: 36},
	}
	rows := make([]table.Row, 0, len(posts))
	for _, p := range posts {
		date := p.Date.Format("2006-01-02")
		if p.Date.IsZero() {
			date = "-"
		}
		title := p.Frontmatter.Title
		if p.Draft {
			title += " (draft)"
		}
		rows = append(rows, table.Row{date, p.Slug, title})
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)+1),
	)
	t.SetStyles(tableStyles())
	return editPickerModel{
		table:  t,
		opts:   o,
		spring: NewSlideSpring(),
		slide:  NewSlide(8),
	}
}

func (m editPickerModel) Init() tea.Cmd { return SlideCmd(m.slide, m.opts.NoAnimations) }

func (m editPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil
	case frameMsg:
		var cmd tea.Cmd
		if m.slide.Update(m.spring) {
			cmd = FrameTick()
		}
		return m, cmd
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if row := m.table.SelectedRow(); row != nil && len(row) > 1 {
				m.picked = row[1]
			}
			m.done = true
			return m, tea.Quit
		case "esc", "q":
			m.done = true
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m editPickerModel) View() tea.View {
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		AnimatedTitle(m.slide.X, "tofu edit"),
		m.table.View(),
		HelpFooter(m.opts),
	))
}

// editFormModel prefills and edits title/date/tags/description/draft + body.
type editFormModel struct {
	root     string
	path     string
	opts     Options
	isDark   bool
	wizard   *wizardModel
	editorFn func() error
}

func runEditForm(root, path string, o Options, isDark bool) error {
	p, err := post.ParseFile(path)
	if err != nil {
		return err
	}

	titleIn := newTextInput("title", p.Frontmatter.Title, o)
	dateIn := newTextInput("YYYY-MM-DD", p.Frontmatter.Date, o)
	tagsIn := newTextInput("a,b,c", strings.Join(p.Frontmatter.Tags, ","), o)
	descIn := newTextInput("description", p.Frontmatter.Description, o)
	body := textarea.New()
	body.SetStyles(textareaStyles())
	body.SetValue(p.BodyMarkdown)
	body.SetHeight(8)
	body.SetWidth(o.Width - 4)
	editorChoice := newChoice([]string{"Quick edit (textarea)", "Open $EDITOR"}, isDark, o.Width)

	fm := p.Frontmatter
	var newBody string
	editorWanted := false

	steps := []step{
		{kind: stepInput, label: "title", input: &titleIn, setter: func(v string) error {
			if v != "" {
				fm.Title = v
			}
			return nil
		}},
		{kind: stepInput, label: "date", input: &dateIn, setter: func(v string) error {
			if v != "" {
				fm.Date = v
			}
			return nil
		}},
		{kind: stepInput, label: "tags", input: &tagsIn, setter: func(v string) error {
			fm.Tags = cli.ParseTags(v)
			return nil
		}},
		{kind: stepInput, label: "description", input: &descIn, setter: func(v string) error {
			fm.Description = v
			return nil
		}},
		{kind: stepChoice, label: "body", choice: &editorChoice, setter: func(v string) error {
			editorWanted = strings.HasPrefix(v, "Open")
			return nil
		}},
		{kind: stepBody, label: "body (textarea)", area: &body, setter: func(v string) error {
			newBody = v
			return nil
		}},
	}

	w := &wizardModel{
		root:     root,
		opts:     o,
		title:    "edit " + p.Slug,
		isDark:   isDark,
		spring:   NewSlideSpring(),
		steps:    steps,
		spinnerM: newSpinnerModel(),
		prog:     newProgressModel(o.Width - 8),
		vp:       newViewport(o.Width, 14),
		finish: func(w *wizardModel) error {
			if editorWanted {
				if ed := os.Getenv("EDITOR"); ed != "" {
					cmd := exec.Command(ed, path)
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Run(); err != nil {
						return fmt.Errorf("editor failed: %w", err)
					}
					return post.UpdateFrontmatter(path, func(f *post.Frontmatter) error {
						applyForm(f, &fm)
						return nil
					})
				}
				return fmt.Errorf("EDITOR is not set; quick edit used instead (or run `tofu edit --title ...`)")
			}
			if err := post.UpdateFrontmatter(path, func(f *post.Frontmatter) error {
				applyForm(f, &fm)
				return nil
			}); err != nil {
				return err
			}
			return post.SetBody(path, newBody)
		},
		summary: func(w *wizardModel) string {
			return strings.Join([]string{
				Title.Render("review"),
				"title: " + fm.Title,
				"date: " + fm.Date,
				"tags: " + strings.Join(fm.Tags, ", "),
				"description: " + fm.Description,
			}, "\n")
		},
	}
	_ = newBody
	return runWizardProgram(w)
}

func applyForm(dst, src *post.Frontmatter) {
	dst.Title = src.Title
	dst.Date = src.Date
	dst.Tags = src.Tags
	dst.Description = src.Description
	dst.Draft = src.Draft
	dst.Slug = src.Slug
}
