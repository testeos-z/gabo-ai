// Package kernel holds the ports shared by several slices. The types are
// owned by the core and never expose adapter details.
package kernel

import (
	"context"
	"io"
)

// Command is a local process invocation. Stdin, when set, is streamed to the
// process; secrets must travel only through Stdin, never through Args or Env.
type Command struct {
	Name  string
	Args  []string
	Env   []string // extra KEY=VALUE pairs appended to the inherited environment
	Dir   string   // working directory; empty means the current one
	Stdin io.Reader
}

// Result is the outcome of a process that ran to completion.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner runs local commands. A non-zero exit code is reported in Result with
// a nil error; an error means the command could not be run at all.
type Runner interface {
	Run(ctx context.Context, cmd Command) (Result, error)
}

// RootUser runs a Script without dropping privileges.
const RootUser = "root"

// Script is a shell script executed on the server as an OS user.
type Script struct {
	User  string
	Body  string
	Stdin io.Reader
}

// RemoteShell runs scripts on the server. Error semantics match Runner.
type RemoteShell interface {
	Exec(ctx context.Context, s Script) (Result, error)
}
