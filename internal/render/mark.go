package render

import (
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// A highlight struct represents a ==highlighted== span.
type highlight struct {
	gast.BaseInline
}

// KindHighlight is a NodeKind of the highlight node.
var KindHighlight = gast.NewNodeKind("Highlight")

// Kind implements Node.Kind.
func (n *highlight) Kind() gast.NodeKind {
	return KindHighlight
}

// Dump implements Node.Dump.
func (n *highlight) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// NewHighlight returns a new highlight node.
func NewHighlight() *highlight {
	return &highlight{}
}

// highlightDelimiterProcessor handles ==highlight== spans.
type highlightDelimiterProcessor struct{}

func (p *highlightDelimiterProcessor) IsDelimiter(b byte) bool {
	return b == '='
}

func (p *highlightDelimiterProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *highlightDelimiterProcessor) OnMatch(consumes int) gast.Node {
	return NewHighlight()
}

var defaultHighlightDelimiterProcessor = &highlightDelimiterProcessor{}

type highlightParser struct{}

func (s *highlightParser) Trigger() []byte {
	return []byte{'='}
}

func (s *highlightParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 2, defaultHighlightDelimiterProcessor)
	if node == nil || node.OriginalLength != 2 || before == '=' {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

func (s *highlightParser) CloseBlock(parent gast.Node, pc parser.Context) {
	// nothing to do
}

// highlightHTMLRenderer renders highlight nodes as <mark>.
type highlightHTMLRenderer struct {
	html.Config
}

// NewHighlightHTMLRenderer returns a renderer for highlight nodes.
func NewHighlightHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &highlightHTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *highlightHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindHighlight, r.renderHighlight)
}

func (r *highlightHTMLRenderer) renderHighlight(
	w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		if n.Attributes() != nil {
			_, _ = w.WriteString("<mark")
			html.RenderAttributes(w, n, html.GlobalAttributeFilter)
			_ = w.WriteByte('>')
		} else {
			_, _ = w.WriteString("<mark>")
		}
	} else {
		_, _ = w.WriteString("</mark>")
	}
	return gast.WalkContinue, nil
}

// Highlight is an extension allowing ==text== to render as <mark>.
var Highlight = &highlightExt{}

type highlightExt struct{}

func (e *highlightExt) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&highlightParser{}, 500),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewHighlightHTMLRenderer(), 500),
	))
}
