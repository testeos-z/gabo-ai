#!/usr/bin/env bash
# gabo-pre-write.sh: the gabo plugin's PreToolUse Write|Edit hook
# (intent 0001 S7, S12, D8).
#
# A thin layer: it applies the S7 filters, pipes the tool-call JSON unchanged
# to `gabo check --pre-write` in the project root and maps the exit code.
# Every rule lives in gabo; the script writes nothing.
#
#   checks option off, no .gabo/VERSION, no gabo on PATH, or the target file
#   is not a gabo path                       -> exit 0, silent
#   gabo exit 0                              -> exit 0, silent
#   gabo exit 1                              -> its lines on stderr, exit 2
#                                               (Claude Code blocks the call)
#   any other gabo exit (2: usage or error)  -> one warning line, exit 0

here="${BASH_SOURCE[0]%/*}"
[[ "$here" == "${BASH_SOURCE[0]}" ]] && here=.
source "$here/gabo-hook-filter.sh"
gabo_hook_filter || exit 0

out="$(cd "$gabo_root" && printf '%s' "$gabo_input" | gabo check --pre-write 2>&1)"
code=$?
case "$code" in
  0) exit 0 ;;
  1)
    printf '%s\n' "$out" >&2
    exit 2
    ;;
  *)
    echo "gabo plugin warning: gabo check --pre-write could not run (exit $code); pre-write check skipped" >&2
    exit 0
    ;;
esac
