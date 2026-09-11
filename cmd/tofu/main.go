package main

import (
	"fmt"
	"os"

	"tofu/internal/cli"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version)
		os.Exit(0)
	}
	if err := cli.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "tofu:", err)
		os.Exit(1)
	}
}
