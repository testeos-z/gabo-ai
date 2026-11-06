# Engine compatibility and integrations

Status for gabo 0.2.6 (tag `apps/cli/v0.2.6`), checked against the code. **Yes** means it ships in 0.2.6; **planned** means it is designed or tracked but does not work yet. The [README](../README.md#compatibility) has the short version.

## Tiers

- **Full**: Claude Code, Codex, Pi and OpenCode. Each has a runtime skill, a plugin or package with hooks, the project-level integration that `gabo init` sets up, and, for Claude Code, Codex and Pi, the engine files that `gabo check` reads.
- **Flexible**: any other engine with subagents and skills. It gets only the gabo skills and configuration: no plugin and no hooks. `gabo init` writes nothing engine-specific for it and prints one line, `<engine>: flexible tier: skills and configuration only; nothing written`, and `gabo check` stays engine-neutral. In 0.2.6, `gabo.yaml` does not accept such an engine name yet (see [Accepted engine names](#notes)).

## Full matrix

| engine | tier | coder | assistant | reviewer | runtime skill | plugin or package | subagents | post-write hook (`gabo check`) | pre-write hook (`gabo check --pre-write`) | `gabo init` writes | `gabo check` reads |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Claude Code | full | yes | yes | yes | `gabo-runtime-claude` | yes: plugin `gabo@gabo-ai` ([Claude Code plugin](#claude-code-plugin)) | built in | yes, in the plugin | yes, in the plugin | `.claude/settings.json`: `extraKnownMarketplaces."gabo-ai"`, `enabledPlugins."gabo@gabo-ai"` | the same keys |
| Codex | full | yes | yes | yes, after the one-time isolation test | `gabo-runtime-codex` | yes: plugin `gabo@gabo-ai` ([Codex plugin](#codex-plugin)) | built in | yes, in the plugin | yes, in the plugin | `.codex/config.toml`: `[marketplaces.gabo-ai]`, `[plugins."gabo@gabo-ai"]` | the same keys |
| Pi | full | yes | yes | yes | `gabo-runtime-pi` | yes: package `@gabo-ai/pi`, `pi install npm:@gabo-ai/pi@<version>` ([Pi package](#pi-package)) | **required**: a subagent package such as gentle-pi; gabo refuses without one | no | yes, in the package extension | `.pi/settings.json`: the `packages` entry `npm:@gabo-ai/pi@<version>` (release builds only) | that entry, any version |
| OpenCode | full | yes | yes | yes, after the one-time isolation test | `gabo-runtime-opencode` | no plugin to install: the runtime skill copies the stage-gate plugin `.opencode/plugins/gabo-stage-gate.js` into the project | built in; `gabo check` requires `experimental.subagent_depth` of 3 or more | no | yes, in the stage-gate plugin | nothing (the runtime skill installs the stage-gate plugin) | no engine key |
| any other engine | flexible | planned | planned | planned | written by `gabo-runtime-builder` | none | required (the builder refuses an engine without subagents and skills) | none | none | nothing; one flexible-tier line | nothing engine-specific |

## Notes

- **Accepted engine names.** `gabo.yaml` and `gabo check` accept `claude`, `codex`, `pi` and `opencode` in `runtimes.coder`, `runtimes.assistant` and `runtimes.reviewer`. Any other value fails `gabo check`, so a flexible-tier engine, including a runtime skill that `gabo-runtime-builder` writes for it, cannot be used until [I0005 (#176)](https://github.com/testeos-z/gabo-ai/issues/176) opens `runtimes.*`.
- **Engine integration by `gabo init`.** For each engine that `runtimes.*` in `gabo.yaml` names, `gabo init` writes the project-level keys of the matrix above, never your global engine config. It only adds missing keys and keeps every value you set. It ends with an `engine integration:` report that names each file it wrote and what you still need to run, such as trusting the project folder in the engine. Without a `gabo.yaml` it writes no engine file and asks you to run `gabo init` again once your agent has written it.
- **gabo-ai dogfoods `gabo init`.** This repository runs `gabo init` on itself and commits the engine file it writes for its own `gabo.yaml` `runtimes` (all `claude`): [`.claude/settings.json`](../.claude/settings.json) with `extraKnownMarketplaces."gabo-ai"` (the GitHub source `testeos-z/gabo-ai`) and `enabledPlugins."gabo@gabo-ai": true`, so its own `gabo check` verifies those keys.
- **Skill directories (on `dev`, ships in 0.2.8).** Codex, OpenCode and Pi share `.agents/skills`: one link per name, the same target for every engine, so one set of links serves all three. Claude Code keeps `.claude/skills`, its own folder of links straight to `.gabo/skills`. When `claude` and `opencode` are both declared, OpenCode finds each gabo skill in both folders with the same target; gabo agents read skills by path, so this changes nothing and `gabo check` does not fail on it. A rebuild with `gabo-runtime-pi` or `gabo-runtime-opencode` removes the `gabo-*` links they made earlier in `.pi/skills` or `.opencode/skills` and keeps your own entries; `gabo check` fails on such a link left behind (`✗ <path>: gabo link left after the move to .agents/skills`).
- **Per-agent checks (on `dev`, ships in 0.2.8).** For every declared engine, `gabo check` reads each agent file where its runtime skill says the skill list and the agent's name are (`metadata.agent_skills`, `metadata.agent_name`): the skill list must be exactly the agent's spec skills plus its fixed-name links (`skill <name> missing` / `skill <name> not in the agent spec`), and a declared name must equal the spec id (`declares name <name>`; OpenCode declares no name, so it gets no name check). Every agent file needs a `gabo-version: x.y.z` line, and without a valid stamp the versions are reported as not compared, never skipped. A `gabo-reviewer` skill in a project skill directory or an agent's skill list fails; `gabo-reviewer-config` does not. A flexible-tier runtime skill without `metadata.agent_skills` gets one warning instead of these checks.
- **Reviewer configuration.** The catalog skill `gabo-reviewer-config` configures any of the four engines as reviewer. For `codex` and `opencode` it runs a one-time isolation test first; the reviewer runs those engines only while its pass marker matches the installed engine version.
- **Pi needs a subagent package.** Pi core has no subagents. `gabo-runtime-pi` builds nothing until a subagent package is present. It recommends [gentle-pi](https://www.npmjs.com/package/gentle-pi) and, as the full setup, the [Gentle AI framework](https://github.com/Gentleman-Programming/gentle-ai) installed with `gentle-ai install --agent pi`. The gabo Pi package is not a subagent package. See [Your engine](setup-prompt.md#your-engine).
- **Private repository.** While `testeos-z/gabo-ai` is private, every engine clones it with your own git credentials. Set them up first: run `gh auth setup-git`, or add an SSH key that can read the repository.
- **Tracking.** All engine work is tracked under the product card [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171).

## Claude Code plugin

The repository is its own Claude Code marketplace, `gabo-ai`, with one plugin, `gabo`. The plugin adds the gabo catalog skills (except the skills that run reviews) and two hooks that run `gabo check` around Claude's writes in a gabo project.

**With `gabo init`.** When `gabo.yaml` names `claude` in `runtimes.*`, `gabo init` adds `extraKnownMarketplaces."gabo-ai"` and `enabledPlugins."gabo@gabo-ai"` to the project's `.claude/settings.json`, never to your global Claude Code settings. Its `engine integration:` report then asks you to trust the project folder in Claude Code, so the marketplace and the plugin apply, and, while the repository is private, to give git credentials that can read it. Commit `.claude/settings.json`. As the alternative, install the plugin at user level by hand:

```sh
claude plugin marketplace add testeos-z/gabo-ai
claude plugin install gabo@gabo-ai
```

- **The `checks` option.** On by default. Before Claude writes or edits a gabo file (`gabo.yaml`, `.gabo/`, `.claude/agents/`, `.claude/skills/`, `docs/product/`), `gabo check --pre-write` blocks a write that breaks the stage gate; after it, `gabo check` runs and reports failures to Claude. Both hooks do nothing outside a gabo project (no `.gabo/VERSION`) or when `gabo` is not on `PATH`. Turn them off with the plugin's `checks` row in `/config`.
- **`gabo init` stays required.** The plugin does not configure a project and ships no agents; `gabo init` writes only the two plugin keys above. Per-project config (`gabo.yaml`), slots, adapters and agents still come from `gabo init` and the setup prompt. Inside a project, the copy in `.gabo/skills/` stays the source of truth: the agents preload it and `gabo check` checks it.
- **Skill names.** Plugin skills appear namespaced as `gabo:<name>`, for example `gabo:gabo-pm`, so they never clash with the project copy. To use only the project copy in one project, disable the plugin there with `"enabledPlugins": { "gabo@gabo-ai": false }` in that project's `.claude/settings.json`.

See [`plugins/claude-code/README.md`](../plugins/claude-code/README.md) for the plugin layout and the release checklist.

## Codex plugin

The repository is also a Codex marketplace, `gabo-ai` ([`.agents/plugins/marketplace.json`](../.agents/plugins/marketplace.json)), with one plugin, `gabo`. It ships the same skills as the Claude Code plugin and two hooks that run `gabo check` around Codex's `apply_patch` edits in a gabo project. These steps are for the Codex CLI.

**With `gabo init`.** When `gabo.yaml` names `codex` in `runtimes.*`, `gabo init` adds `[marketplaces.gabo-ai]` and `[plugins."gabo@gabo-ai"]` to the project's `.codex/config.toml`, never to your global Codex config. Its `engine integration:` report then asks you to trust the project in Codex, so it loads `.codex/config.toml`, to trust both gabo hooks in `/hooks` and, while the repository is private, to set up git credentials that can read it. If Codex does not list the gabo plugin after trust, the report tells you to run the two commands below. Commit `.codex/config.toml`. As the alternative, install the plugin by hand:

```sh
codex plugin marketplace add testeos-z/gabo-ai
codex plugin add gabo@gabo-ai
```

- **Trust the hooks.** Codex runs a plugin hook only after you trust it. In a Codex session, open `/hooks` and trust both gabo hooks: the `PreToolUse` hook denies a patch to a gabo file that breaks the stage gate (`gabo check --pre-write`), and the `PostToolUse` hook runs `gabo check` after the patch and returns its failures to Codex. A hook that changes in an update stays off until you trust it again. Both hooks do nothing outside a gabo project (no `.gabo/VERSION`) or when `gabo` is not on `PATH`. To turn them off, leave them untrusted.
- **Update.** Run `codex plugin marketplace upgrade gabo-ai`. Codex caches the plugin per version, so you get a new plugin when a CLI release bumps its version; then check `/hooks` again.
- **`gabo init` stays required.** The plugin does not configure a project and builds no agents. Run `gabo init` and the setup prompt with `runtimes.coder: codex`: the `gabo-runtime-codex` skill builds the agents as `.codex/agents/<id>.toml` and links the skills under `.agents/skills/`.

See [`plugins/codex/README.md`](../plugins/codex/README.md) for the plugin layout, how its skill and script copies stay equal to their sources, and the release checklist.

## Pi package

The Pi package `@gabo-ai/pi` ([`packages/pi/`](../packages/pi/)) ships the `gabo-pi` and `gabo-reviewer-config` skills and an extension that runs `gabo check --pre-write` at session start and before Pi's `write` and `edit` calls in a gabo project.

**With `gabo init`.** When `gabo.yaml` names `pi` in `runtimes.*`, a release build of `gabo init` adds the `packages` entry `npm:@gabo-ai/pi@<version>`, pinned to its own version, to the project's `.pi/settings.json`, never to your global Pi settings; a development build writes nothing and tells you to add the entry by hand. Its `engine integration:` report then asks you to trust the project in Pi, so it installs and loads `@gabo-ai/pi` from `.pi/settings.json`, and to install a subagent package. Commit `.pi/settings.json`. As the alternative, install the package by hand:

```sh
pi install npm:@gabo-ai/pi@<version>
```

Use the gabo CLI version as `<version>`. Before the package is on npm, install it from a checkout with `pi install /path/to/gabo-ai/packages/pi`. Check it with `pi list`.

- **`gabo init` and `gabo-runtime-pi` stay required.** The package itself writes no project file and builds no agents; with `runtimes.coder: pi`, `gabo-runtime-pi` builds them in `.pi/agents/` and links the skills under `.agents/skills/`.
- **It is not a subagent package.** gabo on Pi still needs one, such as gentle-pi; without it the extension warns at session start and blocks writes under `.pi/agents/`.

See [`packages/pi/README.md`](../packages/pi/README.md) for the package layout, coexistence rules and release pipeline, and [`packages/pi/VERIFY.md`](../packages/pi/VERIFY.md) for the manual verification run before each release.

## OpenCode

OpenCode needs no plugin install. The `gabo-runtime-opencode` skill copies the stage-gate plugin `.opencode/plugins/gabo-stage-gate.js` into the project, which runs `gabo check --pre-write` before writes, and sets `experimental.subagent_depth` to 3 or more in the project config. It builds the agents in `.opencode/agents/` and links the skills under `.agents/skills/`. `gabo init` writes nothing for OpenCode and reports whether the stage-gate plugin is in place.

## Checked engine versions

Each runtime skill rechecks its sources when the installed version differs.

| engine | version | checked on | where it is recorded |
|---|---|---|---|
| Claude Code | 2.1.302 | 2026-10-28 | `gabo-runtime-claude` skill, Sources |
| Codex (`codex-cli`) | 0.163.0 | 2026-11-03 | `gabo-runtime-codex` skill, Sources |
| Pi | 1.4.0 | 2026-11-03 | `gabo-runtime-pi` skill, Sources |
| gentle-pi | 4.5.1 | 2026-11-03 | `gabo-runtime-pi` skill, Sources |
| OpenCode | 2.0.32 | 2026-11-03 | `gabo-runtime-opencode` skill, Sources |
