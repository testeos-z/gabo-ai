#!/usr/bin/env bash
# gabo-pre-write.sh: the gabo plugin's pre-write hook, PreToolUse Write|Edit
# in Claude Code and, run as `gabo-pre-write.sh codex`, PreToolUse
# apply_patch in Codex (intent 0001 S7, S12, D8; intent 0002 S8, S13, D5).
# The Codex plugin ships a byte-identical copy.
#
# A thin layer: it applies the S7 filters, pipes the tool-call JSON unchanged
# to `gabo check --pre-write` in the project root and maps the exit code.
# Every rule lives in gabo; the script writes nothing.
#
#   checks option off, no .gabo/VERSION, no gabo on PATH, or no written file
#   is a gabo path                           -> exit 0, silent
#   gabo exit 0                              -> exit 0, silent
#   gabo exit 1, Claude Code                 -> its lines on stderr, exit 2
#                                               (Claude Code blocks the call)
#   gabo exit 1, codex                       -> Codex PreToolUse deny JSON
#                                               with its lines as
#                                               permissionDecisionReason,
#                                               exit 0 (Codex denies the patch)
#   any other gabo exit (2: usage or error)  -> Claude Code: one warning line,
#                                               exit 0; codex: exit 0, silent

here="${BASH_SOURCE[0]%/*}"
[[ "$here" == "${BASH_SOURCE[0]}" ]] && here=.
source "$here/gabo-hook-filter.sh"
runtime="${1:-}"
gabo_hook_filter "$runtime" || exit 0

out="$(cd "$gabo_root" && printf '%s' "$gabo_input" | gabo check --pre-write 2>&1)"
code=$?
case "$runtime:$code" in
  *:0) exit 0 ;;
  codex:1)
    printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":%s}}\n' "$(gabo_json_string "$out")"
    exit 0
    ;;
  codex:*) exit 0 ;;
  *:1)
    printf '%s\n' "$out" >&2
    exit 2
    ;;
  *)
    echo "gabo plugin warning: gabo check --pre-write could not run (exit $code); pre-write check skipped" >&2
    exit 0
    ;;
esac
