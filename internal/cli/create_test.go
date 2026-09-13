package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/lukgth/tofu/internal/post"
)

func TestHeadlessCommandsRejectExtraArgs(t *testing.T) {
	for _, c := range []*cobra.Command{newVersionCmd(), newNewCmd(), newBuildCmd(), newListCmd(), newServeCmd()} {
		if c.Args == nil {
			t.Errorf("%s has no Args validator", c.Name())
			continue
		}
		if err := c.Args(c, []string{"stray"}); err == nil {
			t.Errorf("%s should reject positional args", c.Name())
		}
	}
}

func TestCreatePostAndCollision(t *testing.T) {
	root := t.TempDir()
	rel, err := CreatePost(root, NewPostInput{Title: "Hello World", Date: "2026-02-03"})
	if err != nil {
		t.Fatal(err)
	}
	if rel != filepath.Join("content", "posts", "hello-world.md") {
		t.Fatalf("rel = %q", rel)
	}
	b, _ := os.ReadFile(filepath.Join(root, rel))
	if !strings.Contains(string(b), "title: \"Hello World\"") || !strings.Contains(string(b), "date: 2026-02-03") {
		t.Errorf("bad frontmatter:\n%s", b)
	}
	if strings.Contains(string(b), "# Hello World") {
		t.Errorf("starter body must not repeat the title:\n%s", b)
	}
	if !strings.Contains(string(b), "write your post here.") {
		t.Errorf("missing starter body:\n%s", b)
	}
	_, err = CreatePost(root, NewPostInput{Title: "Hello World"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("want collision error, got %v", err)
	}
}

func TestCreatePostRejectsUnsafeSlug(t *testing.T) {
	root := t.TempDir()
	if _, err := CreatePost(root, NewPostInput{Title: "X", Slug: "../../escape"}); err == nil {
		t.Fatal("want invalid slug error")
	}
	if _, err := os.Stat(filepath.Join(root, "escape.md")); err == nil {
		t.Error("traversal slug created a file outside content/posts")
	}
}

// Tags are free text; YAML-significant ones must be quoted or the frontmatter
// stops parsing and every command fails on the whole site.
func TestCreatePostQuotesYAMLTags(t *testing.T) {
	root := t.TempDir()
	if _, err := CreatePost(root, NewPostInput{
		Title: "T", Date: "2026-01-01",
		Tags: []string{"a: b", "*star", "#hash", "- dash"},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := post.List(filepath.Join(root, "content"))
	if err != nil {
		t.Fatalf("tagged post does not parse back: %v", err)
	}
	want := []string{"a: b", "*star", "#hash", "- dash"}
	if len(got) != 1 || !reflect.DeepEqual(got[0].Frontmatter.Tags, want) {
		t.Fatalf("tags round-trip = %v, want %v", got[0].Frontmatter.Tags, want)
	}
}

func TestCreatePostWritesAsset(t *testing.T) {
	root := t.TempDir()
	rel, err := CreatePost(root, NewPostInput{Title: "T", Date: "2026-01-01", AssetPath: "img/cover.png"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, rel))
	if !strings.Contains(string(b), `asset: "img/cover.png"`) {
		t.Errorf("--asset not written to frontmatter:\n%s", b)
	}
	got, err := post.List(filepath.Join(root, "content"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Frontmatter.Asset != "img/cover.png" {
		t.Errorf("asset did not round-trip: %+v", got)
	}
}

func TestSlugFor(t *testing.T) {
	if got := SlugFor(NewPostInput{Title: "Café & Fun"}); got != "caf-fun" {
		t.Errorf("slug from title = %q", got)
	}
	if got := SlugFor(NewPostInput{Title: "X", Slug: "explicit"}); got != "explicit" {
		t.Errorf("explicit slug = %q", got)
	}
	got := SlugFor(NewPostInput{Date: "2026-02-03"})
	if got != "post-2026-02-03" {
		t.Errorf("date fallback = %q", got)
	}
}

func TestFindPostBySlugUsesRealPath(t *testing.T) {
	root := t.TempDir()
	posts := filepath.Join(root, "content", "posts")
	if err := os.MkdirAll(posts, 0o755); err != nil {
		t.Fatal(err)
	}
	// Frontmatter slug differs from the filename; the lookup must return the
	// file it parsed, not content/posts/<slug>.md.
	if err := os.WriteFile(filepath.Join(posts, "file-name.md"),
		[]byte("---\ntitle: T\ndate: 2026-01-01\nslug: custom-name\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	got, err := findPostBySlug("custom-name")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("content", "posts", "file-name.md"); got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestParseTags(t *testing.T) {
	got := ParseTags("a, b ,,c")
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("ParseTags = %#v", got)
	}
	if len(ParseTags("")) != 0 {
		t.Error("empty tags should be empty")
	}
}

func TestInitScaffold(t *testing.T) {
	dir := t.TempDir()
	created, err := InitScaffold(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 3 {
		t.Fatalf("expected 3 created files, got %v", created)
	}
	if _, err := os.Stat(filepath.Join(dir, "tofu.toml")); err != nil {
		t.Error("tofu.toml missing")
	}
	// custom.css is opt-in: init must not seed it.
	if _, err := os.Stat(filepath.Join(dir, "assets-blog", "custom.css")); !os.IsNotExist(err) {
		t.Error("init should not create assets-blog/custom.css")
	}
	// second init without force should refuse
	if _, err := InitScaffold(dir, false); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("want not-empty refusal, got %v", err)
	}
	// with force it succeeds and skips existing files
	created2, err := InitScaffold(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(created2) != 0 {
		t.Errorf("force re-init should create nothing new, got %v", created2)
	}
}

func TestInitScaffoldIgnoresDotfiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{".git", ".gitignore", ".DS_Store"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := InitScaffold(dir, false); err != nil {
		t.Fatalf("dotfiles-only dir should scaffold, got %v", err)
	}
}

func TestInitScaffoldRefusesVisibleFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := InitScaffold(dir, false)
	if err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("want not-empty refusal, got %v", err)
	}
}

func TestInitScaffoldWithUsesConfigAndWritesConfigLast(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Title = "Custom Site Title"
	created, err := InitScaffoldWith(dir, false, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 || created[len(created)-1] != "tofu.toml" {
		t.Errorf("tofu.toml must be created last, got %v", created)
	}
	b, err := os.ReadFile(filepath.Join(dir, "tofu.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Custom Site Title") {
		t.Errorf("tofu.toml does not use the caller's config:\n%s", b)
	}
}

// A failure partway through scaffolding must not leave a tofu.toml behind,
// otherwise the half-created site would count as a site and block a retry.
func TestInitScaffoldWithLeavesNoConfigOnFailure(t *testing.T) {
	dir := t.TempDir()
	// A regular file named "content" makes MkdirAll(content/) fail after the
	// config would previously have been written.
	if err := os.WriteFile(filepath.Join(dir, "content"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InitScaffoldWith(dir, true, DefaultConfig()); err == nil {
		t.Fatal("want scaffold failure")
	}
	if _, err := os.Stat(filepath.Join(dir, "tofu.toml")); !os.IsNotExist(err) {
		t.Errorf("tofu.toml written despite scaffold failure (err=%v)", err)
	}
}
