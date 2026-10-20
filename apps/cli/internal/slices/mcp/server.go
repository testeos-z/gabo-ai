package mcp

import (
	"context"
	"slices"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type tool struct {
	name, description string
	readOnly          bool
	register          func(*sdk.Server, *sdk.Tool)
}

func handler[In any](fn func(context.Context, In) (Result, error)) func(*sdk.Server, *sdk.Tool) {
	return func(s *sdk.Server, t *sdk.Tool) {
		sdk.AddTool(s, t, func(ctx context.Context, _ *sdk.CallToolRequest, in In) (*sdk.CallToolResult, Result, error) {
			out, err := fn(ctx, in)
			return nil, out, err
		})
	}
}

func (t *Tools) catalog() []tool {
	return []tool{
		{"status", "One-shot snapshot: server, sessions, open PRs, verdicts, needs-user issues.", true, handler(t.Status)},
		{"doctor", "Read-only checks of the known setup gotchas, locally and on the server.", true, handler(t.Doctor)},
		{"audit", "Check that every merge into the integration branch went through the reviewer bot.", true, handler(t.Audit)},
		{"provision_plan", "Show the server provisioning scripts without running them.", true, handler(t.ProvisionPlan)},
		{"provision_apply", "Provision the server as root (idempotent). Without confirm=true only returns the plan.", false, handler(t.ProvisionApply)},
		{"repo_setup_plan", "Show the GitHub labels and integration ruleset calls without making them.", true, handler(t.RepoSetupPlan)},
		{"repo_setup_apply", "Create or update the labels and ruleset (idempotent). Without confirm=true only returns the plan.", false, handler(t.RepoSetupApply)},
		{"sessions_start", "Open the gbai-coder, gbai-reviewer and gbai-assistant Orca terminals (idempotent).", false, handler(t.SessionsStart)},
		{"sessions_stop", "Close the gbai session terminals. Without confirm=true only says what would close.", false, handler(t.SessionsStop)},
		{"review_request", "After pushing a PR head, open the detached review terminal; the verdict arrives in the notify mailbox.", false, handler(t.ReviewRequest)},
		{"secret_instructions", "Return the exact `gbai secrets set <name>` commands the human must run in their own terminal. Never pass secrets through chat or tools.", true, handler(t.SecretInstructions)},
	}
}

// ToolNames lists the registered tool names in sorted order.
func ToolNames() []string {
	var names []string
	for _, tl := range (&Tools{}).catalog() {
		names = append(names, tl.name)
	}
	slices.Sort(names)
	return names
}

// NewServer registers every tool on a new MCP server.
func NewServer(t *Tools, version string) *sdk.Server {
	s := sdk.NewServer(&sdk.Implementation{Name: "gbai", Version: version}, nil)
	for _, tl := range t.catalog() {
		tl.register(s, &sdk.Tool{
			Name:        tl.name,
			Description: tl.description,
			Annotations: &sdk.ToolAnnotations{ReadOnlyHint: tl.readOnly},
		})
	}
	return s
}
