package tui

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lukgth/tofu/internal/cli"
	"github.com/lukgth/tofu/internal/config"
	"github.com/lukgth/tofu/internal/post"
)

// writePostFile writes a markdown file with frontmatter (slug) and a body.
func writePostFile(t *testing.T, path, slug, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	src := fmt.Sprintf("---\ntitle: T\ndate: 2026-01-01\nslug: %s\n---\n%s", slug, body)
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

// scaffoldSite writes a minimal site: tofu.toml plus content/posts.
func scaffoldSite(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := cli.InitScaffold(root, true); err != nil {
		t.Fatal(err)
	}
	return root
}

// -----------------------------------------------------------------------------
// init: the typed site title must reach the header, not the default's copy.
// -----------------------------------------------------------------------------

// TestInitWizardTitlePropagatesToHeader covers the bug this fixes: the answer
// to "site title" was written to config.title while header.title kept the
// DefaultConfig copy, so every scaffolded site showed "My Tofu Site".
func TestInitWizardTitlePropagatesToHeader(t *testing.T) {
	root := t.TempDir()
	o := normalize(Options{})
	w := newInitWizard(root, o, false)
	w.screen = "prompt"
	w.focusStep()

	// Step 0 is the site title.
	if w.steps[0].label != "site title" {
		t.Fatalf("step 0 = %q, want site title", w.steps[0].label)
	}
	w.steps[0].input.SetValue("Moon Cakes")
	m, _ := w.Update(enterKey())
	w = m.(*wizardModel)
	for range 3 {
		m, _ = w.Update(enterKey())
		w = m.(*wizardModel)
	}
	if w.screen != "review" {
		t.Fatalf("screen = %q, want review", w.screen)
	}
	if got := w.summary(w); !strings.Contains(got, "title: Moon Cakes") {
		t.Fatalf("review summary missing the typed title: %q", got)
	}
	if err := w.finish(w); err != nil {
		t.Fatalf("finish: %v", err)
	}

	cfg, err := config.Load(filepath.Join(root, "tofu.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "Moon Cakes" {
		t.Errorf("config title = %q", cfg.Title)
	}
	if cfg.Header.Title != "Moon Cakes" {
		t.Errorf("header title = %q, want the typed site title", cfg.Header.Title)
	}
	if len(cfg.Header.Nav) == 0 {
		t.Fatal("header nav lost by scaffolding")
	}
	if cfg.Header.Nav[0].URL == "/" && cfg.Header.Nav[0].Label != "Moon Cakes" {
		t.Errorf("home nav label = %q, want the typed site title", cfg.Header.Nav[0].Label)
	}
	if len(cfg.Header.Nav) > 1 && cfg.Header.Nav[1].Label != "blog" {
		t.Errorf("other nav labels must be untouched, got %q", cfg.Header.Nav[1].Label)
	}
}

// -----------------------------------------------------------------------------
// slug lookup: unambiguous, and no silent fall-through to the picker.
// -----------------------------------------------------------------------------

// Two files can claim one frontmatter slug; picking either silently would
// edit the wrong post.
func TestPostPathForSlugAmbiguous(t *testing.T) {
	root := scaffoldSite(t)
	posts := filepath.Join(root, "content", "posts")
	writePostFile(t, filepath.Join(posts, "a.md"), "dup", "A\n")
	writePostFile(t, filepath.Join(posts, "b.md"), "dup", "B\n")

	_, err := postPathForSlug(root, "dup")
	if err == nil {
		t.Fatal("ambiguous slug should error instead of picking one")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("error = %v, want it to say the slug is ambiguous", err)
	}
	for _, want := range []string{"a.md", "b.md"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %v, want it to name %s so the user can pick", err, want)
		}
	}
	// A unique slug in the same directory still resolves.
	p, err := postPathForSlug(root, "hello-tofu")
	if err != nil {
		t.Fatalf("unique slug should resolve: %v", err)
	}
	if filepath.Base(p) != "hello-tofu.md" {
		t.Errorf("path = %q, want hello-tofu.md", p)
	}
}

func TestPostPathForSlugMissing(t *testing.T) {
	root := scaffoldSite(t)
	_, err := postPathForSlug(root, "no-such")
	if err == nil || !strings.Contains(err.Error(), "no-such") {
		t.Fatalf("err = %v, want it to name the missing slug", err)
	}
}

// A path handed to the slug argument is a mistake: post.List would miss it
// with a confusing "no post with slug" message.
func TestPostPathForSlugRejectsPaths(t *testing.T) {
	root := scaffoldSite(t)
	abs := filepath.Join(root, "content", "posts", "hello-tofu.md")
	for _, slug := range []string{abs, "posts/hello-tofu", "..", "../escape", "./hello-tofu"} {
		if _, err := postPathForSlug(root, slug); err == nil {
			t.Errorf("postPathForSlug(%q) = nil error, want rejection", slug)
		} else if !strings.Contains(err.Error(), "not a slug") {
			t.Errorf("postPathForSlug(%q) err = %v, want it to explain the argument is a path", slug, err)
		}
	}
}

// The picker hands back a path, not a slug: a duplicated slug stays
// unambiguous because the row already knows its file.
func TestEditPickerPicksRowPath(t *testing.T) {
	root := scaffoldSite(t)
	posts := filepath.Join(root, "content", "posts")
	writePostFile(t, filepath.Join(posts, "a.md"), "dup", "A\n")
	writePostFile(t, filepath.Join(posts, "b.md"), "dup", "B\n")
	list, err := post.List(filepath.Join(root, "content"))
	if err != nil {
		t.Fatal(err)
	}
	picker := newEditPicker(list, false, normalize(Options{}))
	// post.List sorts date-descending, title-ascending, so the two duplicate
	// rows are not necessarily 0 and 1: find them by their file.
	rowA, rowB := -1, -1
	for i, r := range picker.table.Rows() {
		if len(r) < 2 || r[1] != "dup" || i >= len(picker.paths) {
			continue
		}
		if filepath.Base(picker.paths[i]) == "a.md" {
			rowA = i
		} else {
			rowB = i
		}
	}
	if rowA < 0 || rowB < 0 {
		t.Fatalf("duplicate-slug rows missing from the picker: %v", picker.paths)
	}

	picker.table.SetCursor(rowA)
	m, _ := picker.Update(enterKey())
	got := m.(editPickerModel)
	if !got.done {
		t.Fatal("enter should end the picker")
	}
	if got.picked == "" {
		t.Fatal("enter should pick a file")
	}
	if filepath.Base(got.picked) != "a.md" {
		t.Errorf("picked %q, want the row's own file a.md", got.picked)
	}
	// The pick is a real file, so it opens without a slug round trip.
	if _, err := os.Stat(got.picked); err != nil {
		t.Errorf("picked path is not a file: %v", err)
	}
	// The other row of the same slug picks the other file.
	picker.table.SetCursor(rowB)
	m, _ = picker.Update(enterKey())
	got = m.(editPickerModel)
	if filepath.Base(got.picked) != "b.md" {
		t.Errorf("picked %q, want the other duplicate row's file b.md", got.picked)
	}
}

func TestEditPickerEnterWithNoRows(t *testing.T) {
	picker := newEditPicker(nil, false, normalize(Options{}))
	m, _ := picker.Update(enterKey())
	got := m.(editPickerModel)
	if got.picked != "" {
		t.Errorf("picked = %q, want empty for an empty picker", got.picked)
	}
	if !got.done {
		t.Error("enter on an empty picker should still end the screen")
	}
}

// -----------------------------------------------------------------------------
// any .md browse: plain markdown opens body-only, broken frontmatter errors.
// -----------------------------------------------------------------------------

func TestEditWizardPlainMarkdownIsBodyOnly(t *testing.T) {
	root := scaffoldSite(t)
	plain := filepath.Join(root, "content", "notes", "scratch.md")
	if err := os.MkdirAll(filepath.Dir(plain), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := "# scratch\n\nno frontmatter here\n"
	if err := os.WriteFile(plain, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}

	w, err := newEditWizard(root, plain, normalize(Options{}), false)
	if err != nil {
		t.Fatalf("plain markdown should open for editing: %v", err)
	}
	if w.title != "edit scratch.md" {
		t.Errorf("wizard title = %q", w.title)
	}
	// Body mode choice, then the textarea: no frontmatter steps.
	if len(w.steps) != 2 {
		t.Fatalf("steps = %d, want body-only (choice + textarea)", len(w.steps))
	}
	if w.steps[0].kind != stepChoice || w.steps[1].kind != stepBody {
		t.Errorf("steps = %v, want choice then body", []stepKind{w.steps[0].kind, w.steps[1].kind})
	}
	// The whole file is the body, including any leading markdown.
	w.screen = "prompt"
	w.focusStep()
	if got := w.steps[1].area.Value(); got != orig {
		t.Errorf("body = %q, want the whole file", got)
	}
	if got := w.summary(w); !strings.Contains(got, "none") {
		t.Errorf("summary should say there is no frontmatter: %q", got)
	}

	// Quick edit writes the textarea back verbatim, inventing no frontmatter.
	if err := w.steps[0].setter("Quick edit (textarea)"); err != nil {
		t.Fatal(err)
	}
	w.steps[1].area.SetValue("# scratch\n\nedited\n")
	if err := w.finish(w); err != nil {
		t.Fatalf("finish: %v", err)
	}
	raw, err := os.ReadFile(plain)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "# scratch\n\nedited\n" {
		t.Errorf("file = %q, want the body written back verbatim", raw)
	}
}

// A file that claims frontmatter but has broken YAML is not "plain markdown":
// opening it body-only would hide its metadata inside its own body.
func TestEditWizardMalformedFrontmatterIsStrict(t *testing.T) {
	root := scaffoldSite(t)
	cases := map[string]string{
		"bad.yaml":  "---\ntitle: [unclosed\n---\nbody\n",
		"tabs.yaml": "---\ntitle: T\n\tdate: 2026-01-01\n---\nbody\n",
		"date.md":   "---\ntitle: T\ndate: not-a-date\n---\nbody\n",
		"fence.md":  "---\ntitle: T\nbody with no closing fence\n",
	}
	for name, src := range cases {
		p := filepath.Join(root, "content", "posts", name)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := newEditWizard(root, p, normalize(Options{}), false); err == nil {
			t.Errorf("%s: malformed frontmatter must error, not open body-only", name)
		}
	}
}

// A file with valid frontmatter still gets the full form, not body-only.
func TestEditWizardFrontmatterStillUsesFullForm(t *testing.T) {
	root := scaffoldSite(t)
	p := filepath.Join(root, "content", "posts", "f.md")
	writePostFile(t, p, "f", "hello\n")
	w, err := newEditWizard(root, p, normalize(Options{}), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.steps) < 5 {
		t.Errorf("steps = %d, want the full frontmatter form", len(w.steps))
	}
	if w.steps[0].label != "title" {
		t.Errorf("first step = %q, want title", w.steps[0].label)
	}
}

// -----------------------------------------------------------------------------
// progress: the pointer receiver keeps the animated percent.
// -----------------------------------------------------------------------------

func TestProgressSetPercentMutatesModel(t *testing.T) {
	p := newProgressModel(40)
	cmd := p.setPercent(0.75)
	if cmd == nil {
		t.Fatal("setPercent should return the animation cmd")
	}
	if got := p.m.Percent(); got != 0.75 {
		t.Errorf("Percent() = %v, want 0.75; a value receiver drops the mutation", got)
	}
	// The wizard's progress screen relies on that: set then read back.
	m := MenuModel{prog: newProgressModel(40)}
	_ = m.prog.setPercent(0.5)
	if got := m.prog.m.Percent(); got != 0.5 {
		t.Errorf("menu progress Percent() = %v, want 0.5", got)
	}
}

// -----------------------------------------------------------------------------
// joinLines: linear, exact.
// -----------------------------------------------------------------------------

func TestJoinLines(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"a"}, "a\n"},
		{[]string{"a", "b"}, "a\nb\n"},
		{[]string{"a", "", "c"}, "a\n\nc\n"},
		{[]string{"multi\nline"}, "multi\nline\n"},
	}
	for _, c := range cases {
		if got := joinLines(c.in); got != c.want {
			t.Errorf("joinLines(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// -----------------------------------------------------------------------------
// preview: the listener is bound before the URL is announced, a bind failure
// is reported instead of ignored, and stopping releases the port.
// -----------------------------------------------------------------------------

func newServeTestMenu(t *testing.T, root string, port int) *MenuModel {
	t.Helper()
	return &MenuModel{
		root:   root,
		port:   port,
		prog:   newProgressModel(40),
		vp:     newViewport(80, 14),
		slide:  NewSlide(8),
		spring: NewSlideSpring(),
	}
}

// newServeTestMenuWithList is newServeTestMenu with a properly constructed
// list, for tests that run the model's real Update loop.
func newServeTestMenuWithList(t *testing.T, root string, port int) *MenuModel {
	t.Helper()
	m := newServeTestMenu(t, root, port)
	m.opts = normalize(Options{})
	m.list = newList(menuItems(root), false, m.opts.Width, len(menuActions())+10)
	return m
}

// freePort returns a port that is free right now.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestMenuListenBindsPortAndAnnounces(t *testing.T) {
	root := scaffoldSite(t)
	port := freePort(t)
	m := newServeTestMenu(t, root, port)
	if err := m.listen(); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer m.stopServe()
	if m.srv == nil {
		t.Fatal("listen must keep the server")
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("port not accepting after listen: %v", err)
	}
	conn.Close()
	m.announceServe()
	if m.screen != screenViewport || m.vpTitle != "preview" {
		t.Errorf("screen=%q title=%q, want the preview screen", m.screen, m.vpTitle)
	}
}

func TestMenuListenTwiceReleasesTheOldPort(t *testing.T) {
	root := scaffoldSite(t)
	port := freePort(t)
	m := newServeTestMenu(t, root, port)
	if err := m.listen(); err != nil {
		t.Fatal(err)
	}
	// A second preview must rebind: the first listener has to be closed,
	// otherwise a second "Preview (serve)" reports success on a dead port.
	if err := m.listen(); err != nil {
		t.Fatalf("second listen: %v", err)
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("port not accepting after rebind: %v", err)
	}
	conn.Close()
	m.stopServe()
}

func TestMenuListenReportsBindError(t *testing.T) {
	root := scaffoldSite(t)
	port := freePort(t)
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	m := newServeTestMenu(t, root, port)
	err = m.listen()
	if err == nil {
		t.Fatal("listen on an occupied port must error")
	}
	if !strings.Contains(err.Error(), "preview") {
		t.Errorf("err = %v, want it to name preview", err)
	}
	if m.srv != nil {
		t.Error("a failed listen must not leave a server behind")
	}
}

func TestMenuStopServeReleasesThePort(t *testing.T) {
	root := scaffoldSite(t)
	port := freePort(t)
	m := newServeTestMenu(t, root, port)
	if err := m.listen(); err != nil {
		t.Fatal(err)
	}
	m.stopServe()
	if m.srv != nil {
		t.Error("stopServe must drop the server")
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("port still held after stopServe: %v", err)
	}
	ln.Close()
	// Idempotent: quitting twice must not panic.
	m.stopServe()
}

func TestMenuStopServeWithoutServer(t *testing.T) {
	m := newServeTestMenu(t, scaffoldSite(t), freePort(t))
	m.stopServe() // no server yet
	if err := m.listen(); err != nil {
		t.Fatal(err)
	}
	m.stopServe()
	m.stopServe()
}

// The preview screen binds before it announces, so a busy port fails back to
// the menu instead of showing a URL nothing serves.
func TestRunServeBindsThenAnnounces(t *testing.T) {
	root := scaffoldSite(t)
	port := freePort(t)
	m := newServeTestMenu(t, root, port)
	cmd := m.runServe()
	if cmd == nil {
		t.Fatal("runServe should return the progress commands")
	}
	if m.screen != screenProgress {
		t.Errorf("screen = %q, want progress", m.screen)
	}
	// The build runs in the worker; its result is what the progress screen
	// reacts to.
	if res, _ := runWorkCmd(t, cmd).(progressDoneMsg); res.err != nil {
		t.Fatalf("preview build failed: %v", res.err)
	}
	if _, err := os.Stat(filepath.Join(root, "public", "index.html")); err != nil {
		t.Errorf("preview build output missing: %v", err)
	}

	live := *m
	live.screen = screenProgress
	live.progMsg = "building for preview"
	mm, _ := live.Update(progressDoneMsg{})
	got := mm.(MenuModel)
	if got.srv == nil || got.ln == nil {
		t.Fatal("the preview screen must own a live listener")
	}
	if got.screen != screenViewport || got.vpTitle != "preview" {
		t.Errorf("screen=%q title=%q, want the preview screen", got.screen, got.vpTitle)
	}
	conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("announced port not accepting: %v", err)
	}
	conn.Close()
	got.stopServe()
}

func TestRunServeSurfacesBindError(t *testing.T) {
	root := scaffoldSite(t)
	port := freePort(t)
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	m := newServeTestMenu(t, root, port)
	live := *m
	live.screen = screenProgress
	live.progMsg = "building for preview"
	mm, _ := live.Update(progressDoneMsg{})
	got := mm.(MenuModel)
	if got.screen != screenList {
		t.Errorf("screen = %q, want back to the menu when the bind fails", got.screen)
	}
	if !strings.Contains(got.errorMsg, "preview") {
		t.Errorf("errorMsg = %q, want the bind error", got.errorMsg)
	}
	if got.srv != nil || got.ln != nil {
		t.Error("a failed bind must not leave a server behind")
	}
}

// The preview server is meant to outlive the menu screen, so it is stopped
// explicitly on Quit and ctrl+c. A program that ends any other way (the
// --timeout abort) still must not leave the port bound.
func TestRunProgramReleasesPreviewOnTimeout(t *testing.T) {
	root := scaffoldSite(t)
	port := freePort(t)
	m := newServeTestMenuWithList(t, root, port)
	if err := m.listen(); err != nil {
		t.Fatal(err)
	}
	if _, err := runProgram(*m, Options{
		NoAnimations: true,
		Timeout:      50 * time.Millisecond,
	}, headless()...); err != nil {
		t.Fatalf("runProgram: %v", err)
	}
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("preview port still bound after the program ended: %v", err)
	}
	ln.Close()
}

// Quit and ctrl+c stop the preview explicitly; the exit sweep must not
// double-close a server the quit already stopped.
func TestFinishProgramIsSafeWithoutAServer(t *testing.T) {
	m := newServeTestMenu(t, scaffoldSite(t), freePort(t))
	finishProgram(*m)
	finishProgram(MenuModel{})
}

// -----------------------------------------------------------------------------
// editor scratch files: registered while held, gone when the program ends.
// -----------------------------------------------------------------------------

// trackedTemp makes a real temp file and registers it.
func trackedTemp(t *testing.T, pattern, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return trackTempFile(f.Name())
}

func TestRemoveTempFilesDeletesOnlyTracked(t *testing.T) {
	tracked := trackedTemp(t, "tofu-clean-*.md", "draft")
	untracked := trackedTemp(t, "tofu-keep-*.md", "keep")
	forgetTempFile(untracked)
	defer os.Remove(untracked)

	removeTempFiles()
	if _, err := os.Stat(tracked); !os.IsNotExist(err) {
		t.Errorf("tracked temp %s survived: %v", tracked, err)
	}
	if _, err := os.Stat(untracked); err != nil {
		t.Errorf("untracked file %s was removed: %v", untracked, err)
	}
}

func TestRemoveTempFilesIsIdempotent(t *testing.T) {
	name := trackedTemp(t, "tofu-idem-*.md", "x")
	removeTempFiles()
	removeTempFiles() // second sweep must not panic or fail
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Errorf("temp %s survived: %v", name, err)
	}
}

func TestTrackForgetTempFileEmptyPaths(t *testing.T) {
	// A round trip that never made a temp file must not register "" and must
	// not try to remove it.
	trackTempFile("")
	forgetTempFile("")
	removeTempFiles()
}

func TestOpenInEditorRegistersTempFile(t *testing.T) {
	// With an unusable editor the round trip still owns a temp file: it must
	// be registered so program exit cleans it up.
	t.Setenv("EDITOR", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	done, ok := openInEditor("draft")().(editorDoneMsg)
	if !ok || done.runErr == nil {
		t.Fatalf("want an editor-unavailable editorDoneMsg, got %#v", done)
	}
	if done.tmp == "" {
		t.Fatal("the round trip must report the temp file it owns")
	}
	// Still registered: Update never ran.
	removeTempFiles()
	if _, err := os.Stat(done.tmp); !os.IsNotExist(err) {
		t.Errorf("temp %s not swept: %v", done.tmp, err)
	}
}

// A --timeout abort ends the program while it is suspended in the editor, so
// the round trip never reports back. runProgram must sweep the temp file.
func TestRunProgramSweepsTempFilesOnReturn(t *testing.T) {
	name := trackedTemp(t, "tofu-prog-*.md", "draft")
	if _, err := runProgram(quitImmediately{}, Options{NoAnimations: true}, headless()...); err != nil {
		t.Fatalf("runProgram: %v", err)
	}
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Errorf("temp %s survived program exit: %v", name, err)
	}
}

func TestRunProgramSweepsTempFilesOnTimeout(t *testing.T) {
	name := trackedTemp(t, "tofu-timeout-*.md", "draft")
	// A model that never quits on its own: only the timeout ends it.
	_, err := runProgram(hangModel{}, Options{
		NoAnimations: true,
		Timeout:      50 * time.Millisecond,
	}, headless()...)
	if err == nil {
		t.Log("timeout ended the program cleanly")
	}
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Errorf("temp %s survived the timeout abort: %v", name, err)
	}
}

// runWorkCmd pulls the progressDoneMsg out of a runTask batch: the work
// command, pumped to completion.
func runWorkCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			if msg, ok := c().(progressDoneMsg); ok {
				return msg
			}
		}
		t.Fatal("progress batch carried no progressDoneMsg")
	}
	t.Fatal("runTask did not return a batch")
	return nil
}

// headless runs a program with no renderer, no signal handling and no input,
// so a test needs no terminal. The update loop, the timeout and the exit path
// under test are the real ones.
func headless() []tea.ProgramOption {
	return []tea.ProgramOption{
		tea.WithoutRenderer(),
		tea.WithoutSignalHandler(),
		tea.WithoutSignals(),
		tea.WithInput(nil), // no TTY to open in a test
	}
}

// hangModel never quits by itself.
type hangModel struct{}

func (hangModel) Init() tea.Cmd { return nil }
func (hangModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return hangModel{}, nil
}
func (hangModel) View() tea.View { return tea.NewView("waiting") }

// quitImmediately quits from Init.
type quitImmediately struct{}

func (quitImmediately) Init() tea.Cmd { return tea.Quit }
func (quitImmediately) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return quitImmediately{}, nil
}
func (quitImmediately) View() tea.View { return tea.NewView("") }

// -----------------------------------------------------------------------------
// wizard: the round trip consumes and unregisters the scratch file.
// -----------------------------------------------------------------------------

func TestWizardEditorDoneUnregistersTempFile(t *testing.T) {
	w := newBodyWizard(t, "draft body")
	name := trackedTemp(t, "tofu-wiz-*.md", "edited body")
	m, _ := w.Update(editorDoneMsg{tmp: name})
	w = m.(*wizardModel)
	if got := w.current().area.Value(); got != "edited body" {
		t.Errorf("area = %q, want the edited body", got)
	}
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Errorf("scratch file not removed: %v", err)
	}
	// Consumed: a later program exit must not try to delete it again, and
	// must not delete a file a second ctrl+o happens to reuse.
	removeTempFiles()
}

func TestWizardEditorDoneErrorUnregistersTempFile(t *testing.T) {
	w := newBodyWizard(t, "draft body")
	name := trackedTemp(t, "tofu-wizerr-*.md", "x")
	m, _ := w.Update(editorDoneMsg{tmp: name, runErr: fmt.Errorf("boom")})
	w = m.(*wizardModel)
	if !strings.Contains(w.errorMsg, "boom") {
		t.Errorf("errorMsg = %q", w.errorMsg)
	}
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Errorf("failed round trip must still clean up: %v", err)
	}
	removeTempFiles()
}
