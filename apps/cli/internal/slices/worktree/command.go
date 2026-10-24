package worktree

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// ErrRefused is returned (non-zero exit) after a REFUSED line was printed.
var ErrRefused = errors.New("worktree close refused")

// Factory builds the use case from the config.
type Factory func(ctx context.Context) (*UseCase, error)

// NewCommand builds the `worktree` driving adapter.
func NewCommand(factory Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worktree",
		Short: "Manage coder worktrees",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "close <absolute path>",
		Short: "Close an approved coder worktree and delete its merged local branch",
		Long: "Close one linked worktree of the repository this command runs in, only when it is\n" +
			"approved: directly under <main checkout>-worktrees/, not the main worktree, not on\n" +
			"worktree.keep, clean, fully pushed, without ignored secrets (.env*, keys,\n" +
			"certificates, SQLite files, also under build/ and dist/), worked on, without an open\n" +
			"PR, and merged: its HEAD is in freshly fetched origin/<integration branch>, or a PR\n" +
			"merged into it holds its HEAD. Then it runs `git worktree remove` (never --force),\n" +
			"`git worktree prune` and `git branch -d` (never -D; the remote branch is untouched).\n\n" +
			"Prints exactly one line: CLOSED <path> branch=<b> reason=<in-base|merged-pr>\n" +
			"(exit 0) or REFUSED <path> <reason> (non-zero exit).",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			uc, err := factory(cmd.Context())
			if err != nil {
				return err
			}
			res := uc.Close(cmd.Context(), args[0])
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), res.Line()); err != nil {
				return err
			}
			if res.Warning != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "warning:", res.Warning)
			}
			if !res.Closed {
				return ErrRefused
			}
			return nil
		},
	})
	return cmd
}
