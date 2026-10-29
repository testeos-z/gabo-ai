package initcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Factory builds the init run lazily, so help never touches the filesystem.
type Factory func(ctx context.Context) (*Init, error)

// NewCommand builds the `init` driving adapter. It has no flags (TRD 9.4).
func NewCommand(factory Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Install the reviewer skill and copy the gabo skill catalog into this project",
		Long: "Install the embedded gabo-reviewer skill and its scripts at user level, in\n" +
			"${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer/, never in the repository;\n" +
			"on every run it replaces a copy whose " + ReviewerStamp + " differs from the binary version.\n" +
			"Copy the embedded gabo skill catalog (without gabo-reviewer) into " + CatalogDir + "/\n" +
			"and write the catalog version to " + StampFile + "; commit both.\n" +
			"An older or missing stamp refreshes the catalog; a matching stamp changes nothing;\n" +
			"a newer stamp changes nothing and exits 1 with the upgrade command, unless init runs\n" +
			"at a terminal with CI and GABO_NO_SELF_UPDATE unset and you agree to update gabo to\n" +
			"exactly that version: the release's checksums.txt must be minisign-signed by the key\n" +
			"pinned in gabo and match the asset's SHA-256, then gabo replaces itself and reruns init.\n" +
			"On a fresh project (no stamp, no " + ConfigFile + ") it prints the setup prompt to paste\n" +
			"into your agent; it writes the prompt nowhere.\n" +
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
