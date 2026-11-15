#!/usr/bin/env bash
# gabo-pre-write.sh: the gabo plugin's pre-write hook, PreToolUse Write|Edit
# in Claude Code and, run as `gabo-pre-write.sh codex`, PreToolUse
# apply_patch in Codex (intent 0001 S7, S12, D8; intent 0002 S8, S13, D5;
# intent 0028 S3, S7, S15; plan T10, T11). The Codex plugin ships a
# byte-identical copy.
#
# A thin layer: it pipes the tool-call JSON unchanged to `gabo check
# --pre-write` in the project root and writes gabo's answer in the runtime's
# format (gabo_hook_run in gabo-hook-filter.sh). Every rule lives in gabo; the
# script writes nothing. Every write reaches gabo, gabo paths or not (S3, S7).
#
#   checks option off, no .gabo/VERSION or empty input -> exit 0, silent
#   no gabo on PATH                                    -> exit 0, one warning line
#   gabo exit 0                                        -> exit 0, silent
#   gabo exit 1 or 3                                   -> Claude Code: exit 2, gabo's lines
#                                                         Codex: exit 0, deny JSON with them
#   gabo exit 2, a signal, or any other exit           -> the same block, the failure named

here="${BASH_SOURCE[0]%/*}"
[[ "$here" == "${BASH_SOURCE[0]}" ]] && here=.
source "$here/gabo-hook-filter.sh"
gabo_hook_run write "${1:-}"
exit $?
