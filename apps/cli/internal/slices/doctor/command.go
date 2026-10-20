package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Factory builds the use case lazily, so help never needs configuration.
type Factory func(ctx context.Context) (*UseCase, error)

// NewCommand builds the `doctor` driving adapter.
func NewCommand(factory Factory) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check the known gotchas locally and on the server (read-only)",
		Long: "Check, without changing anything: local tools (railway, gh, orca, go) and gh auth;\n" +
			"on the server, sudo, the orca/reviewer/reviewer-ci users and uids, the sudoers drop-in,\n" +
			"the root-owned review-run wrapper, token variable names visible to the orca user\n" +
			"(names only, never values) and the reviewer home mode. Exits 1 when a check fails.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc, err := factory(cmd.Context())
			if err != nil {
				return err
			}
			rep := uc.Run(cmd.Context())
			if err := write(cmd, rep, asJSON); err != nil {
				return err
			}
			if rep.Failed() {
				return ErrFailed
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the checks as JSON")
	return cmd
}

func write(cmd *cobra.Command, rep Report, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	}
	var b strings.Builder
	for _, c := range rep.Checks {
		fmt.Fprintf(&b, "%-5s %s", c.Status, c.Name)
		if c.Detail != "" {
			fmt.Fprintf(&b, ": %s", c.Detail)
		}
		b.WriteString("\n")
		if c.Fix != "" {
			fmt.Fprintf(&b, "      fix: %s\n", c.Fix)
		}
	}
	_, err := fmt.Fprint(cmd.OutOrStdout(), b.String())
	return err
}
