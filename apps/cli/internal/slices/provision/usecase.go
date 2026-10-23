// Package provision prepares the server, as root, for the reviewer flow in
// single-user mode: packages, a check that the orca user exists, the gbai
// binary and config, removal of the retired three-user sudo path and the
// embedded skills. The reviewer runs as the orca user itself, so no OS user,
// sudo rule or wrapper is created. Every step is idempotent.
package provision

import (
	"bytes"
	"context"
	"debug/elf"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"text/template"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// Fixed install locations on the server.
const (
	BinaryPath = kernel.ServerBinaryPath
	ConfigPath = kernel.ServerConfigPath
)

// ErrNotLinuxBinary is returned when the binary to install is not a linux ELF executable.
var ErrNotLinuxBinary = errors.New("not a linux ELF executable")

var (
	//go:embed provision.sh.tmpl
	templateFS embed.FS
	scripts    = template.Must(template.New("").Funcs(template.FuncMap{"q": shellQuote}).ParseFS(templateFS, "provision.sh.tmpl"))
	userName   = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	safePath   = regexp.MustCompile(`^/[A-Za-z0-9._/'-]*$`)
	stepOrder  = []string{"packages", "user", "config", "binary", "legacy", "skills"}
)

// Account is an OS user on the server.
type Account struct {
	Name string
	Home string
	UID  int
	Path []string
}

// Settings are the values rendered into the provisioning scripts.
type Settings struct {
	Orca      Account // runs the sessions and, in single-user mode, the reviewer
	Config    []byte  // gbai.yaml installed at ConfigPath; must not contain secrets
	SkillsTar []byte  // tar of the embedded skills, installed for the orca user
}

// Input selects the binary to install and whether to only print the plan.
type Input struct {
	BinaryPath string
	DryRun     bool
}

// Output holds the plan (dry run) or one result line per step.
type Output struct {
	Lines []string
}

// UseCase provisions the server.
type UseCase struct {
	shell    kernel.RemoteShell
	binary   BinarySource
	settings Settings
}

// NewUseCase wires the use case with its ports.
func NewUseCase(shell kernel.RemoteShell, binary BinarySource, s Settings) *UseCase {
	return &UseCase{shell: shell, binary: binary, settings: s}
}

// Run renders the steps and, unless DryRun, executes them in order as root.
func (u *UseCase) Run(ctx context.Context, in Input) (Output, error) {
	steps, err := render(u.settings)
	if err != nil {
		return Output{}, err
	}
	if in.DryRun {
		out := Output{}
		for _, s := range steps {
			out.Lines = append(out.Lines, s.body)
		}
		return out, nil
	}

	bin, err := u.binary.ReadBinary(in.BinaryPath)
	if err != nil {
		return Output{}, fmt.Errorf("read binary: %w", err)
	}
	if err := checkLinuxELF(bin); err != nil {
		return Output{}, fmt.Errorf("binary %s: %w", in.BinaryPath, err)
	}

	out := Output{}
	for _, s := range steps {
		script := kernel.Script{User: kernel.RootUser, Body: s.body}
		switch s.name {
		case "binary":
			script.Stdin = bytes.NewReader(bin)
		case "skills":
			script.Stdin = bytes.NewReader(u.settings.SkillsTar)
		}
		res, err := u.shell.Exec(ctx, script)
		if err != nil {
			return out, fmt.Errorf("step %s: %w", s.name, err)
		}
		if res.ExitCode != 0 {
			return out, fmt.Errorf("step %s exited %d: %s", s.name, res.ExitCode, tail(res.Stderr))
		}
		out.Lines = append(out.Lines, "ok "+s.name)
	}
	return out, nil
}

type step struct {
	name string
	body string
}

type view struct {
	Orca                   Account
	ConfigBase64           string
	BinaryPath, ConfigPath string
	LegacyFiles            []string
	LegacyDir              string
	SkillDirs              []string
}

func render(s Settings) ([]step, error) {
	if !userName.MatchString(s.Orca.Name) || !safePath.MatchString(s.Orca.Home) {
		return nil, fmt.Errorf("unsafe account %q", s.Orca.Name)
	}
	v := view{
		Orca:         s.Orca,
		ConfigBase64: base64.StdEncoding.EncodeToString(s.Config),
		BinaryPath:   BinaryPath, ConfigPath: ConfigPath,
		LegacyFiles: kernel.LegacyReviewerFiles(),
		LegacyDir:   kernel.LegacyLibexecDir,
		SkillDirs:   []string{path.Join(s.Orca.Home, ".claude", "skills"), path.Join(s.Orca.Home, ".pi", "agent", "skills")},
	}
	steps := make([]step, 0, len(stepOrder))
	for _, name := range stepOrder {
		var b strings.Builder
		if err := scripts.ExecuteTemplate(&b, name, v); err != nil {
			return nil, fmt.Errorf("render %s: %w", name, err)
		}
		steps = append(steps, step{name: name, body: b.String()})
	}
	return steps, nil
}

func checkLinuxELF(b []byte) error {
	f, err := elf.NewFile(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNotLinuxBinary, err)
	}
	defer f.Close()
	if f.OSABI != elf.ELFOSABI_NONE && f.OSABI != elf.ELFOSABI_LINUX {
		return fmt.Errorf("%w: OS ABI %v", ErrNotLinuxBinary, f.OSABI)
	}
	if f.Type != elf.ET_EXEC && f.Type != elf.ET_DYN {
		return fmt.Errorf("%w: type %v", ErrNotLinuxBinary, f.Type)
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func tail(stderr []byte) string {
	msg := strings.TrimSpace(string(stderr))
	if msg == "" {
		return "no stderr"
	}
	if lines := strings.Split(msg, "\n"); len(lines) > 20 {
		msg = strings.Join(lines[len(lines)-20:], "\n")
	}
	return msg
}
