// Package adapters holds the driven adapters of the selfupdate slice: HTTP
// release downloads, the running binary on the local filesystem and the
// terminal console.
package adapters

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/selfupdate"
)

var (
	_ selfupdate.Releases = HTTPReleases{}
	_ selfupdate.Binary   = OSBinary{}
	_ selfupdate.Console  = Console{}
)

// HTTPReleases downloads release files over HTTP(S), following redirects.
type HTTPReleases struct {
	// Client defaults to a client with a 5-minute timeout.
	Client *http.Client
	// Max caps a body; it defaults to selfupdate.MaxDownload.
	Max int64
}

// Get returns the body of url; 404 matches selfupdate.ErrNotFound.
func (h HTTPReleases) Get(ctx context.Context, url string) ([]byte, error) {
	client, limit := h.Client, h.Max
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	if limit <= 0 {
		limit = selfupdate.MaxDownload
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("GET %s: %s: %w", url, resp.Status, selfupdate.ErrNotFound)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("GET %s: body larger than %d bytes", url, limit)
	}
	return b, nil
}

// OSBinary is the running executable on the local filesystem.
type OSBinary struct{}

// Executable returns os.Executable with symlinks followed.
func (OSBinary) Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// Writable creates and removes a temporary file in dir.
func (OSBinary) Writable(dir string) error {
	f, err := os.CreateTemp(dir, ".gabo-write-check-*")
	if err != nil {
		return err
	}
	name := f.Name()
	_ = f.Close()
	return os.Remove(name)
}

// Replace writes data to a temporary file beside path, makes it executable
// and renames it over path; on error path is unchanged and the temporary
// file is removed.
func (OSBinary) Replace(path string, data []byte) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".gabo-update-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmp)
		}
	}()
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Chmod(0o755); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Exec replaces the process with path (see exec_unix.go).
func (OSBinary) Exec(path string, args, env []string) error {
	return execProcess(path, args, append(os.Environ(), env...))
}

// Console reads answers from In and writes to Out.
type Console struct {
	In  *bufio.Reader
	Out io.Writer
}

// Ask prints the question with a [y/N] hint and returns the answer line.
func (c Console) Ask(question string) (string, error) {
	fmt.Fprint(c.Out, question+" [y/N] ")
	line, err := c.In.ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// Say prints one line.
func (c Console) Say(line string) { fmt.Fprintln(c.Out, line) }

// IsTerminal reports whether f is a terminal; /dev/null, a pipe or a file is
// not.
func IsTerminal(f *os.File) bool { return term.IsTerminal(int(f.Fd())) }
