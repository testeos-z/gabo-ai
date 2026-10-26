package configcmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// NewCommand builds the read-only `config` driving adapter.
func NewCommand(resolve Resolve) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show the config files gbai reads and the resolved values",
		Long: "Show the config layers, lowest to highest precedence (system, user, project), " +
			"whether each file was found, and the resolved config with defaults. " +
			"With --config only that file is read. Credential-looking values are redacted.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rep, err := NewUseCase(resolve).Show(cmd.Context())
			out := cmd.OutOrStdout()
			if werr := writeLayers(out, rep); werr != nil {
				return werr
			}
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(out, "\nresolved config:\n%s", rep.YAML)
			return err
		},
	}
}

func writeLayers(out io.Writer, rep Report) error {
	if _, err := fmt.Fprintln(out, "layers (lowest to highest precedence):"); err != nil {
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, l := range rep.Layers {
		p, state := l.Path, "not found"
		if p == "" {
			p = "(unresolved)"
		}
		if l.Found {
			state = "found"
		}
		if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\n", l.Name, p, state); err != nil {
			return err
		}
	}
	return w.Flush()
}
