// Package railwayssh implements kernel.RemoteShell through the local
// `railway ssh` CLI.
//
// `railway ssh` lands as root with a bare PATH and a stray GH_TOKEN, and sends
// its trailing command as one shell line. The script is therefore shipped
// base64-encoded inside that line (no quoting issues, no secrets: secrets only
// travel through stdin), and a root wrapper drops privileges with setpriv and
// rebuilds HOME and PATH for the target user.
package railwayssh

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

var (
	safeName = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)
	safePath = regexp.MustCompile(`^/[A-Za-z0-9._/-]*$`)
)

const systemPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

// Target identifies the Railway service to reach.
type Target struct {
	ProjectID     string
	EnvironmentID string
	ServiceID     string
}

// User is the environment rebuilt for an OS user on the server.
type User struct {
	Home      string
	ExtraPath []string
}

// Shell runs scripts on the server over `railway ssh`.
type Shell struct {
	runner kernel.Runner
	target Target
	users  map[string]User
}

var _ kernel.RemoteShell = (*Shell)(nil)

// New returns a Shell. users maps OS user names to their environment;
// kernel.RootUser is always available.
func New(runner kernel.Runner, target Target, users map[string]User) *Shell {
	return &Shell{runner: runner, target: target, users: users}
}

// Exec runs s on the server. s.Stdin is streamed to the script.
func (sh *Shell) Exec(ctx context.Context, s kernel.Script) (kernel.Result, error) {
	// An empty id lets `railway ssh` fall back to the project linked in the
	// working directory, which may be an unintended service.
	for _, id := range [][2]string{{"project", sh.target.ProjectID}, {"environment", sh.target.EnvironmentID}, {"service", sh.target.ServiceID}} {
		if id[1] == "" {
			return kernel.Result{}, fmt.Errorf("railway ssh: empty %s id", id[0])
		}
	}
	wrapper, err := sh.wrapper(s)
	if err != nil {
		return kernel.Result{}, err
	}
	args := []string{
		"ssh",
		"--project", sh.target.ProjectID,
		"--environment", sh.target.EnvironmentID,
		"--service", sh.target.ServiceID,
		"--",
		decodeAndRun(wrapper),
	}
	res, err := sh.runner.Run(ctx, kernel.Command{Name: "railway", Args: args, Stdin: s.Stdin})
	if err != nil {
		return res, fmt.Errorf("railway ssh as %s: %w", s.User, err)
	}
	return res, nil
}

func (sh *Shell) wrapper(s kernel.Script) (string, error) {
	var b strings.Builder
	b.WriteString("unset GH_TOKEN\n")
	b.WriteString("export PATH=" + systemPath + "\n")
	if s.User == kernel.RootUser {
		b.WriteString("export HOME=/root\n")
		b.WriteString("cd \"$HOME\"\n")
		b.WriteString(decodeAndRun(s.Body) + "\n")
		return b.String(), nil
	}
	u, ok := sh.users[s.User]
	if !ok {
		return "", fmt.Errorf("unknown server user %q", s.User)
	}
	if !safeName.MatchString(s.User) || !safePath.MatchString(u.Home) {
		return "", fmt.Errorf("unsafe user name or home for %q", s.User)
	}
	for _, p := range u.ExtraPath {
		if !safePath.MatchString(p) {
			return "", fmt.Errorf("unsafe PATH entry for %q", s.User)
		}
	}
	path := systemPath
	if len(u.ExtraPath) > 0 {
		path = strings.Join(u.ExtraPath, ":") + ":" + systemPath
	}
	fmt.Fprintf(&b, "cd %s\n", u.Home)
	fmt.Fprintf(&b,
		"exec setpriv --reuid=%[1]s --regid=%[1]s --init-groups env -u GH_TOKEN HOME=%[2]s USER=%[1]s LOGNAME=%[1]s PATH=%[3]s %[4]s\n",
		s.User, u.Home, path, decodeAndRun(s.Body))
	return b.String(), nil
}

// decodeAndRun returns a shell line running script with bash while keeping
// the caller's stdin attached to it.
func decodeAndRun(script string) string {
	enc := base64.StdEncoding.EncodeToString([]byte(script))
	return fmt.Sprintf(`bash -c "$(printf %%s %s | base64 -d)"`, enc)
}
