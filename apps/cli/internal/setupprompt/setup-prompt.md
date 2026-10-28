Set up gabo-ai in this project. Work in this project's root directory, follow the steps in order, and skip any step whose result already holds, so running this prompt again is safe. Ask me before every choice and apply no value I did not accept. Never write a secret value anywhere; config files hold only the names of environment variables.

1. If the `gabo` command is missing, install the gabo CLI from https://github.com/testeos-z/gabo-ai/blob/main/docs/setup-prompt.md#install-the-cli and check that it runs.
2. If `.gabo/skills/` is missing, run `gabo init` here.
3. Load these skills from `.gabo/skills/`: `gabo-setup`, `gabo-config-writer`, `gabo-agent-templates`, and the runtime skill for the engine you are running in; if `.gabo/skills/` holds none for your engine, load `gabo-runtime-builder` and write that runtime skill first.
4. Following `gabo-setup` and `gabo-config-writer`, ask me the setup questions one at a time, then write the global user file and the project config `gabo.yaml`.
5. With `gabo-agent-templates` and the runtime skill, build every agent of the neutral agent spec in this engine's format.
6. Run `gabo check`, fix every failure it reports and rerun it until it passes.
7. Run the per-slot smoke test from `gabo-setup` and report each slot as verified or unverified with the reason.
