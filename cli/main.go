package main

import (
	"os"

	"github.com/EOEboh/hookdrop/cli/cmd"
)

// Set via goreleaser ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersion(version, commit, date)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
