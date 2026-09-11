# tofu 🧊

a tiny cute static blog generator. one binary, no fuss.

built with [Bubble Tea], [Bubbles], [Lip Gloss], and [Harmonica].

inspired by [Bear Blog], [hugo-bearblog], and [Charm]'s good taste.
not affiliated — just a fan.

```text
$ tofu

  tofu 🧊

  • New site            • Build site
  • New blog post       • Preview (serve)
  • Edit existing posts • List posts
                        • Quit

  enter select • / filter • ctrl+c quit
```

## Tutorial

Let's start a site, write a post, and build it. (`tofu init` also drops
a first post, `hello-tofu.md`.)

```sh
$ tofu init mysite
created tofu.toml
created content/home.md
created assets-blog/custom.css
created content/posts/hello-tofu.md
cute! now try `tofu new`

$ tofu new --title "Hello World" --slug hello-world
created content/posts/hello-world.md

$ tofu build
Built 4 posts -> public/
```

`tofu` with no arguments opens the home menu. Esc goes back, `ctrl+c`
quits, `/` filters.

## Installation

Build (requires Go 1.24+); the binary is static with zero runtime files:

```sh
git clone https://github.com/you/tofu
cd tofu
CGO_ENABLED=0 go build -ldflags="-s -w" -o tofu ./cmd/tofu
```

Drop it anywhere.

## Commands

- `tofu` — open the home menu (interactive)
- `tofu init [dir]` — scaffold a new site (`--force` to write into a non-empty dir)
- `tofu new` — write a new blog post
- `tofu edit [slug]` — edit an existing post
- `tofu build` — render the site to `public/` (`--out`, `--drafts`)
- `tofu list` — list posts, newest first (`--drafts`)
- `tofu serve` — preview locally (`--port`, `--build`)

### tofu new / tofu edit

```sh
$ tofu new --title "Hello World" --slug hello-world
created content/posts/hello-world.md

$ tofu edit hello-world --title "Hello there"
updated content/posts/hello-world.md
```

`new` flags are optional; without them the wizard asks. `edit` flags
(`--title/--tags/--description/--date/--draft-set`) work headless; with
a TTY you get a date|slug|title table and a full editor flow.

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

One file, `tofu.toml`, at the site root:

```toml
title = "My Tofu Site"
author = "Jane Doe"
recent_count = 5
footer = "Made with tofu 🧊"

[theme]
width = "720px"
link = "#3273dc"
```

The full sample — every theme color, nav, homepage heading — is in
[example/tofu.toml](example/tofu.toml). Unknown keys are errors.

## Output

```text
public/
├── index.html
├── articles/        (index.html + one html per post)
├── assets-blog/     (style.css + your custom.css)
└── feed.xml
```

## Customize

Every color is a `[theme]` variable, including the full dark-mode set.
`assets-blog/custom.css` loads after `style.css` and is yours entirely.

## Deploy

`public/` is plain static files — copy it anywhere. Done.

## Contributing

Issues and PRs welcome. MIT license.

[Bubble Tea]: https://github.com/charmbracelet/bubbletea
[Bubbles]: https://github.com/charmbracelet/bubbles
[Lip Gloss]: https://github.com/charmbracelet/lipgloss
[Harmonica]: https://github.com/charmbracelet/harmonica
[Bear Blog]: https://bearblog.dev
[hugo-bearblog]: https://github.com/janraasch/hugo-bearblog
[Charm]: https://charm.sh
