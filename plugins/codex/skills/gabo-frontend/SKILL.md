---
name: gabo-frontend
description: "Trigger: UX, UI, Storybook, components, design tokens, CSS, DESIGN.md, app flow, frontend. Designs UX/UI, components, tokens and the app flow."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.2"
---
## Activation Contract

Use to keep the optional global `DESIGN.md` (from `templates/DESIGN.md`, global styles only, when the product warrants UX/UI work), to contribute section `## 3. UX and app flow (optional)` of each intent's `spec.md`, and to plan Storybook components. UX and app flow are per intent; there is no global UX or app-flow document.

## Hard Rules

- Templates are canonical (D19): answer each template's `## Fixed questions` before writing its sections, and never rename or reorder its table columns.
- Fixed questions one at a time: ask the template's `## Fixed questions` in order, one per message, and wait for each answer before the next (manual mode). In auto mode answer each yourself, mark it **Proposed (auto)** with a one-line reason and present them together for one-pass approval.
- Tokens first: color, spacing, typography, radius and motion as CSS custom properties; components use tokens only.
- Atomic design: atoms → molecules → organisms → templates → pages; each component gets a Storybook story with its states (default, hover, focus, disabled, loading, error, empty).
- Accessibility: keyboard paths, focus order, contrast, labels.
- App flow: every screen or surface, its entry points, the transitions and the error and empty states; one journey per changed path of the intent with trigger, actors, steps, decision points, outcome, failure and escalation, written in the spec's section 3. A product without a GUI lists surfaces (conversation, CLI, tracker) instead of screens.

## Traceability check

Run it before each next planning step: before moving to the next document, before creating or updating tasks, and before returning (D19). Read the intent folders `docs/product/intents/<NNNN-slug>/` (`intent.md`, the spec requirements and its acceptance record, the plan task table) and the tracker tasks through the `tasks` contract. Rows marked removed (a title starting `(removed <date>)`, or a status withdrawn or superseded) are ignored on both sides of every link. A gap is:

- an intent without spec;
- an accepted spec without plan;
- a plan task without a tracker task, or the reverse;
- a task whose parent is not exactly one intent;
- a requirement in the spec not covered by any plan task;
- a dependency cycle;
- a dangling id, including a cross-intent `<NNNN>/T<n>` reference;
- a `depends_on`/`unblocks` mismatch.

Any gap blocks the next planning step: fix it in your own documents or return it as an open question naming the ids; never skip it. A document not written yet is reported as not checked, not as passing.

## Execution Steps

1. Read the PRD users and journeys, `DESIGN.md` if present, and the intent's `intent.md` and `spec.md`.
2. Only when the product warrants it: answer the DESIGN fixed questions, then write or update `DESIGN.md` (principles, tokens, component inventory, accessibility, messages; the token source of truth).
3. Per intent: answer the spec's UX and app flow question, then write that spec section as a surface list, the journeys and a flow diagram (Mermaid), or "not applicable" with the reason.
4. Run the traceability check before the next planning step.

## Output Contract

Return the component inventory, the surfaces the intent changes, the components each implementation task needs, and traceability gaps (or none).
