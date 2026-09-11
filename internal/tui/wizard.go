package tui

import (
	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type stepKind int

const (
	stepInput stepKind = iota
	stepChoice
	stepFile
	stepBody
)

type step struct {
	kind   stepKind
	label  string
	input  *textinput.Model
	area   *textarea.Model
	choice *choiceModel
	pick   *filepicker.Model
	setter func(string) error
}

// normalize fills defaults for Options.
func normalize(o Options) Options {
	if o.CharLimit <= 0 {
		o.CharLimit = 400
	}
	if o.Width <= 0 {
		o.Width = 80
	}
	return o
}

func newTextInput(placeholder, def string, o Options) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(def)
	ti.SetStyles(textInputStyles())
	if o.CharLimit > 0 {
		ti.CharLimit = o.CharLimit
	}
	ti.SetWidth(o.Width - 4)
	return ti
}

func newFilePicker(o Options) filepicker.Model {
	fp := filepicker.New()
	fp.SetHeight(10)
	return fp
}

// choiceModel is a plain cursor over options — every choice always visible,
// no list pagination hiding entries behind a •• pager.
type choiceModel struct {
	cursor  int
	options []string
	width   int
}

func newChoice(options []string, isDark bool, width int) choiceModel {
	return choiceModel{options: options, width: width}
}

func (c *choiceModel) selected() string {
	if c.cursor < 0 || c.cursor >= len(c.options) {
		return ""
	}
	return c.options[c.cursor]
}

func (c *choiceModel) selectOption(v string) {
	for i, o := range c.options {
		if o == v {
			c.cursor = i
			return
		}
	}
}

// update moves the cursor with up/down and returns the model (immutable form
// for bubbletea Update chains).
func (c choiceModel) update(msg tea.KeyPressMsg) choiceModel {
	switch msg.String() {
	case "up", "k":
		if c.cursor > 0 {
			c.cursor--
		}
	case "down", "j":
		if c.cursor < len(c.options)-1 {
			c.cursor++
		}
	}
	return c
}

// view renders all options; the selected row gets an accent border card and
// the others stay dim, so the cursor is visible without stealing the row.
func (c choiceModel) view(isDark bool) string {
	sel := FocusedBorder.Width(c.width-2).Padding(0, 1)
	plain := BlurredBorder.Width(c.width-2).Padding(0, 1)
	var cards []string
	for i, opt := range c.options {
		if i == c.cursor {
			cards = append(cards, sel.Render(Accent.Render("● "+opt)))
		} else {
			cards = append(cards, plain.Render(Dim.Render("○ "+opt)))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, cards...)
}
