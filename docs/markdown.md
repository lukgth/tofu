# Markdown Reference

What you can write in posts and `home.md`, and exactly what it renders.

## Syntax map

| markdown | result |
|---|---|
| `**bold**` | **bold** (heading color) |
| `*italic*` | italic |
| `~~strike~~` | strikethrough |
| `==highlight==` | `<mark>` — lilac highlight chip |
| `inline \`code\`` | mono code chip on `code_bg` |
| `> quote` | blockquote, accent border + tinted surface |
| `- item` | list (1.55 line-height) |
| `1. item` | ordered list |
| `- [x] done` | task list |
| `[^1]` | footnote → "References / Side Notes" block at the end |
| `![alt](url)` | image, rounded corners, centered |
| `# Heading` | section heading (Rubik, no monospace anywhere) |

## Raw HTML

Passes through untouched: `<mark>`, `<details>`, `<summary>`, `<center>`, `<br>`, custom spans, embedded iframes — everything.

Useful with the theme:

```markdown
==Broad Consent==                → lilac chip
<mark>Dynamic Consent</mark>     → same chip, raw HTML form
<details>
<summary>sidenote</summary>
hidden content
</details>
<center>☁️☁️☁️</center>          → section break, reference-style
```

## Code blocks

Fenced blocks with a language get syntax highlighting:

````markdown
```python
def greet(name: str) -> str:
    return f"hello, {name}!"
```
````

- Highlighter: [Chroma](https://github.com/alecthomas/chroma) (200+ lexers — python, powershell, go, js, rust, bash, yaml, toml, …)
- Color scheme: configurable per theme mode, see [theming.md](theming.md):

  ```toml
  [theme]
    code_style      = "catppuccin-latte"  # light
    dark_code_style = "catppuccin-mocha"  # dark
  ```

- Block surface color follows `code_bg` / `dark_code_bg`.
- Without a language, the block renders un-highlighted on the same surface.
- Unknown language names degrade to plain text (no build failure).

## Frontmatter

```yaml
---
title: my cute post
date: 2026-09-12
description: one line, goes to meta + lists
tags:
  - life
  - tech
draft: false
slug: my-cute-post
---
```

- **title** — the single H1 on the post page. If your body repeats it as an `# H1`, tofu drops the duplicate.
- **date** — `YYYY-MM-DD` or RFC3339. Newest first everywhere.
- **tags** — optional list; generates `articles/tag/<name>.html` per-tag pages linked from the post footer (`tags: tech`).
- **draft** — `true` hides from builds unless `--drafts`; lists show `(draft)`.
- **slug** — optional; defaults to slugified title. Duplicate slugs fail the build naming both files.
- Unknown frontmatter keys are preserved on edit (never silently deleted).

## Images

```markdown
![alt text](https://remote/image.jpg)   → remote
![alt text](/photo.png)                 → from your site's static/ folder
```

Images center, never exceed the content column, and get rounded corners (15px).

## What renders where

| content | page |
|---|---|
| `content/home.md` | homepage top (recent-posts list below) |
| `content/posts/*.md` | `articles/<slug>.html` |
| tags | `articles/tag/<tag>.html` |
| frontmatter title | browser tab + list rows |

The homepage template never injects its own heading — `home.md` owns the headline (like `# hello!`).
