package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lukgth/tofu/internal/cli"
	"github.com/lukgth/tofu/internal/post"
)

// RunEdit picks a post and edits it interactively.
// slug empty -> table picker first; "b" on the picker switches to browsing
// content/ with the file picker (any .md, not just posts).
func RunEdit(root, slug string, opts Options) error {
	o := normalize(opts)
	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)

	posts, err := post.List(filepath.Join(root, "content"))
	if err != nil {
		return err
	}
	for {
		m := newEditPicker(posts, isDark, o)
		p := tea.NewProgram(m)
		model, err := p.Run()
		if err != nil {
			return err
		}
		picker := model.(editPickerModel)
		if picker.browse {
			path, err := runFileBrowser(root, o, isDark)
			if err != nil {
				return err
			}
			if path == "" {
				continue // back to the table picker
			}
			return runEditForm(root, path, o, isDark)
		}
		if picker.picked == "" {
			return nil
		}
		slug = picker.picked
		break
	}

	path, err := postPathForSlug(root, slug)
	if err != nil {
		return err
	}
	return runEditForm(root, path, o, isDark)
}

// fileBrowserModel is tofu's own directory browser: name | date | size
// rows over the bubbles table (no permissions column), directories first.
type fileBrowserModel struct {
	table     table.Model
	dir       string
	root      string
	entries   []os.DirEntry
	opts      Options
	picked    string
	cancelled bool
	slide     Slide
	spring    spring
}

func runFileBrowser(root string, o Options, isDark bool) (string, error) {
	b, err := newFileBrowser(filepath.Join(root, "content"), o)
	if err != nil {
		return "", err
	}
	p := tea.NewProgram(b)
	model, err := p.Run()
	if err != nil {
		return "", err
	}
	out := model.(*fileBrowserModel)
	if out.cancelled || out.picked == "" {
		return "", nil
	}
	return out.picked, nil
}

func newFileBrowser(start string, o Options) (*fileBrowserModel, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	b := &fileBrowserModel{
		dir:    abs,
		root:   abs,
		opts:   o,
		spring: NewSlideSpring(),
		slide:  NewSlide(8),
	}
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "name", Width: 30},
			{Title: "date", Width: 12},
			{Title: "size", Width: 8},
		}),
		table.WithFocused(true),
		table.WithHeight(16),
	)
	t.SetStyles(tableStyles())
	b.table = t
	if err := b.load(); err != nil {
		return nil, err
	}
	return b, nil
}

// load (re)reads the current directory into the table, dirs first.
func (m *fileBrowserModel) load() error {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		di, dj := entries[i].IsDir(), entries[j].IsDir()
		if di != dj {
			return di // directories first
		}
		return entries[i].Name() < entries[j].Name()
	})
	m.entries = entries
	rows := make([]table.Row, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		date := info.ModTime().Format("2006-01-02")
		size := "-"
		if !e.IsDir() {
			size = humanizeSize(info.Size())
		}
		rows = append(rows, table.Row{name, date, size})
	}
	m.table.SetRows(rows)
	m.table.SetCursor(0)
	m.table.SetWidth(m.opts.Width)
	m.table.SetHeight(16)
	return nil
}

func humanizeSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1fK", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

func (m *fileBrowserModel) Init() tea.Cmd {
	return tea.Batch(SlideCmd(m.slide, m.opts.NoAnimations))
}

func (m *fileBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(min(24, msg.Height-6))
		return m, nil
	case tea.MouseClickMsg:
		if msg.Mouse().Button == tea.MouseLeft {
			if row := msg.Mouse().Y - 3; row >= 0 && row < len(m.entries) {
				m.table.SetCursor(row)
			}
		}
		return m, nil
	case frameMsg:
		var cmd tea.Cmd
		if m.slide.Update(m.spring) {
			cmd = FrameTick()
		}
		return m, cmd
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			i := m.table.Cursor()
			if i < 0 || i >= len(m.entries) {
				return m, nil
			}
			e := m.entries[i]
			if e.IsDir() {
				next := filepath.Join(m.dir, e.Name())
				if abs, err := filepath.Abs(next); err == nil {
					m.dir = abs
					if err := m.load(); err != nil {
						return m, nil
					}
				}
				return m, nil
			}
			if strings.HasSuffix(e.Name(), ".md") {
				m.picked = filepath.Join(m.dir, e.Name())
				return m, tea.Quit
			}
			return m, nil
		case "backspace", "h":
			up := filepath.Dir(m.dir)
			if up != m.dir && len(up) >= len(m.root) {
				m.dir = up
				if err := m.load(); err != nil {
					return m, nil
				}
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *fileBrowserModel) View() tea.View {
	v := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		AnimatedTitle(m.slide.X, "tofu edit — browse "+m.relDir()),
		m.table.View(),
		HelpStyle.Render("enter open • backspace up • esc quit"),
	))
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// relDir renders the current directory relative to the site root for the title.
func (m *fileBrowserModel) relDir() string {
	if i := strings.Index(m.dir, "content"); i >= 0 {
		return m.dir[i:]
	}
	return m.dir
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
	browse bool
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
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 6)
		return m, nil
	case tea.MouseClickMsg:
		if msg.Mouse().Button == tea.MouseLeft {
			// Rows start after the animated title + header line.
			row := msg.Mouse().Y - 3
			if row >= 0 && row < len(m.table.Rows()) {
				m.table.SetCursor(row)
			}
		}
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
		case "ctrl+b":
			m.browse = true
			m.done = true
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
	v := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		AnimatedTitle(m.slide.X, "tofu edit"),
		m.table.View(),
		HelpStyle.Render("enter select • ctrl+b browse files • esc quit"),
	))
	v.MouseMode = tea.MouseModeCellMotion
	return v
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

// newEditWizard builds the edit-form wizard model. Split out of runEditForm
// so tests can drive the real steps/finish/execFinish closures through
// wizardModel.Update without launching a terminal program. Returns a nil
// model when the file does not parse.
func newEditWizard(root, path string, o Options, isDark bool) (*wizardModel, error) {
	p, err := post.ParseFile(path)
	if err != nil {
		return nil, err
	}

	titleIn := newTextInput("title", p.Frontmatter.Title, o)
	dateIn := newTextInput("YYYY-MM-DD", p.Frontmatter.Date, o)
	tagsIn := newTextInput("a,b,c", strings.Join(p.Frontmatter.Tags, ","), o)
	descIn := newTextInput("description", p.Frontmatter.Description, o)
	body := newBodyArea(p.BodyMarkdown, o)
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
		}, branch: func(w *wizardModel) tea.Cmd {
			// The external editor replaces the textarea step entirely:
			// jump straight to review. Quick edit advances to the
			// textarea; a non-nil branch owns its own advancement, so it
			// must handle both legs.
			if editorWanted {
				return w.gotoReview()
			}
			w.stepIdx++
			w.focusStep()
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
			if err := post.UpdateFrontmatter(path, func(f *post.Frontmatter) error {
				applyForm(f, &fm)
				return nil
			}); err != nil {
				return err
			}
			// Only the textarea path owns the body; the editor path must
			// never rewrite it with an empty newBody.
			if !editorWanted {
				return post.SetBody(path, newBody)
			}
			return nil
		},
		execFinish: func(w *wizardModel) tea.Cmd {
			if !editorWanted {
				err := w.finish(w)
				return func() tea.Msg { return wizardQuitMsg{err: err} }
			}
			sel, err := resolveEditor(path)
			if err != nil {
				return func() tea.Msg { return wizardQuitMsg{err: err} }
			}
			followUp := func(runErr error) tea.Msg {
				if runErr != nil {
					return wizardQuitMsg{err: fmt.Errorf("editor failed: %w", runErr)}
				}
				err := post.UpdateFrontmatter(path, func(f *post.Frontmatter) error {
					applyForm(f, &fm)
					return nil
				})
				return wizardQuitMsg{err: err}
			}
			if sel.needSelection {
				return runSelectEditorCmd(path, func(s2 editorSelection) tea.Msg {
					if s2.err != nil {
						return wizardQuitMsg{err: s2.err}
					}
					if s2.cmd == nil {
						return wizardQuitMsg{err: fmt.Errorf("no editor selected")}
					}
					return execEditorMergeMsg{cmd: s2.cmd, followUp: followUp}
				}, false)
			}
			return tea.ExecProcess(sel.cmd, followUp)
		},

		summary: func(w *wizardModel) string {
			lines := []string{
				Title.Render("review"),
				"title: " + fm.Title,
				"date: " + fm.Date,
				"tags: " + strings.Join(fm.Tags, ", "),
				"description: " + fm.Description,
			}
			if editorWanted {
				sel, _ := resolveEditor(path)
				switch {
				case sel.err != nil:
					lines = append(lines, "editor: "+sel.err.Error())
				case sel.needSelection:
					lines = append(lines, "editor: select-editor will ask (opens after confirm)")
				default:
					name := editorDisplayName()
					if sel.fromSource != "" {
						lines = append(lines, "editor: "+name+" (from "+sel.fromSource+", opens after confirm)")
					} else {
						lines = append(lines, "editor: "+name+" (opens after confirm)")
					}
				}
			}
			return strings.Join(lines, "\n")
		},
	}
	w.repickEditor = func() tea.Cmd {
		return runSelectEditorCmd(path, func(s2 editorSelection) tea.Msg {
			if s2.err != nil {
				return editorRepickErrorMsg{err: s2.err}
			}
			if s2.cmd == nil || len(s2.cmd.Args) == 0 {
				return editorRepickErrorMsg{err: fmt.Errorf("no editor selected")}
			}
			args := s2.cmd.Args
			if n := len(args); n > 0 && args[n-1] == path {
				args = args[:n-1]
			}
			if len(args) == 0 {
				return editorRepickErrorMsg{err: fmt.Errorf("no editor selected")}
			}
			setSessionEditor(strings.Join(args, " "))
			return editorRepickedMsg{}
		}, true)
	}
	return w, nil
}

func runEditForm(root, path string, o Options, isDark bool) error {
	clearSessionEditor()
	w, err := newEditWizard(root, path, o, isDark)
	if err != nil {
		return err
	}
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
