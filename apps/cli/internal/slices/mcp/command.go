package mcp

import (
	"io"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// NewCommand builds the `mcp` driving adapter: an MCP server over stdio.
func NewCommand(tools *Tools, version string) *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Serve the gbai tools over MCP on stdio (no tool accepts or returns secrets)",
		Long: "Serve the gbai use cases as MCP tools over stdin/stdout for a local agent.\n" +
			"Tools are read-only or plan-first: apply tools need confirm=true. Secrets are\n" +
			"never handled: secret_instructions returns the command the human must run.\n\n" +
			"Register it with: claude mcp add --scope user gbai -- <abs path to gbai> mcp",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			t := &sdk.IOTransport{Reader: io.NopCloser(cmd.InOrStdin()), Writer: nopWriteCloser{cmd.OutOrStdout()}}
			return NewServer(tools, version).Run(cmd.Context(), t)
		},
	}
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
