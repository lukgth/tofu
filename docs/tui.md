# TUI Reference

`tofu` with no arguments opens the home menu. Every screen is a Bubble Tea app styled in the purple/pink palette.

## Home menu

```text
  tofu

│ New site
│
  New blog post
  Edit existing posts
  Build site
  Preview (serve)
  List posts
  Quit

enter select • / filter • ctrl+c quit
```

| key | action |
|---|---|
| `↑`/`↓`, `j`/`k` | move cursor |
| `/` | filter items (type to narrow, `esc` clears) |
| `enter` | activate the selected item |
| `ctrl+c` | quit |

Items that need a site show `(need site — run New site first)` when no
`tofu.toml` exists. If you activate one, it shows a `run New site first`
hint.

Mouse: left-click an item to move the cursor to it. Click the already-selected item to activate it.

The menu reopens after every action, so labels reflect the current site.

## Wizards (new site, new post, edit)

Wizards are step-by-step prompts. Each step shows a label, one control, and a footer.

| key | action |
|---|---|
| `enter` | accept and go to the next step |
| `esc` | previous step (on the first step: quit) |
| `ctrl+c` | quit immediately |
| `ctrl+o` | body step only: open the draft in `$EDITOR`. The fallback is the `select-editor` choice and runs `select-editor` once if needed |

- Input steps: text inputs with defaults and placeholders. `enter` on an empty title shows `a title is required`.
- Choice steps: every option is a visible rounded card. The selected
  card has an accent border and `●`. The other cards have a dim border
  and `○`. `↑/↓` moves. `enter` picks. Mouse: click a card to pick it.
- Body step: grows with your content (starts at 14 rows, maximum 30).
  It has no line numbers. On new posts it shows the placeholder
  `empty = starter template`. `ctrl+o` opens the current draft in
  `$EDITOR` (multi-word editors like `code --wait` work). If `$EDITOR`
  is unset or unusable, it falls back to the editor chosen by
  `select-editor` (`~/.selected_editor`) and suspends into
  `select-editor` once if needed. When the editor exits, the text returns
  to the textarea and the step stays in place. A nonzero editor exit
  shows an inline `editor failed: …` message and keeps the draft.
- Review screen: a summary and Yes/No cards. `enter` on Yes runs the
  action. `esc` or No returns to step 1. `e` changes the editor (runs
  `select-editor`, and the pick applies to this session).
- Done screen: `✓ Done! …` plus `press enter to go back to the menu`.

## Editing existing posts

`tofu edit` or the menu item opens the picker:

| key | action |
|---|---|
| `↑`/`↓` | move between posts |
| `enter` | open the edit form for the selected post |
| `ctrl+b` | open the file browser instead |
| `esc`/`q` | back out |

The picker is a date|slug|title table (drafts marked `(draft)`). Mouse: click a row to select it.

### File browser (`ctrl+b`)

The browser uses tofu's own name|date|size view over `content/` (directories first, no permission clutter):

| key | action |
|---|---|
| `↑`/`↓` | move |
| `enter` | open a `.md` file in the edit form / descend into a directory |
| `backspace`, `h` | parent directory (stops at the site root) |
| `esc`/`ctrl+c` | back to the table picker |

Mouse: click a row to select it.

Files without frontmatter open a body-only form. The whole file is the
text and is written back verbatim. A file that opens with `---` but has
broken frontmatter is an error, not a plain file. Its metadata is never
folded into its body.

`tofu edit <slug>` resolves the slug to exactly one post. If two files
claim a slug, the command reports the files instead of picking one. The
command rejects a path as a slug.

### Edit form

The form is prefilled from frontmatter: title, date, tags
(comma-separated), and description. Then comes a body-mode choice
(`Quick edit (textarea)` / `Open $EDITOR`) and the body step.

- `Open $EDITOR` skips the textarea and opens the whole file (frontmatter
  + body) in `$EDITOR` after review. If `$EDITOR` is unset or unusable,
  it uses the `select-editor` choice: it runs `select-editor` once, then
  opens the file in the chosen editor. The review screen shows which
  editor will open and where it came from (`from $EDITOR` /
  `from select-editor`). Press `e` to change the editor by re-running
  `select-editor`, and the pick overrides `$EDITOR` for this session.
  The review screen also shows `editor: … (opens after confirm)`.
  Changes to frontmatter from the form are merged after the editor
  exits.
- `Quick edit` writes frontmatter + body directly at the end.

## Progress / preview screens

- Building shows a spinner and a progress bar. Errors return you to the
  wizard step that failed.
- Preview starts a local static server (`public/`) and shows the URL.
  Any key returns to the menu. The server keeps running until you quit
  tofu.

If another process uses the preview port, the build screen reports it
and returns to the menu. It does not show a URL that nothing serves.
