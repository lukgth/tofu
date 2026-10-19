package render

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// KindAlert is a NodeKind of the alert node.
var KindAlert = gast.NewNodeKind("Alert")

// alert is a blockquote that starts with a GitHub alert marker.
type alert struct {
	gast.BaseBlock

	kind  string
	title string
}

func (n *alert) Kind() gast.NodeKind {
	return KindAlert
}

func (n *alert) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{"Kind": n.kind}, nil)
}

// alertMarker matches the marker line that opens a GitHub alert.
var alertMarker = regexp.MustCompile(`^[ \t]*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\][ \t\r]*`)

var alertTitles = map[string]string{
	"note":      "Note",
	"tip":       "Tip",
	"important": "Important",
	"warning":   "Warning",
	"caution":   "Caution",
}

// alertTransformer rewrites marked blockquotes into alert nodes.
type alertTransformer struct{}

func (t *alertTransformer) Transform(doc *gast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	var marked []*gast.Blockquote
	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}
		bq, ok := n.(*gast.Blockquote)
		if !ok || bq.Parent() != doc {
			return gast.WalkContinue, nil
		}
		para, ok := bq.FirstChild().(*gast.Paragraph)
		if !ok || para.Lines().Len() == 0 {
			return gast.WalkContinue, nil
		}
		line := para.Lines().At(0)
		if alertMarker.Match(line.Value(source)) {
			marked = append(marked, bq)
		}
		return gast.WalkContinue, nil
	})

	for _, bq := range marked {
		para := bq.FirstChild().(*gast.Paragraph)
		line := para.Lines().At(0)
		loc := alertMarker.FindSubmatchIndex(line.Value(source))
		kind := strings.ToLower(string(line.Value(source)[loc[2]:loc[3]]))
		stripAlertMarker(para, line.Start+loc[1])

		a := &alert{kind: kind, title: alertTitles[kind]}
		for c := bq.FirstChild(); c != nil; c = bq.FirstChild() {
			a.AppendChild(a, c)
		}
		parent := bq.Parent()
		parent.ReplaceChild(parent, bq, a)
	}
}

// stripAlertMarker removes the marker line from the paragraph's inline nodes
// and source lines. Text after the marker stays as alert content.
func stripAlertMarker(para *gast.Paragraph, markerEnd int) {
	for c := para.FirstChild(); c != nil; {
		next := c.NextSibling()
		t, ok := c.(*gast.Text)
		if !ok {
			break
		}
		if t.Segment.Stop <= markerEnd {
			para.RemoveChild(para, c)
		} else {
			if t.Segment.Start < markerEnd {
				t.Segment.Start = markerEnd
			}
			break
		}
		c = next
	}
	if para.ChildCount() == 0 {
		parent := para.Parent()
		parent.RemoveChild(parent, para)
		return
	}
	line := para.Lines().At(0)
	if markerEnd < line.Stop-1 {
		line.Start = markerEnd
		para.Lines().Set(0, line)
		return
	}
	para.Lines().SetSliced(1, para.Lines().Len())
}

// alertHTMLRenderer renders alert nodes as a titled, colored callout.
type alertHTMLRenderer struct {
	html.Config
}

// NewAlertHTMLRenderer returns a renderer for alert nodes.
func NewAlertHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &alertHTMLRenderer{Config: html.NewConfig()}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *alertHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindAlert, r.renderAlert)
}

func (r *alertHTMLRenderer) renderAlert(
	w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	a := n.(*alert)
	if entering {
		_, _ = w.WriteString(`<div class="alert alert-` + a.kind + `">` + "\n")
		_, _ = w.WriteString(`<p class="alert-title">` + a.title + "</p>\n")
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return gast.WalkContinue, nil
}

// Alerts is an extension that renders GitHub alert blockquotes.
var Alerts = &alertExt{}

type alertExt struct{}

func (e *alertExt) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithASTTransformers(
		util.Prioritized(&alertTransformer{}, 500),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewAlertHTMLRenderer(), 500),
	))
}
