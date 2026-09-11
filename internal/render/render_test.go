package render

import (
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
	write("tofu.toml", "title = \"Test Site\"\nbase_url = \"https://example.com\"\nrecent_count = 5\nlanguage = \"en\"\ndescription = \"desc\"\nfooter = \"foot\"\n[homepage]\nheading = \"Welcome\"\nbody_file = \"content/home.md\"\n")
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
		"assets-blog/style.css", "feed.xml",
	} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
	css, _ := os.ReadFile(filepath.Join(out, "assets-blog", "style.css"))
	if !strings.Contains(string(css), "--width") || !strings.Contains(string(css), "prefers-color-scheme") {
		t.Error("style.css missing theme vars or dark media query")
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
	if !strings.Contains(string(s), "No posts yet.") {
		t.Error("empty site must show No posts yet.")
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
	if _, err := os.Stat(filepath.Join(out, "assets-blog", "custom.css")); err != nil {
		t.Error("custom.css not copied")
	}
	if _, err := os.Stat(filepath.Join(out, "pic.txt")); err != nil {
		t.Error("static/ not copied")
	}
}
