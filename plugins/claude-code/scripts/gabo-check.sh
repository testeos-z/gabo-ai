#!/usr/bin/env bash
# gabo-check.sh: the gabo plugin's PostToolUse Write|Edit hook
# (intent 0001 S6, S7, S8, D9, D10).
#
# A thin layer: it applies the S7 filters (gabo-hook-filter.sh, shared with
# the pre-write hook), runs `gabo check` in the project root and maps the
# exit code. Every check rule lives in gabo; the script writes nothing.
#
#   checks option off, no .gabo/VERSION, no gabo on PATH, or the written file
#   is not a gabo path                       -> exit 0, silent
#   gabo check exit 0                        -> exit 0, silent
#   gabo check exit 1                        -> its lines on stderr, exit 2
#   any other gabo exit (2: usage or error)  -> one warning line, exit 0

here="${BASH_SOURCE[0]%/*}"
[[ "$here" == "${BASH_SOURCE[0]}" ]] && here=.
source "$here/gabo-hook-filter.sh"
gabo_hook_filter || exit 0

out="$(cd "$gabo_root" && gabo check 2>&1)"
code=$?
case "$code" in
  0) exit 0 ;;
  1)
    printf '%s\n' "$out" >&2
    exit 2
    ;;
  *)
    echo "gabo plugin warning: gabo check could not run (exit $code); post-write check skipped" >&2
    exit 0
    ;;
esac
