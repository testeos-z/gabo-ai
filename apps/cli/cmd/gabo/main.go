// Command gabo compiles the gabo agent team (skills and agents) into a project.
//
// This package is the only composition root: it builds driven adapters,
// injects them into slice use cases and registers the driving adapters.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel/execrunner"
)

// buildVersion is injected at build time with -ldflags "-X main.buildVersion=<v>";
// without it the go install module version is used (resolveVersion).
var buildVersion = ""

func main() {
	root := newRootCommand(deps{
		version: resolveVersion(buildVersion, debug.ReadBuildInfo),
		runner:  execrunner.New(),
	})
	if err := root.Execute(); err != nil {
		os.Exit(exitCode(os.Stderr, err))
	}
}

// exitCode prints err to w and returns the process exit code: the code an
// error carries (for example check's 1 and 2), else 1. An error with an empty
// message prints nothing; its output was already written.
func exitCode(w io.Writer, err error) int {
	if msg := err.Error(); msg != "" {
		fmt.Fprintln(w, "gabo:", msg)
	}
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}
