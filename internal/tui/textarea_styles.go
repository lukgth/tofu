package tui

import (
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
)

// textareaStyles paints the palette onto a textarea.
func textareaStyles() textarea.Styles {
	var s textarea.Styles
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent))
	s.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(colorHeader))
	s.Focused.CursorLine = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent))
	s.Blurred.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	s.Blurred.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(colorSelItem))
	s.Cursor.Color = lipgloss.Color(colorAccent)
	return s
}
