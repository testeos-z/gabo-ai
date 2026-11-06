package initcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
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
			"With a valid " + ConfigFile + ", it writes the project-level gabo integration keys of each\n" +
			"engine in runtimes, merged with the existing project config (unrelated keys kept):\n" +
			engineconfig.ClaudeSettings + ", " + engineconfig.CodexConfigPath + ", " + engineconfig.PiSettings + " and the OpenCode configs\n" +
			"named below are the only engine files it writes; it never writes user-level engine config.\n" +
			"With runtimes.assistant: claude it also sets the top-level \"agent\": \"gabo-assistant\" key of\n" +
			engineconfig.ClaudeSettings + " so Claude Code starts in the gabo assistant; another value of \"agent\" is\n" +
			"yours and kept. With another assistant engine it removes that exact \"agent\" value,\n" +
			"also when claude is no longer in runtimes.\n" +
			"With runtimes.assistant: opencode it sets \"default_agent\": \"gabo-assistant\" in the first existing of\n" +
			strings.Join(engineconfig.OpencodeConfigs[:3], ", ") + " and " + engineconfig.OpencodeConfigs[3] + ", else in a new\n" +
			engineconfig.OpencodeNewConfig + ", comments kept; another value of \"default_agent\" is yours and kept.\n" +
			"With another assistant engine it removes that exact \"default_agent\" value from each of those\n" +
			"files, also when opencode is no longer in runtimes.\n" +
			"With opencode in runtimes it also raises experimental.subagent_depth to 3 in the first existing\n" +
			"of those files, else in a new " + engineconfig.OpencodeNewConfig + ", when it is unset or lower; a whole number\n" +
			"of 3 or more is kept.\n" +
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
