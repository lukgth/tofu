# Content Guide

Where content lives and how to write it.

## Layout

```text
yoursite/
├── tofu.toml
├── content/
│   ├── home.md              ← homepage body
│   └── posts/               ← one .md per post
│       ├── hello-tofu.md
│       └── my-post.md
├── assets-blog/             ← optional; create it to override the theme
│   └── custom.css           ← loaded after style.css
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

- title: shown in lists, the post page, and the browser tab.
- date: `YYYY-MM-DD` (or RFC3339). Posts are listed newest first.
- tags: an optional list. Each tag gets a list page at `articles/tag/<tag>.html`. The post footer links to it (`tags: tech`).
- draft: `true` hides the post from the build unless you run `tofu build --drafts`. Drafts get a `(draft)` suffix in lists.
- slug: an optional override. The default is a slugified title. Duplicate slugs fail the build and report both file names.

## Writing

- GFM everywhere: tables, task lists, strikethrough, autolinks.
- `==highlight==` → purple `<mark>` chip.
- Raw HTML passes through: `<mark>`, `<details>`, `<center>`, custom spans, etc.
- GitHub alerts: `> [!NOTE]`, `> [!TIP]`, `> [!IMPORTANT]`, `> [!WARNING]`, `> [!CAUTION]`.
- Emoji shortcodes: `:smile:`, `:+1:`. Unknown shortcodes stay as written.
- Footnotes: `[^1]` with a `[^1]: text` definition → rendered in a "References / Side Notes" block.
- Images: `![alt](https://…)` remote, `![alt](/file.png)` for files in your site's `static/`. Images get rounded corners and are centered automatically.
- Fenced code blocks with a language get syntax highlighting:

  ````markdown
  ```python
  def hello():
      print("hi")
  ```
  ````

  The color scheme is configurable (`code_style` / `dark_code_style` in `[theme]`).

## Titles

The post page shows exactly one title, and that title comes from
frontmatter. If your body starts with an `# H1` that repeats the title,
tofu drops the duplicate automatically. The homepage works the same
way. `content/home.md` owns the headline, and the template never
injects a second one.

## Homepage

`content/home.md` is regular markdown. Everything you write renders at
the top of the homepage. The recent-posts list follows it, and
`recent_count` sets the count. Set `recent_count = 0` to hide the
recent-posts list. The homepage then shows only `content/home.md`.

## Editing

- `tofu edit` (or the home menu) opens a date|slug|title picker of all posts.
- `ctrl+b` on the picker opens a file browser over `content/`. It shows
  name, date, and size, with directories first. `backspace` goes up.
  `esc` goes back.
- The edit wizard prefills the title, date, tags, and description from
  frontmatter, and puts the body into a textarea.
- On the body step, `ctrl+o` opens the current draft in `$EDITOR`
  (flags like `code --wait` work). When the editor closes, the text goes
  back into the wizard.
- The body choice step can instead open the whole file (frontmatter +
  body) in `$EDITOR` after review.

## Mouse support

The TUI supports the mouse. Click a menu entry to move to it, and click
again to activate it. Click a row in the edit picker. Click a choice
card in wizards.
