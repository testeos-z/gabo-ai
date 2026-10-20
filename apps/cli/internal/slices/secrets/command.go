package secrets

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Factory builds the use case lazily, so help and argument errors never
// need configuration or a server connection.
type Factory func(ctx context.Context) (*UseCase, error)

// NewCommand builds the `secrets` driving adapter.
func NewCommand(factory Factory) *cobra.Command {
	root := &cobra.Command{
		Use:   "secrets",
		Short: "Manage secrets stored on the server",
	}
	root.AddCommand(newSetCommand(factory))
	return root
}

func newSetCommand(factory Factory) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "set <name>",
		Short: "Store a secret on the server for its target OS user",
		Long: "Store a secret on the server for its target OS user.\n\n" +
			"The value is read without echo from the terminal, or from stdin when it is piped.\n" +
			"It is never accepted as an argument, flag or environment variable.\n\n" +
			"Known names: " + strings.Join(Names(), ", "),
		Example:   "  gbai secrets set reviewer-github\n  printf '%s' \"$TOKEN\" | gbai secrets set reviewer-claude",
		Args:      cobra.ExactArgs(1),
		ValidArgs: Names(),
		RunE: func(cmd *cobra.Command, args []string) error {
			uc, err := factory(cmd.Context())
			if err != nil {
				return err
			}
			out, err := uc.Set(cmd.Context(), Input{Name: args[0], DryRun: dryRun})
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if dryRun {
				_, err = fmt.Fprintln(w, strings.Join(out.Scripts, "\n"))
				return err
			}
			_, err = fmt.Fprintln(w, out.Message)
			return err
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the remote script without reading the value or executing")
	return cmd
}
