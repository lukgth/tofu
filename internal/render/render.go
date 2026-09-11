// Package render turns a site into the fixed static output tree.
package render

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/yuin/goldmark"
	extension "github.com/yuin/goldmark/extension"
	"tofu/internal/config"
	"tofu/internal/post"
	"tofu/internal/site"
	"tofu/web"
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM, extension.Footnote, extension.Typographer, Highlight),
)

// MarkdownToHTML converts markdown; on error it returns an escaped <pre> of the source instead of failing.
func MarkdownToHTML(src string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return "<pre>" + html.EscapeString(src) + "</pre>"
	}
	return buf.String()
}

// renderCtx is the data passed to base.html.
type renderCtx struct {
	Lang        string
	Title       string
	SiteTitle   string
	Description string
	ExtraHead   string
	Nav         []navLink
	Content     template.HTML
	Footer      string
	BodyClass   string
}

type navLink struct {
	Label string
	URL   string
}

// entry is one row of the blog-posts list.
type entry struct {
	DateISO    string
	DatePretty string
	Slug       string
	Title      string
}

func entries(posts []postItem) []entry {
	out := make([]entry, 0, len(posts))
	for _, p := range posts {
		out = append(out, entry{
			DateISO:    p.Date.Format("2006-01-02"),
			DatePretty: p.Date.Format("02 Jan, 2006"),
			Slug:       p.Slug,
			Title:      p.Frontmatter.Title,
		})
	}
	return out
}

type postItem = post.Post

func write(dest string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, content, 0o644)
}

// Build renders the whole site into outDir.
func Build(siteRoot, outDir string, includeDrafts bool) error {
	s, err := site.Load(siteRoot, includeDrafts)
	if err != nil {
		return err
	}

	dupes := duplicateSlugs(s.Posts)
	if len(dupes) > 0 {
		return fmt.Errorf("duplicate slug %q in %s and %s", dupes[0].slug, dupes[0].files[0], dupes[0].files[1])
	}

	base := baseCtx(s.Config)
	styles, err := renderCSS(s.Config)
	if err != nil {
		return err
	}

	if err := write(filepath.Join(outDir, "assets-blog", "style.css"), styles); err != nil {
		return err
	}
	if err := copyFonts(outDir); err != nil {
		return err
	}
	if err := copyCustomCSS(siteRoot, outDir); err != nil {
		return err
	}
	if err := copyStatic(siteRoot, outDir); err != nil {
		return err
	}

	if err := writeIndex(siteRoot, s, base, outDir); err != nil {
		return err
	}
	if err := writePostsList(s, base, outDir); err != nil {
		return err
	}
	for i := range s.Posts {
		if err := writePost(s.Posts[i], base, outDir); err != nil {
			return err
		}
	}
	return writeFeed(s, outDir)
}

func baseCtx(cfg config.Site) renderCtx {
	nav := make([]navLink, 0, len(cfg.Header.Nav))
	for _, n := range cfg.Header.Nav {
		nav = append(nav, navLink{Label: n.Label, URL: n.URL})
	}
	siteTitle := cfg.Header.Title
	if siteTitle == "" {
		siteTitle = cfg.Title
	}
	return renderCtx{
		Lang:        cfg.Language,
		SiteTitle:   siteTitle,
		Description: cfg.Description,
		Nav:         nav,
		Footer:      cfg.Footer,
	}
}

func parseTemplates() (*template.Template, error) {
	return template.New("base.html").ParseFS(web.Templates,
		"templates/base.html", "templates/index.html", "templates/post.html", "templates/posts.html")
}

// page renders one fragment inside base.html and writes it to dest.
func page(t *template.Template, frag string, data any, ctx renderCtx, dest string) error {
	var content bytes.Buffer
	if err := t.ExecuteTemplate(&content, frag, data); err != nil {
		return err
	}
	full := struct {
		Lang, Title, SiteTitle, Description, ExtraHead string
		Nav                                            []navLink
		Content                                        template.HTML
		Footer, BodyClass                              string
	}{
		Lang:        ctx.Lang,
		Title:       ctx.Title,
		SiteTitle:   ctx.SiteTitle,
		Description: ctx.Description,
		ExtraHead:   ctx.ExtraHead,
		Nav:         ctx.Nav,
		Content:     template.HTML(content.String()),
		Footer:      ctx.Footer,
		BodyClass:   ctx.BodyClass,
	}
	var out bytes.Buffer
	if err := t.ExecuteTemplate(&out, "base.html", full); err != nil {
		return err
	}
	return write(dest, out.Bytes())
}

func renderCSS(cfg config.Site) ([]byte, error) {
	t, err := texttemplate.ParseFS(web.Static, "static/style.css")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, cfg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func copyCustomCSS(siteRoot, outDir string) error {
	src := filepath.Join(siteRoot, "assets-blog", "custom.css")
	b, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return write(filepath.Join(outDir, "assets-blog", "custom.css"), b)
}

func copyStatic(siteRoot, outDir string) error {
	root := filepath.Join(siteRoot, "static")
	return filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && p == root {
				return nil
			}
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return write(filepath.Join(outDir, rel), b)
	})
}

// duplicateSlugs groups posts sharing a slug after slugify.
type slugDup struct {
	slug  string
	files [2]string
}

func duplicateSlugs(posts []post.Post) []slugDup {
	seen := map[string]int{}
	var dupes []slugDup
	for _, p := range posts {
		i, ok := seen[p.Slug]
		if !ok {
			seen[p.Slug] = len(dupes)
			continue
		}
		d := slugDup{slug: p.Slug, files: [2]string{posts[i].Frontmatter.Title, p.Frontmatter.Title}}
		dupes = append(dupes, d)
	}
	return dupes
}

func writeIndex(siteRoot string, s site.Site, base renderCtx, outDir string) error {
	homeHTML := ""
	if s.Config.Homepage.BodyFile != "" {
		if b, err := os.ReadFile(filepath.Join(siteRoot, s.Config.Homepage.BodyFile)); err == nil {
			homeHTML = MarkdownToHTML(string(b))
		}
	}
	data := struct {
		Heading  string
		HomeHTML template.HTML
		Posts    []entry
	}{
		Heading:  s.Config.Homepage.Heading,
		HomeHTML: template.HTML(homeHTML),
		Posts:    entries(take(s.Posts, s.Config.RecentCount)),
	}
	ctx := base
	ctx.Title = s.Config.Homepage.Heading
	if ctx.Title == "" {
		ctx.Title = s.Config.Title
	}
	ctx.BodyClass = "home"
	t, err := parseTemplates()
	if err != nil {
		return err
	}
	return page(t, "index.html", data, ctx, filepath.Join(outDir, "index.html"))
}

func writePostsList(s site.Site, base renderCtx, outDir string) error {
	ctx := base
	ctx.Title = "Posts"
	t, err := parseTemplates()
	if err != nil {
		return err
	}
	data := struct{ Posts []entry }{Posts: entries(s.Posts)}
	return page(t, "posts.html", data, ctx, filepath.Join(outDir, "articles", "index.html"))
}

func writePost(p post.Post, base renderCtx, outDir string) error {
	ctx := base
	ctx.Title = p.Frontmatter.Title
	if p.Frontmatter.Description != "" {
		ctx.Description = p.Frontmatter.Description
	}
	t, err := parseTemplates()
	if err != nil {
		return err
	}
	data := struct {
		Title      string
		DateISO    string
		DatePretty string
		Tags       []string
		BodyHTML   template.HTML
	}{
		Title:      p.Frontmatter.Title,
		DateISO:    p.Date.Format("2006-01-02"),
		DatePretty: p.Date.Format("02 Jan, 2006"),
		Tags:       p.Frontmatter.Tags,
		BodyHTML:   template.HTML(MarkdownToHTML(p.BodyMarkdown)),
	}
	return page(t, "post.html", data, ctx, filepath.Join(outDir, "articles", p.Slug+".html"))
}

func take[T any](xs []T, n int) []T {
	if n > len(xs) {
		n = len(xs)
	}
	return xs[:n]
}

func writeFeed(s site.Site, outDir string) error {
	base := strings.TrimRight(s.Config.BaseURL, "/")
	if base == "" {
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	fmt.Fprintf(&buf, "<rss version=\"2.0\"><channel>\n")
	fmt.Fprintf(&buf, "<title>%s</title>\n", html.EscapeString(s.Config.Title))
	fmt.Fprintf(&buf, "<link>%s</link>\n", html.EscapeString(base))
	fmt.Fprintf(&buf, "<description>%s</description>\n", html.EscapeString(s.Config.Description))
	latest := take(s.Posts, 20)
	for _, p := range latest {
		if !p.Date.IsZero() {
			fmt.Fprintf(&buf, "<pubDate>%s</pubDate>\n", p.Date.UTC().Format(time.RFC1123Z))
		}
		fmt.Fprintf(&buf, "</item>\n")
	}
	fmt.Fprintf(&buf, "</channel></rss>\n")
	return write(filepath.Join(outDir, "feed.xml"), buf.Bytes())
}
