# gabo-hook-filter.sh: the S7 filters shared by the gabo plugin's hooks
# (intent 0001 S7, D9). Sourced by gabo-check.sh (PostToolUse) and
# gabo-pre-write.sh (PreToolUse); it holds no check rule, only the cases in
# which a hook stays silent without calling gabo.
#
# gabo_hook_filter reads the hook JSON from stdin into gabo_input and the
# project root into gabo_root. It returns 1 (stay silent, exit 0) when the
# checks option is off, the project has no .gabo/VERSION, gabo is not on
# PATH, or the written file is not a gabo path; otherwise 0.
#
# Uses only bash builtins, so it needs no jq.

gabo_hook_filter() {
  gabo_input=""
  [[ "${CLAUDE_PLUGIN_OPTION_CHECKS:-true}" == false ]] && return 1

  gabo_root="${CLAUDE_PROJECT_DIR:-$PWD}"
  gabo_root="${gabo_root%/}"
  [[ -f "$gabo_root/.gabo/VERSION" ]] || return 1
  [[ -n "$(type -P gabo)" ]] || return 1

  # tool_input.file_path from the hook JSON. JSON escapes every quote inside
  # a string value, so the first unescaped "file_path" key is the tool
  # input's.
  IFS= read -r -d '' gabo_input || true
  local re='"file_path"[[:space:]]*:[[:space:]]*"(([^"\\]|\\.)*)"'
  [[ "$gabo_input" =~ $re ]] || return 1
  local path="${BASH_REMATCH[1]}"
  path="${path//\\\//\/}"
  path="${path//\\\\/\\}"

  case "$path" in
    "$gabo_root"/*) path="${path#"$gabo_root"/}" ;;
    /*) return 1 ;;
  esac
  path="${path#./}"

  case "$path" in
    gabo.yaml | .gabo/* | .claude/agents/* | .claude/skills/* | docs/product/*) return 0 ;;
    *) return 1 ;;
  esac
}
