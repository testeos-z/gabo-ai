package review

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// Factories build each use case lazily, so help and argument errors never
// need configuration.
type Factories struct {
	Request func(context.Context) (*RequestUseCase, error)
	Notify  func(context.Context) (*NotifyUseCase, error)
	Run     func(context.Context) (*RunUseCase, error)
	CI      func(context.Context) (*CIUseCase, error)
}

// NewCommand builds the `review` driving adapter.
func NewCommand(f Factories) *cobra.Command {
	root := &cobra.Command{
		Use:   "review",
		Short: "Request, run and relay reviews inside the reviewer boundary",
	}
	root.AddCommand(newRequestCommand(f), newNotifyCommand(f), newRunCommand(f), newCICommand(f))
	return root
}

func newRequestCommand(f Factories) *cobra.Command {
	var in RequestInput
	cmd := &cobra.Command{
		Use:   "request",
		Short: "Open a detached Orca terminal that reviews a pinned PR head (returns immediately)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc, err := f.Request(cmd.Context())
			if err != nil {
				return err
			}
			out, err := uc.Request(cmd.Context(), in)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "review terminal %s opened: %s\n", out.Title, out.Terminal)
			return err
		},
	}
	cmd.Flags().IntVar(&in.PR, "pr", 0, "pull request number")
	cmd.Flags().StringVar(&in.SHA, "sha", "", "pinned 40-hex head SHA")
	cmd.Flags().StringVar(&in.Notify, "notify", "", "coder mailbox (run:<id> or dispatch:<id>)")
	for _, name := range []string{"pr", "sha", "notify"} {
		_ = cmd.MarkFlagRequired(name)
	}
	return cmd
}

func newNotifyCommand(f Factories) *cobra.Command {
	var in NotifyInput
	cmd := &cobra.Command{
		Use:   "notify",
		Short: "Relay the review summary line read from stdin to an Orca mailbox",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc, err := f.Notify(cmd.Context())
			if err != nil {
				return err
			}
			in.Stdin, in.Echo = cmd.InOrStdin(), cmd.OutOrStdout()
			_, err = uc.Notify(cmd.Context(), in)
			return err
		},
	}
	cmd.Flags().StringVar(&in.To, "to", "", "mailbox (run:<id> or dispatch:<id>)")
	cmd.Flags().IntVar(&in.PR, "pr", 0, "pull request number")
	cmd.Flags().StringVar(&in.SHA, "sha", "", "pinned 40-hex head SHA")
	for _, name := range []string{"to", "pr", "sha"} {
		_ = cmd.MarkFlagRequired(name)
	}
	return cmd
}

func newRunCommand(f Factories) *cobra.Command {
	return &cobra.Command{
		Use:   "run head <pr-number> <sha>",
		Short: "Review a pinned PR head as the reviewer and print one summary line",
		Args: cobra.MatchAll(cobra.ExactArgs(3), func(_ *cobra.Command, args []string) error {
			if args[0] != "head" {
				return fmt.Errorf("first argument must be %q", "head")
			}
			n, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("%w, got %q", ErrInvalidPR, args[1])
			}
			return ValidatePR(n)
		}),
		RunE: func(cmd *cobra.Command, args []string) error {
			n, _ := strconv.Atoi(args[1])
			uc, err := f.Run(cmd.Context())
			if err != nil {
				return err
			}
			report, err := uc.Run(cmd.Context(), n, args[2])
			if err != nil {
				return err
			}
			for _, note := range report.Notes {
				fmt.Fprintln(cmd.ErrOrStderr(), "gbai review:", note)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), report.Summary)
			return err
		},
	}
}

func newCICommand(f Factories) *cobra.Command {
	ci := &cobra.Command{
		Use:   "ci",
		Short: "Reviewer-ci side: run untrusted checks and the engine in a copied tree (no GitHub credentials)",
	}
	var dir, runtime string
	withDir := func(c *cobra.Command) *cobra.Command {
		c.Flags().StringVar(&dir, "dir", "", "workspace name (pr-<n>-<sha7>)")
		_ = c.MarkFlagRequired("dir")
		return c
	}
	useCase := func(cmd *cobra.Command) (*CIUseCase, error) { return f.CI(cmd.Context()) }

	prepare := withDir(&cobra.Command{
		Use:   "prepare",
		Short: "Replace the workspace with the tar archive read from stdin",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc, err := useCase(cmd)
			if err != nil {
				return err
			}
			return uc.Prepare(dir, cmd.InOrStdin())
		},
	})
	run := withDir(&cobra.Command{
		Use:   "run --dir <name> -- <command> [args...]",
		Short: "Execute a command (no shell) inside the workspace",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uc, err := useCase(cmd)
			if err != nil {
				return err
			}
			res, err := uc.Run(cmd.Context(), dir, args)
			return relay(cmd, res, err)
		},
	})
	engine := withDir(&cobra.Command{
		Use:   "engine",
		Short: "Run the read-only review engine; stdin is the credential line then the prompt",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc, err := useCase(cmd)
			if err != nil {
				return err
			}
			res, err := uc.Engine(cmd.Context(), dir, runtime, cmd.InOrStdin())
			return relay(cmd, res, err)
		},
	})
	engine.Flags().StringVar(&runtime, "runtime", "", "review runtime (claude or pi)")
	_ = engine.MarkFlagRequired("runtime")
	cleanup := withDir(&cobra.Command{
		Use:   "cleanup",
		Short: "Remove the workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc, err := useCase(cmd)
			if err != nil {
				return err
			}
			return uc.Cleanup(dir)
		},
	})
	ci.AddCommand(prepare, run, engine, cleanup)
	return ci
}

// relay copies a child's output and turns its non-zero exit into an error.
func relay(cmd *cobra.Command, res kernel.Result, err error) error {
	if err != nil {
		return err
	}
	if _, err := cmd.OutOrStdout().Write(res.Stdout); err != nil {
		return err
	}
	if _, err := cmd.ErrOrStderr().Write(res.Stderr); err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("command exited %d", res.ExitCode)
	}
	return nil
}
