# gabo-hook-filter.sh: the S7 filters shared by the gabo plugin's hooks
# (intent 0001 S7, D9; intent 0002 S8, D5). Sourced by gabo-check.sh
# (PostToolUse) and gabo-pre-write.sh (PreToolUse); it holds no check rule,
# only the cases in which a hook stays silent without calling gabo. The Codex
# plugin ships a byte-identical copy.
#
# gabo_hook_filter [codex] reads the hook JSON from stdin into gabo_input and
# the project root into gabo_root. It returns 1 (stay silent, exit 0) when the
# checks option is off, the project has no .gabo/VERSION, gabo is not on
# PATH, or no written file is a gabo path; otherwise 0. Without an argument
# the written file is Claude Code's tool_input.file_path; with `codex` it is
# every file header (Add, Update, Delete File, Move to) of the apply_patch
# text in tool_input.command.
#
# Uses only bash builtins, so it needs no jq.

gabo_hook_filter() {
  gabo_input=""
  [[ "${CLAUDE_PLUGIN_OPTION_CHECKS:-true}" == false ]] && return 1

  gabo_root="${CLAUDE_PROJECT_DIR:-$PWD}"
  gabo_root="${gabo_root%/}"
  [[ -f "$gabo_root/.gabo/VERSION" ]] || return 1
  [[ -n "$(type -P gabo)" ]] || return 1

  IFS= read -r -d '' gabo_input || true
  if [[ "${1:-}" == codex ]]; then
    gabo_hook_codex_paths
    return
  fi

  # tool_input.file_path from the hook JSON. JSON escapes every quote inside
  # a string value, so the first unescaped "file_path" key is the tool
  # input's.
  local re='"file_path"[[:space:]]*:[[:space:]]*"(([^"\\]|\\.)*)"'
  [[ "$gabo_input" =~ $re ]] || return 1
  local path="${BASH_REMATCH[1]}"
  path="${path//\\\//\/}"
  path="${path//\\\\/\\}"
  gabo_hook_is_gabo_path "$path"
}

# gabo_hook_codex_paths returns 0 when a file header of the Codex apply_patch
# in gabo_input's tool_input.command names a gabo path, else 1.
gabo_hook_codex_paths() {
  local re='"tool_input"[[:space:]]*:[[:space:]]*\{[[:space:]]*"command"[[:space:]]*:[[:space:]]*"'
  [[ "$gabo_input" =~ $re ]] || return 1
  local rest="${gabo_input#*"${BASH_REMATCH[0]}"}"

  # Decode the JSON string up to its closing quote. Escaped backslashes and
  # quotes become control bytes first (JSON escapes every control byte, so
  # none occurs raw), which leaves the first quote as the closing one.
  [[ "$rest" == *\"* ]] || return 1
  rest="${rest//\\\\/$'\x01'}"
  rest="${rest//\\\"/$'\x02'}"
  local patch="${rest%%\"*}"
  patch="${patch//\\n/$'\n'}"
  patch="${patch//\\r/$'\r'}"
  patch="${patch//\\t/$'\t'}"
  patch="${patch//\\\//\/}"
  patch="${patch//$'\x02'/\"}"
  patch="${patch//$'\x01'/\\}"

  local line path
  while IFS= read -r line; do
    line="${line%$'\r'}"
    case "$line" in
      '*** Add File: '* | '*** Update File: '* | '*** Delete File: '*) path="${line#*File: }" ;;
      '*** Move to: '*) path="${line#*to: }" ;;
      *) continue ;;
    esac
    gabo_hook_is_gabo_path "$path" && return 0
  done <<<"$patch"
  return 1
}

# gabo_hook_is_gabo_path <path> returns 0 when <path>, absolute inside
# gabo_root or relative to it, is a gabo path, else 1.
gabo_hook_is_gabo_path() {
  local path="$1"
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

# gabo_json_string <text> prints <text> as a JSON string literal.
gabo_json_string() {
  local s="$1" out="" c i
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  # Any other control character as \u00XX.
  if [[ "$s" == *[$'\x01'-$'\x1f']* ]]; then
    for ((i = 0; i < ${#s}; i++)); do
      c="${s:i:1}"
      case "$c" in
        [$'\x01'-$'\x1f']) printf -v c '\\u%04x' "'$c" ;;
      esac
      out+="$c"
    done
    s="$out"
  fi
  printf '"%s"' "$s"
}
