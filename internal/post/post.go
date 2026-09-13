// Package post parses and lists blog posts under content/posts.
package post

import (
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
	Tags        []string `yaml:"tags,omitempty"`
	Slug        string   `yaml:"slug,omitempty"`
	Draft       bool     `yaml:"draft,omitempty"`
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

// splitRaw splits a source file into its frontmatter text and body without
// rewriting the body, so callers that edit frontmatter keep the surrounding
// bytes intact. bodyStart is the index in s where the body begins.
func splitRaw(s string) (fm, body string, bodyStart int, err error) {
	if !strings.HasPrefix(s, "---\n") && !strings.HasPrefix(s, "---\r\n") {
		return "", "", 0, fmt.Errorf("missing frontmatter")
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
	for k, v := range current {
		switch k {
		case "title", "date", "description", "tags", "slug", "draft":
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
