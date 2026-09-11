# tofu 🧊

a tiny cute static blog generator built with [Bubble Tea], [Bubbles],
[Lip Gloss], and [Harmonica].

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

`new` flags are optional, the wizard asks for whatever's missing. `edit`
flags (`--title/--tags/--description/--date/--draft-set`) work headless.
With a TTY you get a date|slug|title table and a full editor flow.

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
footer = "Made with tofu 🧊"

[theme]
width = "720px"
link = "#3273dc"
```

The full sample (every theme color, nav, homepage heading) is in
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
`assets-blog/custom.css` loads after `style.css`, so it wins.

## Deploy

`public/` is a plain static folder. Copy it to your host.

## Contributing

Issues and PRs welcome. MIT license.

[Bubble Tea]: https://github.com/charmbracelet/bubbletea
[Bubbles]: https://github.com/charmbracelet/bubbles
[Lip Gloss]: https://github.com/charmbracelet/lipgloss
[Harmonica]: https://github.com/charmbracelet/harmonica
