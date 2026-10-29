// Package prompt is the `gabo prompt` slice: it prints the embedded setup
// prompt and nothing else (PRD R39, TRD D25, 9.4). It reads no config, writes
// no file and never self-updates.
package prompt

import (
	"io"

	"github.com/spf13/cobra"
)

// NewCommand builds the `prompt` driving adapter that writes text to stdout
// byte for byte.
func NewCommand(text string) *cobra.Command {
	return &cobra.Command{
		Use:   "prompt",
		Short: "Print the setup prompt to paste into your agent",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := io.WriteString(cmd.OutOrStdout(), text)
			return err
		},
	}
}
