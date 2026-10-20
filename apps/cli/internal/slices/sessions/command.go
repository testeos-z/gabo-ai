package sessions

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Factory builds the use case for the Orca host of mode: the server (as the
// orca user) for cloud, this machine for local.
type Factory func(ctx context.Context, mode Mode) (*UseCase, error)

// NewCommand builds the `sessions` driving adapter.
func NewCommand(factory Factory) *cobra.Command {
	root := &cobra.Command{
		Use:   "sessions",
		Short: "Start or stop the gbai-coder, gbai-reviewer and gbai-assistant Orca sessions",
	}
	root.AddCommand(
		newSubcommand(factory, "start", "Open the sessions' Orca terminals (idempotent)", (*UseCase).startFor),
		newSubcommand(factory, "stop", "Close only the Orca terminals titled exactly like the sessions", (*UseCase).stopFor),
	)
	return root
}

func (u *UseCase) startFor(ctx context.Context, roles []Role, mode Mode) (Output, error) {
	return u.Start(ctx, roles, mode)
}

func (u *UseCase) stopFor(ctx context.Context, roles []Role, _ Mode) (Output, error) {
	return u.Stop(ctx, roles)
}

func newSubcommand(factory Factory, use, short string, action func(*UseCase, context.Context, []Role, Mode) (Output, error)) *cobra.Command {
	var role, mode string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			roles, err := ParseRoles(role)
			if err != nil {
				return err
			}
			m, err := ParseMode(mode)
			if err != nil {
				return err
			}
			uc, err := factory(cmd.Context(), m)
			if err != nil {
				return err
			}
			out, err := action(uc, cmd.Context(), roles, m)
			if len(out.Lines) > 0 {
				if _, werr := fmt.Fprintln(cmd.OutOrStdout(), strings.Join(out.Lines, "\n")); werr != nil && err == nil {
					err = werr
				}
			}
			return err
		},
	}
	cmd.Flags().StringVar(&role, "role", "all", "coder, reviewer, assistant or all")
	cmd.Flags().StringVar(&mode, "mode", string(ModeCloud), "cloud (server, as orca) or local (this machine)")
	return cmd
}
