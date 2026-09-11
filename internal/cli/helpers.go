// Package cli holds the headless command implementations shared by cobra and the TUI.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lukgth/tofu/internal/config"
	"github.com/lukgth/tofu/internal/post"
)

// IsTTY reports whether stdout is a terminal. Stdlib only.
func IsTTY() bool {
	st, _ := os.Stdout.Stat()
	return st != nil && st.Mode()&os.ModeCharDevice != 0
}

// DefaultConfig returns the tofu.toml written by init.
func DefaultConfig() config.Site {
	return config.Site{
		Title:       "My Tofu Site",
		Author:      "Jane Doe",
		Description: "A cute little blog",
		BaseURL:     "https://example.com",
		Language:    "en",
		RecentCount: 5,
		Footer:      "powered by <a href=\"https://github.com/lukgth/tofu\">tofu</a>",
		Theme: config.Theme{
			Width:          "47.5rem",
			FontMain:       "\"Ioskeley Mono\", ui-monospace, monospace",
			FontSecondary:  "\"Rubik\", system-ui, sans-serif",
			FontScale:      "1em",
			Background:     "#fffdfa",
			Primary:        "#e9dbf5",
			Text:           "#444444",
			HeaderColor:    "#4c3a63",
			Accent:         "#c8b3e8",
			Blink:          "#e4d9f6",
			Highlight:      "#c9a0e8",
			Link:           "#9d6bb8",
			Visited:        "#b48cb8",
			Blockquote:     "#5c4d6b",
			DarkBackground: "#372947",
			DarkText:       "#e5d9de",
			DarkLink:       "#d5b8f2",
			DarkVisited:    "#c9aee4",
			DarkBlockquote: "#c4b5e0",
			DarkPrimary:    "#8a6fc0",
			DarkSecondary:  "#221e44",
			DarkAccent:     "#bfa6e8",
			DarkBlink:      "#b8a0e8",
			DarkCodeBG:     "#4a3a63",
			CodeStyle:      "tofu",
			DarkCodeStyle:  "tofu-dark",
		},
		Header: config.Header{
			Title: "My Tofu Site",
			Nav: []config.NavItem{
				{Label: "home", URL: "/"},
				{Label: "blog", URL: "/articles/"},
			},
		},
		Homepage: config.Homepage{
			Heading:  "Hello!",
			BodyFile: "content/home.md",
		},
	}
}

const sampleHome = "# hello!\n\nthis is your tofu site. edit `content/home.md` to make it yours.\n"

const samplePost = "write your post here.\n"

const sampleCustomCSS = "/* your custom styles, loaded after style.css */\n"

// InitScaffold creates a new site in dir. Refuses non-empty dirs unless force.
// Hidden dotfiles (.git, .DS_Store, ...) don't count as content: scaffolding
// never overwrites existing files, so they're safe to scaffold alongside.
func InitScaffold(dir string, force bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	visible := 0
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			visible++
		}
	}
	if err == nil && visible > 0 && !force {
		return nil, fmt.Errorf("%s is not empty (use --force to write anyway)", dir)
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	var created []string
	mk := func(rel, content string) error {
		p := filepath.Join(dir, rel)
		if _, statErr := os.Stat(p); statErr == nil {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			return err
		}
		created = append(created, rel)
		return nil
	}
	cfg := DefaultConfig()
	tomlPath := filepath.Join(dir, "tofu.toml")
	if _, statErr := os.Stat(tomlPath); statErr != nil {
		if err := config.Save(tomlPath, cfg); err != nil {
			return nil, err
		}
		created = append(created, "tofu.toml")
	}
	if err := mk("content/home.md", sampleHome); err != nil {
		return nil, err
	}
	if err := mk("assets-blog/custom.css", sampleCustomCSS); err != nil {
		return nil, err
	}
	today := time.Now().Format("2006-01-02")
	fm := fmt.Sprintf("---\ntitle: hello, tofu\ndate: %s\ndescription: your first tofu post\ntags:\n  - intro\n---\n\n%s", today, samplePost)
	if err := mk(filepath.Join("content", "posts", "hello-tofu.md"), fm+"\n"); err != nil {
		return nil, err
	}
	return created, nil
}

// NewPostInput is everything needed to create a post file.
type NewPostInput struct {
	Title       string
	Slug        string
	Date        string
	Tags        []string
	Description string
	Draft       bool
	Body        string
	AssetPath   string
}

// SlugFor derives the slug for a new post: explicit slug, else slugified title,
// else post-<date>.
func SlugFor(in NewPostInput) string {
	if in.Slug != "" {
		return in.Slug
	}
	if s := post.Slugify(in.Title); s != "" {
		return s
	}
	date := in.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	return "post-" + date
}

// CreatePost writes content/posts/<slug>.md. Existing files error (use edit).
func CreatePost(root string, in NewPostInput) (string, error) {
	slug := SlugFor(in)
	rel := filepath.Join("content", "posts", slug+".md")
	path := filepath.Join(root, rel)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%s already exists; use `tofu edit` to change it", rel)
	}
	date := in.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		if _, err2 := time.Parse(time.RFC3339, date); err2 != nil {
			return "", fmt.Errorf("bad date %q (want YYYY-MM-DD)", date)
		}
	}
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %q\n", in.Title)
	fmt.Fprintf(&b, "date: %s\n", date)
	if in.Description != "" {
		fmt.Fprintf(&b, "description: %q\n", in.Description)
	}
	if len(in.Tags) > 0 {
		b.WriteString("tags:\n")
		for _, t := range in.Tags {
			fmt.Fprintf(&b, "  - %s\n", t)
		}
	}
	if in.Draft {
		b.WriteString("draft: true\n")
	}
	b.WriteString("---\n\n")
	body := in.Body
	if strings.TrimSpace(body) == "" {
		body = samplePost
	}
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return rel, nil
}

// ParseTags splits a comma-separated tag list.
func ParseTags(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		t := strings.TrimSpace(part)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// ListPosts returns plain list lines for the list command and TUI pager.
func ListPosts(root string) ([]string, error) {
	posts, err := post.List(filepath.Join(root, "content"))
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, p := range posts {
		line := fmt.Sprintf("%s %s %s", p.Date.Format("2006-01-02"), p.Slug, p.Frontmatter.Title)
		if p.Draft {
			line += " (draft)"
		}
		lines = append(lines, line)
	}
	if lines == nil {
		lines = []string{"no posts yet"}
	}
	return lines, nil
}

// CountPosts returns the number of non-draft posts under root/content/posts.
func CountPosts(root string) (int, error) {
	posts, err := post.List(filepath.Join(root, "content"))
	if err != nil {
		return 0, err
	}
	n := 0
	for _, p := range posts {
		if !p.Draft {
			n++
		}
	}
	return n, nil
}
