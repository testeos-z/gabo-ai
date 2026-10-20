// Command gbai provisions and operates autonomous cloud coding agents.
//
// This package is the only composition root: it builds driven adapters,
// injects them into slice use cases and registers the driving adapters.
package main

import (
	"fmt"
	"os"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/execrunner"
)

// buildVersion is injected at build time with -ldflags "-X main.buildVersion=<v>".
var buildVersion = ""

func main() {
	root := newRootCommand(deps{
		version: buildVersion,
		runner:  execrunner.New(),
		stdin:   os.Stdin,
		stderr:  os.Stderr,
	})
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "gbai:", err)
		os.Exit(1)
	}
}
