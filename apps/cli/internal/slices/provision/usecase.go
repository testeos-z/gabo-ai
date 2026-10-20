// Package provision prepares the server, as root, for the reviewer flow:
// packages, OS users, the gbai binary and config, the root-owned review
// wrappers and the sudoers drop-in. Every step is idempotent.
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
	"strconv"
	"strings"
	"text/template"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// Fixed install locations on the server.
const (
	BinaryPath  = kernel.ServerBinaryPath
	ConfigPath  = kernel.ServerConfigPath
	ReviewRun   = kernel.ReviewRunPath
	ReviewCI    = kernel.ReviewCIPath
	SudoersPath = "/etc/sudoers.d/gbai-reviewer"
	systemPath  = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
)

// ErrNotLinuxBinary is returned when the binary to install is not a linux ELF executable.
var ErrNotLinuxBinary = errors.New("not a linux ELF executable")

var (
	//go:embed provision.sh.tmpl
	templateFS embed.FS
	scripts    = template.Must(template.New("").Funcs(template.FuncMap{"q": shellQuote}).ParseFS(templateFS, "provision.sh.tmpl"))
	userName   = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	safePath   = regexp.MustCompile(`^/[A-Za-z0-9._/'-]*$`)
	stepOrder  = []string{"packages", "users", "config", "binary", "wrappers", "sudoers", "skills"}
)

// Account is an OS user on the server. Path entries are prepended to the system PATH.
type Account struct {
	Name string
	Home string
	UID  int
	Path []string
}

// Settings are the values rendered into the provisioning scripts.
type Settings struct {
	Orca       Account
	Reviewer   Account
	ReviewerCI Account
	Config     []byte // gbai.yaml installed at ConfigPath; must not contain secrets
	SkillsTar  []byte // tar of the embedded skills, installed for the orca user
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

type account struct {
	Account
	UIDText  string
	PathText string
}

type view struct {
	Orca, Reviewer, ReviewerCI account
	Managed                    []account
	WorkDir                    string
	ConfigBase64               string
	BinaryPath, ConfigPath     string
	ReviewRun, ReviewCI        string
	SudoersPath                string
	SkillDirs                  []string
}

func render(s Settings) ([]step, error) {
	for _, a := range []Account{s.Orca, s.Reviewer, s.ReviewerCI} {
		if !userName.MatchString(a.Name) || !safePath.MatchString(a.Home) {
			return nil, fmt.Errorf("unsafe account %q", a.Name)
		}
		for _, p := range a.Path {
			if !safePath.MatchString(p) || strings.Contains(p, ":") {
				return nil, fmt.Errorf("unsafe PATH entry for %q", a.Name)
			}
		}
	}
	for _, a := range []Account{s.Reviewer, s.ReviewerCI} {
		if a.UID <= 0 {
			return nil, fmt.Errorf("account %q needs a managed uid", a.Name)
		}
	}
	v := view{
		Orca: wrap(s.Orca), Reviewer: wrap(s.Reviewer), ReviewerCI: wrap(s.ReviewerCI),
		WorkDir:      path.Join(s.Reviewer.Home, "work"),
		ConfigBase64: base64.StdEncoding.EncodeToString(s.Config),
		BinaryPath:   BinaryPath, ConfigPath: ConfigPath,
		ReviewRun: ReviewRun, ReviewCI: ReviewCI, SudoersPath: SudoersPath,
	}
	v.Managed = []account{v.Reviewer, v.ReviewerCI}
	v.SkillDirs = []string{path.Join(s.Orca.Home, ".claude", "skills"), path.Join(s.Orca.Home, ".pi", "agent", "skills")}

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

func wrap(a Account) account {
	p := systemPath
	if len(a.Path) > 0 {
		p = strings.Join(a.Path, ":") + ":" + systemPath
	}
	return account{Account: a, UIDText: strconv.Itoa(a.UID), PathText: p}
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
