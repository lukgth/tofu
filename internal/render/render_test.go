package render

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukgth/tofu/internal/post"
)

func scaffoldSite(t *testing.T) string {
	t.Helper()
	return scaffoldSiteIn(t, t.TempDir(), "")
}

// scaffoldSiteIn builds the fixture site under parent/name, so a test can place
// a sibling of the site to try to reach with a traversing path.
func scaffoldSiteIn(t *testing.T, parent, name string) string {
	t.Helper()
	root := filepath.Join(parent, name)
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tofu.toml", "title = \"Test Site\"\nbase_url = \"https://example.com\"\nrecent_count = 5\nlanguage = \"en\"\ndescription = \"desc\"\nfooter = \"foot\"\n[theme]\nfont_header_style = \"italic\"\nfont_header_weight = \"400\"\n[homepage]\nheading = \"Welcome\"\nbody_file = \"content/home.md\"\n")
	os.MkdirAll(filepath.Join(root, "content", "posts"), 0o755)
	write("content/home.md", "# hello!\n")
	return root
}

// setBodyFile repoints [homepage] body_file in the fixture's tofu.toml.
func setBodyFile(t *testing.T, root, value string) {
	t.Helper()
	p := filepath.Join(root, "tofu.toml")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	flipped := strings.Replace(string(b), `body_file = "content/home.md"`, fmt.Sprintf("body_file = %q", value), 1)
	if flipped == string(b) {
		t.Fatal("body_file not found in the fixture config")
	}
	if err := os.WriteFile(p, []byte(flipped), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePostFile(t *testing.T, root, name, fm string) {
	t.Helper()
	p := filepath.Join(root, "content", "posts", name)
	if err := os.WriteFile(p, []byte("---\n"+fm+"\n---\n\nHello **world** body.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildGoldenIndex(t *testing.T) {
	root := scaffoldSite(t)
	for i := 1; i <= 6; i++ {
		writePostFile(t, root, fmt.Sprintf("p%d.md", i),
			fmt.Sprintf("title: P%d\ndate: 2026-01-%02d\n", i, i))
	}
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	idx, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(idx)
	if !strings.Contains(s, "<ul class=\"blog-posts\">") {
		t.Error("index missing ul.blog-posts")
	}
	if n := strings.Count(s, "<li>"); n != 5 {
		t.Errorf("index has %d <li>, want 5 (recent_count)", n)
	}
	// p6 is newest; recent_count=5 shows p6..p2 and drops the oldest, p1.
	if !strings.Contains(s, "/articles/p6.html") || strings.Contains(s, "/articles/p1.html") {
		t.Error("recent list should include p6..p2, exclude oldest p1")
	}
	if !strings.Contains(s, "06 Jan, 2026") {
		t.Errorf("date format 02 Jan, 2006 missing; got: %.300s", s)
	}
	post, err := os.ReadFile(filepath.Join(out, "articles", "p6.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(post), "<i><time") {
		t.Error("post page missing <i><time> italic date wrapper")
	}
}

func TestBuildOutputTree(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "hello.md", "title: Hello\ndate: 2026-03-01\n")
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{
		"index.html", "articles/index.html", "articles/hello.html",
		"assets-blog/style.css", "assets-blog/theme-and-visited.js", "feed.xml",
	} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
	css, _ := os.ReadFile(filepath.Join(out, "assets-blog", "style.css"))
	s := string(css)
	if !strings.Contains(s, "--width") || !strings.Contains(s, "light-dark(") {
		t.Error("style.css missing theme vars or light-dark() values")
	}
	if strings.Contains(s, "prefers-color-scheme") || strings.Contains(s, "html.dark") {
		t.Error("style.css must not use class/media dark scoping")
	}
	if !strings.Contains(string(css), "@font-face") || !strings.Contains(string(css), "Rubik") {
		t.Error("style.css missing @font-face Rubik")
	}
	if !strings.Contains(s, ".title {\n  font-family: var(--font-header, var(--font-secondary));") {
		t.Error("style.css missing .title header-font rule")
	}
	if strings.Contains(string(css), "cursor-blink") {
		t.Error("style.css must not have cursor-blink keyframes")
	}
	if !strings.Contains(string(css), "font-style: italic") {
		t.Error("style.css missing italic time")
	}
}

func TestFooterRendersHTML(t *testing.T) {
	root := scaffoldSite(t)
	os.WriteFile(filepath.Join(root, "tofu.toml"), []byte("title = \"Test Site\"\nbase_url = \"https://example.com\"\nrecent_count = 5\nlanguage = \"en\"\ndescription = \"desc\"\nfooter = \"powered by <a href='https://github.com/lukgth/tofu'>tofu</a>\"\n[homepage]\nheading = \"Welcome\"\nbody_file = \"content/home.md\"\n"), 0o644)
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	idx, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(idx), "powered by <a href='https://github.com/lukgth/tofu'>tofu</a>") {
		t.Errorf("footer link got escaped or lost: %.300s", idx)
	}
}

// base.html must load behaviour from theme-and-visited.js rather than inline scripts;
// the pre-paint theme boot is the only intentional exception.
func TestScriptsAreExternal(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "p.md", "title: P\ndate: 2026-01-01\n")
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	idx, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(idx)
	if !strings.Contains(s, `<script defer src="/assets-blog/theme-and-visited.js"></script>`) {
		t.Error("index does not link the deferred theme-and-visited.js")
	}
	if strings.Contains(s, `getElementById("theme-toggle")`) {
		t.Error("toggle logic must live in theme-and-visited.js, not inline")
	}
	if _, err := os.Stat(filepath.Join(out, "assets-blog", "theme-and-visited.js")); err != nil {
		t.Error("theme-and-visited.js not emitted into assets-blog/")
	}
}

// A channel description alone is not a feed: each post needs a real <item>
// with title/link, and a stray </item> would make the XML ill-formed.
func TestFeedEmitsItems(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "p.md", "title: Hello & <World>\ndate: 2026-01-02\n")
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(out, "feed.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var feed struct {
		Channel struct {
			Items []struct {
				Title string `xml:"title"`
				Link  string `xml:"link"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(raw, &feed); err != nil {
		t.Fatalf("feed.xml is not well-formed: %v\n%s", err, raw)
	}
	if len(feed.Channel.Items) != 1 {
		t.Fatalf("feed has %d items, want 1", len(feed.Channel.Items))
	}
	it := feed.Channel.Items[0]
	if it.Title != "Hello & <World>" {
		t.Errorf("item title = %q, want escaped-then-decoded title", it.Title)
	}
	if it.Link != "https://example.com/articles/p.html" {
		t.Errorf("item link = %q", it.Link)
	}
}

func TestBuildEmptyPosts(t *testing.T) {
	root := scaffoldSite(t)
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	s, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(s), "<ul") {
		t.Error("empty site must not render <ul>")
	}
	if !strings.Contains(string(s), "no posts yet.") {
		t.Error("empty site must show No posts yet.")
	}
}

func TestBuildHidesRecentWhenZero(t *testing.T) {
	root := scaffoldSite(t)
	// scaffoldSite writes recent_count = 5; flip it to the hide sentinel.
	tomlPath := filepath.Join(root, "tofu.toml")
	b, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatal(err)
	}
	flipped := strings.Replace(string(b), "recent_count = 5", "recent_count = 0", 1)
	if err := os.WriteFile(tomlPath, []byte(flipped), 0o644); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "content", "home.md")
	if err := os.WriteFile(home, []byte("just the body\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writePostFile(t, root, "p1.md", "title: P1\ndate: 2026-01-01\n")
	writePostFile(t, root, "p2.md", "title: P2\ndate: 2026-01-02\n")
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	idx, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(idx)
	if !strings.Contains(s, "just the body") {
		t.Error("home body missing from hidden index")
	}
	for _, banned := range []string{`<h1>recent posts</h1>`, `<ul class="blog-posts">`, "/articles/p", "no posts yet."} {
		if strings.Contains(s, banned) {
			t.Errorf("hidden index must not contain %q", banned)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "articles", "index.html")); err != nil {
		t.Error("blog output must still build when recents are hidden")
	}
}

func TestBuildDuplicateSlugsFails(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "a.md", "title: A\ndate: 2026-01-01\n")
	writePostFile(t, root, "b.md", "title: B\ndate: 2026-01-02\n")
	// a.md has slug "a"; force collision via frontmatter slug
	p := filepath.Join(root, "content", "posts", "b.md")
	os.WriteFile(p, []byte("---\ntitle: B\ndate: 2026-01-02\nslug: a\n---\nbody\n"), 0o644)
	err := Build(root, filepath.Join(t.TempDir(), "public"), false)
	if err == nil || !strings.Contains(err.Error(), "duplicate slug") {
		t.Fatalf("want duplicate slug error, got %v", err)
	}
}

// The duplicate report must name the two colliding posts, not an unrelated
// earlier post (the index bookkeeping once pointed at the wrong entry).
func TestDuplicateSlugsNamesBothPosts(t *testing.T) {
	posts := []post.Post{
		{Frontmatter: post.Frontmatter{Title: "First A"}, Slug: "a"},
		{Frontmatter: post.Frontmatter{Title: "First B"}, Slug: "b"},
		{Frontmatter: post.Frontmatter{Title: "Second B"}, Slug: "b"},
		{Frontmatter: post.Frontmatter{Title: "Second A"}, Slug: "a"},
	}
	dupes := duplicateSlugs(posts)
	if len(dupes) != 2 {
		t.Fatalf("got %d dupes, want 2", len(dupes))
	}
	want := map[string][2]string{
		"b": {"First B", "Second B"},
		"a": {"First A", "Second A"},
	}
	for _, d := range dupes {
		if d.files != want[d.slug] {
			t.Errorf("slug %q reported %v, want %v", d.slug, d.files, want[d.slug])
		}
	}
}

func TestBuildDraftsExcludedUnlessAsked(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "pub.md", "title: Pub\ndate: 2026-01-01\n")
	writePostFile(t, root, "d.md", "title: D\ndate: 2026-01-02\ndraft: true\n")
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "articles", "d.html")); !os.IsNotExist(err) {
		t.Error("draft must be excluded by default")
	}
	if err := Build(root, out, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "articles", "d.html")); err != nil {
		t.Error("draft must be included with includeDrafts")
	}
}

func TestSiteTitleHeader(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "p.md", "title: P\ndate: 2026-01-01\n")
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}

	idx, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(idx)
	t.Run("index has site title h1 and squiggle", func(t *testing.T) {
		if !strings.Contains(s, `<h1 class="title"><a href="/">Test Site</a></h1>`) {
			t.Errorf("index missing site title h1; got: %.400s", s)
		}
		if !strings.Contains(s, `class="squiggle"`) {
			t.Error("index missing squiggle div")
		}
	})
	t.Run("post page has site title h1 too", func(t *testing.T) {
		html, err := os.ReadFile(filepath.Join(out, "articles", "p.html"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(html), `<h1 class="title"><a href="/">Test Site</a></h1>`) {
			t.Error("post page missing site title h1")
		}
	})
	t.Run("style.css time rule is italic without monospace", func(t *testing.T) {
		css, err := os.ReadFile(filepath.Join(out, "assets-blog", "style.css"))
		if err != nil {
			t.Fatal(err)
		}
		c := string(css)
		timeRule := timeRuleCSS(c)
		if !strings.Contains(timeRule, "font-style: italic") {
			t.Errorf("time rule missing font-style italic; got: %s", timeRule)
		}
		if strings.Contains(timeRule, "monospace") {
			t.Error("time rule must not set a monospace font-family")
		}
		if !strings.Contains(c, "h1, h2, h3, h4, h5, h6 {\n  font-family: var(--font-secondary)") {
			t.Error("style.css headings must use Rubik (--font-secondary)")
		}
	})
	t.Run("style.css header title uses font_header", func(t *testing.T) {
		css, err := os.ReadFile(filepath.Join(out, "assets-blog", "style.css"))
		if err != nil {
			t.Fatal(err)
		}
		c := string(css)
		if !strings.Contains(c, ".title {\n  font-family: var(--font-header, var(--font-secondary));") {
			t.Error("style.css .title must use --font-header")
		}
		if !strings.Contains(c, `--font-header: "Georgia", "Gelasio", serif`) {
			t.Errorf("style.css --font-header missing default; got: %.400s", c)
		}
		if !strings.Contains(c, "--font-header-style: italic") || !strings.Contains(c, "--font-header-weight: 400") {
			t.Errorf("configured header style/weight not rendered; got: %.400s", c)
		}
		if !strings.Contains(c, `font-family: "Gelasio"`) {
			t.Error("style.css missing Gelasio @font-face")
		}
	})
}

// timeRuleCSS extracts the top-level `time { ... }` rule body from CSS.
func timeRuleCSS(css string) string {
	start := strings.Index(css, "\ntime {")
	if start < 0 {
		return ""
	}
	rest := css[start+len("\ntime {"):]
	end := strings.Index(rest, "}")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func TestBuildHomeBodyAndStaticCopy(t *testing.T) {
	root := scaffoldSite(t)
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), 0o644)
	}
	write("content/home.md", "# Welcome\n\n*home body*\n")
	write("assets-blog/custom.css", "/* custom */\n")
	write("static/pic.txt", "pic\n")
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	idx, _ := os.ReadFile(filepath.Join(out, "index.html"))
	if !strings.Contains(string(idx), "<em>home body</em>") {
		t.Error("home.md not rendered into index")
	}
	if !strings.Contains(string(idx), `href="/assets-blog/custom.css"`) {
		t.Error("index does not link custom.css")
	}
	if _, err := os.Stat(filepath.Join(out, "assets-blog", "custom.css")); err != nil {
		t.Error("custom.css not copied")
	}
	if _, err := os.Stat(filepath.Join(out, "pic.txt")); err != nil {
		t.Error("static/ not copied")
	}
}

// A site without assets-blog/custom.css must not link a stylesheet that
// isn't there (it would 404).
func TestCustomCSSLinkOnlyWhenPresent(t *testing.T) {
	root := scaffoldSite(t)
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	idx, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(idx), "custom.css") {
		t.Error("index links custom.css though none exists")
	}
}

func TestMarkdownHardWraps(t *testing.T) {
	got := MarkdownToHTML("here's my:\ndiscord username: `luk_`\nmy email: me (at) lukgth.cloud\ni host @ [nyara.cloud](https://nyara.cloud)\n")
	for _, want := range []string{"here&rsquo;s my:<br>", "<code>luk_</code><br>", "<a href=\"https://nyara.cloud\">nyara.cloud</a></p>"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "<p>line one\nline two") {
		t.Error("soft breaks were joined")
	}
	if got := MarkdownToHTML("a\n\nb\n"); !strings.Contains(got, "<p>a</p>\n<p>b</p>") {
		t.Errorf("blank-line paragraphs broken: %q", got)
	}
}

// Tags are untrusted frontmatter: a traversal-looking tag must not escape the
// tag output directory, and the post's tag link must match the emitted file.
func TestTagNamesAreSanitized(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "p.md", "title: P\ndate: 2026-01-01\ntags: [\"../escape\", \"Hello World\"]\n")
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "articles", "escape.html")); err == nil {
		t.Error("tag ../escape escaped the tag directory")
	}
	post, err := os.ReadFile(filepath.Join(out, "articles", "p.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(post), `href="/articles/tag/escape.html"`) {
		t.Errorf("post missing sanitized tag link: %.400s", post)
	}
	if !strings.Contains(string(post), `href="/articles/tag/hello-world.html">Hello World</a>`) {
		t.Error("tag link should use the slug but keep the label")
	}
	for _, slug := range []string{"escape", "hello-world"} {
		if _, err := os.Stat(filepath.Join(out, "articles", "tag", slug+".html")); err != nil {
			t.Errorf("tag page %q missing", slug)
		}
	}
}

// A static/ symlink resolving outside the site must not leak into the output.
func TestStaticSymlinkEscapeRejected(t *testing.T) {
	root := scaffoldSite(t)
	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("TOP SECRET\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "static", "leak")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	err := Build(root, filepath.Join(t.TempDir(), "public"), false)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("want symlink-escape error, got %v", err)
	}
}

// A leading body heading that repeats the title but carries inline markup must
// still be stripped, so the post does not render two visible titles.
func TestStripDuplicateTitleWithInlineMarkup(t *testing.T) {
	root := scaffoldSite(t)
	os.WriteFile(filepath.Join(root, "content", "posts", "dup.md"),
		[]byte("---\ntitle: Hello\ndate: 2026-01-01\n---\n\n# *Hello*\n\ntext\n"), 0o644)
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "articles", "dup.html"))
	if err != nil {
		t.Fatal(err)
	}
	// Exactly two <h1>: the header site title and the post template's title.
	if n := strings.Count(string(b), "<h1"); n != 2 {
		t.Errorf("post has %d <h1>, want 2 (duplicate heading not stripped)", n)
	}
}

// The font_scale knob and reduced-motion overrides must reach the output CSS.
func TestCSSKnobsAndReducedMotion(t *testing.T) {
	root := scaffoldSite(t)
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	css, err := os.ReadFile(filepath.Join(out, "assets-blog", "style.css"))
	if err != nil {
		t.Fatal(err)
	}
	c := string(css)
	if !strings.Contains(c, "font-size: var(--font-scale") {
		t.Error("font_scale knob is not consumed (no var(--font-scale) rule)")
	}
	i := strings.Index(c, "prefers-reduced-motion")
	if i < 0 {
		t.Fatal("missing prefers-reduced-motion block")
	}
	block := c[i:]
	if !strings.Contains(block, "transition: none") || !strings.Contains(block, "transform: none") {
		t.Error("reduced-motion must also disable hover transitions/transforms")
	}
}

// A rebuild must not leave pages from renamed or deleted posts behind, so the
// output directory is emptied before the new tree is written.
func TestBuildReplacesStaleOutput(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "old.md", "title: Old\ndate: 2026-01-01\n")
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(out, "articles", "old.html")
	if _, err := os.Stat(stale); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(out, "orphan.html")
	if err := os.WriteFile(orphan, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(filepath.Join(root, "content", "posts", "old.md")); err != nil {
		t.Fatal(err)
	}
	writePostFile(t, root, "new.md", "title: New\ndate: 2026-01-02\n")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("page of a removed post survived the rebuild")
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Error("unrelated file from a previous build survived the rebuild")
	}
	if _, err := os.Stat(filepath.Join(out, "articles", "new.html")); err != nil {
		t.Errorf("new post not written: %v", err)
	}
}

// The output target is checked before anything is deleted or written, so a
// rejected build leaves the previous output and the sources intact.
func TestBuildRejectsOverlappingOutput(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "p.md", "title: P\ndate: 2026-01-01\n")
	keep := filepath.Join(root, "content", "posts", "p.md")
	before, err := os.ReadFile(keep)
	if err != nil {
		t.Fatal(err)
	}
	for _, out := range []string{
		root,                                    // the site itself
		filepath.Join(root, ".."),               // a parent of the site
		filepath.Join(root, "content"),          // a source directory
		filepath.Join(root, "content", "posts"), // the posts source
		filepath.Join(root, "static"),           // the static source
		filepath.Join(root, "assets-blog"),      // the custom.css source
		filepath.Join(root, "tofu.toml"),        // a file, not a directory
	} {
		if err := Build(root, out, false); err == nil {
			t.Errorf("Build accepted overlapping output %q", out)
		}
	}
	after, err := os.ReadFile(keep)
	if err != nil {
		t.Fatalf("source post was destroyed: %v", err)
	}
	if string(before) != string(after) {
		t.Error("rejected build modified a source file")
	}
	if _, err := os.Stat(filepath.Join(root, "tofu.toml")); err != nil {
		t.Errorf("rejected build destroyed tofu.toml: %v", err)
	}
}

// A rejected target must not clear the previous output, otherwise the safety
// check itself costs the user a working site.
func TestBuildRejectionLeavesPreviousOutput(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "p.md", "title: P\ndate: 2026-01-01\n")
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	idx := filepath.Join(out, "index.html")
	before, err := os.ReadFile(idx)
	if err != nil {
		t.Fatal(err)
	}
	if err := Build(root, filepath.Join(root, "content"), false); err == nil {
		t.Fatal("overlapping output was accepted")
	}
	after, err := os.ReadFile(idx)
	if err != nil {
		t.Fatalf("previous output was destroyed by the rejected build: %v", err)
	}
	if string(before) != string(after) {
		t.Error("rejected build rewrote the previous output")
	}
}

// The default output lives inside the site (public/); that must keep working,
// and a nested, not-yet-existing path must resolve through its nearest
// existing ancestor.
func TestBuildIntoNestedSiteSubdir(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "p.md", "title: P\ndate: 2026-01-01\n")
	out := filepath.Join(root, "public", "site", "deep")
	if err := Build(root, out, false); err != nil {
		t.Fatalf("build into a nested in-site dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "index.html")); err != nil {
		t.Errorf("index.html not written: %v", err)
	}
}

// resolvePath must follow existing symlinks and resolve a path that does not
// exist yet against its nearest existing ancestor.
func TestResolvePathResolvesNearestAncestor(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	got, err := resolvePath(filepath.Join(link, "missing", "deep"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(real, "missing", "deep")
	if got != want {
		t.Errorf("resolvePath = %q, want %q", got, want)
	}
	// An existing symlink must be followed, not reported verbatim.
	got, err = resolvePath(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != real {
		t.Errorf("resolvePath(%q) = %q, want %q", link, got, real)
	}
}

// body_file is site-relative: a path escaping the site must be rejected with
// the configured name in the message.
func TestHomepageBodyFileMustStayInsideSite(t *testing.T) {
	// The secret is a sibling of the site, so ../secret.md really exists: the
	// rejection must come from the escape check, not from a missing file.
	parent := t.TempDir()
	secret := filepath.Join(parent, "secret.md")
	if err := os.WriteFile(secret, []byte("TOP SECRET\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := scaffoldSiteIn(t, parent, "site")
	setBodyFile(t, root, "../secret.md")
	out := filepath.Join(t.TempDir(), "public")
	err := Build(root, out, false)
	if err == nil {
		t.Fatal("body_file outside the site was accepted")
	}
	if !strings.Contains(err.Error(), "body_file") {
		t.Errorf("error lacks the configured name: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "index.html")); !os.IsNotExist(err) {
		t.Error("a homepage was written despite the rejection")
	}
}

// A body_file pointing at a symlink that leaves the site must be rejected too.
func TestHomepageBodyFileSymlinkEscapeRejected(t *testing.T) {
	root := scaffoldSite(t)
	secret := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(secret, []byte("TOP SECRET\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "content", "home.md")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	err := Build(root, filepath.Join(t.TempDir(), "public"), false)
	if err == nil || !strings.Contains(err.Error(), "body_file") {
		t.Fatalf("want body_file rejection, got %v", err)
	}
}

// A body_file naming a directory is a config mistake worth reporting.
func TestHomepageBodyFileMustBeRegularFile(t *testing.T) {
	root := scaffoldSite(t)
	if err := os.Remove(filepath.Join(root, "content", "home.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "content", "home.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := Build(root, filepath.Join(t.TempDir(), "public"), false)
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("want not-a-regular-file error, got %v", err)
	}
}

// A configured but missing home.md is an error; an empty body_file disables
// the body entirely.
func TestHomepageBodyFileRequired(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		root := scaffoldSite(t)
		if err := os.Remove(filepath.Join(root, "content", "home.md")); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(t.TempDir(), "public")
		if err := Build(root, out, false); err == nil || !strings.Contains(err.Error(), "body_file") {
			t.Fatalf("missing configured home.md error = %v, want body_file error", err)
		}
		if _, err := os.Stat(filepath.Join(out, "index.html")); !os.IsNotExist(err) {
			t.Errorf("build emitted index despite missing configured body: %v", err)
		}
	})
	t.Run("empty file", func(t *testing.T) {
		root := scaffoldSite(t)
		if err := os.WriteFile(filepath.Join(root, "content", "home.md"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := Build(root, filepath.Join(t.TempDir(), "public"), false); err != nil {
			t.Fatalf("empty existing body must be valid: %v", err)
		}
	})
	t.Run("empty config", func(t *testing.T) {
		root := scaffoldSite(t)
		setBodyFile(t, root, "")
		if err := os.WriteFile(filepath.Join(root, "content", "home.md"), []byte("ignored\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(t.TempDir(), "public")
		if err := Build(root, out, false); err != nil {
			t.Fatalf("empty body_file must not fail the build: %v", err)
		}
		idx, err := os.ReadFile(filepath.Join(out, "index.html"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(idx), "ignored") {
			t.Error("empty body_file must disable the homepage body")
		}
	})
}

// articles/index.html is the generated archive; a post claiming that slug must
// be rejected before anything is written.
func TestBuildRejectsIndexSlug(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "index.md", "title: Home\ndate: 2026-01-01\n")
	out := filepath.Join(t.TempDir(), "public")
	err := Build(root, out, false)
	if err == nil {
		t.Fatal("post with the reserved index slug was accepted")
	}
	if !strings.Contains(err.Error(), "index") {
		t.Errorf("error does not name the reserved slug: %v", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Error("rejected build created the output directory")
	}
}

// The reserved-slug check runs before the output is cleared, so a previous
// build's archive is not left half-overwritten.
func TestIndexSlugRejectedBeforeWriting(t *testing.T) {
	root := scaffoldSite(t)
	writePostFile(t, root, "p.md", "title: P\ndate: 2026-01-01\n")
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(out, "articles", "index.html")
	before, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	writePostFile(t, root, "index.md", "title: Home\ndate: 2026-01-02\n")
	if err := Build(root, out, false); err == nil {
		t.Fatal("reserved index slug was accepted")
	}
	after, err := os.ReadFile(archive)
	if err != nil {
		t.Fatalf("archive was destroyed by the rejected build: %v", err)
	}
	if string(before) != string(after) {
		t.Error("the post archive was overwritten by the colliding post")
	}
}

// static/ may itself be a symlink to a directory; its contents are still
// copied.
func TestStaticRootSymlinkToDirectory(t *testing.T) {
	root := scaffoldSite(t)
	elsewhere := t.TempDir()
	if err := os.WriteFile(filepath.Join(elsewhere, "pic.txt"), []byte("pic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(root, "static")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatalf("static symlinked to a directory must build: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(out, "pic.txt"))
	if err != nil {
		t.Fatalf("contents of the linked static dir not copied: %v", err)
	}
	if string(b) != "pic\n" {
		t.Errorf("copied content = %q, want %q", b, "pic\n")
	}
}

// A symlink to a file inside the tree is copied as a plain file at its alias
// path; one pointing at a directory is skipped, not recursed into.
func TestStaticNestedSymlinks(t *testing.T) {
	root := scaffoldSite(t)
	staticDir := filepath.Join(root, "static", "sub")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "static", "real.txt"), []byte("real\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "real.txt"), filepath.Join(staticDir, "alias.txt")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "sub", "alias.txt"))
	if err != nil {
		t.Fatalf("in-tree file symlink not copied at its alias path: %v", err)
	}
	if string(b) != "real\n" {
		t.Errorf("alias content = %q, want %q", b, "real\n")
	}
}

// A symlink to a directory would drag in a whole subtree WalkDir never
// visited; it is skipped instead.
func TestStaticNestedDirectorySymlinkSkipped(t *testing.T) {
	root := scaffoldSite(t)
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(filepath.Join(staticDir, "shared", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "shared", "deep", "big.txt"), []byte("big\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("shared"), filepath.Join(staticDir, "link")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "link")); !os.IsNotExist(err) {
		t.Error("symlinked directory was copied instead of skipped")
	}
	// The real directory is still copied on its own.
	if _, err := os.Stat(filepath.Join(out, "shared", "deep", "big.txt")); err != nil {
		t.Errorf("real in-tree directory not copied: %v", err)
	}
}

// A symlink with no target cannot be read, and silently skipping it would
// ship a 404; the build reports it instead.
func TestStaticBrokenSymlinkRejected(t *testing.T) {
	root := scaffoldSite(t)
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(staticDir, "gone.txt"), filepath.Join(staticDir, "dangling")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	err := Build(root, filepath.Join(t.TempDir(), "public"), false)
	if err == nil || !strings.Contains(err.Error(), "dangling") {
		t.Fatalf("want broken-symlink error naming the link, got %v", err)
	}
}
