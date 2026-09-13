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

// ParseFile reads a markdown file, splits leading YAML frontmatter, and fills defaults.
func ParseFile(path string) (Post, error) {
	var p Post
	raw, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
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
	if !strings.HasPrefix(norm, "---\n") {
		return nil, "", fmt.Errorf("missing frontmatter")
	}
	rest := norm[4:]
	// Empty frontmatter: "---\n---\n" (body follows) or "---\n---" at EOF.
	if strings.HasPrefix(rest, "---\n") {
		return nil, rest[4:], nil
	}
	if rest == "---" {
		return nil, "", nil
	}
	if end := strings.Index(rest, "\n---\n"); end >= 0 {
		return []byte(rest[:end]), rest[end+len("\n---\n"):], nil
	}
	// Closing delimiter at EOF with no trailing newline: "---\n…\n---".
	if strings.HasSuffix(rest, "\n---") {
		return []byte(rest[:len(rest)-4]), "", nil
	}
	return nil, "", fmt.Errorf("missing frontmatter")
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

// UpdateFrontmatter rewrites only the frontmatter, preserving unknown keys and the body.
func UpdateFrontmatter(path string, mutate func(*Frontmatter) error) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	norm := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(norm, "---\n") {
		return fmt.Errorf("%s: missing frontmatter", path)
	}
	rest := norm[4:]
	end := strings.Index(rest, "\n---\n")
	var fmText, body string
	if end < 0 {
		if strings.HasSuffix(rest, "\n---") {
			fmText = rest[:len(rest)-4]
			body = ""
		} else {
			return fmt.Errorf("%s: missing frontmatter", path)
		}
	} else {
		fmText = rest[:end]
		body = rest[end+len("\n---\n"):]
	}
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
// frontmatter untouched.
func SetBody(path, body string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := strings.ReplaceAll(string(raw), "\r\n", "\n")
	end := strings.Index(s, "\n---\n")
	if end < 0 {
		return fmt.Errorf("%s: missing frontmatter", path)
	}
	bodyEnd := end + len("\n---\n")
	return os.WriteFile(path, []byte(s[:bodyEnd]+body), 0o644)
}
