// Package web embeds templates and static assets so the binary has zero runtime file deps.
package web

import "embed"

var (
	//go:embed templates
	Templates embed.FS

	//go:embed static
	Static embed.FS
)
