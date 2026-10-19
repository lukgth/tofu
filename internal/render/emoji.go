package render

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// maxEmojiAlias bounds the scan between the two colons. The longest gemoji
// alias is far shorter, and the bound keeps a stray colon from scanning a
// whole line.
const maxEmojiAlias = 64

// emojiParser replaces :shortcode: with the matching Unicode emoji.
type emojiParser struct{}

func (p *emojiParser) Trigger() []byte {
	return []byte{':'}
}

func (p *emojiParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	if block.PrecendingCharacter() == ':' {
		return nil
	}
	line, _ := block.PeekLine()
	if len(line) < 3 {
		return nil
	}
	end := bytes.IndexByte(line[1:], ':')
	if end < 1 || end > maxEmojiAlias {
		return nil
	}
	name := line[1 : end+1]
	for _, b := range name {
		if !isEmojiAliasByte(b) {
			return nil
		}
	}
	emoji, ok := emojiAliases[string(name)]
	if !ok {
		return nil
	}
	block.Advance(end + 2)
	return ast.NewString([]byte(emoji))
}

func isEmojiAliasByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '_' || b == '+' || b == '-'
}

// Emoji is an extension that renders GitHub emoji shortcodes.
var Emoji = &emojiExt{}

type emojiExt struct{}

func (e *emojiExt) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&emojiParser{}, 500),
	))
}
