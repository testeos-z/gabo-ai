#!/usr/bin/env bash
# gabo-check.sh: the gabo plugin's PostToolUse Write|Edit hook
# (intent 0001 S6, S7, S8, D9, D10).
#
# A thin layer: it filters, runs `gabo check` in the project root and maps the
# exit code. Every check rule lives in gabo; the script writes nothing.
#
#   checks option off, no .gabo/VERSION, no gabo on PATH, or the written file
#   is not a gabo path                       -> exit 0, silent
#   gabo check exit 0                        -> exit 0, silent
#   gabo check exit 1                        -> its lines on stderr, exit 2
#   any other gabo exit (2: usage or error)  -> one warning line, exit 0
#
# Uses only bash builtins and gabo, so it needs no jq.

[[ "${CLAUDE_PLUGIN_OPTION_CHECKS:-true}" == false ]] && exit 0

root="${CLAUDE_PROJECT_DIR:-$PWD}"
root="${root%/}"
[[ -f "$root/.gabo/VERSION" ]] || exit 0
[[ -n "$(type -P gabo)" ]] || exit 0

# tool_input.file_path from the hook JSON on stdin. JSON escapes every quote
# inside a string value, so the first unescaped "file_path" key is the
# tool input's.
input=""
IFS= read -r -d '' input || true
re='"file_path"[[:space:]]*:[[:space:]]*"(([^"\\]|\\.)*)"'
[[ "$input" =~ $re ]] || exit 0
path="${BASH_REMATCH[1]}"
path="${path//\\\//\/}"
path="${path//\\\\/\\}"

case "$path" in
  "$root"/*) path="${path#"$root"/}" ;;
  /*) exit 0 ;;
esac
path="${path#./}"

case "$path" in
  gabo.yaml | .gabo/* | .claude/agents/* | .claude/skills/* | docs/product/*) ;;
  *) exit 0 ;;
esac

out="$(cd "$root" && gabo check 2>&1)"
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
