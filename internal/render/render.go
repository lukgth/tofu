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

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/lukgth/tofu/internal/config"
	"github.com/lukgth/tofu/internal/post"
	"github.com/lukgth/tofu/internal/site"
	"github.com/lukgth/tofu/web"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-highlighting/v2"
	extension "github.com/yuin/goldmark/extension"
	rendererhtml "github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM, extension.Footnote, extension.Typographer, Highlight,
		highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
			highlighting.WithFormatOptions(
				chromahtml.WithClasses(true),
			),
		),
	),
	goldmark.WithRendererOptions(rendererhtml.WithHardWraps(), rendererhtml.WithUnsafe()),
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
	CustomCSS   bool
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

// tagLink is a tag's display label paired with the slug of its page.
type tagLink struct {
	Label string
	Slug  string
}

// tagSlug maps a free-form tag to a filesystem/URL-safe slug. Tags come from
// untrusted frontmatter, so they never reach the filesystem raw.
func tagSlug(tag string) string { return post.Slugify(tag) }

// tagLinks pairs each tag with its page slug, dropping tags with no safe slug
// (all-punctuation) so links and generated pages stay in sync.
func tagLinks(tags []string) []tagLink {
	out := make([]tagLink, 0, len(tags))
	for _, t := range tags {
		if slug := tagSlug(t); slug != "" {
			out = append(out, tagLink{Label: t, Slug: slug})
		}
	}
	return out
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
	if err := writeJS(outDir); err != nil {
		return err
	}
	if err := copyFonts(outDir); err != nil {
		return err
	}
	custom, err := copyCustomCSS(siteRoot, outDir)
	if err != nil {
		return err
	}
	base.CustomCSS = custom
	if err := copyStatic(siteRoot, outDir); err != nil {
		return err
	}

	if err := writeIndex(siteRoot, s, base, outDir); err != nil {
		return err
	}
	if err := writePostsList(s, base, outDir); err != nil {
		return err
	}
	if err := writeTagPages(s, base, outDir); err != nil {
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
		Content, Footer                                template.HTML
		BodyClass                                      string
		CustomCSS                                      bool
	}{
		Lang:        ctx.Lang,
		Title:       ctx.Title,
		SiteTitle:   ctx.SiteTitle,
		Description: ctx.Description,
		ExtraHead:   ctx.ExtraHead,
		Nav:         ctx.Nav,
		Content:     template.HTML(content.String()),
		Footer:      template.HTML(ctx.Footer),
		BodyClass:   ctx.BodyClass,
		CustomCSS:   ctx.CustomCSS,
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
	if err := writeChromaCSS(&buf, cfg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeChromaCSS appends Chroma's class-based styles as a single rule set:
// each color property that differs between the light and dark palettes becomes
// a light-dark() value, so code highlighting follows the page's color-scheme.
func writeChromaCSS(buf *bytes.Buffer, cfg config.Site) error {
	light, err := chromaSheet(pickChromaStyle(cfg.Theme.CodeStyle, "github"))
	if err != nil {
		return err
	}
	dark, err := chromaSheet(pickChromaStyle(cfg.Theme.DarkCodeStyle, "github-dark"))
	if err != nil {
		return err
	}
	darkBySel := map[string]map[string]string{}
	for _, r := range dark {
		darkBySel[r.sel] = r.vals
	}
	for _, r := range light {
		parts := make([]string, 0, len(r.keys))
		for _, k := range r.keys {
			lv := r.vals[k]
			dv, ok := darkBySel[r.sel][k]
			if !ok || dv == lv {
				parts = append(parts, k+": "+lv)
				continue
			}
			if isColorProp(k) {
				parts = append(parts, k+": light-dark("+lv+", "+dv+")")
			} else {
				parts = append(parts, k+": "+lv)
			}
		}
		buf.WriteString(r.sel + " { " + strings.Join(parts, "; ") + " }\n")
	}
	return nil
}

// chromaRule is one `selector { props }` line from a Chroma WriteCSS sheet.
type chromaRule struct {
	sel  string
	keys []string
	vals map[string]string
}

// chromaSheet parses Chroma's one-liner CSS into ordered rules, dropping the
// `/* comment */ ` prefixes, the `.bg` rule, and hardcoded black/white block
// backgrounds (the site's --code-bg owns those).
func chromaSheet(style *chroma.Style) ([]chromaRule, error) {
	var raw bytes.Buffer
	if err := chromahtml.New(chromahtml.WithClasses(true)).WriteCSS(&raw, style); err != nil {
		return nil, err
	}
	var rules []chromaRule
	for _, line := range strings.Split(raw.String(), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "/*") {
			if i := strings.Index(trimmed, "*/"); i >= 0 {
				trimmed = strings.TrimSpace(trimmed[i+2:])
			} else {
				continue
			}
		}
		if trimmed == "" {
			continue
		}
		open := strings.Index(trimmed, "{")
		if open < 0 {
			continue
		}
		sel := strings.TrimSpace(trimmed[:open])
		if sel == ".bg" {
			continue
		}
		props := strings.TrimSuffix(strings.TrimSpace(trimmed[open+1:]), "}")
		r := chromaRule{sel: sel, vals: map[string]string{}}
		for _, decl := range strings.Split(props, ";") {
			decl = strings.TrimSpace(decl)
			if decl == "" {
				continue
			}
			colon := strings.Index(decl, ":")
			if colon < 0 {
				continue
			}
			k := strings.TrimSpace(decl[:colon])
			v := strings.TrimSpace(decl[colon+1:])
			if k == "background-color" && (v == "#ffffff" || v == "#000000") {
				continue
			}
			r.keys = append(r.keys, k)
			r.vals[k] = v
		}
		rules = append(rules, r)
	}
	return rules, nil
}

// isColorProp reports whether a CSS property carries a color, so its value is
// eligible for light-dark() merging.
func isColorProp(k string) bool {
	return k == "color" || strings.HasSuffix(k, "-color")
}

// pickChromaStyle resolves a Chroma style name, falling back when the knob
// is empty or names an unknown style.
func pickChromaStyle(name, fallback string) *chroma.Style {
	if name != "" {
		if s := styles.Get(name); s != nil {
			return s
		}
	}
	return styles.Get(fallback)
}

// copyCustomCSS copies the optional custom.css into the output and reports
// whether it was present, so pages only link a stylesheet that exists.
func copyCustomCSS(siteRoot, outDir string) (bool, error) {
	src := filepath.Join(siteRoot, "assets-blog", "custom.css")
	b, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if err := write(filepath.Join(outDir, "assets-blog", "custom.css"), b); err != nil {
		return false, err
	}
	return true, nil
}

// writeJS copies the embedded theme-and-visited.js (theme toggle + visited-post tracking)
// to assets-blog/, where base.html links it with defer.
func writeJS(outDir string) error {
	b, err := web.Static.ReadFile("static/theme-and-visited.js")
	if err != nil {
		return err
	}
	return write(filepath.Join(outDir, "assets-blog", "theme-and-visited.js"), b)
}

func copyStatic(siteRoot, outDir string) error {
	root := filepath.Join(siteRoot, "static")
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		// WalkDir does not follow symlinks, but os.ReadFile would: a symlink
		// resolving outside static/ would copy an arbitrary readable file
		// into the output. Reject those; allow links within the tree.
		if d.Type()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(p)
			if err != nil {
				return err
			}
			if inside, err := filepath.Rel(resolvedRoot, target); err != nil ||
				inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
				return fmt.Errorf("static/%s: symlink escapes the static directory", rel)
			}
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
	showRecent := s.Config.RecentCount != 0
	var recent []entry
	if showRecent {
		recent = entries(take(s.Posts, s.Config.RecentCount))
	}
	data := struct {
		Heading    string
		HomeHTML   template.HTML
		Posts      []entry
		ShowRecent bool
	}{
		Heading:    s.Config.Homepage.Heading,
		HomeHTML:   template.HTML(homeHTML),
		Posts:      recent,
		ShowRecent: showRecent,
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
	data := struct {
		Heading string
		Posts   []entry
	}{"posts", entries(s.Posts)}
	return page(t, "posts.html", data, ctx, filepath.Join(outDir, "articles", "index.html"))
}

// writeTagPages emits articles/tag/<tag>.html listing every post with that
// tag; the post footer's tag links point here.
func writeTagPages(s site.Site, base renderCtx, outDir string) error {
	if len(s.Posts) == 0 {
		return nil
	}
	t, err := parseTemplates()
	if err != nil {
		return err
	}
	bySlug := map[string]*tagPage{}
	var order []string
	for _, p := range s.Posts {
		for _, tag := range p.Frontmatter.Tags {
			slug := tagSlug(tag)
			if slug == "" {
				continue
			}
			tp := bySlug[slug]
			if tp == nil {
				tp = &tagPage{label: tag}
				bySlug[slug] = tp
				order = append(order, slug)
			}
			tp.posts = append(tp.posts, entry{
				DateISO:    p.Date.Format("2006-01-02"),
				DatePretty: p.Date.Format("02 Jan, 2006"),
				Slug:       p.Slug,
				Title:      p.Frontmatter.Title,
			})
		}
	}
	for _, slug := range order {
		tp := bySlug[slug]
		ctx := base
		ctx.Title = "posts tagged " + tp.label
		data := struct {
			Heading string
			Posts   []entry
		}{"posts tagged “" + tp.label + "”", tp.posts}
		dest := filepath.Join(outDir, "articles", "tag", slug+".html")
		if err := page(t, "posts.html", data, ctx, dest); err != nil {
			return err
		}
	}
	return nil
}

// tagPage collects the posts carrying one tag (by slug) and the label to show.
type tagPage struct {
	label string
	posts []entry
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
		Tags       []tagLink
		BodyHTML   template.HTML
	}{
		Title:      p.Frontmatter.Title,
		DateISO:    p.Date.Format("2006-01-02"),
		DatePretty: p.Date.Format("02 Jan, 2006"),
		Tags:       tagLinks(p.Frontmatter.Tags),
		BodyHTML:   template.HTML(stripDuplicateTitle(MarkdownToHTML(p.BodyMarkdown), p.Frontmatter.Title)),
	}
	return page(t, "post.html", data, ctx, filepath.Join(outDir, "articles", p.Slug+".html"))
}

// stripDuplicateTitle removes a leading <h1> from the rendered body when it
// merely repeats the frontmatter title — the post template already prints it.
func stripDuplicateTitle(bodyHTML, title string) string {
	trimmed := strings.TrimLeft(bodyHTML, " \t\n")
	if !strings.HasPrefix(trimmed, "<h1>") {
		return bodyHTML
	}
	end := strings.Index(trimmed, "</h1>")
	if end < 0 {
		return bodyHTML
	}
	heading := strings.TrimSpace(html.UnescapeString(trimmed[4:end]))
	if strings.EqualFold(heading, strings.TrimSpace(title)) {
		return trimmed[end+5:]
	}
	return bodyHTML
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
	for _, p := range take(s.Posts, 20) {
		link := base + "/articles/" + p.Slug + ".html"
		buf.WriteString("<item>\n")
		fmt.Fprintf(&buf, "<title>%s</title>\n", html.EscapeString(p.Frontmatter.Title))
		fmt.Fprintf(&buf, "<link>%s</link>\n", html.EscapeString(link))
		fmt.Fprintf(&buf, "<guid isPermaLink=\"true\">%s</guid>\n", html.EscapeString(link))
		if !p.Date.IsZero() {
			fmt.Fprintf(&buf, "<pubDate>%s</pubDate>\n", p.Date.UTC().Format(time.RFC1123Z))
		}
		buf.WriteString("</item>\n")
	}
	fmt.Fprintf(&buf, "</channel></rss>\n")
	return write(filepath.Join(outDir, "feed.xml"), buf.Bytes())
}
