// Package site loads a tofu site: config plus posts.
package site

import (
	"path/filepath"

	"tofu/internal/config"
	"tofu/internal/post"
)

type Site struct {
	Config config.Site
	Posts  []post.Post
}

// Load reads tofu.toml and content/posts. Drafts are dropped unless includeDrafts.
func Load(siteRoot string, includeDrafts bool) (Site, error) {
	cfg, err := config.Load(filepath.Join(siteRoot, "tofu.toml"))
	if err != nil {
		return Site{}, err
	}
	posts, err := post.List(filepath.Join(siteRoot, "content"))
	if err != nil {
		return Site{}, err
	}
	if !includeDrafts {
		kept := posts[:0]
		for _, p := range posts {
			if !p.Draft {
				kept = append(kept, p)
			}
		}
		posts = kept
	}
	return Site{Config: cfg, Posts: posts}, nil
}
