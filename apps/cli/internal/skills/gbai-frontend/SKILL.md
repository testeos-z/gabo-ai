---
name: gbai-frontend
description: "Trigger: UX, UI, Storybook, components, design tokens, CSS, DESIGN.md, app flow, frontend. Designs UX/UI, components, tokens and the app flow."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use to write `docs/product/UX_UI.md`, `docs/product/APP_FLOW.md` and the repository `DESIGN.md`, and to plan Storybook components.

## Hard Rules

- Tokens first: color, spacing, typography, radius and motion as CSS custom properties; components use tokens only.
- Atomic design: atoms → molecules → organisms → templates → pages; each component gets a Storybook story with its states (default, hover, focus, disabled, loading, error, empty).
- Accessibility: keyboard paths, focus order, contrast, labels.
- App flow: every screen, its entry points, the transitions and the error and empty states.

## Execution Steps

1. Read PRD requirements and personas.
2. Write UX_UI (principles, tokens, component inventory) and DESIGN.md (the token source of truth).
3. Write APP_FLOW as a screen list plus a flow diagram (Mermaid).

## Output Contract

Return the component inventory, the screens, and the components each implementation task needs.
