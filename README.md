# tofu

tofu is a static blog generator written in Go. It reads Markdown posts with
YAML frontmatter and writes a static site that can be served from any web
host. A terminal interface handles site setup, post creation and editing,
builds, and local previews.

You can use tofu to:

- Write posts in Markdown, with GFM tables, task lists, footnotes, alerts,
  emoji shortcodes, and syntax-highlighted code blocks.
- Style the site with a light and dark theme, configurable colors, and an
  optional `assets-blog/custom.css` override.
- Publish tag pages and, when `base_url` is set, an RSS feed.
- Work through menus and wizards, or use flag-driven commands in scripts.

## Features

- Static output: `public/` contains HTML, CSS, JavaScript, and fonts, with no
  server-side runtime.
- Self-hosted Rubik, Ioskeley Mono, and Gelasio fonts. The built-in theme
  makes no external requests.
- A light and dark theme with a visitor toggle that persists in
  `localStorage`.
- Configuration in `tofu.toml`, with unknown keys rejected.

## Getting started

tofu requires Go 1.25 or later.

```sh
git clone https://github.com/lukgth/tofu
cd tofu
make build

./tofu init mysite
cd mysite
../tofu new --title "Hello World" --slug hello-world
../tofu build
../tofu serve
```

`init` creates `tofu.toml`, `content/home.md`, and a first post. `build`
writes the site to `public/` and leaves drafts out unless you add `--drafts`.
`serve` hosts the existing build at `http://127.0.0.1:8787`. Add `--build` to
rebuild before serving.

The build clears the output directory before writing. Do not point `--out` at
a directory that contains other files.

## Documentation

Check out the documentation in [`docs/`](docs/) for configuration, Markdown
syntax, theming, the TUI reference, and deployment. Run `tofu help` or
`tofu <command> --help` for the command list and flags.

## Contributing

Issues and pull requests are welcome. Run `make test` before sending a patch.
tofu is released under the MIT license, see [LICENSE](LICENSE).
