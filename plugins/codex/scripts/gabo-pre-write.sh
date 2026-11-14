#!/usr/bin/env bash
# gabo-pre-write.sh: the gabo plugin's pre-write hook, PreToolUse Write|Edit
# in Claude Code and, run as `gabo-pre-write.sh codex`, PreToolUse
# apply_patch in Codex (intent 0001 S7, S12, D8; intent 0002 S8, S13, D5;
# intent 0028 S3, S15; plan T10). The Codex plugin ships a byte-identical copy.
#
# A thin layer: it pipes the tool-call JSON unchanged to `gabo check
# --pre-write` in the project root and maps the exit code. Every rule lives in
# gabo; the script writes nothing.
#
# Claude Code (no argument): every file path of the call reaches gabo, gabo
# paths or not (intent 0028 S3), and the exit maps as in gabo-hook-filter.sh:
#   checks option off, no .gabo/VERSION or empty input -> exit 0, silent
#   no gabo on PATH                                   -> exit 0, one warning line
#   gabo exit 0                                       -> exit 0, silent
#   gabo exit 1 or 3                                  -> exit 2, gabo's lines
#   gabo exit 2, a signal, or any other exit          -> exit 2, the failure named
#
# Codex (`codex` argument): the S7 path filter, then
#   checks option off, no .gabo/VERSION, no gabo on PATH, or no written file
#   is a gabo path                           -> exit 0, silent
#   gabo exit 0                              -> exit 0, silent
#   gabo exit 1                              -> Codex PreToolUse deny JSON
#                                               with its lines as
#                                               permissionDecisionReason,
#                                               exit 0 (Codex denies the patch)
#   any other gabo exit                      -> exit 0, silent

here="${BASH_SOURCE[0]%/*}"
[[ "$here" == "${BASH_SOURCE[0]}" ]] && here=.
source "$here/gabo-hook-filter.sh"
runtime="${1:-}"

if [[ "$runtime" != codex ]]; then
  gabo_hook_judge_input || exit 0
  gabo_hook_judge_relay write
  exit $?
fi

gabo_hook_filter codex || exit 0
out="$(cd "$gabo_root" && printf '%s' "$gabo_input" | gabo check --pre-write 2>&1)"
code=$?
case "$code" in
  0) exit 0 ;;
  1)
    printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":%s}}\n' "$(gabo_json_string "$out")"
    exit 0
    ;;
  *) exit 0 ;;
esac
