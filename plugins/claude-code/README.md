# gabo plugin for Claude Code

The `gabo` plugin of the `gabo-ai` marketplace (this repository, [`.claude-plugin/marketplace.json`](../../.claude-plugin/marketplace.json)). It brings the gabo catalog skills into Claude Code and runs `gabo check` around Claude's writes in a gabo project. Spec: [intent 0001](../../docs/product/intents/0001-claude-code-plugin/spec.md).

## Install

```sh
claude plugin marketplace add testeos-z/gabo-ai
claude plugin install gabo@gabo-ai
```

While the repository is private, Claude Code clones it with your own git credentials: run `gh auth setup-git` or add an SSH key that can read it.

The plugin does not replace `gabo init`. Per-project config (`gabo.yaml`), slots, adapters and agents still come from `gabo init` and the setup prompt (see the [Quick start](../../README.md#quick-start)).

## What it ships

| path | what it is |
|---|---|
| `.claude-plugin/plugin.json` | Manifest: name `gabo`, `version`, and the `checks` option (boolean, default `true`). |
| `skills/` | One relative symlink per catalog skill into `apps/cli/internal/skills/`, every catalog skill except those that run reviews (`gabo-reviewer`, `gabo-reviewer-placement-local`). Claude Code copies them into its plugin cache on install. |
| `hooks/hooks.json` | `PreToolUse` and `PostToolUse` on `Write\|Edit`. |
| `scripts/gabo-pre-write.sh` | Pre-write hook: pipes the tool call to `gabo check --pre-write`; a write that breaks the stage gate is blocked with gabo's reason. |
| `scripts/gabo-check.sh` | Post-write hook: runs `gabo check` and reports its failures to Claude. |
| `scripts/gabo-hook-filter.sh` | Shared filter: both hooks stay silent when `checks` is off, the project has no `.gabo/VERSION`, `gabo` is not on `PATH`, or the file is not a gabo path (`gabo.yaml`, `.gabo/`, `.claude/agents/`, `.claude/skills/`, `docs/product/`). |
| `scripts/tests/` | Shell tests for both hooks, run by the Go tests in `apps/cli/internal/skills`. |

The hooks hold no check rule; every rule lives in `gabo check`. Turn both off with the plugin's `checks` row in `/config`.

## Open owner decisions

These questions from the [spec](../../docs/product/intents/0001-claude-code-plugin/spec.md#6-open-questions) are still open. This file documents the recommended option; update it when the owner decides.

| # | question | recommended (not decided) |
|---|---|---|
| 3 | Which marketplace hosts the plugin while the repository is private? | This repository as the marketplace; users need git credentials (`gh auth setup-git` or an SSH key) until it is public. Anthropic's plugin directory when the repository goes public. |
| 4 | Which ref do users install from, and who bumps the version? | The default branch `main`, with `version` bumped by hand in the release commit until a release pipeline exists (see the release checklist below). |
| 5 | Inside a gabo project, how should the main session treat the plugin skills that duplicate the project copy in `.gabo/skills/`? | Accept both copies (agents are unaffected). To use only the project copy in one project, disable the plugin there with `"enabledPlugins": { "gabo@gabo-ai": false }` in that project's `.claude/settings.json`. |

## Release checklist

The plugin `version` equals the gabo CLI release version `x.y.z`. It lives in the two plugin manifests, this plugin's and the [Codex plugin](../codex/README.md)'s, and in the [Pi package](../../packages/pi/README.md)'s `package.json`; all three always carry the same version, and the marketplace entries have no `version`. There is no release pipeline yet, so these are manual steps of every CLI release `apps/cli/v<x.y.z>`, covering both plugins and the Pi package:

1. In the release commit, set `version` to `x.y.z`, the same version as the CLI tag `apps/cli/v<x.y.z>`, in `plugins/claude-code/.claude-plugin/plugin.json`, `plugins/codex/.codex-plugin/plugin.json` and `packages/pi/package.json`. Check the Pi package with `test "$(jq -r .version packages/pi/package.json)" = "<x.y.z>"`; it must exit 0.
2. Run `claude plugin validate .` and `claude plugin validate plugins/claude-code`; both must pass.
3. Run `go -C apps/cli test ./...`; it asserts both manifests carry the same version, the marketplace entries carry no `version`, and the Codex plugin's skill and script copies equal their sources.
4. After the release commit is on the release branch, run `claude plugin tag plugins/claude-code` (`--dry-run` first). It must succeed and create `gabo--v<x.y.z>`, matching `apps/cli/v<x.y.z>`; push it with `--push`.
5. Run `bun test` in `packages/pi`; it must pass. Then run the [Pi package verification procedure](../../packages/pi/VERIFY.md) on a local Pi 1.4.0 at the release commit and record each step's observed output and result in the intent's review findings ([intent 0003](../../docs/product/intents/0003-pi-package/)). Every step must pass.
6. After `apps/cli/v<x.y.z>` is pushed, the owner runs `npm publish` from `packages/pi` with their own npm account. Agents never publish: they have no npm account or token.
