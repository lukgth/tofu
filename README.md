# tofu

a tiny cute static blog generator built with [Bubble Tea], [Bubbles],
[Lip Gloss], and [Harmonica].

```text
$ tofu

  tofu

│ New site

  New blog post

  Edit existing posts

  Build site

  Preview (serve)

  List posts

  Quit


enter select • / filter • ctrl+c quit
```

## Tutorial

Start a site, write a post, build it. `tofu init` also drops a first
post, `hello-tofu.md`.

```sh
$ tofu init mysite
created tofu.toml
created content/home.md
created assets-blog/custom.css
created content/posts/hello-tofu.md
done! try `tofu new` to write a post

$ tofu new --title "Hello World" --slug hello-world
created content/posts/hello-world.md

$ tofu build
Built 4 posts -> public/
```

`tofu` with no arguments opens the home menu. Esc goes back, `ctrl+c`
quits, `/` filters.

## Installation

Requires Go 1.24+.

```sh
git clone https://github.com/lukgth/tofu
cd tofu
CGO_ENABLED=0 go build -ldflags="-s -w" -o tofu ./cmd/tofu
```

## Commands

- `tofu` opens the home menu
- `tofu init [dir]` scaffolds a new site (`--force` for non-empty dirs)
- `tofu new` writes a new blog post
- `tofu edit [slug]` edits an existing post
- `tofu build` renders the site to `public/` (`--out`, `--drafts`)
- `tofu list` lists posts, newest first (`--drafts`)
- `tofu serve` previews locally (`--port`, `--build`)

### tofu new / tofu edit

```sh
$ tofu new --title "Hello World" --slug hello-world
created content/posts/hello-world.md

$ tofu edit hello-world --title "Hello there"
updated content/posts/hello-world.md
```

`new` flags are optional, the wizard asks for whatever's missing. The body
step grows with your text and accepts `ctrl+o` to jump into `$EDITOR`.
`edit` flags (`--title/--tags/--description/--date/--draft-set`) work
headless; with a TTY you get a date|slug|title picker (or `ctrl+b` to
browse files) and a full edit form.

### tofu build / tofu list

```sh
$ tofu build
Built 4 posts -> public/

$ tofu list
2026-09-11 hello-tofu Hello, tofu
2026-09-11 why-tofu Why tofu is cute
2026-09-10 markdown-everywhere Markdown everywhere
```

Drafts get a ` (draft)` suffix and are hidden unless `--drafts`.

### tofu serve

```sh
$ tofu serve --port 8787
serving public at http://127.0.0.1:8787 (ctrl+c to stop)
```

## Config

Everything lives in `tofu.toml` at the site root:

```toml
title = "My Tofu Site"
author = "Jane Doe"
recent_count = 5
footer = "powered by <a href='https://github.com/lukgth/tofu'>tofu</a>"

[theme]
width = "720px"
link = "#9d6bb8"
code_style = "github"
dark_code_style = "github-dark"
```

The full sample (every theme color, nav, homepage, code color schemes) is in
[example/tofu.toml](example/tofu.toml). Unknown keys are errors.

## Docs

- [Configuration](docs/configuration.md) — every `tofu.toml` key, theme colors, code color schemes
- [Content](docs/content.md) — post format, frontmatter, tags, homepage, editing
- [Markdown](docs/markdown.md) — syntax map, `==highlight==`, raw HTML, code blocks, images
- [Theming](docs/theming.md) — layers, fonts, header anatomy, custom.css recipes
- [TUI Reference](docs/tui.md) — menu, wizards, keybindings, mouse, file browser
- [Deploying](docs/deploying.md) — output tree, static hosts, rsync/CI

## Features

- Interactive TUI (menu, wizards, mouse support) — all styling through Bubble Tea + Lip Gloss
- Wizards for site creation, new posts, and editing; `ctrl+o` opens `$EDITOR` on the body step
- Self-hosted fonts (Rubik + Ioskeley Mono) — zero external requests
- Light/dark theme with a visitor toggle, `localStorage` persistence, no flash on load
- Per-tag list pages, RSS feed, drafts
- `==highlight==` markdown, raw HTML passthrough, footnotes
- Syntax-highlighted code blocks with 60+ selectable color schemes
- Everything configurable from `tofu.toml`; `assets-blog/custom.css` loads after the theme

## Deploy

`public/` is a plain static folder. Copy it to your host.

## Contributing

Issues and PRs welcome. MIT license.

[Bubble Tea]: https://github.com/charmbracelet/bubbletea
[Bubbles]: https://github.com/charmbracelet/bubbles
[Lip Gloss]: https://github.com/charmbracelet/lipgloss
[Harmonica]: https://github.com/charmbracelet/harmonica
