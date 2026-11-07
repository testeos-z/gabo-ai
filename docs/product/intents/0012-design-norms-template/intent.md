# Intent 0012 Design norms template

| field | value |
|---|---|
| intent id | 0012-design-norms-template |
| status | Accepted |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-06 |

## Problem

A project with a UI asks `gabo-frontend` for its design norms, and the `DESIGN.md` template it gets is a short global style sheet: principles, a token table, a component inventory, a free-text accessibility paragraph and a messages table. It has no place for layout and responsive behavior, type scale, icons, content and writing, forms or full accessibility (keyboard, focus order, target size, labels), and its rules have no common shape: no normative level, no stated verification, no rationale, no examples and no source. A rule nobody can check is not enforced, and the review checks of a consumer project cannot be wired to rules that do not say how they are verified.

The vulcam session hit this first (cappy-lab/vulcam#624, owner decisions of 2026-10-30): a review of vulcam's design norms against open design systems (Carbon, Polaris, GOV.UK) and WCAG 2.2 AA found those six areas missing, and the owner approved one rule format, verbatim from that issue: "Normative keywords per RFC 2119 (MUST / SHOULD / MAY). One rule, one idea, with a stable ID [...]. Checkable: each rule states how it is verified (automated test or lint, or a concrete manual review step). A one-line rationale. Do / don't examples (image or Storybook story). Exceptions recorded with date, reason and owner. Source and governance: each rule cites its source (WCAG criterion or owner decision); changes go through a reviewed proposal and a changelog." The owner decided the template lives in gabo-ai (issue #162) and approved, in the same session, that the "what" of the UX (principles, inventory, screens, flows, concrete messages) stays apart from the "how" (the rules), so the same rule is never written in two documents (issue #164). vulcam's own rewrite of its `DESIGN.md` waits for this template.

The first attempt (PR #163, the template; PR #168, its plan) was closed on 2026-11-06 because it predates the per-change model: it edited `docs/product/PRD.md` inside a code PR (TRD D7) and split against a global `UX_UI.md` template that the per-change model has since retired (TRD D14).

## Who is affected

- Owners of projects with a UI (vulcam first), each time `gabo-frontend` writes or updates their `DESIGN.md`, and the reviewers and checks that enforce those norms on every UI PR.
- `gabo-frontend` itself, when it writes section 3 of each intent's spec and must cite the norms instead of repeating them.
- Products without a GUI, gabo-ai included, only as far as the template must still let them say "not applicable".

## Desired outcome

- `gabo-frontend` scaffolds `DESIGN.md` from a norms template that covers the fifteen areas of the approved format, each rule in one table shape with level, rationale, verification, examples and source, plus exceptions and a changelog.
- The rules live only in `DESIGN.md`; each change's concrete surfaces, journeys and messages live in its spec section 3, which links the rules instead of repeating them.
- Catalog tests pin the new template shape, so a later edit cannot silently drop a section or a column.
- `docs/product/PRD.md` is not touched by any code PR of this intent.

## Constraints

- Per-change model (TRD D14, D19): no global UX or app-flow document comes back; templates keep stable table columns; directiva skills keep no reference to the retired `UX_UI` and `APP_FLOW` documents (catalog test `retiredPlanningTerms`).
- Plan changes never travel with code changes (TRD D7): the PRD and TRD are edited only in `docs/product` PRs.
- Every change assesses whether `gabo check` must change (TRD 8.5, intent 0008 S10).
- `plugins/` is a `review.human_only_paths` entry: the `plugins/codex/skills/gabo-frontend` copy needs a human-only merge.
- About 400 changed lines per reviewable task.

## Open questions

| # | question | status |
|---|---|---|
| 1 | With no global `UX_UI.md` any more, where do the global "what" items of issue #164 live: the product principles and the shared component inventory? | carried to the spec (D2, open question 1), passed to the owner with a proposed answer; answered by the owner 2026-11-07: the principles stay in `DESIGN.md`, the component inventory goes in each change's spec section 3 (spec D2 C) |
| 2 | Should `gabo check` validate a project's `DESIGN.md` rule tables? | carried to the spec (D4, open question 2), passed to the owner with a proposed answer |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#374 I0012: Design norms template](https://github.com/testeos-z/gabo-ai/issues/374) |
| PRD | [PRD](../../PRD.md) R20 |
| spec | [spec](spec.md) (accepted 2026-11-06) |
| plan | [plan](plan.md) |
| origin | [cappy-lab/vulcam#624](https://github.com/cappy-lab/vulcam/issues/624) (owner decisions 2026-10-30) |
| earlier requests | [#162](https://github.com/testeos-z/gabo-ai/issues/162) (DESIGN.md norms template), [#164](https://github.com/testeos-z/gabo-ai/issues/164) (UX_UI split) |
| earlier PRs (closed) | [#163](https://github.com/testeos-z/gabo-ai/pull/163) (branch `feat/design-template`, the template), [#168](https://github.com/testeos-z/gabo-ai/pull/168) (branch `docs/plan-t078-t079-design-template`, plan T078 and T079) |
| related intent | 0008-stricter-gabo-check (TRD 8.5) |
