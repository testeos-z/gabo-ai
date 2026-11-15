# Verifying the gabo Pi package

The manual verification of `@gabo-ai/pi` (spec [S9](../../docs/product/intents/0003-pi-package/spec.md), D8), run by the owner on a local Pi 1.4.0 before each release. Pi has no headless test harness for packages; `bun test` covers the extension against a fake Pi, and this procedure covers the real one. Record each step's observed output in the intent's review findings.

## Before you start

- `<v>` is the package version: `jq -r .version packages/pi/package.json`.
- `<pkg>` is the absolute path of `packages/pi` in a gabo-ai checkout at the release commit.
- Pi 1.4.0 (`pi --version`) with gentle-pi installed: `pi list` names `npm:gentle-pi`.
- The gabo CLI built at `<v>` and first on `PATH`: from the checkout, `go -C apps/cli install -ldflags "-X main.buildVersion=<v>" ./cmd/gabo`, then `gabo version` prints `<v>`.
- A test project: a git checkout of a gabo project set up by `gabo init` (CLI `<v>`) and the setup prompt with `gabo-runtime-pi`, so it has `gabo.yaml`, `.gabo/VERSION` holding `<v>` and the agents in `.pi/agents/`. `gabo check` passes and `git status --short` prints nothing. Run every step below in its root.
- An intent folder for the stage gate, committed in the test project: `docs/product/intents/0901-verify-probe/` with an `intent.md` (any text), no `plan.md`, and this `spec.md`, whose Acceptance table has no filled row. `gabo check` still passes.

  ```markdown
  # Spec 0901 verify probe

  Status: Draft

  ## Acceptance

  | accepted by | date |
  |---|---|
  ```

In the steps, "ask the agent" means a prompt in the Pi session, and a blocked call means Pi reports the tool call as blocked with the quoted reason.

The subagent refusal quoted in steps 4 and 5 is this text, from `gabo-runtime-pi` (Prerequisite step 4):

```text
Pi has no subagents, so gabo cannot work on Pi without a subagent package. For subagents I recommend gentle-pi, installed with `npm i -g gentle-pi` (adds the `gentle-shell` launcher) or `pi install npm:gentle-pi` (inside plain Pi; npm is used only because gentle-pi's postinstall needs it); another subagent package is acceptable. As the recommended setup I recommend the full Gentle AI framework (https://github.com/Gentleman-Programming/gentle-ai): its methodology with RDD and the Engram memory stack, set up for Pi with `gentle-ai install --agent pi`. Then ask me to rebuild.
```

## Steps

| # | step | run | expected output |
|---|---|---|---|
| 1 | Install from the local path | `pi install <pkg>` | `Installed <pkg>`; `git status --short` prints nothing. |
| 2 | List packages | `pi list` | Under `User packages:`, a line naming `<pkg>`, next to `npm:gentle-pi`. |
| 3 | Start with gentle-pi | `pi`, then quit | No gabo notification: no warning and no version notice. In the session, `/skill:gabo-pi` and `/skill:gabo-reviewer-config` load. |
| 4 | Start without a subagent package | `pi --no-extensions -e <pkg>` | One warning notification whose text is the subagent refusal above, unchanged. Nothing else from gabo. |
| 5 | `.pi/agents/` write blocked | In the step 4 session, ask the agent: "Use the write tool to create `.pi/agents/verify-probe.md` containing `probe`." | The `write` call is blocked with the subagent refusal as its reason; `.pi/agents/verify-probe.md` does not exist. Quit the session. |
| 6 | Stage-gate block | `pi`, then ask the agent: "Use the write tool to create `docs/product/intents/0901-verify-probe/plan.md` containing `probe`." | The `write` call is blocked with the reason `✗ docs/product/intents/0901-verify-probe/plan.md: spec.md is not accepted; a human fills a row (accepted by, date) of its Acceptance table before the plan`; no `plan.md` exists. Quit the session. |
| 7 | Version notice | `printf '0.0.1\n' > .gabo/VERSION`, then `pi`, then quit; then `git restore .gabo/VERSION` | One info notification: `The gabo Pi package is <v> but this project's .gabo/VERSION is 0.0.1; next: run gabo init with the gabo CLI <v> to update the project's catalog.` No warning. |
| 8 | No project change | `git status --short`, then `gabo check` | `git status --short` prints nothing; `gabo check` passes as before step 1. |
| 9 | Remove | `pi remove <pkg>`, then `pi list` | `Removed <pkg>`; `pi list` no longer names `<pkg>`. |

## Recording

For each step record the command, the observed output (notification text, block reason or command output) and pass or fail. A step whose observed text differs from the expected text by any character fails: the extension shows gabo's text unchanged, so a difference is a bug in the extension or in `gabo check --pre-write`.
