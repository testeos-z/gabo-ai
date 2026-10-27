---
name: gabo-frontend
description: "Trigger: UX, UI, Storybook, components, design tokens, CSS, DESIGN.md, app flow, frontend. Designs UX/UI, components, tokens and the app flow."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.1"
---
## Activation Contract

Use to write `docs/product/UX_UI.md` (from `templates/UX_UI.md`), `docs/product/APP_FLOW.md` (from `templates/APP_FLOW.md`) and the repository `DESIGN.md`, and to plan Storybook components.

## Hard Rules

- Templates are canonical (D19): answer each template's `## Fixed questions` before writing its sections, and never rename or reorder its table columns.
- Tokens first: color, spacing, typography, radius and motion as CSS custom properties; components use tokens only.
- Atomic design: atoms → molecules → organisms → templates → pages; each component gets a Storybook story with its states (default, hover, focus, disabled, loading, error, empty).
- Accessibility: keyboard paths, focus order, contrast, labels.
- App flow: every screen or surface, its entry points, the transitions and the error and empty states; one journey per story with trigger, actors, steps, decision points, outcome, failure and escalation. A product without a GUI lists surfaces (conversation, CLI, tracker) instead of screens.

## Execution Steps

1. Read PRD requirements and personas.
2. Answer the UX_UI fixed questions, then write UX_UI (principles, tokens, component inventory, accessibility, messages) and DESIGN.md (the token source of truth).
3. Answer the APP_FLOW fixed questions, then write APP_FLOW as a surface list, the journeys and a flow diagram (Mermaid).

## Output Contract

Return the component inventory, the screens, and the components each implementation task needs.
