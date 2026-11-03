package check

import (
	"context"
	"errors"
	"fmt"
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
		Use:   "check [--pre-write <path>...]",
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
			"they happen, for the engines' pre-write hooks. Each path is relative to the\n" +
			"project root or absolute inside it. Writing " + IntentsDir + "/<NNNN-slug>/plan.md\n" +
			"is blocked while that folder's spec.md is missing, unreadable or has no filled\n" +
			"Acceptance row; any other path, or one outside the root, passes. It reads\n" +
			"nothing else and writes nothing. Exit 0 with no output when every write\n" +
			"passes, 1 with one \"✗ <path>: <reason>\" line per blocked path, 2 on a\n" +
			"usage error.",
		// The command parses its own flags, so a flag error knows the mode
		// wherever --pre-write stands among the arguments.
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, raw []string) error {
			if err := cmd.Flags().Parse(raw); err != nil {
				if hasPreWrite(raw) {
					return preWriteUsage(err)
				}
				return usage(err)
			}
			if help, _ := cmd.Flags().GetBool("help"); help {
				return cmd.Help()
			}
			args := cmd.Flags().Args()
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
	cmd.Flags().BoolVar(&preWrite, preWriteFlag, false, "judge the pending writes to the given paths with the intent stage gate (hook mode)")
	return cmd
}

// hasPreWrite reports whether raw sets the hook mode flag before a "--".
func hasPreWrite(raw []string) bool {
	for _, a := range raw {
		if a == "--" {
			return false
		}
		if a == "--"+preWriteFlag || strings.HasPrefix(a, "--"+preWriteFlag+"=") {
			return true
		}
	}
	return false
}

// judgePreWrite runs the hook mode: silent exit 0 when every write passes,
// the blocked lines and exit 1 otherwise, exit 2 on a usage error.
func judgePreWrite(cmd *cobra.Command, factory Factory, paths []string) error {
	if err := checkPreWritePaths(paths); err != nil {
		return &ExitError{Code: ExitUsage, Err: err}
	}
	uc, err := factory(cmd.Context())
	if err != nil {
		return &ExitError{Code: ExitUsage, Err: err}
	}
	rep, err := uc.PreWrite(paths)
	if err != nil {
		return &ExitError{Code: ExitUsage, Err: err}
	}
	if !rep.Failed {
		return nil
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), strings.Join(rep.Lines, "\n")); err != nil {
		return &ExitError{Code: ExitUsage, Err: err}
	}
	return &ExitError{Code: ExitFailed}
}

func preWriteUsage(err error) error {
	return &ExitError{Code: ExitUsage, Err: errors.Join(err, ErrPreWriteUsage)}
}

func usage(err error) error {
	return &ExitError{Code: ExitUsage, Err: errors.Join(err, errors.New("usage: gabo check"))}
}
