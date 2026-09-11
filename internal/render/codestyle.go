package render

import (
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
)

// tofuCodeStyles builds the tofu code color scheme from the site palette.
// Light mode: plum keywords, purple names, pink strings on transparent.
// Dark mode: bright lavender keywords, soft pink strings on transparent.
// Backgrounds stay unset so the site's --code-bg owns the code block.
func init() {
	styles.Register(chroma.MustNewStyle("tofu", chroma.StyleEntries{
		chroma.Text:                  "#444444",
		chroma.Error:                 "#fb7185",
		chroma.Comment:               "#8a7a99",
		chroma.CommentPreproc:        "#9d6bb8",
		chroma.Keyword:               "#7c5a9e",
		chroma.KeywordType:           "#7c5a9e",
		chroma.Operator:              "#6d5a9e",
		chroma.Punctuation:           "#6d5a9e",
		chroma.Name:                  "#4c3a63",
		chroma.NameBuiltin:           "#9d6bb8",
		chroma.NameTag:               "#9d6bb8",
		chroma.NameAttribute:         "#7c5a9e",
		chroma.NameDecorator:         "#c084fc",
		chroma.NameFunction:          "#a855d8",
		chroma.NameClass:             "#a855d8",
		chroma.NameConstant:          "#b48cb8",
		chroma.LiteralString:         "#c25aa0",
		chroma.LiteralStringEscape:   "#e879b8",
		chroma.LiteralStringInterpol: "#c25aa0",
		chroma.LiteralNumber:         "#8d5bb8",
		chroma.Literal:               "#8d5bb8",
	}))
	styles.Register(chroma.MustNewStyle("tofu-dark", chroma.StyleEntries{
		chroma.Text:                  "#e5d9de",
		chroma.Error:                 "#fb7185",
		chroma.Comment:               "#9c8bb0",
		chroma.CommentPreproc:        "#c9aee4",
		chroma.Keyword:               "#d5b8f2",
		chroma.KeywordType:           "#d5b8f2",
		chroma.Operator:              "#b8a0e8",
		chroma.Punctuation:           "#cfc2e6",
		chroma.Name:                  "#e9d8f5",
		chroma.NameBuiltin:           "#c9aee4",
		chroma.NameTag:               "#c9aee4",
		chroma.NameAttribute:         "#d5b8f2",
		chroma.NameDecorator:         "#e2b8f0",
		chroma.NameFunction:          "#e2b8f0",
		chroma.NameClass:             "#e2b8f0",
		chroma.NameConstant:          "#c9aee4",
		chroma.LiteralString:         "#f2a6d8",
		chroma.LiteralStringEscape:   "#f5c1e3",
		chroma.LiteralStringInterpol: "#f2a6d8",
		chroma.LiteralNumber:         "#d8b3f0",
		chroma.Literal:               "#d8b3f0",
	}))
}
