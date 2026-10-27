package initcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Factory builds the use case lazily, so help never touches the filesystem.
type Factory func(ctx context.Context) (*UseCase, error)

// NewCommand builds the `init` driving adapter. It has no flags (TRD 9.4).
func NewCommand(factory Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Copy the gbai skill catalog into this project",
		Long: "Copy the embedded gbai skill catalog (without gabo-reviewer) into " + CatalogDir + "/\n" +
			"and write the catalog version to " + StampFile + "; commit both.\n" +
			"An older or missing stamp refreshes the catalog; a matching stamp changes nothing;\n" +
			"a newer stamp changes nothing and exits 1 with the upgrade command.\n" +
			"init never creates or edits " + ConfigFile + " or the user config.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc, err := factory(cmd.Context())
			if err != nil {
				return err
			}
			out, err := uc.Run(cmd.Context())
			if len(out.Lines) > 0 {
				if _, werr := fmt.Fprintln(cmd.OutOrStdout(), strings.Join(out.Lines, "\n")); werr != nil && err == nil {
					err = werr
				}
			}
			return err
		},
	}
}
