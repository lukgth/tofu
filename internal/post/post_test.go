package post

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writePost(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSlugify(t *testing.T) {
	cases := [][2]string{
		{"Hello World", "hello-world"},
		{"  Spaced  Out  ", "spaced-out"},
		{"Café & Crêpes!", "caf-cr-pes"},
		{"multiple---dashes", "multiple-dashes"},
		{"", ""},
		{"!!!", ""},
		{"already-slug-42", "already-slug-42"},
	}
	for _, c := range cases {
		if got := Slugify(c[0]); got != c[1] {
			t.Errorf("Slugify(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}

func TestParseFileDates(t *testing.T) {
	dir := t.TempDir()
	p, err := ParseFile(writePost(t, dir, "a.md", "---\ntitle: A\ndate: 2026-01-05\n---\nbody\n"))
	if err != nil {
		t.Fatal(err)
	}
	if p.Date.Format("2006-01-02") != "2026-01-05" {
		t.Fatalf("date = %v", p.Date)
	}
	if p.Slug != "a" {
		t.Fatalf("slug = %q", p.Slug)
	}
	p, err = ParseFile(writePost(t, dir, "b.md", "---\ntitle: B\ndate: \"2026-01-05T10:30:00Z\"\nslug: custom\n---\nbody\n"))
	if err != nil {
		t.Fatal(err)
	}
	if p.Slug != "custom" {
		t.Fatalf("slug = %q, want custom", p.Slug)
	}
	if p.Date.Format(time.RFC3339) == "" {
		t.Fatal("rfc3339 date lost")
	}
}

func TestParseFileMissingFrontmatter(t *testing.T) {
	_, err := ParseFile(writePost(t, t.TempDir(), "bad.md", "just text\n"))
	if err == nil || err.Error() == "" || !contains(err.Error(), "missing frontmatter") {
		t.Fatalf("want missing frontmatter error, got %v", err)
	}
}

func TestParseFileBadDateNamesFile(t *testing.T) {
	_, err := ParseFile(writePost(t, t.TempDir(), "x.md", "---\ntitle: X\ndate: not-a-date\n---\n"))
	if err == nil || !contains(err.Error(), "x.md") || !contains(err.Error(), "not-a-date") {
		t.Fatalf("want error naming file+value, got %v", err)
	}
}

func TestParseFileEmptySlugErrors(t *testing.T) {
	_, err := ParseFile(writePost(t, t.TempDir(), "---.md", "---\ntitle: '---'\n---\n"))
	if err == nil || !contains(err.Error(), "empty slug") {
		t.Fatalf("want empty slug error, got %v", err)
	}
}

func TestListSortOrderAndFilters(t *testing.T) {
	dir := t.TempDir()
	posts := filepath.Join(dir, "posts")
	os.MkdirAll(posts, 0o755)
	writePost(t, posts, "old.md", "---\ntitle: Old\ndate: 2025-01-01\n---\n")
	writePost(t, posts, "new.md", "---\ntitle: New\ndate: 2026-01-01\n---\n")
	writePost(t, posts, "tie-a.md", "---\ntitle: Alpha\ndate: 2026-01-02\n---\n")
	writePost(t, posts, "tie-b.md", "---\ntitle: Beta\ndate: 2026-01-02\n---\n")
	writePost(t, posts, "notes.txt", "not a post")
	os.MkdirAll(filepath.Join(posts, "sub"), 0o755)
	writePost(t, posts, "sub/nested.md", "---\ntitle: Nested\ndate: 2026-02-01\n---\n")

	got, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	wantOrder := []string{"tie-a", "tie-b", "new", "old"}
	if len(got) != len(wantOrder) {
		t.Fatalf("got %d posts %+v", len(got), got)
	}
	for i, w := range wantOrder {
		if got[i].Slug != w {
			t.Fatalf("pos %d slug = %q, want %q", i, got[i].Slug, w)
		}
	}
}

func TestListMissingDirEmpty(t *testing.T) {
	got, err := List(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty, got %d", len(got))
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
