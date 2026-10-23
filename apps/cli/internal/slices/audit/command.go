package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/verdict"
)

// Factory builds the use case lazily, so help never needs configuration.
type Factory func(ctx context.Context) (*UseCase, error)

const dateLayout = "2006-01-02"

// NewCommand builds the `audit` driving adapter; now sets the default window.
func NewCommand(factory Factory, now func() time.Time) *cobra.Command {
	var since string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Check that every merge into the integration branch went through the reviewer",
		Long: "For every PR merged into the integration branch since --since, check that it was\n" +
			"merged by the bot login, never by the coder login, and that its head commit has a\n" +
			"successful " + ReviewContext + " status set by the bot, and that the newest verdict comment\n" +
			"the bot posted on the PR signs that head and starts with \"" + verdict.Clean + "\".\n" +
			"Exits 1 when violations exist.\n\n" +
			"Only GitHub is audited; the Engram per-step log is not cross-checked yet.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			from := now().UTC().AddDate(0, 0, -30).Truncate(24 * time.Hour)
			if since != "" {
				t, err := time.Parse(dateLayout, since)
				if err != nil {
					return fmt.Errorf("--since must be a date like 2026-10-01: %w", err)
				}
				from = t
			}
			uc, err := factory(cmd.Context())
			if err != nil {
				return err
			}
			rep, err := uc.Audit(cmd.Context(), from)
			if err != nil {
				return err
			}
			if err := write(cmd, rep, asJSON); err != nil {
				return err
			}
			if len(rep.Violations) > 0 {
				return fmt.Errorf("%w: %d", ErrViolations, len(rep.Violations))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&since, "since", "", "first merge date to audit, YYYY-MM-DD (default: 30 days ago)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the report as JSON")
	return cmd
}

func write(cmd *cobra.Command, rep Report, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "audited %d merged PRs since %s: %d violations\n", rep.Checked, rep.Since.Format(dateLayout), len(rep.Violations))
	for _, v := range rep.Violations {
		fmt.Fprintf(&b, "  #%d: %s\n", v.PR, v.Reason)
	}
	_, err := fmt.Fprint(cmd.OutOrStdout(), b.String())
	return err
}
