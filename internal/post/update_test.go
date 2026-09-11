package post

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateFrontmatterPreservesUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.md")
	src := "---\ntitle: Old\ndate: 2026-01-01\nfavorite_snack: tofu\n---\n\n# Body\n\nkept.\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	err := UpdateFrontmatter(path, func(f *Frontmatter) error {
		f.Title = "New"
		f.Tags = []string{"a", "b"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	s := string(b)
	if !strings.Contains(s, "title: New") {
		t.Errorf("title not updated:\n%s", s)
	}
	if !strings.Contains(s, "favorite_snack: tofu") {
		t.Errorf("unknown key lost:\n%s", s)
	}
	if !strings.Contains(s, "# Body") {
		t.Errorf("body lost:\n%s", s)
	}
	if !strings.Contains(s, "- a") {
		t.Errorf("tags not written:\n%s", s)
	}
}

func TestUpdateFrontmatterSetsAndClearsDraft(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.md")
	os.WriteFile(path, []byte("---\ntitle: T\ndate: 2026-01-01\n---\nbody\n"), 0o644)
	if err := UpdateFrontmatter(path, func(f *Frontmatter) error { f.Draft = true; return nil }); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "draft: true") {
		t.Errorf("draft not set:\n%s", b)
	}
	if err := UpdateFrontmatter(path, func(f *Frontmatter) error { f.Draft = false; return nil }); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(path)
	if strings.Contains(string(b), "draft: true") {
		t.Errorf("draft not cleared:\n%s", b)
	}
}
