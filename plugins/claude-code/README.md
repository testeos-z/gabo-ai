# gabo plugin for Claude Code

The `gabo` plugin of the `gabo-ai` marketplace (this repository, [`.claude-plugin/marketplace.json`](../../.claude-plugin/marketplace.json)). It brings the gabo catalog skills into Claude Code and runs `gabo check` around Claude's writes in a gabo project. Spec: [intent 0001](../../docs/product/intents/0001-claude-code-plugin/spec.md).

## Install

**With `gabo init`.** When `gabo.yaml` names `claude` in `runtimes.*`, `gabo init` writes the plugin's project keys `extraKnownMarketplaces."gabo-ai"` (the GitHub source `testeos-z/gabo-ai`) and `enabledPlugins."gabo@gabo-ai": true` to the project's `.claude/settings.json`, never to your global Claude Code settings. It only adds missing keys and keeps every value you set. What you still do, as its `engine integration:` report says: trust the project folder in Claude Code, so the marketplace and the plugin apply. Commit `.claude/settings.json`.

**By hand.** As the alternative, install the plugin at user level:

```sh
claude plugin marketplace add testeos-z/gabo-ai
claude plugin install gabo@gabo-ai
```

The plugin does not replace `gabo init`. Per-project config (`gabo.yaml`), slots, adapters and agents still come from `gabo init` and the setup prompt (see the [Install and set up](../../README.md#install-and-set-up)).

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
| 3 | Which marketplace hosts the plugin? | The repository is public since 2026-11-06, so this repository works as the marketplace and needs no git credentials. Anthropic's plugin directory is the recommended host; that move is not decided. |
| 4 | Which ref do users install from, and who bumps the version? | The default branch `main`, with `version` bumped by hand in the release commit until a release pipeline exists (see the release checklist below). |
| 5 | Inside a gabo project, how should the main session treat the plugin skills that duplicate the project copy in `.gabo/skills/`? | Accept both copies (agents are unaffected). To use only the project copy in one project, disable the plugin there with `"enabledPlugins": { "gabo@gabo-ai": false }` in that project's `.claude/settings.json`. |

## Release checklist

The plugin `version` equals the gabo CLI release version `x.y.z`, currently `0.2.6`. It lives in the two plugin manifests, this plugin's and the [Codex plugin](../codex/README.md)'s, and in the [Pi package](../../packages/pi/README.md) `@gabo-ai/pi`'s `package.json`; all three always carry the same version, and the marketplace entries have no `version`. These are the steps of every CLI release `apps/cli/v<x.y.z>`, covering both plugins and the Pi package. The Pi package is published to npm by the [release pipeline](../../packages/pi/README.md#release-pipeline); the rest is manual:

1. In the release commit, set `version` to `x.y.z`, the same version as the CLI tag `apps/cli/v<x.y.z>`, in `plugins/claude-code/.claude-plugin/plugin.json`, `plugins/codex/.codex-plugin/plugin.json` and `packages/pi/package.json`. Check the Pi package with `test "$(jq -r .version packages/pi/package.json)" = "<x.y.z>"`; it must exit 0.
2. Run `claude plugin validate .` and `claude plugin validate plugins/claude-code`; both must pass.
3. Run `go -C apps/cli test ./...`; it asserts both manifests carry the same version, the marketplace entries carry no `version`, and the Codex plugin's skill and script copies equal their sources.
4. After the release commit is on the release branch, run `claude plugin tag plugins/claude-code` (`--dry-run` first). It must succeed and create `gabo--v<x.y.z>`, matching `apps/cli/v<x.y.z>`; push it with `--push`.
5. Run `bun test` in `packages/pi`; it must pass. Then run the [Pi package verification procedure](../../packages/pi/VERIFY.md) on a local Pi 1.4.0 at the release commit and record each step's observed output and result in the intent's review findings ([intent 0003](../../docs/product/intents/0003-pi-package/)). Every step must pass. The owner runs this procedure before merging the release to `main`.
6. The owner merges the release to `main`; only the owner merges to `main`. That merge publishes the Pi package: `main` is merged into `pi/pkg-up`, and `publish-pi.yml` runs `bun test` and publishes `@gabo-ai/pi@<x.y.z>` to npm when that version is not on npm yet, with the repository secret `NPM_TOKEN`. Nobody runs `npm publish` by hand; check the workflow run in GitHub Actions. Workflow changes on `main` do not reach `pi/pkg-up` automatically (the sync pushes with `GITHUB_TOKEN`, which may not change `.github/workflows/`, so it keeps `pi/pkg-up`'s own); when a release changes `publish-pi.yml`, the owner pushes it to `pi/pkg-up` by hand: `git fetch origin && git switch -C pi/pkg-up origin/pi/pkg-up && git checkout origin/main -- .github/workflows && git commit -m "ci: update pi/pkg-up workflows from main" && git push origin pi/pkg-up` (see [Updating workflows on `pi/pkg-up`](../../packages/pi/README.md#updating-workflows-on-pipkg-up)).
