# gabo-ai

CLI and harness to provision and operate autonomous cloud coding agents (first target: cappy-lab/vulcam).

Read `docs/cloud-agent-flow.md` before any change: it defines the agent flow, the hard rules (no polling loops, separation of duties, secrets via stdin only, cleanup of merged branches and finished worktrees only), the server layout, and what the CLI must automate.

## Stack

- Codebase is Go: one module at `apps/cli` (`github.com/testeos-z/gabo-ai/apps/cli`) producing a single binary `gabo` that is both the CLI and the MCP server.
- Bun workspaces + Turborepo (`apps/*`, `packages/*`) only orchestrate tasks; root scripts delegate via `turbo run`, and each workspace `package.json` delegates `build`, `test`, `lint` and `check-types` to the Go toolchain.
- `tsconfig.base.json` stays as the shared base for any future TypeScript package.
- No OpenTUI for now.

## Layout (hexagonal, vertical slices)

- `cmd/gabo/main.go` is the only composition root: it builds driven adapters, injects them into slice use cases and registers the driving adapters (cobra commands, MCP tools).
- `internal/kernel/`: ports shared by several slices (our own types), with driven adapters in subpackages and hand-written fakes in `internal/kernel/kerneltest`.
- `internal/config/`: loads and validates `gabo.yaml`.
- `internal/engineconfig/`: pure engine integration domain (engine set, keys, merge rules, report) shared by `initcmd` and `check`; no ports, no I/O.
- `internal/slices/<name>/`: `usecase.go` (core logic and domain types), `ports.go` (slice-specific ports), `command.go` (driving adapter), `adapters/` (slice-specific driven adapters). The core never imports cobra, `os/exec` or adapters; use-case tests use hand-written in-memory fakes.

## Conventions

- Conventional Commits, no AI attribution.
- Generated artifacts (code, docs, commits) in English.
- Test-first for behavior changes.

## Commands

The reviewer runs these commands on every PR (`review.checks` in `gabo.yaml`). Run each one from the repository root before you push. A line starting with `$` is the command; the lines under it are its healthy output, and each output ends with `exit: 0`. A command passed when it prints that output.

```text
$ go -C apps/cli build ./...
(no output)
exit: 0
$ go -C apps/cli vet ./...
(no output)
exit: 0
$ go -C apps/cli test ./...
ok  	github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo	(cached)
?   	github.com/testeos-z/gabo-ai/apps/cli/e2e	[no test files]
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec	(cached)
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/config	0.112s
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig	(cached)
?   	github.com/testeos-z/gabo-ai/apps/cli/internal/kernel	[no test files]
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/kernel/execrunner	(cached)
?   	github.com/testeos-z/gabo-ai/apps/cli/internal/kernel/kerneltest	[no test files]
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/kernel/osfs	(cached)
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/setupprompt	(cached)
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/skills	(cached)
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/slices/check	(cached)
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/slices/initcmd	(cached)
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/slices/prompt	(cached)
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/slices/selfupdate	(cached)
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/slices/selfupdate/adapters	(cached)
?   	github.com/testeos-z/gabo-ai/apps/cli/internal/slices/selfupdate/selfupdatetest	[no test files]
ok  	github.com/testeos-z/gabo-ai/apps/cli/internal/slices/version	(cached)
exit: 0
```

## Repeated mistakes

**Two-times rule:** a mistake seen once is recorded, not added here, so that a second occurrence can be recognized. When an agent makes the same mistake a second time, its correction is proposed for this section, naming both occurrences (the PR or task of each). No correction is added on the first occurrence.

No entries yet.
