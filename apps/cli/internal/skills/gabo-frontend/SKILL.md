---
name: gabo-frontend
description: "Trigger: UX, UI, Storybook, components, design tokens, CSS, DESIGN.md, app flow, frontend. Designs UX/UI, components, tokens and the app flow."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.2"
---
## Activation Contract

Use to write the repository `DESIGN.md` (from `templates/DESIGN.md`, optional, global styles only), section `## 3. UX and app flow (optional)` of each intent's `spec.md`, and to plan Storybook components.

## Hard Rules

- Templates are canonical (D19): answer each template's `## Fixed questions` before writing its sections, and never rename or reorder its table columns.
- Fixed questions one at a time: ask the template's `## Fixed questions` in order, one per message, and wait for each answer before the next (manual mode). In auto mode answer each yourself, mark it **Proposed (auto)** with a one-line reason and present them together for one-pass approval.
- Tokens first: color, spacing, typography, radius and motion as CSS custom properties; components use tokens only.
- Atomic design: atoms → molecules → organisms → templates → pages; each component gets a Storybook story with its states (default, hover, focus, disabled, loading, error, empty).
- Accessibility: keyboard paths, focus order, contrast, labels.
- App flow: every screen or surface, its entry points, the transitions and the error and empty states; one journey per story with trigger, actors, steps, decision points, outcome, failure and escalation. A product without a GUI lists surfaces (conversation, CLI, tracker) instead of screens.

## Traceability check

Run it before each next planning step: before moving to the next document, before creating or updating tasks, and before returning (D19). Read the stable template tables: PRD stories and requirements, TRD decisions and requirement coverage, plan tasks and coverage. Rows marked removed (a title starting `(removed <date>)`, or a status withdrawn or superseded) are ignored on both sides of every link. A gap is:

- a story without requirement;
- a requirement without decision, task or verification;
- a task without story;
- a dependency cycle, a dangling id (a story, requirement, decision or task id no live row defines), or a `depends_on` / `unblocks` mismatch.

Any gap blocks the next planning step: fix it in your own documents or return it as an open question naming the ids; never skip it. A document not written yet is reported as not checked, not as passing.

## Execution Steps

1. Read PRD requirements and personas.
2. Answer the DESIGN fixed questions, then write DESIGN.md (principles, tokens, component inventory, accessibility, messages; the token source of truth).
3. Answer the spec's UX and app flow question, then write that spec section as a surface list, the journeys and a flow diagram (Mermaid).
4. Run the traceability check before the next planning step.

## Output Contract

Return the component inventory, the screens, the components each implementation task needs, and traceability gaps (or none).
