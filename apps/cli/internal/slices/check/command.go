package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Exit codes (TRD 9.4): 0 nothing failed, 1 a check failed, 2 a usage or
// read error.
const (
	ExitFailed = 1
	ExitUsage  = 2
)

// ExitError carries the process exit code. A nil Err means the report was
// already printed and nothing else is said.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

// ExitCode is read by the composition root.
func (e *ExitError) ExitCode() int { return e.Code }

// Factory builds the run lazily, so help and usage errors never read the project.
type Factory func(ctx context.Context) (*UseCase, error)

// preWriteFlag is the hook mode flag (intent 0001 S11).
const preWriteFlag = "pre-write"

// NewCommand builds the `check` driving adapter. Without --pre-write it takes
// no arguments and no flags; with it, the paths of the pending writes.
func NewCommand(factory Factory) *cobra.Command {
	var preWrite bool
	cmd := &cobra.Command{
		Use:   "check [--pre-write [<path>...]]",
		Short: "Validate the config, the agents, slot coverage and the catalog version, read-only",
		Long: "Validate the project read-only and engine-neutral (PRD R33):\n" +
			"  - the global user file and gabo.yaml, each layer and the merged result,\n" +
			"    against the embedded JSON Schemas;\n" +
			"  - no config left under its name from before the gabo-ai rename (a project\n" +
			"    gbai.yaml, or a global gbai config dir holding config.yaml or\n" +
			"    reviewer.env): it fails with the rename to make and is never read;\n" +
			"  - every agent of the neutral agent spec (" + CatalogDir + "/) in the agent\n" +
			"    directory each runtime skill declares (metadata.agent_dir);\n" +
			"  - each agent's gabo-version line against " + StampFile + ";\n" +
			"  - for a runtime skill declaring metadata.skill_dir, a link <skill_dir>/<name>\n" +
			"    to " + CatalogDir + "/<name> for each skill a spec agent lists, and a link\n" +
			"    <skill_dir>/gabo-<slot> to each slot adapter that is a skill folder;\n" +
			"  - for the reviewer runtime's metadata.reviewer_link, a user-level link to\n" +
			"    ${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer holding\n" +
			"    scripts/reviewer.sh (not checked with CI set);\n" +
			"  - when a role runs on opencode, experimental.subagent_depth at 3 or more in\n" +
			"    the first existing of opencode.json, opencode.jsonc, .opencode/opencode.json\n" +
			"    and .opencode/opencode.jsonc (never ~/.config/opencode);\n" +
			"  - every slots.<name>.adapter file against its contract skill (" + CatalogDir + "/):\n" +
			"    a ### <operation> heading under ## Operations for each operation;\n" +
			"  - " + StampFile + " against the CLI version: older warns, missing or newer fails.\n" +
			"  - each " + IntentsDir + "/<NNNN-slug>/ folder: its name, intent.md, an accepted\n" +
			"    spec.md before any plan.md, and the plan's task ids and dependency graph.\n" +
			"With GABO_UPGRADE_PR=1 only the config is validated.\n" +
			"It writes nothing and never updates itself.\n" +
			"Exit 0 when nothing fails, 1 on any failure, 2 on a usage or read error.\n" +
			"\n" +
			"Hook mode, gabo check --pre-write <path>..., judges pending writes before\n" +
			"they happen, for the engines' pre-write hooks. Every argument after\n" +
			"--pre-write is a path, relative to the working directory or absolute inside\n" +
			"the project root (the git top level). Writing " + IntentsDir + "/<NNNN-slug>/plan.md\n" +
			"is blocked while that folder's spec.md is missing, unreadable or has no filled\n" +
			"Acceptance row; any other path, or one outside the root, passes. It reads\n" +
			"nothing else and writes nothing. Exit 0 with no output when every write\n" +
			"passes, 1 with one \"✗ <path>: <reason>\" line per blocked path (every path\n" +
			"when the project cannot be read), 2 on a usage error.\n" +
			"With no path, gabo check --pre-write < tool-call.json reads the engine hook's\n" +
			"tool-call JSON from stdin and judges the paths it would write: Claude Code's\n" +
			"tool_input.file_path, or the \"*** Add File:\", \"*** Update File:\" and\n" +
			"\"*** Move to:\" headers of a Codex apply_patch in tool_input.command. A patch\n" +
			"that writes no file passes; empty stdin or stdin that is not such a JSON\n" +
			"object exits 2 with one usage line.\n" +
			"Stdin holding a JSON object with the key gabo_facts is an engine extension's\n" +
			"session facts (intent 0003): {\"gabo_facts\": 1, \"event\": \"session_start\" or\n" +
			"\"tool_call\", \"engine\": \"pi\", \"tools\": [<registered tool names>],\n" +
			"\"package_version\": \"x.y.z\", \"target_paths\": [<paths>, tool_call only]}.\n" +
			"While no tool is subagent_run, session_start warns with the subagent refusal\n" +
			"and a tool_call writing under .pi/agents/ is blocked with it. session_start\n" +
			"also adds one info notification when package_version differs from the\n" +
			"project's " + StampFile + " (none when it is missing), and a tool_call is\n" +
			"blocked by the stage gate above with its \"✗\" lines, after the refusal,\n" +
			"one reason per line. It prints one JSON object, {\"notify\": [{\"level\":\n" +
			"\"warning\" or \"info\", \"text\": ...}], \"block\": null or {\"reason\": ...}},\n" +
			"and exits 0 without a block, 1 with one, 2 with one usage line on a bad\n" +
			"object (malformed, unknown schema version, engine or event, or a tool_call\n" +
			"without target_paths).",
		// The command parses its own flags, so a flag error knows the mode
		// wherever --pre-write stands among the arguments.
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, raw []string) error {
			flags, paths := splitPreWrite(raw)
			if err := cmd.Flags().Parse(flags); err != nil {
				if hasPreWrite(raw) {
					return preWriteUsage(err)
				}
				return usage(err)
			}
			if help, _ := cmd.Flags().GetBool("help"); help {
				return cmd.Help()
			}
			args := append(cmd.Flags().Args(), paths...)
			if preWrite {
				return judgePreWrite(cmd, factory, args)
			}
			if len(args) > 0 {
				return usage(fmt.Errorf("check takes no arguments, got %q", strings.Join(args, " ")))
			}
			var set []string
			cmd.Flags().Visit(func(f *pflag.Flag) { set = append(set, "--"+f.Name) })
			if len(set) > 0 {
				return usage(fmt.Errorf("check takes no flags, got %s", strings.Join(set, " ")))
			}
			uc, err := factory(cmd.Context())
			if err != nil {
				return &ExitError{Code: ExitUsage, Err: err}
			}
			rep, err := uc.Run()
			if err != nil {
				return &ExitError{Code: ExitUsage, Err: err}
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), strings.Join(rep.Lines, "\n")); err != nil {
				return &ExitError{Code: ExitUsage, Err: err}
			}
			if rep.Failed {
				return &ExitError{Code: ExitFailed}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&preWrite, preWriteFlag, false, "judge the pending writes to the given paths, or to the paths of the tool-call JSON on stdin, with the intent stage gate (hook mode)")
	return cmd
}

// splitPreWrite splits raw after the first --pre-write in any form (bare or
// --pre-write=<value>, before any "--"): every later argument is a path,
// also one that starts with "-", so a path's name can never turn a hook's
// call into a usage error. One "--" right after the flag is dropped.
func splitPreWrite(raw []string) (flags, paths []string) {
	for i, a := range raw {
		if a == "--" {
			break
		}
		if isPreWriteArg(a) {
			paths = raw[i+1:]
			if len(paths) > 0 && paths[0] == "--" {
				paths = paths[1:]
			}
			return raw[:i+1], paths
		}
	}
	return raw, nil
}

// isPreWriteArg reports whether a is the hook mode flag, bare or with a
// value.
func isPreWriteArg(a string) bool {
	return a == "--"+preWriteFlag || strings.HasPrefix(a, "--"+preWriteFlag+"=")
}

// hasPreWrite reports whether raw sets the hook mode flag before a "--".
func hasPreWrite(raw []string) bool {
	for _, a := range raw {
		if a == "--" {
			return false
		}
		if isPreWriteArg(a) {
			return true
		}
	}
	return false
}

// judgePreWrite runs the hook mode: silent exit 0 when every write passes,
// the blocked lines and exit 1 otherwise, exit 2 on a usage error. A project
// that cannot be loaded (the factory fails, e.g. the working directory
// cannot be read) blocks every path with exit 1: the hook fails closed.
//
// With no path the targets come from the engine hook's tool-call JSON on
// stdin (PreWriteTargets); stdin that cannot be read or is no tool call
// exits 2, and a tool call that writes no file passes.
func judgePreWrite(cmd *cobra.Command, factory Factory, paths []string) error {
	if len(paths) == 0 {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return &ExitError{Code: ExitUsage, Err: fmt.Errorf("%w: stdin cannot be read (%v)", ErrPreWriteUsage, err)}
		}
		if IsFacts(data) {
			return judgeFacts(cmd, factory, data)
		}
		if paths, err = PreWriteTargets(data); err != nil {
			return &ExitError{Code: ExitUsage, Err: err}
		}
		if len(paths) == 0 {
			return nil
		}
	}
	if err := checkPreWritePaths(paths); err != nil {
		return &ExitError{Code: ExitUsage, Err: err}
	}
	uc, err := factory(cmd.Context())
	if err != nil {
		var r Report
		for _, p := range paths {
			r.fail("✗ " + p + ": the project cannot be read (" + err.Error() + "); it counts as not accepted")
		}
		return printBlocked(cmd, r)
	}
	rep, err := uc.PreWrite(paths)
	if err != nil {
		return &ExitError{Code: ExitUsage, Err: err}
	}
	if !rep.Failed {
		return nil
	}
	return printBlocked(cmd, rep)
}

// judgeFacts runs the session-facts mode (intent 0003 D9): it prints one
// FactsResult JSON object and exits 0 without a block, 1 with one and 2 on
// a bad facts object. A project that cannot be loaded blocks a tool_call
// (fail closed), after the subagent refusal when it applies; session_start
// then gets the subagent rule alone, with no version notification.
func judgeFacts(cmd *cobra.Command, factory Factory, data []byte) error {
	f, err := ParseFacts(data)
	if err != nil {
		return &ExitError{Code: ExitUsage, Err: err}
	}
	uc, err := factory(cmd.Context())
	var r FactsResult
	switch {
	case err == nil:
		r = uc.JudgeFacts(f)
	case f.Event == EventToolCall:
		reasons := []string{"the project cannot be read (" + err.Error() + "); the write counts as blocked"}
		if !hasSubagentTool(f.Tools) {
			reasons = append([]string{SubagentRefusal}, reasons...)
		}
		r = FactsResult{Notify: []Notification{}, Block: &Block{Reason: strings.Join(reasons, "\n")}}
	default:
		r = (&UseCase{}).JudgeFacts(f)
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r); err != nil {
		return &ExitError{Code: ExitFailed, Err: err}
	}
	if r.Block != nil {
		return &ExitError{Code: ExitFailed}
	}
	return nil
}

// printBlocked prints the blocked lines and exits 1; an output error still
// exits 1, so a hook never lets the writes through.
func printBlocked(cmd *cobra.Command, rep Report) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), strings.Join(rep.Lines, "\n")); err != nil {
		return &ExitError{Code: ExitFailed, Err: err}
	}
	return &ExitError{Code: ExitFailed}
}

func preWriteUsage(err error) error {
	return &ExitError{Code: ExitUsage, Err: errors.Join(err, ErrPreWriteUsage)}
}

func usage(err error) error {
	return &ExitError{Code: ExitUsage, Err: errors.Join(err, errors.New("usage: gabo check"))}
}
