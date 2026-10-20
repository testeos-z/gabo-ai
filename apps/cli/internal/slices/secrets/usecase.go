// Package secrets stores operator secrets on the server for a target OS user.
// Values travel only through the remote shell's stdin; they never appear in
// argv, the environment, scripts or output.
package secrets

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// Sentinel errors.
var (
	ErrUnknownSecret = errors.New("unknown secret")
	ErrEmptyValue    = errors.New("secret value is empty")
)

// Account is an OS user on the server.
type Account struct {
	Name string
	Home string
}

type strategy int

const (
	strategyGHLogin strategy = iota + 1
	strategyFile
)

type secret struct {
	name     string
	owner    Account
	strategy strategy
	file     string // absolute target path for strategyFile
}

// Input selects the secret to set.
type Input struct {
	Name   string
	DryRun bool
}

// Output is either a confirmation message or, on dry run, the remote scripts.
type Output struct {
	Message string
	Scripts []string
}

// UseCase sets secrets on the server.
type UseCase struct {
	shell   kernel.RemoteShell
	reader  SecretReader
	catalog map[string]secret
}

// NewUseCase wires the use case with its ports and the reviewer account.
func NewUseCase(shell kernel.RemoteShell, reader SecretReader, reviewer Account) *UseCase {
	file := func(name, base string) secret {
		return secret{name: name, owner: reviewer, strategy: strategyFile, file: path.Join(reviewer.Home, "secrets", base)}
	}
	return &UseCase{
		shell:  shell,
		reader: reader,
		catalog: map[string]secret{
			"reviewer-github": {name: "reviewer-github", owner: reviewer, strategy: strategyGHLogin},
			"reviewer-claude": file("reviewer-claude", "claude-oauth-token"),
			"reviewer-pi":     file("reviewer-pi", "pi-api-key"),
		},
	}
}

// Names lists the known secret names in sorted order.
func Names() []string {
	return []string{"reviewer-claude", "reviewer-github", "reviewer-pi"}
}

// Set reads the secret value and stores it on the server.
func (u *UseCase) Set(ctx context.Context, in Input) (Output, error) {
	s, ok := u.catalog[in.Name]
	if !ok {
		return Output{}, fmt.Errorf("%w %q (known: %s)", ErrUnknownSecret, in.Name, strings.Join(Names(), ", "))
	}
	steps := s.steps()
	if in.DryRun {
		out := Output{}
		for _, st := range steps {
			out.Scripts = append(out.Scripts, fmt.Sprintf("# as %s\n%s", s.owner.Name, st.body))
		}
		return out, nil
	}

	value, err := u.reader.ReadSecret(fmt.Sprintf("Value for %s: ", s.name))
	if err != nil {
		return Output{}, fmt.Errorf("read secret: %w", err)
	}
	defer clear(value)
	if len(bytes.TrimSpace(value)) == 0 {
		return Output{}, ErrEmptyValue
	}

	for _, st := range steps {
		script := kernel.Script{User: s.owner.Name, Body: st.body}
		if st.stdin {
			script.Stdin = bytes.NewReader(value)
		}
		res, err := u.shell.Exec(ctx, script)
		if err != nil {
			return Output{}, fmt.Errorf("%s: %s: %w", s.name, st.label, err)
		}
		if res.ExitCode != 0 {
			return Output{}, fmt.Errorf("%s: %s exited %d: %s", s.name, st.label, res.ExitCode, redact(res.Stderr, value))
		}
	}
	return Output{Message: fmt.Sprintf("%s stored for %s", s.name, s.owner.Name)}, nil
}

type step struct {
	label string
	body  string
	stdin bool
}

func (s secret) steps() []step {
	switch s.strategy {
	case strategyGHLogin:
		return []step{
			{label: "gh auth login", body: "set -eu\ngh auth login --hostname github.com --git-protocol https --with-token\n", stdin: true},
			{label: "gh auth status", body: "set -eu\ngh auth status --hostname github.com\n"},
		}
	case strategyFile:
		return []step{{label: "write " + s.file, body: fileScript(s.file), stdin: true}}
	default:
		panic(fmt.Sprintf("secrets: unhandled strategy %d", s.strategy))
	}
}

// fileScript writes stdin atomically to target with mode 0600 inside a 0700
// directory owned by the running user.
func fileScript(target string) string {
	dir := path.Dir(target)
	return strings.Join([]string{
		"set -eu",
		"umask 077",
		"dir=" + shellQuote(dir),
		"target=" + shellQuote(target),
		`mkdir -p "$dir"`,
		`if [ "$(stat -c %u "$dir")" != "$(id -u)" ]; then echo "$dir is not owned by $(id -un)" >&2; exit 1; fi`,
		`chmod 700 "$dir"`,
		`tmp=$(mktemp "$dir/.tmp.XXXXXX")`,
		`trap 'rm -f "$tmp"' EXIT`,
		`cat > "$tmp"`,
		`chmod 600 "$tmp"`,
		`mv -f "$tmp" "$target"`,
		"trap - EXIT",
		"",
	}, "\n")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// redact trims remote stderr and removes any echo of the secret from it.
func redact(stderr, value []byte) string {
	msg := string(bytes.TrimSpace(stderr))
	if v := string(bytes.TrimSpace(value)); v != "" {
		msg = strings.ReplaceAll(msg, v, "[REDACTED]")
	}
	if msg == "" {
		return "no stderr"
	}
	return msg
}
