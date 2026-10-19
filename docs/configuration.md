# Site Configuration (`tofu.toml`)

All configuration lives in `tofu.toml` at the site root. The file uses TOML. Unknown keys cause an error.

## Top level

```toml
title       = "My Tofu Site"       # site name (feeds, titles)
author      = "Jane Doe"
description = "A cute little blog" # meta description, feed description
base_url    = "https://example.com" # used for feed links
language    = "en"                  # <html lang>; defaults to en
recent_count = 5                   # posts on the homepage; 0 hides them (else clamped 1–20)
footer      = "powered by <a href='https://github.com/lukgth/tofu'>tofu</a>"
```

- `footer` renders raw HTML. Links, `<hr>`, and images (88×31 buttons) all work.
- `homepage.heading` only sets the browser tab title. The build does not
  render it into the page body. The headline on the homepage comes from
  `content/home.md` itself.

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

- `header.title` is the single header `h1`. It falls back to the top-level `title` and links home.
- Nav labels are free text. You can use any casing.

## `[homepage]`

```toml
[homepage]
  heading  = "Hello!"         # browser tab title only
  body_file = "content/home.md" # markdown rendered as the homepage body
```

- If `body_file` is empty, the build disables the homepage body. When it
  names a file, that file must exist, must be readable, and must stay
  inside the site. If not, the build fails with a contextual error. An
  existing empty file is valid. To hide recent posts, set
  `recent_count = 0`. The homepage then shows only the rendered
  `home.md`, or just the header and footer when the body file is empty
  or disabled.

## `[theme]`

### Layout & fonts

| key | default | notes |
|---|---|---|
| `width` | `47.5rem` | content column width |
| `font_main` | `"Ioskeley Mono", ui-monospace, monospace` | used for code, `pre`, tags |
| `font_secondary` | `"Rubik", sans-serif` | body text and all headings |
| `font_header` | `"Georgia", "Gelasio", serif` | top header site title only |
| `font_header_style` | `normal` | header title `font-style` (e.g. `italic`, `oblique`) |
| `font_header_weight` | `bold` | header title `font-weight` (e.g. `400`, `bold`, `800`) |
| `font_scale` | `1em` | base font size |

Each font named here must exist on the visitor's system. You can also
declare it with your own `@font-face` in a `custom.css` file that you
add. The tofu theme self-hosts Rubik (regular/italic/bold), Ioskeley
Mono (regular/bold/italic), and Gelasio (regular/italic/bold)
automatically. Gelasio is a metric-compatible stand-in for Georgia and
is registered as the `"Gelasio"` family. The build populates
`assets-blog/fonts/`.

### Colors (light mode)

| key | default | used for |
|---|---|---|
| `background` | `#fffdfa` | page background |
| `text` | `#444444` | body text and all headings |
| `header_color` | `#4c3a63` | only the top header title |
| `link` | `#9d6bb8` | links, tags, unvisited post titles |
| `visited` | `#b48cb8` | visited post titles |
| `blockquote` | `#5c4d6b` | blockquote text |
| `primary` | `#e9dbf5` | reserved tint |
| `secondary` | `#f7f5fa` | blockquote surface |
| `accent` | `#c8b3e8` | squiggle divider under the header |
| `highlight` | `#c9a0e8` | unused in light mode (the light pill derives from `accent`) |
| `code_bg` | `#efedf2` | inline code + code block surface |
| `blink` | `#e4d9f6` | reserved |

### Colors (dark mode)

The dark mode keys have the prefix `dark_`. The keys are
`dark_background` `#372947`, `dark_text` `#e5d9de`,
`dark_header_color` `#f3e8ff`, `dark_link` `#d5b8f2`,
`dark_visited` `#c9aee4`, `dark_blockquote` `#c4b5e0`,
`dark_primary` `#8a6fc0`, `dark_secondary` `#221e44`,
`dark_accent` `#bfa6e8`, `dark_code_bg` `#4a3a63`,
`dark_highlight` `#c9a0e8`.

### Code color schemes

```toml
[theme]
  code_style      = "github"        # light-mode code colors
  dark_code_style = "github-dark"   # dark-mode code colors
```

Any [Chroma](https://github.com/alecthomas/chroma) style name works. Notable entries in the catalog:

- Dark: `github-dark`, `dracula`, `tokyonight-night`, `tokyonight-storm`, `tokyonight-moon`, `gruvbox`, `nord`, `nordic`, `onedark`, `catppuccin-mocha`, `catppuccin-frappe`, `catppuccin-macchiato`, `doom-one`, `rose-pine`, `hrdark`, `monokai`, `solarized-dark`, `vulcan`, `witchhazel`, `evergarden`, `xcode-dark`
- Light styles include `github`, `catppuccin-latte`, `rose-pine-dawn`,
  `tokyonight-day`, `solarized-light`, `gruvbox-light`, `friendly`,
  `tango`, `autumn`, `manni`, `lovelace`. Other light styles are
  `xcode`, `emacs`, `algol`, `pastie`, `perldoc`, `paraiso-light`,
  `modus-operandi`, `bw`, `base16-snazzy`, `native`, `vim`, `fruity`,
  `rainbow_dash`, `colorful`

If a name is unknown or empty, it falls back to `github` / `github-dark`.

### Highlight (mark + date pill)

`highlight` and `dark_highlight` set the color of the `<mark>` highlights and the date pill. In the default theme:

- Light mode: the pill derives from your `accent` (55% toward white), so `highlight` is unused in light mode
- Dark mode: `dark_highlight` is the actual pill and highlight color (`#c9a0e8`)

Text on top of these pills is fixed at `#1d1530` for contrast (WCAG-checked). You cannot configure it.

## Markdown features

- GFM tables, task lists, strikethrough
- `==highlight==` renders `<mark>` (custom extension)
- Footnotes (`[^1]`), rendered in a "References / Side Notes" block
- GitHub alerts: `> [!NOTE]`, `> [!TIP]`, `> [!IMPORTANT]`, `> [!WARNING]`, `> [!CAUTION]`
- Emoji shortcodes: `:smile:`, `:+1:`
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

- Unknown top-level keys, unknown `[theme]` keys, and unknown sections cause errors.
- `recent_count` clamps to the range 1–20. A value of `0` hides the
  recent-posts section on the homepage. Negative values become 5.
  Values `>20` become 20. A missing key becomes 5.
