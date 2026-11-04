#!/usr/bin/env bash
# gabo-check.sh: the gabo plugin's post-write hook, PostToolUse Write|Edit in
# Claude Code and, run as `gabo-check.sh codex`, PostToolUse apply_patch in
# Codex (intent 0001 S6, S7, S8, D9, D10; intent 0002 S7, S8, S9, D5). The
# Codex plugin ships a byte-identical copy.
#
# A thin layer: it applies the S7 filters (gabo-hook-filter.sh, shared with
# the pre-write hook), runs `gabo check` in the project root and maps the
# exit code. Every check rule lives in gabo; the script writes nothing.
#
#   checks option off, no .gabo/VERSION, no gabo on PATH, or no written file
#   is a gabo path                           -> exit 0, silent
#   gabo check exit 0                        -> exit 0, silent
#   gabo check exit 1, Claude Code           -> its lines on stderr, exit 2
#   gabo check exit 1, codex                 -> its lines as Codex JSON
#                                               hookSpecificOutput.
#                                               additionalContext, exit 0
#   any other gabo exit (2: usage or error)  -> Claude Code: one warning line,
#                                               exit 0; codex: exit 0, silent

here="${BASH_SOURCE[0]%/*}"
[[ "$here" == "${BASH_SOURCE[0]}" ]] && here=.
source "$here/gabo-hook-filter.sh"
runtime="${1:-}"
gabo_hook_filter "$runtime" || exit 0

out="$(cd "$gabo_root" && gabo check 2>&1)"
code=$?
case "$runtime:$code" in
  *:0) exit 0 ;;
  codex:1)
    printf '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":%s}}\n' "$(gabo_json_string "$out")"
    exit 0
    ;;
  codex:*) exit 0 ;;
  *:1)
    printf '%s\n' "$out" >&2
    exit 2
    ;;
  *)
    echo "gabo plugin warning: gabo check could not run (exit $code); post-write check skipped" >&2
    exit 0
    ;;
esac
