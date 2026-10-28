# Setup prompt

The setup prompt is the only entry text of gabo-ai setup (TRD 9.4, D25, PRD R36). Paste it into the coding agent you use, opened in your project: the agent installs the CLI when it is missing, runs `gabo init`, and continues with the guided setup. `gabo init` prints the same text on a fresh project, so you can also start from the terminal and paste what it prints.

The block between the two marker comments is the prompt, byte for byte the file embedded in the CLI (`apps/cli/internal/setupprompt/setup-prompt.md`). Do not edit it here: change the embedded file and copy it over; a Go test fails when the two differ.

## Install the CLI

The CLI needs Go. Install it with:

```sh
go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@latest
```

## Prompt

<!-- gabo-setup-prompt:begin -->
Set up gabo-ai in this project. Work in this project's root directory, follow the steps in order, and skip any step whose result already holds, so running this prompt again is safe. Ask me before every choice and apply no value I did not accept. Never write a secret value anywhere; config files hold only the names of environment variables.

1. If the `gabo` command is missing, install the gabo CLI from https://github.com/testeos-z/gabo-ai/blob/main/docs/setup-prompt.md#install-the-cli and check that it runs.
2. If `.gabo/skills/` is missing, run `gabo init` here.
3. Load these skills from `.gabo/skills/`: `gabo-setup`, `gabo-config-writer`, `gabo-agent-templates`, and the runtime skill for the engine you are running in; if `.gabo/skills/` holds none for your engine, load `gabo-runtime-builder` and write that runtime skill first.
4. Following `gabo-setup` and `gabo-config-writer`, ask me the setup questions one at a time, then write the global user file and the project config `gabo.yaml`.
5. With `gabo-agent-templates` and the runtime skill, build every agent of the neutral agent spec in this engine's format.
6. Run `gabo check`, fix every failure it reports and rerun it until it passes.
7. Run the per-slot smoke test from `gabo-setup` and report each slot as verified or unverified with the reason.

About memory: before you configure the memory slot, explain that one memory backend and one memory project name should be shared by every agent of this project, local and cloud, for every developer, except the reviewer, which never reads or writes it; a local-only memory is fine, for example for a solo developer. Offer Engram with Engram Cloud sync first as the recommendation, list the other backends `gabo-setup` names, record only the memory adapter and the memory project name, and never set up the backend or its credentials. The memory smoke test writes only under a scratch topic and deletes the probe afterwards.
<!-- gabo-setup-prompt:end -->
