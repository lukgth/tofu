package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"

	"tofu/internal/cli"
)

// RunNew is the new-post wizard.
func RunNew(root string, opts Options) error {
	o := normalize(opts)
	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)

	titleIn := newTextInput("my cute post", "", o)
	dateIn := newTextInput("YYYY-MM-DD", time.Now().Format("2006-01-02"), o)
	tagsIn := newTextInput("intro, tips (comma separated)", "", o)
	descIn := newTextInput("one line description", "", o)
	slugIn := newTextInput("my-cute-post", "", o)
	body := newBodyArea("", o)
	body.Placeholder = "empty = starter template"

	in := cli.NewPostInput{Title: "Untitled"}
	draft := false
	draftChoice := newChoice([]string{"No", "Yes"}, isDark, o.Width)

	steps := []step{
		{kind: stepInput, label: "title", input: &titleIn, setter: func(v string) error {
			if v == "" {
				return fmt.Errorf("a title is required")
			}
			in.Title = v
			return nil
		}},
		{kind: stepInput, label: "slug", input: &slugIn, setter: func(v string) error {
			in.Slug = v
			return nil
		}},
		{kind: stepInput, label: "date", input: &dateIn, setter: func(v string) error {
			if v == "" {
				v = time.Now().Format("2006-01-02")
			}
			if _, err := time.Parse("2006-01-02", v); err != nil {
				return fmt.Errorf("bad date %q, want YYYY-MM-DD", v)
			}
			in.Date = v
			return nil
		}},
		{kind: stepInput, label: "tags", input: &tagsIn, setter: func(v string) error {
			in.Tags = cli.ParseTags(v)
			return nil
		}},
		{kind: stepInput, label: "description", input: &descIn, setter: func(v string) error {
			in.Description = v
			return nil
		}},
		{kind: stepChoice, label: "draft?", choice: &draftChoice, setter: func(v string) error {
			draft = v == "Yes"
			in.Draft = draft
			return nil
		}},
		{kind: stepBody, label: "body", area: &body, setter: func(v string) error {
			if strings.TrimSpace(v) != "" {
				in.Body = v
			}
			return nil
		}},
	}

	w := &wizardModel{
		root:     root,
		opts:     o,
		title:    "new post",
		isDark:   isDark,
		spring:   NewSlideSpring(),
		steps:    steps,
		spinnerM: newSpinnerModel(),
		prog:     newProgressModel(o.Width - 8),
		vp:       newViewport(o.Width, 14),
		finish: func(w *wizardModel) error {
			in.Slug = cli.SlugFor(in)
			rel, err := cli.CreatePost(w.root, in)
			if err != nil {
				return err
			}
			w.doneMsg = "created " + rel
			return nil
		},
		summary: func(w *wizardModel) string {
			slug := cli.SlugFor(in)
			lines := []string{
				Title.Render("review"),
				"title: " + in.Title,
				"slug: " + slug,
				"date: " + in.Date,
				"tags: " + strings.Join(in.Tags, ", "),
				"description: " + in.Description,
				fmt.Sprintf("draft: %v", in.Draft),
				"file: " + filepath.Join("content", "posts", slug+".md"),
			}
			return strings.Join(lines, "\n")
		},
	}
	w.slide = NewSlide(8)
	return runWizardProgram(w)
}

// ensure the editor choice and key bindings stay referenced if reused later.
var _ = key.NewBinding
var _ = lipgloss.JoinVertical
