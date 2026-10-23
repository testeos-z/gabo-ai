package provision

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Factory builds the use case lazily, so help never needs configuration.
type Factory func(ctx context.Context) (*UseCase, error)

// DefaultBinary resolves the binary to install when --binary is not set.
type DefaultBinary func() (string, error)

// NewCommand builds the `provision` driving adapter.
func NewCommand(factory Factory, defaultBinary DefaultBinary) *cobra.Command {
	var in Input
	cmd := &cobra.Command{
		Use:   "provision",
		Short: "Prepare the server for the reviewer flow (idempotent, as root)",
		Long: "Prepare the server for the reviewer flow, as root and idempotently, in single-user mode:\n" +
			"packages (git), a check that the orca user exists, " + ConfigPath + ", the gbai binary at\n" +
			BinaryPath + ", removal of the retired reviewer sudoers drop-in and wrappers, and the\n" +
			"embedded gbai skills for the orca user (claude and pi skill dirs; gbai-owned skills are replaced).\n" +
			"The reviewer runs as the orca user with the owner's gh login: no extra users, no sudo.\n\n" +
			"Secrets are not handled here: use `gbai secrets set`.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc, err := factory(cmd.Context())
			if err != nil {
				return err
			}
			if in.BinaryPath == "" && !in.DryRun {
				if in.BinaryPath, err = defaultBinary(); err != nil {
					return fmt.Errorf("resolve running executable: %w", err)
				}
			}
			out, err := uc.Run(cmd.Context(), in)
			if len(out.Lines) > 0 {
				if _, werr := fmt.Fprintln(cmd.OutOrStdout(), strings.Join(out.Lines, "\n")); werr != nil && err == nil {
					err = werr
				}
			}
			return err
		},
	}
	cmd.Flags().StringVar(&in.BinaryPath, "binary", "", "linux gbai binary to install (default: the running executable)")
	cmd.Flags().BoolVar(&in.DryRun, "dry-run", false, "print the step scripts without reading the binary or executing")
	return cmd
}
