# Deploying

`public/` is a plain static folder — upload it anywhere that serves files.

## Output tree

```text
public/
├── index.html            ← homepage
├── articles/
│   ├── index.html        ← all posts (newest first)
│   ├── tag/              ← one page per tag
│   │   └── tech.html
│   └── <slug>.html       ← one page per post
├── assets-blog/
│   ├── style.css         ← theme (with your colors baked in)
│   ├── custom.css        ← yours, if present
│   ├── theme-and-visited.js           ← theme toggle + visited-post tracking
│   └── fonts/            ← Rubik + Ioskeley Mono (self-hosted)
├── feed.xml              ← RSS 2.0
└── …                     ← anything from your site's static/
```

## Build & upload

```sh
tofu build
# then, however you deploy:
rsync -av --delete public/ user@host:/var/www/blog/
```

CI one-liner example:

```sh
CGO_ENABLED=0 go build -ldflags="-s -w" -o tofu ./cmd/tofu && ./tofu build && rsync -av --delete public/ deploy@host:/srv/blog/
```

## Static hosts

- **Netlify / Cloudflare Pages / Vercel**: set the build command to `go build -o tofu ./cmd/tofu && ./tofu build` and the publish directory to `public`.
- **GitHub Pages**: push `public/` to your `gh-pages` branch or deploy from a workflow.
- **Any VPS**: `tofu serve` is only for previewing; for production prefer nginx/caddy pointing at the folder.

## Notes

- `base_url` in `tofu.toml` is used for absolute links in `feed.xml` only — pages use root-relative links, so the site works on any domain or subdirectory preview.
- Fonts are self-hosted from `assets-blog/fonts/`: no third-party requests, works offline.
- Drafts are excluded from the output and the feed unless you build with `--drafts`.

## Local preview

```sh
tofu serve --port 8787        # builds first with --build
# open http://127.0.0.1:8787
```
