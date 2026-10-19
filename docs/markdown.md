# Markdown Reference

This page lists what you can write in posts and `home.md`, and exactly what it renders.

## Syntax map

| markdown | result |
|---|---|
| `**bold**` | **bold** (heading color) |
| `*italic*` | italic |
| `~~strike~~` | strikethrough |
| `==highlight==` | `<mark>`, a lilac highlight chip |
| `inline \`code\`` | mono code chip on `code_bg` |
| `> quote` | blockquote, accent border + tinted surface |
| `- item` | list (1.55 line-height) |
| `1. item` | ordered list |
| `- [x] done` | task list |
| `[^1]` | footnote → "References / Side Notes" block at the end |
| `> [!NOTE]` | alert callout (also `TIP`, `IMPORTANT`, `WARNING`, `CAUTION`) |
| `:smile:` | emoji from the GitHub emoji set |
| `![alt](url)` | image, rounded corners, centered |
| `# Heading` | section heading with an auto `id` anchor (Rubik, no monospace anywhere) |

## Alerts

GitHub alert syntax turns a blockquote into a callout. The marker is the
first line of the blockquote. The alert content follows on the next
lines.

```markdown
> [!NOTE]
> Useful information that users should know.

> [!TIP]
> Helpful advice for doing things better.

> [!IMPORTANT]
> Key information users need to reach their goal.

> [!WARNING]
> Urgent info that needs immediate attention.

> [!CAUTION]
> Advice about risks or negative outcomes.
```

The marker is uppercase. Text on the marker line stays in the alert. Alerts cannot nest inside other elements.

## Emoji

`:shortcode:` renders an emoji from the GitHub emoji set. Unknown names stay as written.

```markdown
Ready to merge :+1: :tada:
```

Code spans keep the shortcode: `` `:smile:` `` renders as text.

## Raw HTML

Raw HTML passes through untouched: `<mark>`, `<details>`, `<summary>`, `<center>`, `<br>`, custom spans, embedded iframes, and everything else.

These examples work with the theme:

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

- Highlighter: [Chroma](https://github.com/alecthomas/chroma) (200+ lexers: python, powershell, go, js, rust, bash, yaml, toml, …)
- Color scheme: configurable per theme mode, see [theming.md](theming.md):

  ```toml
  [theme]
    code_style      = "catppuccin-latte"  # light
    dark_code_style = "catppuccin-mocha"  # dark
  ```

- Block surface color follows `code_bg` / `dark_code_bg`.
- Without a language, the block renders without highlighting on the same surface.
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

- title: the single H1 on the post page. If your body repeats it as an `# H1`, tofu drops the duplicate.
- date: `YYYY-MM-DD` or RFC3339. Newest first everywhere.
- tags: an optional list. It generates `articles/tag/<name>.html` per-tag pages, and the post footer links to them (`tags: tech`).
- draft: `true` hides the post from builds unless you pass `--drafts`. Lists show `(draft)`.
- slug: optional. The default is a slugified title. Duplicate slugs fail the build and name both files.
- Unknown frontmatter keys are preserved on edit (never silently deleted).

## Images

```markdown
![alt text](https://remote/image.jpg)   → remote
![alt text](/photo.png)                 → from your site's static/ folder
```

Images are centered, never exceed the content column, and get rounded corners (15px).

## What renders where

| content | page |
|---|---|
| `content/home.md` | homepage top (recent-posts list below) |
| `content/posts/*.md` | `articles/<slug>.html` |
| tags | `articles/tag/<tag>.html` |
| frontmatter title | browser tab + list rows |

The homepage template never injects its own heading. `home.md` owns the headline (like `# hello!`).
