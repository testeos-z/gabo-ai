# Intent 0011 Docs for the public repository

| field | value |
|---|---|
| intent id | 0011-docs-for-the-public-repository |
| status | Accepted |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-06 |

## Problem

`testeos-z/gabo-ai` is now public (owner, 2026-11-06), but the docs, the `gabo init` report and their tests still describe a private repository: they tell every user to set `GOPRIVATE`, to give git credentials before an engine can clone the marketplace, and to copy the "Made with GaboAI" sticker because its URL does not load. New users follow steps they do not need, and a user without GitHub credentials is told setup cannot work. Owner's brief, verbatim: "testeos-z/gabo-ai is now public (owner, 2026-11-06). Remove or update every "while the repository is private" text: README (GOPRIVATE in the install command and Upgrading, Known limitations Private repository, the Made with gabo-ai sticker note), docs/compatibility.md, docs/setup-prompt.md, plugin and package READMEs, gabo init report lines and runtime/setup skills that tell the user to give git credentials, with their tests. Verify go install without GOPRIVATE works from the public module proxy."

## Who is affected

Every new user installing the `gabo` CLI or an engine plugin, every owner who reads the `gabo init` report after each run, and the maintainers who read the operating docs (`packages/pi/README.md`, `docs/cloud-agent-flow.md`) where they still rely on private-repository limits.

## Desired outcome

- No user-facing text says the repository is private or asks for `GOPRIVATE` or git credentials to read it.
- The install command is the plain `go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@latest`, verified to work from the public module proxy.
- The `gabo init` report no longer prints the credential steps, and its tests and golden files agree.
- The sticker snippet works as written, with no copy-it-yourself note.
- The change ships in gabo release 0.2.10. The version bump is not part of this intent.

## Constraints

- Docs and report text only: no change to what `gabo init` writes or to any `gabo check` rule.
- Every change assesses whether `gabo check` must change (TRD 8.5, intent 0008 S10).
- Accepted intent folders and archived documents are history: they keep what was true when written.
- `plugins/` is a `review.human_only_paths` entry: the plugin README changes need a human-only merge.

## Open questions

| # | question | status |
|---|---|---|
| 1 | `packages/pi/README.md` records the owner's acceptance of the npm publishing risk partly because branch protection and GitHub Environments "are not available on this private repository's plan". Public repositories have them. Does locking down `pi/pkg-up` and `NPM_TOKEN` belong to this intent? | answered by the owner on 2026-11-06: "ok", proposal A (spec D2, open question 1) |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#355 I0011: Docs for the public repository](https://github.com/testeos-z/gabo-ai/issues/355) |
| PRD | [PRD](../../PRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) |
| related intent | 0001-claude-code-plugin (open question 3, marketplace while private), 0006-engine-integration-at-init (the credential steps of the report, its open question 7), 0007-update-documentation-for-0-2-6 |
