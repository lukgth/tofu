# Content Guide

Where content lives and how it's written.

## Layout

```text
yoursite/
├── tofu.toml
├── content/
│   ├── home.md              ← homepage body
│   └── posts/               ← one .md per post
│       ├── hello-tofu.md
│       └── my-post.md
├── assets-blog/
│   └── custom.css           ← optional, loads after the theme
└── static/                  ← optional, copied to output root
    └── buttons/88x31.gif
```

## Post format

Each post is markdown with YAML frontmatter:

```markdown
---
title: my cute post
date: 2026-09-12
description: one line for the meta description
tags:
  - life
  - tech
draft: false
slug: my-cute-post        # optional, defaults to slugified title
---

your body here.
```

- **title**: shown in lists, the post page, and the browser tab.
- **date**: `YYYY-MM-DD` (or RFC3339). Posts are listed newest first.
- **tags**: optional list. Each tag gets a list page at `articles/tag/<tag>.html`; the post footer links to it (`tags: tech`).
- **draft**: `true` hides the post from the build unless `tofu build --drafts`. Drafts get a `(draft)` suffix in lists.
- **slug**: optional override; defaults to a slugified title. Duplicate slugs fail the build with both file names.

## Writing

- GFM everywhere: tables, task lists, strikethrough, autolinks.
- `==highlight==` → purple `<mark>` chip.
- Raw HTML passes through: `<mark>`, `<details>`, `<center>`, custom spans, etc.
- Footnotes: `[^1]` with a `[^1]: text` definition → rendered in a "References / Side Notes" block.
- Images: `![alt](https://…)` remote, `![alt](/file.png)` for files in your site's `static/`. Images get rounded corners and center automatically.
- Fenced code blocks with a language get syntax highlighting:

  ````markdown
  ```python
  def hello():
      print("hi")
  ```
  ````

  The color scheme is configurable (`code_style` / `dark_code_style` in `[theme]`).

## Titles

The post page shows exactly one title, from frontmatter. If your body starts with an `# H1` that repeats the title, tofu drops the duplicate automatically. The homepage works the same way: `content/home.md` owns the headline — the template never injects a second one.

## Homepage

`content/home.md` is regular markdown. Everything you write renders at the top of the homepage, followed by the recent-posts list (count from `recent_count`). Set `recent_count = 0` to hide the recent-posts list entirely; the homepage then shows only `content/home.md`.

## Editing

- `tofu edit` (or the home menu) opens a date|slug|title picker of all posts.
- `ctrl+b` on the picker opens a file browser over `content/` — name, date, size, directories first, `backspace` up, `esc` back.
- The edit wizard prefills title/date/tags/description from frontmatter and the body into a textarea.
- On the body step, `ctrl+o` opens the current draft in `$EDITOR` (flags like `code --wait` work). Closing the editor puts the text back in the wizard.
- The body choice step can instead open the *whole file* (frontmatter + body) in `$EDITOR` after review.

## Mouse support

The TUI is mouse-enabled: click a menu entry to move to it, click again to activate; click a row in the edit picker; click a choice card in wizards.
