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

Items needing a site show `(need site — run New site first)` when no `tofu.toml` exists, and activating one shows a `run New site first` hint.

**Mouse**: left-click an item to move the cursor to it; click the already-selected item to activate it.

The menu re-opens after every action so labels reflect the current site.

## Wizards (new site, new post, edit)

Wizards are step-by-step prompts: each step shows a label, one control, and a footer.

| key | action |
|---|---|
| `enter` | accept and go to the next step |
| `esc` | previous step (on the first step: quit) |
| `ctrl+c` | quit immediately |
| `ctrl+o` | **body step only**: open the draft in `$EDITOR` |

- **Input steps**: text inputs with defaults/placeholder. `enter` on an empty title shows `a title is required`.
- **Choice steps**: every option is a visible rounded card. Selected card: accent border + `●`; others: dim border + `○`. `↑/↓` moves, `enter` picks. **Mouse**: click a card to pick it.
- **Body step**: grows with your content (starts 14 rows, max 30), no line numbers, placeholder `empty = starter template` on new posts. `ctrl+o` opens the current draft in `$EDITOR` (multi-word editors like `code --wait` work); on editor exit the text lands back in the textarea and the step stays put. Nonzero editor exit shows an inline `editor failed: …` message and keeps the draft.
- **Review screen**: summary + Yes/No cards. `enter` on Yes runs the action; `esc`/No returns to step 1.
- **Done screen**: `✓ Done! …` plus `press enter to go back to the menu`.

## Editing existing posts

`tofu edit` or the menu item opens the picker:

| key | action |
|---|---|
| `↑`/`↓` | move between posts |
| `enter` | open the edit form for the selected post |
| `ctrl+b` | open the **file browser** instead |
| `esc`/`q` | back out |

The picker is a date|slug|title table (drafts marked `(draft)`). **Mouse**: click a row to select it.

### File browser (`ctrl+b`)

Browses `content/` with tofu's own name|date|size browser (directories first, no permission clutter):

| key | action |
|---|---|
| `↑`/`↓` | move |
| `enter` | open a `.md` file in the edit form / descend into a directory |
| `backspace`, `h` | parent directory (stops at the site root) |
| `esc`/`ctrl+c` | back to the table picker |

**Mouse**: click a row to select it.

### Edit form

Prefilled from frontmatter: title, date, tags (comma-separated), description, then a body-mode choice (`Quick edit (textarea)` / `Open $EDITOR`) and the body step.

- `Open $EDITOR` skips the textarea and, after review, opens the whole file (frontmatter + body) in `$EDITOR`; frontmatter changes from the form are merged after the editor exits.
- `Quick edit` writes frontmatter + body directly at the end.

## Progress / preview screens

- Building shows a spinner + progress bar; errors return you to the wizard step that failed.
- Preview starts a local static server (`public/`) and shows the URL; any key returns to the menu (the server keeps running until you quit tofu).
