package status

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// Factory builds the use case; mode picks where the Orca sessions are listed.
type Factory func(ctx context.Context, mode string) (*UseCase, error)

// NewCommand builds the `status` driving adapter.
func NewCommand(factory Factory) *cobra.Command {
	var asJSON bool
	var mode string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "One-shot snapshot: server, sessions, open PRs, verdicts and needs-user issues",
		Long: "Take a one-shot snapshot (no polling). Each probe is independent: a failing\n" +
			"probe reports unknown with its error instead of aborting the status.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if mode != "cloud" && mode != "local" {
				return fmt.Errorf("mode must be cloud or local, got %q", mode)
			}
			uc, err := factory(cmd.Context(), mode)
			if err != nil {
				return err
			}
			snap := uc.Snapshot(cmd.Context())
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(snap)
			}
			return render(cmd.OutOrStdout(), snap)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the snapshot as JSON")
	cmd.Flags().StringVar(&mode, "mode", "cloud", "where the sessions run: cloud (server) or local")
	return cmd
}

func head(p Probe, detail string) string {
	if p.State != StateOK {
		return StateUnknown + " (" + p.Error + ")"
	}
	return StateOK + detail
}

func render(w io.Writer, s Snapshot) error {
	var b strings.Builder
	fmt.Fprintf(&b, "server: %s\n", head(s.Server.Probe, " (reachable)"))
	titles := make([]string, 0, len(s.Sessions.Alive))
	for _, t := range s.Sessions.Alive {
		titles = append(titles, t.Title)
	}
	fmt.Fprintf(&b, "sessions: %s\n", head(s.Sessions.Probe, " ["+strings.Join(titles, ", ")+"]"))
	fmt.Fprintf(&b, "pull requests: %s\n", head(s.PullRequests.Probe, fmt.Sprintf(" (%d open)", len(s.PullRequests.Items))))
	for _, p := range s.PullRequests.Items {
		fmt.Fprintf(&b, "  #%d [%s] review=%s %s\n", p.Number, p.Mode, p.Review, p.Title)
		if p.Verdict != "" {
			fmt.Fprintf(&b, "      verdict: %s\n", p.Verdict)
		}
	}
	fmt.Fprintf(&b, "needs-user: %s\n", head(s.NeedsUser.Probe, fmt.Sprintf(" (%d open)", len(s.NeedsUser.Items))))
	for _, i := range s.NeedsUser.Items {
		fmt.Fprintf(&b, "  #%d %s\n", i.Number, i.Title)
	}
	_, err := io.WriteString(w, b.String())
	return err
}
