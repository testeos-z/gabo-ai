# gbai-team

CLI and harness to provision and operate autonomous cloud coding agents (first target: cappy-lab/vulcam).

Read `docs/cloud-agent-flow.md` before any change: it defines the agent flow, the hard rules (no polling loops, separation of duties, secrets via stdin only, never delete), the server layout, and what the CLI must automate.

## Stack

- Bun workspaces + Turborepo (`apps/*`, `packages/*`); root scripts only delegate via `turbo run`.
- TypeScript, single latest version repo-wide; `tsconfig.base.json` is the shared base.
- No OpenTUI for now.

## Conventions

- Conventional Commits, no AI attribution.
- Generated artifacts (code, docs, commits) in English.
- Test-first for behavior changes.
