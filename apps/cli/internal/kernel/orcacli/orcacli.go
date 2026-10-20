// Package orcacli implements kernel.Terminals with the orca CLI, either
// locally through kernel.Runner or on the server through kernel.RemoteShell.
package orcacli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

type execFunc func(ctx context.Context, argv []string) (kernel.Result, error)

// Client runs orca terminal commands.
type Client struct {
	exec execFunc
}

var _ kernel.Terminals = (*Client)(nil)

// NewLocal runs bin through the local runner.
func NewLocal(r kernel.Runner, bin string) *Client {
	return &Client{exec: func(ctx context.Context, argv []string) (kernel.Result, error) {
		return kernel.RunChecked(ctx, r, kernel.Command{Name: bin, Args: argv})
	}}
}

// NewRemote runs `orca` on the server as user, every argument shell-quoted.
func NewRemote(sh kernel.RemoteShell, user string) *Client {
	return &Client{exec: func(ctx context.Context, argv []string) (kernel.Result, error) {
		quoted := make([]string, 0, len(argv)+1)
		for _, a := range append([]string{"orca"}, argv...) {
			quoted = append(quoted, shellQuote(a))
		}
		res, err := sh.Exec(ctx, kernel.Script{User: user, Body: "exec " + strings.Join(quoted, " ") + "\n"})
		if err != nil {
			return res, err
		}
		if res.ExitCode != 0 {
			return res, fmt.Errorf("orca %s exited %d: %s", argv[0], res.ExitCode, kernel.StderrTail(res.Stderr))
		}
		return res, nil
	}}
}

// List returns the live terminals.
func (c *Client) List(ctx context.Context) ([]kernel.Terminal, error) {
	res, err := c.exec(ctx, []string{"terminal", "list", "--json"})
	if err != nil {
		return nil, err
	}
	return parseList(res.Stdout)
}

// Create opens a terminal titled title that runs command on startup.
func (c *Client) Create(ctx context.Context, title, command string) (string, error) {
	res, err := c.exec(ctx, []string{"terminal", "create", "--title", title, "--command", command, "--json"})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(res.Stdout)), nil
}

// Close closes the terminal's tab.
func (c *Client) Close(ctx context.Context, handle string) error {
	_, err := c.exec(ctx, []string{"terminal", "close", "--terminal", handle, "--tab", "--json"})
	return err
}

type row struct {
	Handle string `json:"handle"`
	ID     string `json:"id"`
	Title  string `json:"title"`
	Name   string `json:"name"`
}

// parseList accepts a bare array or an object holding "terminals", possibly
// under "result"; the exact shape of orca's JSON is not pinned yet.
func parseList(b []byte) ([]kernel.Terminal, error) {
	var rows []row
	if err := json.Unmarshal(b, &rows); err != nil {
		var doc struct {
			Terminals []row `json:"terminals"`
			Result    struct {
				Terminals []row `json:"terminals"`
			} `json:"result"`
		}
		if err := json.Unmarshal(b, &doc); err != nil {
			return nil, fmt.Errorf("decode orca terminal list: %w", err)
		}
		rows = append(doc.Terminals, doc.Result.Terminals...)
	}
	out := make([]kernel.Terminal, 0, len(rows))
	for _, r := range rows {
		t := kernel.Terminal{Handle: r.Handle, Title: r.Title}
		if t.Handle == "" {
			t.Handle = r.ID
		}
		if t.Title == "" {
			t.Title = r.Name
		}
		out = append(out, t)
	}
	return out, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
