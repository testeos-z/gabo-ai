package initcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Factory builds the use case lazily, so help never touches the filesystem.
type Factory func(ctx context.Context) (*UseCase, error)

// NewCommand builds the `init` driving adapter.
func NewCommand(factory Factory) *cobra.Command {
	var runtime, scope string
	var in Input
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Install the gbai agent skills locally and register the gbai MCP server",
		Long: "Install the embedded gbai skills for the local agent:\n" +
			"  claude: ~/.claude/skills/<name>/ (user) or .claude/skills/ (project)\n" +
			"  pi:     ~/.pi/agent/skills/<name>/ (user) or .pi/skills/ (project)\n" +
			"A file whose content differs is never overwritten without --force.\n\n" +
			"Then register the MCP server: for claude, `claude mcp add --scope user gbai -- <gbai> mcp`\n" +
			"unless `claude mcp get gbai` succeeds; for pi, print the config snippet to add.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			if in.Runtimes, err = ParseRuntimes(runtime); err != nil {
				return err
			}
			if in.Scope, err = ParseScope(scope); err != nil {
				return err
			}
			uc, err := factory(cmd.Context())
			if err != nil {
				return err
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
	cmd.Flags().StringVar(&runtime, "runtime", "claude", "claude, pi or both")
	cmd.Flags().StringVar(&scope, "scope", "user", "user (home skill dirs) or project (current directory)")
	cmd.Flags().BoolVar(&in.DryRun, "dry-run", false, "print what would be installed and run, changing nothing")
	cmd.Flags().BoolVar(&in.Force, "force", false, "overwrite skill files whose content differs")
	return cmd
}
