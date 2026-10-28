---
name: gabo-config-writer
description: "Trigger: write gbai config, edit gbai.yaml, set a config key, set alias, interaction level, tone or language, credential env var. Edits the right YAML file directly, then runs gabo check."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Load whenever an agent writes a gbai config value: the setup skill at J1 step 7 (write the config), the assistant at J4 step 5 (comfort answer: `interaction_level` and `tone`) and when it stores an alias, and any later slot or runtime change (J7). This skill replaces `gbai config set` (T015, withdrawn; TRD D24): there is no config command, the agent edits the YAML directly and `gabo check` validates it. Until the rename lands the binary also answers to `gbai`, so `gbai check` is the same command. It uses the `memory` slot adapter only in auto mode.

## Files

| file | holds | path |
|---|---|---|
| global user file | personal keys of one person, for every repository (D11, D22) | `$XDG_CONFIG_HOME/gbai/config.yaml` when that variable is an absolute path, else `~/.config/gbai/config.yaml` |
| project file | team keys of this project | `gbai.yaml` at the repository root |

- **Personal keys** are exactly `alias`, `interaction_level`, `tone`, `language`. Write them only to the global user file, never to `gbai.yaml` (there they are ignored with a warning).
- **Team keys** are every other key (`version`, `mode`, `repo`, `slots`, `runtimes`, `plan_change`, `review`, `worktree`, `developers`, and the rest of the project schema). Write them only to `gbai.yaml`, never to the global user file, even though that file would accept them as a fallback.
- The allowed keys, types and enums come from the JSON Schemas (`gbai.user.schema.json`, `gbai.project.schema.json`, draft 2020-12, `additionalProperties: false`), published at the URL on each file's first line and embedded in the CLI that `gabo check` runs.

## Hard Rules

- **Edit the YAML directly, minimally.** Change only the lines of the key you write. Keep every comment, the key order, the indentation style and the first line `# yaml-language-server: $schema=<url>`. Never reformat, re-serialize or sort the whole file, and never drop a key you did not touch.
- **Owner values only (R12).** Write a value only when the owner gave it in this session, or it is already in the file. Never fill a default, a guess or an example value. In manual mode, a missing value is one question for the owner, asked through the caller (the assistant or setup skill); a subagent returns it as a question in its outcome instead of writing.
- **Auto mode.** When `mode: auto` in `gbai.yaml` and the owner is not asked, the agent may choose a value only after recording the decision through `memory` (`save decision`: what, question, choice, reason, mode, wave, task) and reading it back. A decision that does not show up after one retry means the value is not written; report it as an unverified write.
- **No secrets (R21).** Never write a token, password, API key, credential or any secret value to either file. A credential is configured by its env var name only (for example `pi_credential_env: PI_API_KEY`); the value stays in the environment or the owner's secret store. If the owner pastes a secret, do not write it, tell the owner it was not stored and ask for the env var name instead.
- **Refuse keys outside the schemas.** Before writing, confirm the key path exists in the schema of the target file and the value matches its type, enum or pattern. A key the schemas do not define (including the former cloud keys `server`, `users`, `bot`, `sessions`; agent hosts go in `slots.agent_host`) is refused and reported with the key path; never add it, rename it into something that looks valid, or put it under another key.
- **`gabo check` after each write.** Run `gabo check` once after every write, read its output, and fix what fails before the next write: a failure on a key you wrote is fixed in place (the owner's value stays, only the shape changes; a different value needs the owner or, in auto mode, a new decision record). A failure you cannot fix without a value the owner did not give, or one on a key you did not write, is reported with the check's key path, file and `next:` line, never hidden or worked around by deleting the key.

## Steps

1. Classify the key: personal ⇒ global user file; team ⇒ `gbai.yaml`. A key in neither schema ⇒ refuse (Hard Rules).
2. Get the value: from the owner, from the file, or in auto mode from a recorded decision. A secret ⇒ its env var name only.
3. Open the target file. If it does not exist, create it with exactly one first line and then the key:
   - global user file: `# yaml-language-server: $schema=https://raw.githubusercontent.com/testeos-z/gbai-team/main/schemas/gbai.user.schema.json`
   - `gbai.yaml`: `# yaml-language-server: $schema=https://raw.githubusercontent.com/testeos-z/gbai-team/main/schemas/gbai.project.schema.json`, then `version: 1`.
   If an existing file lacks that first line, add it as the first line; change nothing else for it.
4. Edit the key in place: replace the value on its line, or add the key under its parent mapping, keeping the surrounding comments. A personal key found in `gbai.yaml` is moved to the global user file only with the owner's consent (manual) or a recorded decision (auto).
5. Run `gabo check` and act on it (Hard Rules). Repeat steps 1 to 5 for the next key.

## Output Contract

Return, per key written: key path, file, value source (`owner`, `existing`, or `auto decision <id>`), and the `gabo check` result after that write (`pass` or the failures left with their `next:` lines). Then list the refused keys with the reason, the questions for the owner, and any unverified write.
