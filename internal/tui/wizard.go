package tui

import (
	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
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

// choiceModel is an option list (Yes/No, editor choice, ...).
type choiceModel struct {
	list    list.Model
	options []string
}

func newChoice(options []string, isDark bool, width int) choiceModel {
	items := make([]list.Item, len(options))
	for i, opt := range options {
		items[i] = menuItem{label: opt}
	}
	l := newList(items, isDark, width, len(options)+2)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	return choiceModel{list: l, options: options}
}

func (c *choiceModel) selected() string {
	i := c.list.Index()
	if i < 0 || i >= len(c.options) {
		return ""
	}
	return c.options[i]
}

func (c *choiceModel) selectOption(v string) {
	for i, o := range c.options {
		if o == v {
			c.list.Select(i)
			return
		}
	}
}
