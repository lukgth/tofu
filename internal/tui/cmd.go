package tui

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lukgth/tofu/internal/cli"
)

// AddWizardFlags adds the shared TUI flags to a command.
func AddWizardFlags(cmd *cobra.Command) *Options {
	o := &Options{}
	cmd.Flags().BoolVar(&o.NoColor, "no-color", false, "disable colors and animations")
	cmd.Flags().BoolVar(&o.NoAnimations, "no-animations", false, "render final state instantly")
	cmd.Flags().DurationVar(&o.Timeout, "timeout", 0, "abort after this duration (0 = never)")
	cmd.Flags().BoolVar(&o.ShowHelp, "show-help", true, "show the help footer")
	cmd.Flags().IntVar(&o.Width, "width", 0, "UI width (0 = terminal width)")
	cmd.Flags().IntVar(&o.CharLimit, "char-limit", 400, "per-prompt character limit (0 = unlimited)")
	return o
}

// WizardCommands returns the interactive wizard commands for the root.
func WizardCommands() []*cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init-wizard [dir]",
		Short: "create a new site interactively",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			o := AddWizardFlags(cmd)
			if !cli.IsTTY() {
				_, err := cli.InitScaffold(dir, false)
				return err
			}
			return RunInit(dir, *o)
		},
	}

	newCmd := &cobra.Command{
		Use:   "new-wizard",
		Short: "create a new blog post interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			o := AddWizardFlags(cmd)
			if !cli.IsTTY() {
				_, err := cli.CreatePost(".", cli.NewPostInput{Title: "Untitled"})
				return err
			}
			return RunNew(".", *o)
		},
	}

	editCmd := &cobra.Command{
		Use:   "edit-wizard [slug]",
		Short: "edit a post interactively",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o := AddWizardFlags(cmd)
			slug := ""
			if len(args) == 1 {
				slug = args[0]
			}
			if !cli.IsTTY() {
				return fmt.Errorf("interactive edit needs a TTY; use `tofu edit <slug> --title ...`")
			}
			return RunEdit(".", slug, *o)
		},
	}

	menuCmd := &cobra.Command{
		Use:   "menu",
		Short: "open the tofu home menu",
		RunE: func(cmd *cobra.Command, args []string) error {
			o := AddWizardFlags(cmd)
			if !cli.IsTTY() {
				return fmt.Errorf("the menu needs a TTY")
			}
			return RunMenu(".", *o)
		},
	}

	return []*cobra.Command{initCmd, newCmd, editCmd, menuCmd}
}
