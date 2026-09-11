package main

import (
	"fmt"
	"os"

	"tofu/internal/cli"
	"tofu/internal/tui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version)
		os.Exit(0)
	}
	root := cli.NewRoot()
	for _, c := range tui.WizardCommands() {
		root.AddCommand(c)
	}
	// bare tofu with a TTY opens the home menu
	if len(os.Args) == 1 && cli.IsTTY() {
		if err := tui.RunMenu(".", tui.Options{}); err != nil {
			fmt.Fprintln(os.Stderr, "tofu:", err)
			os.Exit(1)
		}
		return
	}
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "tofu:", err)
		os.Exit(1)
	}
}
