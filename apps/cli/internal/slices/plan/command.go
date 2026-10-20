package plan

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// NewCommand builds the `plan` driving adapter.
func NewCommand(uc *UseCase) *cobra.Command {
	root := &cobra.Command{
		Use:   "plan",
		Short: "Scaffold and check the six core product docs (PRD, TRD, plan, DB, UX/UI, app flow)",
	}
	var dir string
	var force, asJSON bool
	newCmd := &cobra.Command{
		Use:   "new",
		Short: "Write " + strings.Join(Docs, ", ") + " from templates (never overwrites without --force)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := uc.New(cmd.Context(), dir, force)
			if len(out.Lines) > 0 {
				if _, werr := fmt.Fprintln(cmd.OutOrStdout(), strings.Join(out.Lines, "\n")); werr != nil && err == nil {
					err = werr
				}
			}
			return err
		},
	}
	newCmd.Flags().BoolVar(&force, "force", false, "overwrite existing docs")
	status := &cobra.Command{
		Use:   "status",
		Short: "Report which docs exist and which still hold " + Placeholder + " placeholders",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			st, err := uc.Status(cmd.Context(), dir)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(st)
			}
			var b strings.Builder
			for _, s := range st {
				fmt.Fprintf(&b, "%-8s %s", s.State, s.Path)
				if s.Placeholders > 0 {
					fmt.Fprintf(&b, " (%d placeholders)", s.Placeholders)
				}
				b.WriteString("\n")
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), b.String())
			return err
		},
	}
	status.Flags().BoolVar(&asJSON, "json", false, "print the status as JSON")
	for _, c := range []*cobra.Command{newCmd, status} {
		c.Flags().StringVar(&dir, "dir", DefaultDir, "directory of the docs")
		root.AddCommand(c)
	}
	return root
}
