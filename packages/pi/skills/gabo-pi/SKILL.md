---
name: gabo-pi
description: "Trigger: gabo, gabo-ai, set up gabo, start gabo in this project, gabo missing, gabo init, gabo prompt. Bootstraps gabo-ai in Pi: installs the CLI if missing, runs gabo init, then follows gabo prompt."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.0"
---
## Activation Contract

Load in Pi when the user asks to set up or start gabo-ai in a project, or asks for a gabo skill the project does not have yet. This skill only bootstraps: the CLI prints the setup text and the project's `.gabo/skills/` copy holds every other gabo skill.

## Hard Rules

- Work in the project root. Skip a step whose result already holds, so running this skill again is safe.
- Never write the setup steps from memory or from this skill: `gabo prompt` prints the only setup text, and it matches the installed CLI version.
- This package ships no catalog skill and no reviewer. The role skills, the slot adapters and `gabo-runtime-pi` come from `.gabo/skills/` after `gabo init`; never run a review from this skill.
- Building gabo agents for Pi needs a subagent package (gentle-pi); `gabo-runtime-pi` refuses without one and says why. Report its refusal to the user unchanged.

## Steps

1. Run `gabo version`. When the command is missing, install the gabo CLI with `go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@latest` (it needs Go), then run `gabo version` again. When the install fails, or `gabo init` later refuses the installed version, follow the "Install the CLI" section of https://github.com/testeos-z/gabo-ai/blob/main/docs/setup-prompt.md and stop if it still fails.
2. When `.gabo/skills/` is missing in the project root, run `gabo init` there.
3. Run `gabo prompt` and follow its output, step by step, as the user's request.
