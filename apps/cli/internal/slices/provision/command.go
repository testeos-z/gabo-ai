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
		Long: "Prepare the server for the reviewer flow, as root and idempotently:\n" +
			"packages (sudo, git), the reviewer and reviewer-ci users, " + ConfigPath + ",\n" +
			"the gbai binary at " + BinaryPath + ", the review wrappers under /usr/local/libexec/gbai\n" +
			"and the sudoers drop-in " + SudoersPath + " (validated with visudo before install).\n\n" +
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
