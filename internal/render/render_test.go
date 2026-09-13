package render

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func scaffoldSite(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tofu.toml", "title = \"Test Site\"\nbase_url = \"https://example.com\"\nrecent_count = 5\nlanguage = \"en\"\ndescription = \"desc\"\nfooter = \"foot\"\n[theme]\nfont_header_style = \"italic\"\nfont_header_weight = \"400\"\n[homepage]\nheading = \"Welcome\"\nbody_file = \"content/home.md\"\n")
	os.MkdirAll(filepath.Join(root, "content", "posts"), 0o755)
	return root
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
