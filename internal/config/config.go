// Package config defines the only site config format: tofu.toml at the site root.
package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Theme struct {
	Width          string `toml:"width"`
	FontMain       string `toml:"font_main"`
	FontSecondary  string `toml:"font_secondary"`
	FontScale      string `toml:"font_scale"`
	Background     string `toml:"background"`
	Heading        string `toml:"heading"`
	Text           string `toml:"text"`
	Link           string `toml:"link"`
	Visited        string `toml:"visited"`
	Blockquote     string `toml:"blockquote"`
	DarkBackground string `toml:"dark_background"`
	DarkHeading    string `toml:"dark_heading"`
	DarkText       string `toml:"dark_text"`
	DarkLink       string `toml:"dark_link"`
	DarkVisited    string `toml:"dark_visited"`
	DarkBlockquote string `toml:"dark_blockquote"`
}

type NavItem struct {
	Label string `toml:"label"`
	URL   string `toml:"url"`
}

type Header struct {
	Title string    `toml:"title"`
	Nav   []NavItem `toml:"nav"`
}

type Homepage struct {
	Heading  string `toml:"heading"`
	BodyFile string `toml:"body_file"`
}

type Site struct {
	Title       string   `toml:"title"`
	Author      string   `toml:"author"`
	Description string   `toml:"description"`
	BaseURL     string   `toml:"base_url"`
	Language    string   `toml:"language"`
	RecentCount int      `toml:"recent_count"`
	Footer      string   `toml:"footer"`
	Theme       Theme    `toml:"theme"`
	Header      Header   `toml:"header"`
	Homepage    Homepage `toml:"homepage"`
}

// Load parses a tofu.toml. Unknown keys are errors; recent_count clamps to 1..20 with default 5.
func Load(path string) (Site, error) {
	var s Site
	b, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	md, err := toml.Decode(string(b), &s)
	if err != nil {
		return s, fmt.Errorf("parse %s: %w", path, err)
	}
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		return s, fmt.Errorf("unknown key %q in %s", undecoded[0], path)
	}
	if s.RecentCount <= 0 {
		s.RecentCount = 5
	}
	if s.RecentCount > 20 {
		s.RecentCount = 20
	}
	return s, nil
}

func Save(path string, s Site) error {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(s); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
