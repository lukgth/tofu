package tui

import (
	"charm.land/bubbles/v2/viewport"
)

// viewportModel wraps a bubbles viewport with sizing helpers.
type viewportModel struct {
	vp viewport.Model
}

func newViewport(width, height int) viewportModel {
	return viewportModel{vp: viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))}
}

func (v *viewportModel) setContent(s string) {
	v.vp.SetContent(s)
}

func (v *viewportModel) view() string {
	return v.vp.View()
}

func (v *viewportModel) setSize(width, height int) {
	v.vp.SetWidth(width)
	v.vp.SetHeight(height)
}
