# Design norms

| field | value |
|---|---|
| status | Accepted |
| owner | gabo-frontend |
| date | 2026-11-07 |
| linked docs | [PRD](PRD.md), [TRD](TRD.md) |

## 1. Principles and sources

- Clarity first, it serves PRD R1.
- Standards followed: WCAG 2.2 AA, DTCG 2025.10.

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| P1 | Every screen MUST have one primary action. | MUST | Focus | manual: review the screen | one button / two | PRD R1 |

## 2. Tokens

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| T1 | Components MUST read colors from tokens only. | MUST | One source of truth | automated: lint no-raw-color | var(--c) / #fff | owner 2026-11-07 |

## 3. Color and themes

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| C1 | The dark theme SHOULD mirror the light roles. | SHOULD | Consistency | manual: theme review | — | owner 2026-11-07 |

## 4. Typography

| id | level | rule | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| Y1 | Headings MAY use the display face. | MAY | Brand | — | — | owner 2026-11-07 |

## 5. Layout and responsive behavior

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| L1 | Pages MUST NOT scroll sideways at 320 px. | MUST | Reflow | automated: e2e reflow test | wrap / overflow | WCAG 1.4.10 |

## 6. Shape and density

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| S1 | Controls SHOULD use the 8 px spacing rhythm. | SHOULD | Rhythm | manual: spacing review | 8 / 7 | owner 2026-11-07 |

## 7. Motion and state

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| M1 | Animations MUST respect prefers-reduced-motion. | MUST | Vestibular safety | automated: story test | off / on | WCAG 2.3.3 |

## 8. Icons

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| I1 | Icons MAY be filled in the navigation bar. | MAY | Weight | manual: icon review | — | owner 2026-11-07 |

## 9. Accessibility

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| A1 | Focus MUST stay visible on every control. | MUST | Keyboard users | automated: axe focus-visible | ring / none | WCAG 2.4.7 |

## 10. Content and writing

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| W1 | Buttons SHOULD start with a verb. | SHOULD | Clarity | manual: copy review | Save / OK | owner 2026-11-07 |

## 11. Forms

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| F1 | Every field MUST have a visible label. | MUST | Accessibility | automated: axe label | label / placeholder | WCAG 3.3.2 |

## 12. Components

Atomic design: atom, molecule, organism, template, page.

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| K1 | Each component MUST have a story per state. | MUST | Review | automated: storybook coverage test | Button.stories / none | owner 2026-11-07 |

## 13. Exceptions

| rule id | exception | reason | owner | date |
|---|---|---|---|---|
| L1 | The data grid scrolls sideways | Wide tables | owner | 2026-11-07 |

## 14. Changelog

| date | rule ids | change | decision |
|---|---|---|---|
| 2026-11-07 | P1, T1 | added | owner decision 2026-11-07 |

## 15. Open questions

1. Which icon set for marketing pages? (gabo-frontend)
