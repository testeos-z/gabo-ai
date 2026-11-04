# gabo plugin for Codex

The `gabo` plugin of the `gabo-ai` Codex marketplace (this repository, [`.agents/plugins/marketplace.json`](../../.agents/plugins/marketplace.json)). It brings the gabo catalog skills into Codex and runs `gabo check` around Codex's `apply_patch` edits in a gabo project. Spec: [intent 0002](../../docs/product/intents/0002-codex-plugin/spec.md). These docs cover the Codex CLI.

## Install

**With `gabo init`.** When `gabo.yaml` names `codex` in `runtimes.*`, `gabo init` writes the plugin's project keys `[marketplaces.gabo-ai]` (`source_type = "git"`, `source = "https://github.com/testeos-z/gabo-ai.git"`) and `[plugins."gabo@gabo-ai"]` (`enabled = true`) to the project's `.codex/config.toml`, never to your global Codex config or anything under `CODEX_HOME`. It only adds missing keys and keeps every value you set. What you still do, as its `engine integration:` report says: trust the project in Codex, so it loads `.codex/config.toml`; trust both gabo hooks in `/hooks` (see below); and, while `testeos-z/gabo-ai` is private, set up git credentials that can read it. While the fallback is printed, the report also says: if Codex does not list the gabo plugin after trust, run the two manual commands below. Commit `.codex/config.toml`.

**By hand.** As the alternative, install the plugin by hand:

```sh
codex plugin marketplace add testeos-z/gabo-ai
codex plugin add gabo@gabo-ai
```

While the repository is private, Codex clones it with your own git credentials: run `gh auth setup-git` or add an SSH key that can read it.

Then trust the two hooks: in a Codex session, open `/hooks` and trust the gabo `PreToolUse` and `PostToolUse` hooks. Codex runs a plugin hook only after you trust it, and skips a hook that changed until you trust it again. Hook trust is your own approval in `~/.codex/config.toml`; neither the plugin nor gabo writes it.

Update with `codex plugin marketplace upgrade gabo-ai`. Codex caches the plugin per version (`~/.codex/plugins/cache/gabo-ai/gabo/<version>/`), so an update arrives when a release bumps `version`. Check `/hooks` again after an update.

The plugin does not replace `gabo init` and builds no agents. Per-project config (`gabo.yaml`), slots, adapters and agents still come from `gabo init` and the setup prompt (see the [Install and set up](../../README.md#install-and-set-up)); with `runtimes.coder: codex`, the `gabo-runtime-codex` skill builds the agents as `.codex/agents/<id>.toml` and links the skills under `.agents/skills/`.

## What it ships

| path | what it is |
|---|---|
| `.codex-plugin/plugin.json` | Manifest: name `gabo`, `version`, `skills` (`./skills/`) and `hooks` (`./hooks/hooks.codex.json`). No apps and no MCP servers. |
| `skills/` | A real copy (not a symlink) of every catalog skill in `apps/cli/internal/skills/` except those that run reviews (`gabo-reviewer`, `gabo-reviewer-placement-local`): the same set as the Claude Code plugin. Codex leaves symlinked skill folders and files out of its plugin cache, so the copies are real files. |
| `hooks/hooks.codex.json` | `PreToolUse` and `PostToolUse` on `apply_patch`, each running a script with the argument `codex`. A file of its own, so Codex does not load the Claude Code `hooks/hooks.json`. |
| `scripts/gabo-pre-write.sh` | Pre-write hook: pipes the tool call to `gabo check --pre-write`; a patch that breaks the stage gate is denied with gabo's reason. |
| `scripts/gabo-check.sh` | Post-write hook: runs `gabo check` and returns its failures to Codex as additional context. |
| `scripts/gabo-hook-filter.sh` | Shared filter: both hooks stay silent when the project has no `.gabo/VERSION`, `gabo` is not on `PATH`, or no file in the patch is a gabo path (`gabo.yaml`, `.gabo/`, `.claude/agents/`, `.claude/skills/`, `docs/product/`). |
| `scripts/tests/` | Shell tests for both hooks on Codex inputs, run by the Go tests in `apps/cli/internal/skills`. |

The hooks hold no check rule; every rule lives in `gabo check`. To turn them off, leave them untrusted in `/hooks`.

## Keeping the copies equal

The three scripts are real copies of `plugins/claude-code/scripts/`, and each skill folder is a real copy of its catalog folder. `apps/cli/internal/skills/codexplugin_test.go` fails unless every copy is byte-identical, with the same mode, to its source, and unless `skills/` holds exactly the expected skill set. It names the first drifted path.

When you change a catalog skill or a shared hook script, refresh its copy in the same commit, from the repository root:

```sh
rm -r plugins/codex/skills/<name> && cp -r apps/cli/internal/skills/<name> plugins/codex/skills/<name>
cp plugins/claude-code/scripts/<script> plugins/codex/scripts/<script>
```

A catalog skill added or removed needs the same change in `plugins/codex/skills/`. Then run `go -C apps/cli test ./internal/skills/`.

## Release checklist

Both plugins share one version: the gabo CLI release version `x.y.z` (spec S11, D8), currently `0.2.6`; the [Pi package](../../packages/pi/README.md) `@gabo-ai/pi` carries the same version. Follow the [release checklist](../claude-code/README.md#release-checklist) of the Claude Code plugin; it sets `version` in both `plugins/claude-code/.claude-plugin/plugin.json` and `plugins/codex/.codex-plugin/plugin.json` in the same release commit as the CLI tag `apps/cli/v<x.y.z>`. The marketplace entries carry no `version`. `go -C apps/cli test ./...` fails when the two manifests differ or a copy has drifted.
