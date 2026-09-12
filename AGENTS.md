# AGENTS

## build/test

- `go build ./...`
- `go test ./...`
- release binary: `CGO_ENABLED=0 go build -o tofu ./cmd/tofu`
- `make build`, `make test` wrap the same

## layout

- `cmd/tofu/` — main(); assembles cobra root, dispatches bare `tofu` to the menu
- `internal/config/` — `tofu.toml` model (`config.Site`, strict TOML load/save)
- `internal/post/` — frontmatter parse, slugify, list, `Create`-style writes, `UpdateFrontmatter`
- `internal/render/` — goldmark markdown + Bear-style HTML output tree
- `internal/site/` — config + posts loader (drops drafts unless asked)
- `internal/cli/` — headless command implementations shared by cobra and the TUI
- `internal/tui/` — every bubbletea screen: menu, wizards, skin (`styles.go`), springs (`anim.go`)
- `web/templates/`, `web/static/` — embedded via `web/embed.go`
- `example/` — fixture site built by `tofu init` plus two posts

## rules

- site config is TOML (`tofu.toml`, BurntSushi); post frontmatter is YAML between `---` (yaml.v3). never mix.
- output literals are fixed: `articles/`, `assets-blog/`, `public/`. never rename.
- all TUI goes through `internal/tui`: styles from `styles.go`, springs from `anim.go`, bubbles + lipgloss + harmonica only. no gum binary, no huh.
- palette is purple + pink only: title `#C084FC`, accent `#F472B6`, dim/help `#B8A6E3`, error `#FB7185`, borders `#F472B6`/`#6D5A9E`, list header/cursor/selected `#E9D5FF`/`#C084FC`/`#F9A8D4`. progress uses `progress.WithDefaultBlend()`.
- menu actions reuse the cli helpers (`InitScaffold`, `CreatePost`, `ListPosts`, `CountPosts`, `render.Build`) — never duplicate logic.
- `recent_count` clamps to 1–20, default 5; `0` hides the homepage recents (negatives → 5, >20 → 20, missing → 5).
- comments are minimal: only for non-obvious whys. no narration, no restating code.
- TTY detection is stdlib only (`os.Stdout.Stat()` + `ModeCharDevice`), no isatty dep.
- empty/missing `content/home.md` or `static/` never fails a build.

## e2e

From repo root, no network needed (commands assume a built binary at `/tmp/tofu`):

1. `CGO_ENABLED=0 go build -o /tmp/tofu ./cmd/tofu && /tmp/tofu version`
2. `/tmp/tofu init /tmp/tofusite && cd /tmp/tofusite && /tmp/tofu new --title "Hello" --slug hello --tags intro --description hi`
3. `/tmp/tofu build --out /tmp/tofusite/public` → `test -f public/index.html public/articles/hello.html public/articles/index.html public/assets-blog/style.css public/feed.xml`, `grep -c '<li>' public/index.html` shows the recent count, `grep assets-blog/style.css -e '--width' -e 'prefers-color-scheme'`
4. `/tmp/tofu serve --port 8787 &` then `curl -s localhost:8787/ | grep -i tofu`; kill the server
5. `echo | /tmp/tofu edit --help` works headless; `/tmp/tofu edit hello --title "Hi"` updates the file without a TTY
