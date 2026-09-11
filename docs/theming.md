# Theming

How the generated site looks and how to change it.

## Layered model

1. **Theme defaults** (built in): Rubik body text, Ioskeley Mono code, purple/pink light theme, plum dark theme.
2. **`tofu.toml` `[theme]`**: every color and font is a knob. See [configuration.md](configuration.md) for the full table.
3. **`assets-blog/custom.css`**: loaded after `style.css`, wins over everything. Your escape hatch for anything the knobs don't cover.
4. **`static/`**: raw files copied to the output root (buttons, images, fonts, favicon).

## Theme variables

Colors are rendered into `assets-blog/style.css` as CSS custom properties:

```css
:root {
  --text-color: #444444;      /* body + all headings */
  --header-color: #4c3a63;    /* top header title only */
  --link-color: #9d6bb8;      /* links, tags, unread titles */
  --visited-color: #b48cb8;   /* read post titles */
  --accent: #c8b3e8;          /* squiggle divider */
  --highlight: …              /* mark + date pill (see below) */
  --code-bg: …                /* code chips and blocks */
  --primary: …; --secondary: …  /* reserved tints */
}
```

Dark mode flips the same variables via `prefers-color-scheme`, plus a manual override: the site ships a light/dark toggle that sets `html.dark` / `html.light` classes, and the stylesheet mirrors every dark value under both the media query and `html.dark` (manual choice wins over the OS).

## Highlight & date pill

- **Light mode**: the pill derives from your `accent` — `color-mix(in srgb, var(--accent) 55%, white)` — a brighter pastel of whatever accent you pick. The `highlight` knob is unused in light mode.
- **Dark mode**: `dark_highlight` is used directly (default lilac `#c9a0e8`).
- Text on pills/highlights is fixed `#1d1530` (contrast-checked) and is not configurable.

## Fonts

| use | knob | default |
|---|---|---|
| headings + body | `font_secondary` | Rubik (self-hosted: regular, italic, bold) |
| code, `pre`, tags | `font_main` | Ioskeley Mono (self-hosted: regular, bold, italic) |

Self-hosted fonts are embedded in the binary and copied to `assets-blog/fonts/` at build time — zero external requests, works offline. To use your own fonts: point the knobs at any system/fallback stack and declare your own `@font-face` in `custom.css`.

## Code color schemes

Fenced code blocks are highlighted with [Chroma](https://github.com/alecthomas/chroma). Choose the palette per mode in `tofu.toml`:

```toml
[theme]
  code_style      = "catppuccin-latte"   # light mode
  dark_code_style = "catppuccin-mocha"   # dark mode
```

Any Chroma style name is valid; unknown/empty falls back to `github` / `github-dark`. Some pairings that suit a purple site:

| vibe | light | dark |
|---|---|---|
| soft pastel | `catppuccin-latte` | `catppuccin-mocha`, `-frappe`, `-macchiato` |
| purple-ish | `rose-pine-dawn`, `tokyonight-day` | `rose-pine`, `tokyonight-storm`, `tokyonight-night` |
| classic | `github`, `solarized-light`, `friendly` | `github-dark`, `onedark`, `dracula`, `nord`, `gruvbox`, `monokai` |
| warm/retro | `gruvbox-light`, `autumn`, `manni` | `gruvbox`, `doom-one`, `hrdark`, `witchhazel` |

Unknown or empty names fall back to `github` / `github-dark`. Code block *backgrounds* always follow `code_bg` / `dark_code_bg`, so chips and blocks share one surface color regardless of scheme.

## Header anatomy

```text
[big site title h1 — hidden by CSS]
tagline h2 (site title, Rubik, header_color) ← what visitors see
~~~~ squiggle divider (accent color) ~~~~
home  blog            [theme toggle]
```

- The visible header title is `header.title` (falls back to top-level `title`).
- The squiggle divider and the theme toggle button both use `--accent`.

## Common customizations (custom.css recipes)

Make images square-cornered:

```css
img { border-radius: 0; }
```

Un-round the date pill:

```css
time { border-radius: 0; }
```

Style the footer:

```css
footer { font-size: 0.9rem; letter-spacing: 0.05em; }
footer hr { border-top: 1px dashed var(--accent); }
```

Style your 88×31 buttons:

```css
footer img { image-rendering: pixelated; display: inline-block; margin: 0 2px; }
```

## Fonts fallback note

If you replace `font_secondary`/`font_main` with fonts that aren't embedded, visitors without the font installed get your fallback stack. Keep at least one system fallback in the stack.
