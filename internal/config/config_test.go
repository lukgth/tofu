package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `title = "My Tofu Site"
author = "Jane Doe"
description = "A cute little blog"
base_url = "https://example.com"
language = "en"
recent_count = 5
footer = "powered by tofu"

[theme]
width = "720px"
font_main = "Verdana, sans-serif"
font_secondary = "Verdana, sans-serif"
font_scale = "1em"
background = "#fff"
heading = "#222"
text = "#444"
link = "#3273dc"
visited = "#8b6fcb"
blockquote = "#222"
dark_background = "#01242e"
dark_heading = "#eee"
dark_text = "#ddd"
dark_link = "#8cc2dd"
dark_visited = "#8b6fcb"
dark_blockquote = "#ccc"

[header]
title = "My Tofu Site"
[[header.nav]]
label = "Home"
url = "/"
[[header.nav]]
label = "Blog"
url = "/articles/"

[homepage]
heading = "Hello!"
body_file = "content/home.md"
`

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "tofu.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadDefaultsAndClamp(t *testing.T) {
	minimal := "title = \"t\"\n"
	s, err := Load(writeTemp(t, minimal))
	if err != nil {
		t.Fatal(err)
	}
	if s.RecentCount != 5 {
		t.Fatalf("default recent_count = %d, want 5", s.RecentCount)
	}
	s, err = Load(writeTemp(t, "recent_count = 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.RecentCount != 5 {
		t.Fatalf("recent_count 0 => %d, want 5", s.RecentCount)
	}
	s, err = Load(writeTemp(t, "recent_count = -3\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.RecentCount != 5 {
		t.Fatalf("recent_count -3 => %d, want 5", s.RecentCount)
	}
	s, err = Load(writeTemp(t, "recent_count = 99\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.RecentCount != 20 {
		t.Fatalf("recent_count 99 => %d, want 20", s.RecentCount)
	}
}

func TestLoadFullSample(t *testing.T) {
	s, err := Load(writeTemp(t, sample))
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "My Tofu Site" || s.Footer != "powered by tofu" {
		t.Fatalf("bad scalars: %+v", s)
	}
	if len(s.Theme.DarkBackground) == 0 || s.Theme.Link != "#3273dc" {
		t.Fatalf("bad theme: %+v", s.Theme)
	}
	if len(s.Header.Nav) != 2 || s.Header.Nav[1].URL != "/articles/" {
		t.Fatalf("bad nav: %+v", s.Header.Nav)
	}
	if s.Homepage.BodyFile != "content/home.md" {
		t.Fatalf("bad homepage body_file: %q", s.Homepage.BodyFile)
	}
}

func TestLoadUnknownKeyFails(t *testing.T) {
	_, err := Load(writeTemp(t, "title = \"t\"\nbogus_key = 1\n"))
	if err == nil || !strings.Contains(err.Error(), "bogus_key") {
		t.Fatalf("want unknown-key error naming bogus_key, got %v", err)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	s, err := Load(writeTemp(t, sample))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "out.toml")
	if err := Save(out, s); err != nil {
		t.Fatal(err)
	}
	s2, err := Load(out)
	if err != nil {
		t.Fatal(err)
	}
	if s2.Title != s.Title || s2.RecentCount != 5 || len(s2.Header.Nav) != 2 {
		t.Fatalf("round trip mismatch: %+v vs %+v", s2, s)
	}
	if s2.Theme.Link != "#3273dc" {
		t.Fatalf("theme lost: %+v", s2.Theme)
	}
}
