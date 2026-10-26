package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Factory builds the use case lazily, so help never needs configuration.
type Factory func(ctx context.Context) (*UseCase, error)

// NewCommand builds the `repo` driving adapter.
func NewCommand(factory Factory) *cobra.Command {
	root := &cobra.Command{
		Use:   "repo",
		Short: "Prepare the GitHub repository for the agent flow",
	}
	var dryRun bool
	setup := &cobra.Command{
		Use:   "setup",
		Short: "Create or update the mode/needs-user labels and the integration branch ruleset (idempotent)",
		Long: "Create or update, with the owner's gh session:\n" +
			"  labels mode:cloud (reviewer may auto-merge), mode:local (a human merges) and needs-user;\n" +
			"  the repository ruleset " + RulesetName + " on the integration branch: pull request with no\n" +
			"  required approval (the reviewer comments, it never approves), required statuses\n" +
			"  reviewer/tests and reviewer/review,\n" +
			"  no force pushes or deletions, and an empty bypass list.\n\n" +
			"Admins of the repository or organization can still edit or bypass rulesets.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc, err := factory(cmd.Context())
			if err != nil {
				return err
			}
			out, err := uc.Setup(cmd.Context(), dryRun)
			if len(out.Lines) > 0 {
				if _, werr := fmt.Fprintln(cmd.OutOrStdout(), strings.Join(out.Lines, "\n")); werr != nil && err == nil {
					err = werr
				}
			}
			return err
		},
	}
	setup.Flags().BoolVar(&dryRun, "dry-run", false, "print the planned API calls and bodies without calling gh")
	root.AddCommand(setup)
	return root
}
