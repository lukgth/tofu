package tui

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lukgth/tofu/internal/cli"
	"github.com/lukgth/tofu/internal/render"
)

type screenKind int

const (
	screenList screenKind = iota
	screenViewport
	screenProgress
)

// menuItem adapts a plain label to a list.Item.
type menuItem struct{ label string }

func (m menuItem) Title() string       { return m.label }
func (m menuItem) Description() string { return "" }
func (m menuItem) FilterValue() string { return m.label }

// MenuModel is the tofu home menu with its nested busy screens.
type MenuModel struct {
	opts     Options
	root     string
	port     int
	list     list.Model
	slide    Slide
	spring   spring
	screen   screenKind
	vp       viewportModel
	vpTitle  string
	spinnerM spinnerModel
	prog     progressModel
	progMsg  string
	errorMsg string
	width    int
	height   int
	// srv is the preview server, ln its listener. Closing an http.Server
	// closes the listener it was handed, and the OS holds that port until
	// the close lands, so rebinding and stopping go through ln.Close: a
	// second Preview can take the same port back immediately.
	srv    *http.Server
	ln     net.Listener
	wizard func(root string, o Options) error
}

type viewportDoneMsg struct{}

type progressDoneMsg struct{ err error }

// RunMenu shows the home menu in a loop. Wizard actions quit the menu, run
// their wizard, and the menu re-opens fresh so labels reflect the site.
func RunMenu(root string, opts Options) error {
	errMsg := ""
	for {
		final, err := runMenuOnce(root, opts, errMsg)
		if err != nil {
			return err
		}
		errMsg = ""
		mm, ok := final.(MenuModel)
		if !ok || mm.wizard == nil {
			return nil // quit or ctrl+c
		}
		if err := mm.wizard(root, opts); err != nil {
			errMsg = err.Error()
		}
	}
}

func runMenuOnce(root string, opts Options, initErr string) (tea.Model, error) {
	o := normalize(opts)
	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	l := newList(menuItems(root), isDark, o.Width, len(menuActions())+10)
	l.SetShowStatusBar(false)
	l.KeyMap.Filter = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter"))

	m := MenuModel{
		opts:     o,
		root:     root,
		port:     8787,
		list:     l,
		spring:   NewSlideSpring(),
		screen:   screenList,
		vp:       newViewport(o.Width, 20),
		spinnerM: newSpinnerModel(),
		prog:     newProgressModel(o.Width - 8),
		width:    o.Width,
		height:   24,
		slide:    NewSlide(8),
		errorMsg: initErr,
	}
	return runProgram(m, o)
}

func hasSite(root string) bool {
	_, err := os.Stat(filepath.Join(root, "tofu.toml"))
	return err == nil
}

type menuAction struct {
	label    string
	needSite bool
	run      func(*MenuModel) tea.Cmd
}

func menuActions() []menuAction {
	return []menuAction{
		{"New site", false, (*MenuModel).runInit},
		{"New blog post", true, (*MenuModel).runNew},
		{"Edit existing posts", true, (*MenuModel).runEdit},
		{"Build site", true, (*MenuModel).runBuild},
		{"Preview (serve)", true, (*MenuModel).runServe},
		{"List posts", true, (*MenuModel).runList},
		{"Quit", false, (*MenuModel).runQuit},
	}
}

func menuItems(root string) []list.Item {
	site := hasSite(root)
	actions := menuActions()
	items := make([]list.Item, len(actions))
	for i, a := range actions {
		label := a.label
		if a.needSite && !site {
			label += " (need site — run New site first)"
		}
		items[i] = menuItem{label: label}
	}
	return items
}

func (m *MenuModel) selectedAction() (menuAction, error) {
	i := m.list.Index()
	actions := menuActions()
	if i < 0 || i >= len(actions) {
		return menuAction{}, fmt.Errorf("no selection")
	}
	return actions[i], nil
}

func (m *MenuModel) runInit() tea.Cmd {
	if hasSite(m.root) {
		m.errorMsg = "a site already exists here (tofu.toml)"
		return nil
	}
	return m.runWizard(RunInit)
}

func (m *MenuModel) runNew() tea.Cmd {
	return m.runWizard(RunNew)
}

func (m *MenuModel) runEdit() tea.Cmd {
	return m.runWizard(func(root string, o Options) error {
		return RunEdit(root, "", o)
	})
}

func (m *MenuModel) runBuild() tea.Cmd {
	return m.runTask("building site", func() error {
		return render.Build(m.root, filepath.Join(m.root, "public"), false)
	})
}

// runServe builds the site; the listener is bound by the progress screen
// itself once the build reports success, so the server belongs to the model
// that carries it into the preview screen.
func (m *MenuModel) runServe() tea.Cmd {
	return m.runTask("building for preview", func() error {
		return render.Build(m.root, filepath.Join(m.root, "public"), false)
	})
}

func (m *MenuModel) runList() tea.Cmd {
	lines, err := cli.ListPosts(m.root)
	if err != nil {
		m.errorMsg = err.Error()
		return nil
	}
	m.vpTitle = "posts"
	m.vp.setContent(joinLines(lines))
	m.screen = screenViewport
	return nil
}

func (m *MenuModel) runQuit() tea.Cmd {
	m.stopServe()
	return tea.Quit
}

// runTask wraps blocking work in the spinner+progress screen.
func (m *MenuModel) runTask(label string, work func() error) tea.Cmd {
	m.screen = screenProgress
	m.progMsg = label
	m.errorMsg = ""
	workErr := make(chan error, 1)
	go func() { workErr <- work() }()
	return tea.Batch(
		m.spinnerM.tick(),
		m.prog.setPercent(0.5),
		func() tea.Msg {
			err := <-workErr
			return progressDoneMsg{err: err}
		},
	)
}

// runWizard records the wizard and quits the menu; the RunMenu loop runs it
// and re-opens the menu fresh afterwards.
func (m *MenuModel) runWizard(fn func(root string, o Options) error) tea.Cmd {
	m.wizard = fn
	return tea.Quit
}

// joinLines renders list lines for the viewport: one trailing newline each,
// so the last line is not glued to the frame edge.
func joinLines(lines []string) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return b.String()
}

func (m MenuModel) Init() tea.Cmd {
	return SlideCmd(m.slide, m.opts.NoAnimations)
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// The menu is a short single-line-item list; keep it content-sized
		// instead of stretching to the full terminal height. The floor keeps
		// the list positive: a very short terminal (or a window the size of
		// the message itself) would otherwise ask the list to paginate into
		// negative space.
		h := len(menuActions()) + 10
		if lim := msg.Height - 5; h > lim {
			h = max(lim, 1)
		}
		m.list.SetSize(msg.Width, h)
		return m, nil

	case tea.BackgroundColorMsg:
		return m, nil

	case frameMsg:
		var cmd tea.Cmd
		if m.slide.Update(m.spring) {
			cmd = FrameTick()
		}
		return m, cmd

	case tea.MouseClickMsg:
		if msg.Mouse().Button != tea.MouseLeft {
			return m, nil
		}
		if m.screen != screenList {
			// any click on a viewport-ish screen pops back to the menu
			if m.screen == screenViewport {
				m.screen = screenList
			}
			return m, nil
		}
		// List rows start after the animated title row; item rows are
		// single-line. Convert the click row into a cursor move, then
		// activate if it was already selected.
		row := msg.Mouse().Y - 2 // title row + list top offset
		idx := m.list.Index()
		if row >= 0 && row < len(menuItems(m.root)) {
			m.list.Select(row)
			if idx == row {
				action, err := m.selectedAction()
				if err == nil {
					if action.needSite && !hasSite(m.root) {
						m.errorMsg = "run New site first"
						return m, nil
					}
					m.errorMsg = ""
					return m, action.run(&m)
				}
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			m.stopServe()
			return m, tea.Quit
		}
		switch m.screen {
		case screenList:
			if msg.String() == "enter" {
				action, err := m.selectedAction()
				if err != nil {
					return m, nil
				}
				if action.needSite && !hasSite(m.root) {
					m.errorMsg = "run New site first"
					return m, nil
				}
				m.errorMsg = ""
				cmd := action.run(&m)
				return m, cmd
			}
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd

		case screenViewport:
			switch msg.String() {
			case "esc", "q", "enter":
				m.screen = screenList
				return m, nil
			}
			var cmd tea.Cmd
			m.vp.vp, cmd = m.vp.vp.Update(msg)
			return m, cmd

		case screenProgress:
			return m, nil
		}
		return m, nil

	case progressFrameMsg:
		if m.screen != screenProgress {
			return m, nil
		}
		return m, m.prog.updateFrame(msg)

	case spinnerTickMsg:
		if m.screen != screenProgress {
			return m, nil
		}
		return m, m.spinnerM.update(msg)

	case progressDoneMsg:
		if msg.err != nil {
			m.screen = screenList
			m.errorMsg = msg.err.Error()
			return m, nil
		}
		switch m.progMsg {
		case "building site":
			n, _ := cli.CountPosts(m.root)
			m.vpTitle = "build"
			m.vp.setContent(fmt.Sprintf("✓ Done! Built %d posts -> %s/\n", n, filepath.Join(m.root, "public")))
			m.screen = screenViewport
			return m, nil
		case "building for preview":
			// Bind here, on the model that keeps the server: a port already
			// in use is a preview failure, and announcing a URL no listener
			// backs would be a lie.
			if err := m.listen(); err != nil {
				m.screen = screenList
				m.errorMsg = err.Error()
				return m, nil
			}
			m.announceServe()
			return m, nil
		}
		m.screen = screenList
		return m, nil

	}
	return m, nil
}

// listen binds the preview listener and starts serving public/ in the
// background. A bind failure goes back to the progress screen, which reports
// it. A live listener is closed first, so a second "Preview (serve)" can take
// the same port back instead of failing on its own predecessor.
func (m *MenuModel) listen() error {
	m.stopServe()
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(m.port)))
	if err != nil {
		return fmt.Errorf("preview: %w", err)
	}
	srv := &http.Server{
		Handler:           http.FileServer(http.Dir(filepath.Join(m.root, "public"))),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()
	m.srv = srv
	m.ln = ln
	return nil
}

// stopServe closes the preview server and its listener, and is safe to call
// when nothing is running.
func (m *MenuModel) stopServe() {
	if m.ln != nil {
		// The listener is closed directly: closing the http.Server alone
		// leaves the port held until its own close lands, which is too late
		// for a rebind on the very next action.
		_ = m.ln.Close()
		m.ln = nil
	}
	if m.srv != nil {
		_ = m.srv.Close()
		m.srv = nil
	}
}

// announceServe shows the URL of an already-listening server.
func (m *MenuModel) announceServe() {
	addr := fmt.Sprintf("127.0.0.1:%d", m.port)
	m.vpTitle = "preview"
	m.vp.setContent(fmt.Sprintf("✓ serving %s\nopen http://%s in your browser\n\nany key returns to the menu (server keeps running)", filepath.Join(m.root, "public"), addr))
	m.screen = screenViewport
}

func (m MenuModel) View() tea.View {
	switch m.screen {
	case screenViewport:
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			AnimatedTitle(m.slide.X, "tofu "+m.vpTitle),
			m.vp.view(),
			HelpFooter(m.opts),
		))
	case screenProgress:
		v := tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			AnimatedTitle(m.slide.X, "tofu"),
			m.spinnerM.view()+" "+m.progMsg,
			m.prog.view(),
		))
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}
	footer := MenuFooter()
	if m.errorMsg != "" {
		footer = ErrorStyle.Render(m.errorMsg) + "\n" + footer
	}
	v := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		AnimatedTitle(m.slide.X, "tofu"),
		m.list.View(),
		footer,
	))
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
