# Site Configuration (`tofu.toml`)

Everything lives in `tofu.toml` at your site root. TOML, strict: unknown keys are errors.

## Top level

```toml
title       = "My Tofu Site"       # site name (feeds, titles)
author      = "Jane Doe"
description = "A cute little blog" # meta description, feed description
base_url    = "https://example.com" # used for feed links
language    = "en"
recent_count = 5                   # posts on the homepage (clamped 1–20)
footer      = "powered by <a href='https://github.com/lukgth/tofu'>tofu</a>"
```

- `footer` renders **raw HTML**, so links, `<hr>`, images (88×31 buttons) all work.
- `homepage.heading` only feeds the browser tab title — it is not rendered into the page body. The homepage headline comes from `content/home.md` itself.

## `[header]`

```toml
[header]
  title = "My Tofu Site"   # shown in the header (falls back to top-level title)

  [[header.nav]]
    label = "home"
    url = "/"

  [[header.nav]]
    label = "blog"
    url = "/articles/"
```

- `header.title` is displayed as the big header title (CSS hides the duplicate site-title link behind it).
- Nav labels are free text; use any casing you want.

## `[homepage]`

```toml
[homepage]
  heading  = "Hello!"         # browser tab title only
  body_file = "content/home.md" # markdown rendered as the homepage body
```

- Empty/missing `home.md` never fails a build — the homepage just shows the recent-posts list.

## `[theme]`

### Layout & fonts

| key | default | notes |
|---|---|---|
| `width` | `47.5rem` | content column width |
| `font_main` | `"Ioskeley Mono", ui-monospace, monospace` | used for code, `pre`, tags |
| `font_secondary` | `"Rubik", sans-serif` | body text and all headings |
| `font_scale` | `1em` | base font size |

Fonts named here must exist on the visitor's system or via your own `@font-face` in `custom.css`. The tofu theme self-hosts Rubik (regular/italic/bold) and Ioskeley Mono (regular/bold/italic) automatically — `assets-blog/fonts/` is populated at build time.

### Colors (light mode)

| key | default | used for |
|---|---|---|
| `background` | `#fffdfa` | page background |
| `text` | `#444444` | body text **and all headings** |
| `header_color` | `#4c3a63` | only the top header title |
| `link` | `#9d6bb8` | links, tags, unvisited post titles |
| `visited` | `#b48cb8` | visited post titles |
| `blockquote` | `#5c4d6b` | blockquote text |
| `primary` | `#e9dbf5` | reserved tint |
| `secondary` | `#f7f5fa` | blockquote surface |
| `accent` | `#c8b3e8` | squiggle divider under the header |
| `highlight` | `#c9a0e8` | **unused in light mode** (light pill derives from `accent`) |
| `code_bg` | `#efedf2` | inline code + code block surface |
| `blink` | `#e4d9f6` | reserved |

### Colors (dark mode)

Same keys prefixed `dark_`: `dark_background` `#372947`, `dark_text` `#e5d9de`, `dark_header_color` `#f3e8ff`, `dark_link` `#d5b8f2`, `dark_visited` `#c9aee4`, `dark_blockquote` `#c4b5e0`, `dark_primary` `#8a6fc0`, `dark_secondary` `#221e44`, `dark_accent` `#bfa6e8`, `dark_code_bg` `#4a3a63`, `dark_highlight` `#c9a0e8`.

### Code color schemes

```toml
[theme]
  code_style      = "github"        # light-mode code colors
  dark_code_style = "github-dark"   # dark-mode code colors
```

Any [Chroma](https://github.com/alecthomas/chroma) style name works. Highlights of the catalog:

- **Dark:** `github-dark`, `dracula`, `tokyonight-night`, `tokyonight-storm`, `tokyonight-moon`, `gruvbox`, `nord`, `nordic`, `onedark`, `catppuccin-mocha`, `catppuccin-frappe`, `catppuccin-macchiato`, `doom-one`, `rose-pine`, `hrdark`, `monokai`, `solarized-dark`, `vulcan`, `witchhazel`, `evergarden`, `xcode-dark`
- **Light:** `github`, `catppuccin-latte`, `rose-pine-dawn`, `tokyonight-day`, `solarized-light`, `gruvbox-light`, `friendly`, `tango`, `autumn`, `manni`, `lovelace`, `xcode`, `emacs`, `algol`, `pastie`, `perldoc`, `paraiso-light`, `modus-operandi`, `bw`, `base16-snazzy`, `native`, `vim`, `fruity`, `rainbow_dash`, `colorful`

Unknown or empty names fall back to `github` / `github-dark`.

### Highlight (mark + date pill)

`highlight` / `dark_highlight` color the `<mark>` highlights and the date pill. In the default theme:

- **light mode**: the pill *derives from your `accent`* (55% toward white), so `highlight` is unused in light mode
- **dark mode**: `dark_highlight` is the actual pill/highlight color (`#c9a0e8`)

Text on top of these pills is fixed `#1d1530` for contrast (WCAG-checked) — not configurable.

## Markdown features

- GFM tables, task lists, strikethrough
- `==highlight==` renders `<mark>` (custom extension)
- Footnotes (`[^1]`), rendered in a "References / Side Notes" block
- Raw HTML passes through (`<mark>`, `<details>`, `<center>`, …)
- Fenced code blocks with a language get syntax highlighting:
  ````markdown
  ```python
  def hello(): ...
  ```
  ````
- Images: `![alt](url)` for remote, `![alt](/file.png)` for files in your site's `static/`
- Tags in frontmatter generate per-tag list pages at `articles/tag/<name>.html`

## Strictness

- Unknown top-level keys, unknown `[theme]` keys, unknown sections: **errors**.
- `recent_count` clamps to 1–20 (0/negative → 5, >20 → 20).
