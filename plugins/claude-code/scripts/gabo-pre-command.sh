#!/usr/bin/env bash
# gabo-pre-command.sh: the gabo plugin's pre-command hook, PreToolUse Bash in
# Claude Code (intent 0028 S3, S10, S12, S15, S16; plan T10). A thin layer: it
# pipes every shell call, unchanged, to `gabo check --pre-command` in the
# project root and maps the exit (see gabo_hook_judge_relay in
# gabo-hook-filter.sh). Every rule lives in gabo; the script writes nothing.
#
#   checks option off, no .gabo/VERSION, empty input -> exit 0, silent
#   no gabo on PATH                                  -> exit 0, one warning line
#   gabo exit 0                                      -> exit 0, silent
#   gabo exit 1 or 3                                 -> exit 2, gabo's lines
#   gabo exit 2, a signal, or any other exit         -> exit 2, the failure named

here="${BASH_SOURCE[0]%/*}"
[[ "$here" == "${BASH_SOURCE[0]}" ]] && here=.
source "$here/gabo-hook-filter.sh"
gabo_hook_judge_input || exit 0
gabo_hook_judge_relay command
exit $?
