package main

import (
	"os"

	"github.com/dwlhm/nova/internal/cli"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	os.Exit(cli.Run(os.Args[1:], cwd, os.Stdout, os.Stderr))
}
