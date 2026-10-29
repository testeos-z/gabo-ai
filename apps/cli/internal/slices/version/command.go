package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewCommand builds the `version` driving adapter.
func NewCommand(uc *UseCase) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the gabo version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "gabo %s\n", uc.Show().Version)
			return err
		},
	}
}
