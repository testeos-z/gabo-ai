# Setup prompt

The setup prompt is the only entry text of gabo-ai setup (TRD 9.4, D25, PRD R36). Paste it into the coding agent you use, opened in your project: the agent installs the CLI when it is missing, runs `gabo init`, and continues with the guided setup. `gabo init` prints the same text on a fresh project, so you can also start from the terminal and paste what it prints.

The block between the two marker comments is the prompt, byte for byte the file embedded in the CLI (`apps/cli/internal/setupprompt/setup-prompt.md`). Do not edit it here: change the embedded file and copy it over; a Go test fails when the two differ.

## Install the CLI

The CLI needs Go. Install it with:

```sh
go install github.com/testeos-z/gbai-team/apps/cli/cmd/gbai@latest
```

Until the gabo-ai rename lands, the installed command is `gbai`; the prompt says `gabo` and tells the agent to use `gbai` when only that exists.

## Prompt

<!-- gabo-setup-prompt:begin -->
Set up gabo-ai in this project. Work in this project's root directory, follow the steps in order, and skip any step whose result already holds, so running this prompt again is safe. Ask me before every choice and apply no value I did not accept. Never write a secret value anywhere; config files hold only the names of environment variables.

1. If the `gabo` command is missing, install the gabo CLI from https://github.com/testeos-z/gabo-ai/blob/main/docs/setup-prompt.md#install-the-cli and check that it runs. Releases before the gabo-ai rename install the command as `gbai`; if only `gbai` exists, use it wherever these steps say `gabo`.
2. If `.gbai/skills/` is missing, run `gabo init` here.
3. Load these skills from `.gbai/skills/`: `gabo-setup`, `gabo-config-writer`, `gabo-agent-templates`, and the runtime skill for the engine you are running in; if `.gbai/skills/` holds none for your engine, load `gabo-runtime-builder` and write that runtime skill first.
4. Following `gabo-setup` and `gabo-config-writer`, ask me the setup questions one at a time, then write the global user file and the project config `gbai.yaml`.
5. With `gabo-agent-templates` and the runtime skill, build every agent of the neutral agent spec in this engine's format.
6. Run `gabo check`, fix every failure it reports and rerun it until it passes.
7. Run the per-slot smoke test from `gabo-setup` and report each slot as verified or unverified with the reason.
<!-- gabo-setup-prompt:end -->
