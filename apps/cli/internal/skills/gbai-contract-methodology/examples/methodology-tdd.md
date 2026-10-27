# methodology-tdd: example adapter for the gbai methodology contract

Fills the `methodology` slot with plain test-driven development and no coder-side review tool. Copy it to `.gbai/adapters/methodology-tdd.md` and point `slots.methodology.adapter` at it; coders load it as `gbai-methodology`.

## Operations

### coder instructions

1. Read the task, its acceptance criteria and the code it touches.
2. Per behavior: write a failing test and run it (RED), write the least code that passes (GREEN), then refactor with the tests green.
3. No runnable deterministic test applies (pure documentation, configuration without a test seam): state the exception and run a proportional check instead.
4. Commit each behavior with its tests as one Conventional Commit.
5. Before every push run every command in `review.checks` of `gbai.yaml`; fix failures before pushing.

### coder-side review

`none`. The gbai reviewer is the only review; the coder requests it after each push.

### report per task

- RED and GREEN evidence per behavior (failing then passing test names), or the exception with the check run instead.
- Commits (short SHA and subject).
- Each `review.checks` command with its result.
- Coder-side review: `none`.
