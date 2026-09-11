// Package tui holds every bubbletea screen: the home menu and the wizards.
package tui

import (
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// Palette. Purple + pink, no blues, no grays outside these.
const (
	colorTitle   = "#C084FC"
	colorAccent  = "#F472B6"
	colorDim     = "#B8A6E3"
	colorError   = "#FB7185"
	colorBorder  = "#6D5A9E"
	colorHeader  = "#E9D5FF"
	colorCursor  = "#C084FC"
	colorSelItem = "#F9A8D4"
)

// Options configures any TUI screen.
type Options struct {
	NoColor      bool
	NoAnimations bool
	ShowHelp     bool
	Timeout      time.Duration
	Width        int
	CharLimit    int
}

func opts(o Options) Options {
	if o.CharLimit <= 0 {
		o.CharLimit = 400
	}
	if o.Width <= 0 {
		o.Width = defaultWidth
	}
	return o
}

const defaultWidth = 80

var (
	Title         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorTitle))
	Accent        = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent))
	Dim           = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	ErrorStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorError))
	HelpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	FocusedBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(colorAccent))
	BlurredBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(colorBorder))
)

// HelpFooter is the standard wizard footer.
func HelpFooter(o Options) string {
	if !o.ShowHelp {
		return ""
	}
	return HelpStyle.Render("enter next • esc back • ctrl+c quit")
}

// MenuFooter is the home menu footer.
func MenuFooter() string {
	return HelpStyle.Render("enter select • / filter • ctrl+c quit")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// textInputStyles returns the palette for a text input.
func textInputStyles() textinput.Styles {
	var s textinput.Styles
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent))
	s.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(colorHeader))
	s.Blurred.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	s.Blurred.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(colorBorder))
	s.Blurred.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(colorSelItem))
	s.Cursor.Color = lipgloss.Color(colorAccent)
	return s
}

func newListStyles(isDark bool) (list.Styles, list.DefaultItemStyles) {
	styles := list.DefaultStyles(isDark)
	styles.Title = Title
	styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 1)
	styles.Spinner = Accent
	styles.NoItems = Dim
	items := list.NewDefaultItemStyles(isDark)
	items.NormalTitle = items.NormalTitle.Foreground(lipgloss.Color(colorHeader)).PaddingLeft(2)
	items.NormalDesc = items.NormalTitle.Foreground(lipgloss.Color(colorDim))
	items.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color(colorCursor)).
		Foreground(lipgloss.Color(colorSelItem)).
		Bold(true).
		PaddingLeft(1)
	items.SelectedDesc = items.SelectedTitle.Foreground(lipgloss.Color(colorAccent))
	items.DimmedTitle = items.NormalTitle.Foreground(lipgloss.Color(colorDim))
	items.DimmedDesc = items.DimmedTitle
	items.FilterMatch = Accent
	return styles, items
}

func newList(items []list.Item, isDark bool, width, height int) list.Model {
	styles, itemStyles := newListStyles(isDark)
	delegate := list.NewDefaultDelegate()
	delegate.Styles = itemStyles
	l := list.New(items, delegate, width, height)
	l.Styles = styles
	l.SetShowHelp(false)
	l.SetShowTitle(true)
	return l
}

// tableStyles restyles the table for the palette.
func tableStyles() table.Styles {
	return table.Styles{
		Header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorHeader)).Padding(0, 1),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
		Selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorSelItem)),
	}
}

func newSpinner() spinner.Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = Accent
	return sp
}

func newProgress(width int) progress.Model {
	return progress.New(
		progress.WithWidth(width),
		progress.WithDefaultBlend(),
		progress.WithSpringOptions(12, 0.8),
	)
}

func newHelp() help.Model {
	h := help.New()
	h.Styles = help.Styles{
		Ellipsis:       HelpStyle,
		FullKey:        HelpStyle,
		FullDesc:       HelpStyle,
		ShortKey:       HelpStyle,
		ShortDesc:      HelpStyle,
		FullSeparator:  HelpStyle,
		ShortSeparator: HelpStyle,
	}
	return h
}
