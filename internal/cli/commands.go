package cli

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/lukgth/tofu/internal/config"
	"github.com/lukgth/tofu/internal/post"
	"github.com/lukgth/tofu/internal/render"
	"github.com/spf13/cobra"
)

// NewRoot builds the tofu command tree.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "tofu",
		Short:         "a tiny cute static blog generator",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newInitCmd(), newNewCmd(), newEditCmd(), newBuildCmd(), newListCmd(), newServeCmd(), newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print the tofu version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(Version)
			return nil
		},
	}
}

// Version is set at build time via -ldflags "-X tofu/internal/cli.Version=X".
var Version = "dev"

func newInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "create a new site (default: current directory)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			created, err := InitScaffold(dir, force)
			if err != nil {
				return err
			}
			if len(created) == 0 {
				fmt.Println("nothing to do; site files already exist")
				return nil
			}
			for _, rel := range created {
				fmt.Println("created", rel)
			}
			fmt.Println("done! try `tofu new` to write a post")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "write into a non-empty directory")
	return cmd
}

func newNewCmd() *cobra.Command {
	var title, slug, tags, desc, date, body, asset string
	var draft bool
	cmd := &cobra.Command{
		Use:   "new",
		Short: "create a new blog post",
		RunE: func(cmd *cobra.Command, args []string) error {
			in := NewPostInput{
				Title:       title,
				Slug:        slug,
				Date:        date,
				Tags:        ParseTags(tags),
				Description: desc,
				Draft:       draft,
				Body:        body,
				AssetPath:   asset,
			}
			if in.Title == "" {
				in.Title = "Untitled"
			}
			rel, err := CreatePost(".", in)
			if err != nil {
				return err
			}
			fmt.Println("created", rel)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "post title")
	cmd.Flags().StringVar(&slug, "slug", "", "post slug (default: slugified title)")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	cmd.Flags().StringVar(&desc, "description", "", "post description")
	cmd.Flags().StringVar(&date, "date", "", "post date, YYYY-MM-DD (default: today)")
	cmd.Flags().StringVar(&body, "body", "", "markdown body (default: starter template)")
	cmd.Flags().StringVar(&asset, "asset", "", "optional asset path to note in frontmatter")
	cmd.Flags().BoolVar(&draft, "draft", false, "mark the post as draft")
	return cmd
}

func newEditCmd() *cobra.Command {
	var title, tags, desc, date string
	var draftSet bool
	var draft bool
	cmd := &cobra.Command{
		Use:   "edit [slug]",
		Short: "edit an existing post's frontmatter",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("usage: tofu edit [slug] (interactive editing needs a TTY)")
			}
			slug := args[0]
			path, err := findPostBySlug(slug)
			if err != nil {
				return err
			}
			any := false
			var mutators []func(*post.Frontmatter) error
			if cmd.Flags().Changed("title") {
				mutators = append(mutators, func(f *post.Frontmatter) error { f.Title = title; return nil })
				any = true
			}
			if cmd.Flags().Changed("tags") {
				mutators = append(mutators, func(f *post.Frontmatter) error { f.Tags = ParseTags(tags); return nil })
				any = true
			}
			if cmd.Flags().Changed("description") {
				mutators = append(mutators, func(f *post.Frontmatter) error { f.Description = desc; return nil })
				any = true
			}
			if cmd.Flags().Changed("date") {
				if _, err := time.Parse("2006-01-02", date); err != nil {
					if _, err2 := time.Parse(time.RFC3339, date); err2 != nil {
						return fmt.Errorf("bad date %q (want YYYY-MM-DD)", date)
					}
				}
				mutators = append(mutators, func(f *post.Frontmatter) error { f.Date = date; return nil })
				any = true
			}
			if draftSet {
				mutators = append(mutators, func(f *post.Frontmatter) error { f.Draft = draft; return nil })
				any = true
			}
			if !any {
				cmd.Println("use interactive mode (TTY) or pass flags like --title")
				os.Exit(2)
			}
			if err := post.UpdateFrontmatter(path, func(f *post.Frontmatter) error {
				for _, m := range mutators {
					if err := m(f); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				return err
			}
			fmt.Println("updated", path)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&tags, "tags", "", "new comma-separated tags")
	cmd.Flags().StringVar(&desc, "description", "", "new description")
	cmd.Flags().StringVar(&date, "date", "", "new date, YYYY-MM-DD")
	cmd.Flags().BoolVar(&draft, "draft", false, "draft state")
	cmd.Flags().BoolVar(&draftSet, "draft-set", false, "actually change the draft flag")
	return cmd
}

func findPostBySlug(slug string) (string, error) {
	posts, err := post.List("content")
	if err != nil {
		return "", err
	}
	for _, p := range posts {
		// Return the file we actually parsed: a post's frontmatter slug may
		// differ from its filename, so rebuilding the path from the slug
		// would miss the file (or hit a different one).
		if p.Slug == slug {
			return p.Path, nil
		}
	}
	return "", fmt.Errorf("no post with slug %q", slug)
}

func newBuildCmd() *cobra.Command {
	var out string
	var drafts bool
	cmd := &cobra.Command{
		Use:   "build",
		Short: "render the site to the output directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := render.Build(".", out, drafts); err != nil {
				return err
			}
			n, _ := countPosts()
			fmt.Printf("Built %d posts -> %s/\n", n, out)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "public", "output directory")
	cmd.Flags().BoolVar(&drafts, "drafts", false, "include draft posts")
	return cmd
}

func countPosts() (int, error) {
	posts, err := post.List("content")
	if err != nil {
		return 0, err
	}
	n := 0
	for _, p := range posts {
		if !p.Draft {
			n++
		}
	}
	return n, nil
}

func newListCmd() *cobra.Command {
	var drafts bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list posts, newest first",
		RunE: func(cmd *cobra.Command, args []string) error {
			posts, err := post.List("content")
			if err != nil {
				return err
			}
			for _, p := range posts {
				if p.Draft && !drafts {
					continue
				}
				line := fmt.Sprintf("%s %s %s", p.Date.Format("2006-01-02"), p.Slug, p.Frontmatter.Title)
				if p.Draft {
					line += " (draft)"
				}
				fmt.Println(line)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&drafts, "drafts", false, "include draft posts")
	return cmd
}

func newServeCmd() *cobra.Command {
	var out string
	var port int
	var build bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "preview the built site locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			if build {
				if err := render.Build(".", out, false); err != nil {
					return err
				}
				n, _ := countPosts()
				fmt.Printf("Built %d posts -> %s/\n", n, out)
			}
			addr := fmt.Sprintf("127.0.0.1:%d", port)
			fmt.Printf("serving %s at http://%s (ctrl+c to stop)\n", out, addr)
			handler := http.FileServer(http.Dir(out))
			srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
			return srv.ListenAndServe()
		},
	}
	cmd.Flags().StringVar(&out, "out", "public", "directory to serve")
	cmd.Flags().IntVar(&port, "port", 8787, "port to listen on")
	cmd.Flags().BoolVar(&build, "build", false, "rebuild before serving")
	return cmd
}

var _ = config.Site{}
