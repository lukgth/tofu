// Package render turns a site into the fixed static output tree.
package render

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
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

// resolvePath returns the absolute, symlink-free form of path. Every component
// that exists is resolved, so the result cannot be redirected anywhere else
// later; for a path that does not exist yet the nearest existing ancestor is
// resolved and the remaining elements are appended verbatim.
func resolvePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	cur := abs
	var tail []string
	for {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return filepath.Clean(resolved), nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("no existing ancestor of %s", abs)
		}
		tail = append(tail, filepath.Base(cur))
		cur = parent
	}
}

// under reports whether path sits inside root (or is root itself). Both sides
// must already be resolved, or a symlinked output dir escapes the check.
func under(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// siteSources are the paths a build reads. The output directory may sit
// inside the site (public/ is the default), but it must stay clear of these.
var siteSources = []string{"tofu.toml", "content", "static", "assets-blog"}

// prepareOutDir validates the requested output directory and clears any
// earlier build so renamed or deleted sources cannot leave stale pages behind.
// It runs before a single byte is written, so a rejected target leaves the
// previous output and every source file untouched.
func prepareOutDir(siteRoot, outDir string) error {
	absSite, err := resolvePath(siteRoot)
	if err != nil {
		return err
	}
	absOut, err := resolvePath(outDir)
	if err != nil {
		return err
	}
	if absOut == absSite || under(absOut, absSite) {
		return fmt.Errorf("output directory %s must not be the site root or contain it (%s)",
			outDir, absSite)
	}
	// The wipe deletes everything below absOut, so the two must be disjoint in
	// both directions: absOut inside a source would delete that source.
	for _, rel := range siteSources {
		src := filepath.Join(absSite, rel)
		// A source that is itself a symlink is judged by where it points, so
		// an outDir cannot dodge the check via a link.
		if resolved, err := resolvePath(src); err == nil {
			src = resolved
		}
		if under(absOut, src) || under(src, absOut) {
			return fmt.Errorf("output directory %s overlaps the site source %s: "+
				"the build would delete its own input", outDir, src)
		}
	}
	entries, err := os.ReadDir(absOut)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(absOut, 0o755)
		}
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(absOut, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// reservedIndexSlug is the slug of the generated articles/index.html archive.
const reservedIndexSlug = "index"

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
	// articles/index.html is the generated post archive; a post with that slug
	// would overwrite it. Reject before anything is written, so the archive of
	// a previous build stays intact.
	for i := range s.Posts {
		if s.Posts[i].Slug == reservedIndexSlug {
			return fmt.Errorf("post %q uses reserved slug %q: it would overwrite the articles archive",
				s.Posts[i].Frontmatter.Title, reservedIndexSlug)
		}
	}
	if err := prepareOutDir(siteRoot, outDir); err != nil {
		return err
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

// copyStatic mirrors the site's static/ directory into outDir. WalkDir does
// not follow symlinks, but os.ReadFile would, so a link resolving outside the
// static tree is rejected: it would copy an arbitrary readable file into the
// output. Links staying inside the tree are copied as plain files, and a link
// to a directory is skipped rather than recursed into.
func copyStatic(siteRoot, outDir string) error {
	root := filepath.Join(siteRoot, "static")
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("static: %w", err)
	}
	fi, err := os.Stat(resolvedRoot)
	if err != nil {
		return fmt.Errorf("static: %w", err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("static: %s is not a directory", root)
	}
	// static/ itself may be a symlink to a directory; walk the resolved tree so
	// its contents are copied instead of the link being read as a file.
	walkRoot := root
	if lfi, err := os.Lstat(root); err == nil && lfi.Mode()&os.ModeSymlink != 0 {
		walkRoot = resolvedRoot
	}
	return filepath.WalkDir(walkRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(walkRoot, p)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(p)
			if err != nil {
				return fmt.Errorf("static/%s: broken symlink: %w", rel, err)
			}
			if !under(resolvedRoot, target) {
				return fmt.Errorf("static/%s: symlink escapes the static directory", rel)
			}
			tfi, err := os.Stat(target)
			if err != nil {
				return fmt.Errorf("static/%s: %w", rel, err)
			}
			if tfi.IsDir() {
				// Copying a linked directory would recurse out of the walked
				// tree; the real directory is copied on its own if in-tree.
				return nil
			}
			if !tfi.Mode().IsRegular() {
				return fmt.Errorf("static/%s: not a regular file", rel)
			}
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("static/%s: %w", rel, err)
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
	for i, p := range posts {
		j, ok := seen[p.Slug]
		if !ok {
			seen[p.Slug] = i
			continue
		}
		d := slugDup{slug: p.Slug, files: [2]string{posts[j].Frontmatter.Title, p.Frontmatter.Title}}
		dupes = append(dupes, d)
	}
	return dupes
}

// homeBody renders the configured homepage body file. The path is
// site-relative: it must stay inside the site and name a readable regular
// file, so a traversal or a stray symlink cannot pull unrelated bytes onto
// the homepage. Missing or unreadable configured files are errors.
func homeBody(siteRoot, bodyFile string) (string, error) {
	absSite, err := resolvePath(siteRoot)
	if err != nil {
		return "", err
	}
	p := filepath.Join(siteRoot, bodyFile)
	resolved, err := resolvePath(p)
	if err != nil {
		return "", fmt.Errorf("homepage body_file %q: %w", bodyFile, err)
	}
	if !under(absSite, resolved) {
		return "", fmt.Errorf("homepage body_file %q resolves outside the site (%s)", bodyFile, resolved)
	}
	fi, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("homepage body_file %q: %w", bodyFile, err)
	}
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("homepage body_file %q is not a regular file", bodyFile)
	}
	b, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("homepage body_file %q: %w", bodyFile, err)
	}
	return MarkdownToHTML(string(b)), nil
}

func writeIndex(siteRoot string, s site.Site, base renderCtx, outDir string) error {
	homeHTML := ""
	if s.Config.Homepage.BodyFile != "" {
		var err error
		if homeHTML, err = homeBody(siteRoot, s.Config.Homepage.BodyFile); err != nil {
			return err
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

// htmlTagRe matches HTML tags, used to compare a heading's visible text.
var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

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
	// Compare the heading's visible text so inline markup (# *Hello*) still
	// matches the plain-text title.
	heading := strings.TrimSpace(html.UnescapeString(htmlTagRe.ReplaceAllString(trimmed[4:end], "")))
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
