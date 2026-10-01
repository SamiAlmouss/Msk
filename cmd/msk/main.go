package main

import (
	"msk/internal/cli"
	"msk/internal/clipboard"
	"os"
)

var version = "dev"
var commit = "unknown"
var date = "unknown"

func main() {
	os.Exit(cli.App{Clipboard: clipboard.System{}, Out: os.Stdout, Err: os.Stderr, Version: version, Commit: commit, Date: date}.Run(os.Args[1:]))
}
