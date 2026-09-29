// Package post parses and lists blog posts under content/posts.
package post

import (
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Frontmatter struct {
	Title       string   `yaml:"title"`
	Date        string   `yaml:"date"`
	Description string   `yaml:"description,omitempty"`
	Asset       Asset    `yaml:"asset,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Slug        string   `yaml:"slug,omitempty"`
	Draft       bool     `yaml:"draft,omitempty"`
}

// Asset carries an optional `asset:` frontmatter value in whatever shape the
// author wrote it. YAML accepts a string, a mapping, or a sequence, and the
// generator only ever carries the value through, so parsing must not reject
// the non-string forms. Node keeps the parsed document (string nodes return
// their literal); Marshal renders the same value back to YAML.
type Asset struct {
	Node *yaml.Node
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (a *Asset) UnmarshalYAML(n *yaml.Node) error {
	a.Node = n
	return nil
}

// MarshalYAML implements yaml.Marshaler.
func (a Asset) MarshalYAML() (any, error) {
	if a.Node == nil {
		return nil, nil
	}
	return a.Node, nil
}

// IsZero reports whether no asset value is present, so a round-tripped
// frontmatter omits the key instead of writing a null.
func (a Asset) IsZero() bool { return a.Node == nil }

// String returns the asset as a single string, or "" when it is absent or has
// a mapping/sequence shape.
func (a Asset) String() string {
	if a.Node != nil && a.Node.Kind == yaml.ScalarNode {
		return a.Node.Value
	}
	return ""
}

// Any returns the asset as a plain Go value (map, slice, string, or nil),
// for callers that want structured access.
func (a Asset) Any() any {
	if a.Node == nil {
		return nil
	}
	var v any
	if err := a.Node.Decode(&v); err != nil {
		return a.Node.Value
	}
	return v
}

// Marshal renders the asset as a YAML fragment (empty when absent), for
// frontmatter written by hand.
func (a Asset) Marshal() ([]byte, error) {
	if a.Node == nil {
		return nil, nil
	}
	return yaml.Marshal(&a)
}

// ParseAsset decodes a YAML document into an Asset. A scalar document is
// read as its literal string; anything else keeps its structured shape.
func ParseAsset(src string) (Asset, error) {
	var n yaml.Node
	if err := yaml.Unmarshal([]byte(src), &n); err != nil {
		return Asset{}, err
	}
	return assetFromNode(&n), nil
}

func assetFromNode(n *yaml.Node) Asset {
	if n == nil || n.Kind == 0 || len(n.Content) == 0 {
		return Asset{}
	}
	if n.Kind == yaml.DocumentNode {
		return assetFromNode(n.Content[0])
	}
	// Scalar documents decode with a null tag; read those by value so plain
	// text keeps its literal string.
	if n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		return Asset{Node: &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: n.Value}}
	}
	return Asset{Node: n}
}

type Post struct {
	Frontmatter
	Path         string
	Slug         string
	Date         time.Time
	BodyMarkdown string
	BodyHTML     template.HTML
}

var slugStrip = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(s string) string {
	s = strings.ToLower(s)
	s = slugStrip.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

var dateFormats = []string{"2006-01-02", time.RFC3339}

// ValidSlug reports whether s is safe as a single filename/URL component:
// non-empty, not "." or "..", and free of path separators. A slug is used
// verbatim as a filename (content/posts/<slug>.md, articles/<slug>.html),
// so an unsanitized value could escape its directory.
func ValidSlug(s string) bool {
	return s != "" && s != "." && s != ".." &&
		!strings.ContainsAny(s, `/\`) &&
		filepath.Base(s) == s
}

// NormalizeSlug maps an author-supplied slug onto the canonical lowercase
// dash-separated form so a value is used exactly as written wherever it
// appears: the file name, the frontmatter slug, and every link built from it.
func NormalizeSlug(s string) string { return Slugify(s) }

// effectiveSlug is the slug a post actually publishes under: its own value
// when set, else the slugified base name (what ParseFile fills in).
func effectiveSlug(fm Frontmatter, fileName string) string {
	s := strings.TrimSuffix(filepath.Base(fileName), ".md")
	if fm.Slug != "" {
		s = fm.Slug
	}
	return Slugify(s)
}

// SlugOwner returns the path of the post already published under slug
func SlugOwner(contentDir, slug string) (string, bool) {
	want := NormalizeSlug(slug)
	if want == "" {
		return "", false
	}
	dir := filepath.Join(contentDir, "posts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		path := filepath.Join(dir, name)
		var fm Frontmatter
		if text, _, err := splitFrontmatter(mustRead(path)); err == nil {
			// A post whose frontmatter does not parse is reported by
			// List/Build; here it only must not be treated as the owner.
			_ = yaml.Unmarshal([]byte(text), &fm)
		}
		if effectiveSlug(fm, name) == want {
			return path, true
		}
	}
	return "", false
}

func mustRead(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

// ParseFile reads a markdown file, splits leading YAML frontmatter, and fills defaults.
func ParseFile(path string) (Post, error) {
	var p Post
	raw, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
	p.Path = path
	fm, body, err := splitFrontmatter(string(raw))
	if err != nil {
		return p, fmt.Errorf("%s: %w", path, err)
	}
	if err := yaml.Unmarshal(fm, &p.Frontmatter); err != nil {
		return p, fmt.Errorf("%s: bad frontmatter: %w", path, err)
	}
	if p.Frontmatter.Date != "" {
		p.Date, err = parseDate(p.Frontmatter.Date)
		if err != nil {
			return p, fmt.Errorf("%s: bad date %q: %w", path, p.Frontmatter.Date, err)
		}
	}
	slug := p.Frontmatter.Slug
	if slug == "" {
		slug = Slugify(strings.TrimSuffix(filepath.Base(path), ".md"))
	} else {
		// An explicit slug is a filename/URL component: reject a traversing
		// value outright, then normalize so the published URL matches the
		// slug every other component (links, tags, `tofu edit`) uses.
		if !ValidSlug(slug) {
			return p, fmt.Errorf("%s: invalid slug %q", path, slug)
		}
		slug = NormalizeSlug(slug)
	}
	if slug == "" {
		return p, fmt.Errorf("%s: empty slug after slugify", path)
	}
	if !ValidSlug(slug) {
		return p, fmt.Errorf("%s: invalid slug %q", path, slug)
	}
	p.Slug = slug
	p.BodyMarkdown = body
	return p, nil
}

func parseDate(v string) (time.Time, error) {
	for _, layout := range dateFormats {
		t, err := time.Parse(layout, v)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unparsable")
}

func splitFrontmatter(src string) ([]byte, string, error) {
	norm := strings.TrimPrefix(strings.ReplaceAll(src, "\r\n", "\n"), "\ufeff")
	fm, body, _, err := splitRaw(norm)
	if err != nil {
		return nil, "", err
	}
	return []byte(fm), body, nil
}

// ErrNoFrontmatter reports a file that does not open with a `---` fence.
// Callers that also handle ordinary markdown (the TUI's file browser opens
// any .md) can tell "this is not a post" from "this post is broken".
var ErrNoFrontmatter = errors.New("missing frontmatter")

// splitRaw splits a source file into its frontmatter text and body without
// rewriting the body, so callers that edit frontmatter keep the surrounding
// bytes intact. bodyStart is the index in s where the body begins.
func splitRaw(s string) (fm, body string, bodyStart int, err error) {
	if !strings.HasPrefix(s, "---\n") && !strings.HasPrefix(s, "---\r\n") {
		return "", "", 0, ErrNoFrontmatter
	}
	rest := s[strings.IndexByte(s, '\n')+1:]
	for start := 0; ; {
		lineEnd := strings.IndexByte(rest[start:], '\n')
		next := len(rest)
		var line string
		if lineEnd < 0 {
			line = rest[start:]
		} else {
			line = rest[start : start+lineEnd]
			next = start + lineEnd + 1
		}
		if strings.TrimRight(line, "\r") == "---" {
			end := start
			if end > 0 && rest[end-1] == '\n' {
				end--
			}
			if end > 0 && rest[end-1] == '\r' {
				end--
			}
			return rest[:end], rest[next:], len(s) - len(rest) + next, nil
		}
		if lineEnd < 0 {
			return "", "", 0, fmt.Errorf("missing frontmatter")
		}
		start = next
	}
}

// List reads content/posts/*.md sorted date-descending, title-ascending tiebreak.
func List(contentDir string) ([]Post, error) {
	dir := filepath.Join(contentDir, "posts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var posts []Post
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		p, err := ParseFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	sort.SliceStable(posts, func(i, j int) bool {
		a, b := posts[i], posts[j]
		if !a.Date.Equal(b.Date) {
			return a.Date.After(b.Date)
		}
		return a.Frontmatter.Title < b.Frontmatter.Title
	})
	return posts, nil
}

// PathForSlug returns the conventional file path for a slug under contentDir.
func PathForSlug(contentDir, slug string) string {
	return filepath.Join(contentDir, "posts", slug+".md")
}

// UpdateFrontmatter rewrites only the frontmatter, preserving unknown keys and
// the body (including its original line endings) byte for byte.
func UpdateFrontmatter(path string, mutate func(*Frontmatter) error) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fmText, body, _, err := splitRaw(string(raw))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	fmText = strings.ReplaceAll(fmText, "\r\n", "\n")
	var current map[string]any
	if err := yaml.Unmarshal([]byte(fmText), &current); err != nil {
		return fmt.Errorf("%s: bad frontmatter: %w", path, err)
	}
	var known Frontmatter
	if err := yaml.Unmarshal([]byte(fmText), &known); err != nil {
		return fmt.Errorf("%s: bad frontmatter: %w", path, err)
	}
	if err := mutate(&known); err != nil {
		return err
	}
	merged, err := yaml.Marshal(&known)
	if err != nil {
		return err
	}
	var mergedMap map[string]any
	if err := yaml.Unmarshal(merged, &mergedMap); err != nil {
		return err
	}
	// yaml.Marshal of the Asset type drops nothing, but re-assert the node so a
	// mapping/sequence asset survives the round trip exactly as authored.
	if a := known.Asset.Node; a != nil {
		mergedMap["asset"] = a
	}
	for k, v := range current {
		switch k {
		case "title", "date", "description", "asset", "tags", "slug", "draft":
			continue
		}
		if _, ok := mergedMap[k]; !ok {
			mergedMap[k] = v
		}
	}
	out, err := yaml.Marshal(mergedMap)
	if err != nil {
		return err
	}
	var final strings.Builder
	final.WriteString("---\n")
	final.Write(out)
	final.WriteString("---\n")
	final.WriteString(body)
	return os.WriteFile(path, []byte(final.String()), 0o644)
}

// SetBody replaces the markdown body after the frontmatter, leaving the
// frontmatter (and its line endings) untouched.
func SetBody(path, body string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(raw)
	_, _, bodyStart, err := splitRaw(s)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return os.WriteFile(path, []byte(s[:bodyStart]+body), 0o644)
}
