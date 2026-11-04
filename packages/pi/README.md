# gabo package for Pi

`@gabo-ai/pi` is the gabo package for [Pi](https://github.com/earendil-works/pi). It brings two gabo skills into Pi and an extension that runs `gabo check --pre-write` at session start and before Pi's `write` and `edit` tool calls in a gabo project. Spec: [intent 0003](../../docs/product/intents/0003-pi-package/spec.md).

## Install

From npm, pinned to a release version (the gabo CLI version you use):

```sh
pi install npm:@gabo-ai/pi@<version>
```

In a gabo project you can skip that command: when `gabo.yaml` declares `pi` in `runtimes`, `gabo init` writes the `packages` entry `npm:@gabo-ai/pi@<gabo version>` to the project's `.pi/settings.json` (pinned to the version of the `gabo` binary that ran it; a development build writes nothing and prints the entry to add by hand). Pi reads that file, and installs and loads the package, only after you trust the project.

During development, or to try unreleased changes, install it from a local checkout. Pi loads a local package from its path without copying it:

```sh
pi install /path/to/gabo-ai/packages/pi
```

Check it with `pi list`: the package appears under `User packages:`. Pi loads its skills and its extension at the next start. Remove it with `pi remove <source>`, using the same source you installed.

The extension calls the `gabo` CLI, so install the CLI too (see [Install](../../README.md#install)) and keep `gabo` on `PATH`.

## What it ships

| path | what it is |
|---|---|
| `package.json` | Manifest: name `@gabo-ai/pi`, `version`, keyword `pi-package`, `pi.skills` and `pi.extensions`. The Pi host package `@earendil-works/pi-coding-agent` is an optional peer dependency only. |
| `skills/gabo-pi/` | Bootstrap skill: how to install the CLI, run `gabo init` and print the setup prompt with `gabo prompt`. |
| `skills/gabo-reviewer-config/` | A byte-for-byte copy of the catalog skill `apps/cli/internal/skills/gabo-reviewer-config/`: it configures the reviewer engine. No skill in the package runs reviews. |
| `extensions/gabo.ts` | The `gabo` extension (below). |
| `test/` | `bun test` suites: the manifest and tarball rules, and both extension handlers against a fake Pi API and a fake `gabo`. Not shipped. |

## The extension

The extension is a thin connection layer. It collects the session facts (the names of every registered tool and the package version), pipes them as JSON to `gabo check --pre-write` and shows or applies what gabo returns. Every rule, comparison and message text lives in the gabo CLI (`gabo check --help` describes the input and output). Outside a gabo project (no `gabo.yaml` in the working directory) it does nothing.

- **`session_start`.** Runs `gabo` once and shows each notification it returns, level and text unchanged. gabo warns when no subagent package is loaded (no `subagent_run` tool), with the same refusal text as `gabo-runtime-pi`, and adds one info notification when the package version differs from the project's `.gabo/VERSION`.
- **`tool_call`.** For `write` and `edit` only, runs `gabo` once with the call's `path` as the target. When gabo blocks, the call is blocked with gabo's reason: a write under `.pi/agents/` while no subagent package is loaded, or a `plan.md` whose intent `spec.md` is not accepted (the stage gate).
- **gabo unavailable.** When `gabo` is not on `PATH`, exits 2, times out (30 s) or prints unreadable output, the extension blocks nothing and shows one warning that gabo's checks were skipped.

## Coexistence with `gabo init` and `gabo-runtime-pi`

- **`gabo init` stays required.** The package does not configure a project. Per-project config (`gabo.yaml`), the `.gabo/skills/` catalog, slots and adapters still come from `gabo init` and the setup prompt (see the [Quick start](../../README.md#quick-start)).
- **`gabo-runtime-pi` stays required.** It builds the agents in `.pi/agents/` and the skill links in `.pi/skills/`. The package replaces none of them.
- **It writes no project file.** Installing the package or running a Pi session with it changes nothing in the project: `git status` stays clean.
- **It is not a subagent package.** gabo on Pi needs a subagent package such as gentle-pi (`pi install npm:gentle-pi`). The `gabo-runtime-pi` Prerequisite still refuses when the gabo package is the only one installed.

## Versioning

The package version follows the gabo release and equals the gabo CLI release version (today 0.2.6): one version, one release tag `apps/cli/v<x.y.z>`. A package release follows every CLI release, even with no package change. The release steps are in the [release checklist](../../plugins/claude-code/README.md#release-checklist).

## Release pipeline

Publishing to npm is automated:

1. A merge to `main` runs [`sync-pi-pkg-up.yml`](../../.github/workflows/sync-pi-pkg-up.yml): it runs [`scripts/sync-pkg-up.sh`](scripts/sync-pkg-up.sh), which merges `main` into the `pi/pkg-up` branch (fast-forward when possible, a merge commit otherwise; it fails on a conflict outside `.github/workflows/`) and pushes it. The push uses `GITHUB_TOKEN`, which GitHub refuses for any change under `.github/workflows/`, so the merge always keeps `pi/pkg-up`'s own `.github/workflows/`: workflow changes on `main` do not reach `pi/pkg-up` automatically (see [Updating workflows on `pi/pkg-up`](#updating-workflows-on-pipkg-up)). `test/sync-pkg-up.test.ts` covers it against a temporary git repository.
2. It then dispatches [`publish-pi.yml`](../../.github/workflows/publish-pi.yml) on `pi/pkg-up` (a push made with the workflow token starts no workflow by itself). A direct push to `pi/pkg-up` that touches `packages/pi/` starts it too.
3. `publish-pi.yml` runs `bun test` in `packages/pi`, and publishes with `npm publish --access public` only when `@gabo-ai/pi@<version>` is not on npm yet. An unchanged version is a no-op, so bump `version` to release.

The owner's gate is the merge into `main`, which only the owner performs. Before merging a release to `main`, the owner runs the [verification procedure](VERIFY.md) on a real Pi.

It needs the repository secret `NPM_TOKEN`: an npm granular access token with read and write on the `@gabo-ai` scope and 2FA bypass for publishing. Two npm limits apply (the npm token page shows both):

- Granular tokens with write permissions have a maximum lifetime of 90 days ([GitHub changelog, 2025-11-05](https://github.blog/changelog/2025-11-05-npm-security-update-classic-token-creation-disabled-and-granular-token-changes/): "All npm existing granular tokens with write permissions are now capped at 90-day maximum lifetime"), so rotate the secret before it expires.
- npm removes direct publishing with granular tokens in January 2027 ([npm docs, About access tokens](https://docs.npmjs.com/about-access-tokens): "The ability to publish new package versions directly with a granular access token will be removed in January 2027"). After that this workflow can no longer publish with `NPM_TOKEN`; the follow-up is npm trusted publishing (OIDC from GitHub Actions), which needs no stored token.

The publish decision lives in [`scripts/publish-decision.sh`](scripts/publish-decision.sh) (already published: exit 0, no publish; `npm view` E404 or empty: publish; any other `npm view` error: the job fails). `test/publish-decision.test.ts` runs it against a fake `npm`.

### Updating workflows on `pi/pkg-up`

To change `publish-pi.yml` (or any workflow) on `pi/pkg-up`, the owner first merges the change to `main`, then pushes it to `pi/pkg-up` by hand with their own credentials:

```sh
git fetch origin
git switch -C pi/pkg-up origin/pi/pkg-up
git checkout origin/main -- .github/workflows
git commit -m "ci: update pi/pkg-up workflows from main"
git push origin pi/pkg-up
```

### Accepted risk: who can publish

Anyone with write access to this repository can push to `pi/pkg-up` and so start `publish-pi.yml`, which publishes with `NPM_TOKEN`; a writer could also change a workflow on that branch to read the secret. The merge into `main` is therefore not a hard gate for npm publishing.

The owner accepts this risk (2026-11-04, verbatim: "acepto ese riesgo") because:

- only the owner account `leomerida15` has write access to the repository; every other collaborator is read-only;
- branch protection and GitHub Environments are not available on this private repository's plan, so the branch and the secret cannot be locked down further today.

Mitigation path: move to npm trusted publishing (OIDC), which removes the stored token; if the plan changes, put the publish job behind a GitHub Environment with required reviewers that holds `NPM_TOKEN`, and protect `pi/pkg-up`. Revisit this decision whenever a second account gets write access.

## Development

```sh
cd packages/pi
bun test
```

Before each release, the owner runs the manual [verification procedure](VERIFY.md) on a local Pi.
