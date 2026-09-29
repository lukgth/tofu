// Package cli holds the headless command implementations shared by cobra and the TUI.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

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
			Width:            "47.5rem",
			FontMain:         "\"Ioskeley Mono\", ui-monospace, monospace",
			FontSecondary:    "\"Rubik\", system-ui, sans-serif",
			FontHeader:       "\"Georgia\", \"Gelasio\", serif",
			FontHeaderStyle:  "normal",
			FontHeaderWeight: "bold",
			FontScale:        "1em",
			Background:       "#fffdfa",
			Primary:          "#e9dbf5",
			Text:             "#444444",
			HeaderColor:      "#4c3a63",
			Accent:           "#c8b3e8",
			Blink:            "#e4d9f6",
			Highlight:        "#c9a0e8",
			Link:             "#9d6bb8",
			Visited:          "#b48cb8",
			Blockquote:       "#5c4d6b",
			DarkBackground:   "#372947",
			DarkText:         "#e5d9de",
			DarkLink:         "#d5b8f2",
			DarkVisited:      "#a98fc4",
			DarkBlockquote:   "#c4b5e0",
			DarkPrimary:      "#8a6fc0",
			DarkSecondary:    "#221e44",
			DarkAccent:       "#bfa6e8",
			DarkBlink:        "#b8a0e8",
			DarkCodeBG:       "#4a3a63",
			CodeStyle:        "tofu",
			DarkCodeStyle:    "tofu-dark",
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

// InitScaffold creates a new site in dir with the default config. Refuses
// non-empty dirs unless force.
// Hidden dotfiles (.git, .DS_Store, ...) don't count as content: scaffolding
// never overwrites existing files, so they're safe to scaffold alongside.
func InitScaffold(dir string, force bool) ([]string, error) {
	return InitScaffoldWith(dir, force, DefaultConfig())
}

// InitScaffoldWith creates a new site in dir using cfg. tofu.toml is written
// last, so a failure partway through leaves no site config behind (and thus no
// half-created site that blocks a retry).
func InitScaffoldWith(dir string, force bool, cfg config.Site) ([]string, error) {
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
	if err := mk("content/home.md", sampleHome); err != nil {
		return nil, err
	}
	today := time.Now().Format("2006-01-02")
	fm := fmt.Sprintf("---\ntitle: hello, tofu\ndate: %s\ndescription: your first tofu post\ntags:\n  - intro\n---\n\n%s", today, samplePost)
	if err := mk(filepath.Join("content", "posts", "hello-tofu.md"), fm+"\n"); err != nil {
		return nil, err
	}
	tomlPath := filepath.Join(dir, "tofu.toml")
	if _, statErr := os.Stat(tomlPath); statErr != nil {
		if err := config.Save(tomlPath, cfg); err != nil {
			return nil, err
		}
		created = append(created, "tofu.toml")
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
	// Asset is any YAML value (string, mapping or sequence); it is written to
	// the frontmatter verbatim, and may be a yaml fragment.
	Asset any
}

// AssetNode is a NewPostInput asset supplied as raw YAML.
type AssetNode struct{ Node *yaml.Node }

// UnmarshalYAML lets CreatePost emit an arbitrary asset mapping or sequence.
func (a AssetNode) UnmarshalYAML(n *yaml.Node) error { a.Node = n; return nil }

// MarshalYAML emits the supplied node.
func (a AssetNode) MarshalYAML() (any, error) {
	if a.Node == nil {
		return nil, nil
	}
	return a.Node, nil
}

// assetNode converts an input asset to a YAML node: an already-parsed value,
// a raw YAML fragment (a string that reads as a mapping or sequence), or a
// plain string. A string that does not parse as a document node keeps its
// literal value.
func assetNode(v any) (*yaml.Node, error) {
	switch a := v.(type) {
	case nil:
		return nil, nil
	case AssetNode:
		return a.Node, nil
	case *yaml.Node:
		return a, nil
	}
	if s, ok := v.(string); ok {
		// A fragment is raw YAML (mapping, sequence, multi-line); anything
		// else is a literal path or name, which must be quoted so YAML
		// punctuation in it cannot corrupt the frontmatter.
		if strings.ContainsAny(s, "\n\r") || strings.Contains(s, ": ") {
			var doc yaml.Node
			if err := yaml.Unmarshal([]byte(s), &doc); err != nil {
				return nil, fmt.Errorf("bad asset: %w", err)
			}
			if len(doc.Content) == 0 {
				return nil, nil
			}
			return doc.Content[0], nil
		}
		// A quoted style keeps YAML-significant characters from ending the
		// scalar early and corrupting the rest of the frontmatter.
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s, Style: yaml.DoubleQuotedStyle}, nil
	}
	var doc yaml.Node
	if err := doc.Encode(v); err != nil {
		return nil, fmt.Errorf("bad asset: %w", err)
	}
	if doc.Kind == yaml.DocumentNode {
		if len(doc.Content) == 0 {
			return nil, nil
		}
		return doc.Content[0], nil
	}
	return &doc, nil
}

// assetYAML renders an input asset for hand-written frontmatter.
func assetYAML(v any) ([]byte, error) {
	n, err := assetNode(v)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, nil
	}
	return yaml.Marshal(map[string]any{"asset": n})
}

// SlugFor derives the slug for a new post: explicit slug, else slugified title,
// else post-<date>. Every branch is slugified so the stored value and the
// published URL are the same string.
func SlugFor(in NewPostInput) string {
	if s := post.NormalizeSlug(in.Slug); s != "" {
		return s
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
	// Reject a traversing explicit slug rather than quietly rewriting it: the
	// author asked for a path escape, and normalizing it into a different post
	// would be worse than refusing.
	if in.Slug != "" && !post.ValidSlug(in.Slug) {
		return "", fmt.Errorf("invalid slug %q", in.Slug)
	}
	slug := SlugFor(in)
	if !post.ValidSlug(slug) {
		return "", fmt.Errorf("invalid slug %q", slug)
	}
	rel := filepath.Join("content", "posts", slug+".md")
	path := filepath.Join(root, rel)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%s already exists; use `tofu edit` to change it", rel)
	}
	// The file name is not the only way a post claims a slug: a frontmatter
	// `slug:` on another file publishes to the same articles/<slug>.html and
	// overwrites it, so refuse before writing anything.
	if owner, ok := post.SlugOwner(filepath.Join(root, "content"), slug); ok {
		ownerRel, relErr := filepath.Rel(root, owner)
		if relErr != nil {
			ownerRel = owner
		}
		return "", fmt.Errorf("slug %q is already used by %s; use `tofu edit` to change it", slug, ownerRel)
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
	if in.Asset != nil {
		asset, err := assetYAML(in.Asset)
		if err != nil {
			return "", fmt.Errorf("%s: %w", rel, err)
		}
		b.Write(asset)
	}
	if len(in.Tags) > 0 {
		b.WriteString("tags:\n")
		for _, t := range in.Tags {
			// %q: tags are free text and may contain YAML-significant
			// characters (`a: b`, `*star`); raw, they corrupt the frontmatter
			// and make the whole site unparseable.
			fmt.Fprintf(&b, "  - %q\n", t)
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
