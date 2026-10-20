package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"slices"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func connect(t *testing.T, tl *Tools) *sdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := sdk.NewInMemoryTransports()
	if _, err := NewServer(tl, "test").Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func listTools(t *testing.T, cs *sdk.ClientSession) []*sdk.Tool {
	t.Helper()
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return res.Tools
}

func TestRegistryListsExactlyTheTools(t *testing.T) {
	t.Parallel()
	var names []string
	for _, tool := range listTools(t, connect(t, newTools(&calls{}))) {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	want := []string{"audit", "doctor", "provision_apply", "provision_plan", "repo_setup_apply", "repo_setup_plan",
		"review_request", "secret_instructions", "sessions_start", "sessions_stop", "status"}
	if !slices.Equal(names, want) {
		t.Fatalf("tools = %v\nwant    %v", names, want)
	}
	if !slices.Equal(ToolNames(), want) {
		t.Fatalf("ToolNames() = %v", ToolNames())
	}
}

func TestNoSchemaFieldSuggestsASecret(t *testing.T) {
	t.Parallel()
	banned := []string{"token", "password", "key", "value", "secret", "credential"}
	for _, tool := range listTools(t, connect(t, newTools(&calls{}))) {
		for _, schema := range []any{tool.InputSchema, tool.OutputSchema} {
			for _, field := range propertyNames(schema) {
				for _, b := range banned {
					if strings.Contains(strings.ToLower(field), b) {
						t.Errorf("tool %s has field %q", tool.Name, field)
					}
				}
			}
		}
	}
}

func propertyNames(v any) []string {
	var out []string
	switch x := v.(type) {
	case map[string]any:
		if props, ok := x["properties"].(map[string]any); ok {
			for name, sub := range props {
				out = append(out, name)
				out = append(out, propertyNames(sub)...)
			}
		}
		for _, k := range []string{"items", "additionalProperties"} {
			out = append(out, propertyNames(x[k])...)
		}
	case []any:
		for _, e := range x {
			out = append(out, propertyNames(e)...)
		}
	}
	return out
}

func TestEndToEndApplyWithoutConfirmReturnsPlan(t *testing.T) {
	t.Parallel()
	c := &calls{}
	cs := connect(t, newTools(c))
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: "provision_apply", Arguments: map[string]any{}})
	if err != nil || res.IsError {
		t.Fatalf("CallTool = %+v, %v", res, err)
	}
	var out Result
	b, _ := json.Marshal(res.StructuredContent)
	if err := json.Unmarshal(b, &out); err != nil || out.Kind != KindPlan || out.Data != "provision-plan-data" {
		t.Fatalf("structured = %s (%v)", b, err)
	}
	if !slices.Equal(c.got, []string{"provision-plan"}) {
		t.Fatalf("calls = %v", c.got)
	}
}

func TestEndToEndInvalidInputIsToolError(t *testing.T) {
	t.Parallel()
	c := &calls{}
	cs := connect(t, newTools(c))
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: "status", Arguments: map[string]any{"mode": "prod"}})
	if err != nil || !res.IsError || len(c.got) != 0 {
		t.Fatalf("CallTool = %+v, %v, calls %v", res, err, c.got)
	}
}

func TestCommandServesStdio(t *testing.T) {
	t.Parallel()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	cmd := NewCommand(newTools(&calls{}), "test")
	cmd.SetIn(inR)
	cmd.SetOut(outW)
	cmd.SetArgs(nil)
	done := make(chan error, 1)
	go func() { done <- cmd.ExecuteContext(context.Background()); _ = outW.Close() }()

	lines := bufio.NewScanner(outR)
	send := func(msg string) {
		if _, err := io.WriteString(inW, msg+"\n"); err != nil {
			t.Fatal(err)
		}
	}
	send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`)
	if !lines.Scan() || !strings.Contains(lines.Text(), `"serverInfo"`) {
		t.Fatalf("initialize response = %q", lines.Text())
	}
	send(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	send(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if !lines.Scan() || !strings.Contains(lines.Text(), `"secret_instructions"`) {
		t.Fatalf("tools/list response = %q", lines.Text())
	}
	_ = inW.Close()
	<-done
}
