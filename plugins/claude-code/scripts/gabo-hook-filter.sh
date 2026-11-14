# gabo-hook-filter.sh: the S7 filters and the lock relay shared by the gabo
# plugin's hooks (intent 0001 S7, D9; intent 0002 S8, D5; intent 0028 S3, S10,
# S15; plan T10, T11). Sourced by gabo-check.sh (PostToolUse), gabo-pre-write.sh
# (PreToolUse Write|Edit, and apply_patch in Codex) and gabo-pre-command.sh
# (PreToolUse Bash); it holds no check rule, only the cases in which a hook
# stays silent without calling gabo, the mapping of gabo's exit and the answer
# each runtime reads. The Codex plugin ships a byte-identical copy.
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

# gabo_hook_judge_input reads the hook JSON from stdin into gabo_input and the
# project root into gabo_root for the lock relay (intent 0028 S3). Unlike
# gabo_hook_filter it names no path and no command: gabo judges every call. It
# returns 1 (stay silent, exit 0) when the checks option is off, the project has
# no .gabo/VERSION or the input is empty. With gabo not on PATH it prints one
# warning line and returns 1, so the call runs (S15).
gabo_hook_judge_input() {
  gabo_input=""
  [[ "${CLAUDE_PLUGIN_OPTION_CHECKS:-true}" == false ]] && return 1

  gabo_root="${CLAUDE_PROJECT_DIR:-$PWD}"
  gabo_root="${gabo_root%/}"
  [[ -f "$gabo_root/.gabo/VERSION" ]] || return 1
  if [[ -z "$(type -P gabo)" ]]; then
    echo "gabo plugin warning: gabo is not on PATH; this call runs without gabo's checks" >&2
    return 1
  fi

  IFS= read -r -d '' gabo_input || true
  [[ -n "$gabo_input" ]]
}

# gabo_hook_judge_relay <kind> <engine> <label> pipes gabo_input unchanged to
# gabo check --pre-<kind> in gabo_root with GABO_ENGINE=<engine>, so the call is
# judged by that engine's session rules (intent 0028 S3). GABO_TASK and
# GABO_OUTCOME_FILE pass through when the cycle sets them. The exit maps (S10,
# S15): 0 runs the call (returns 0); 1 blocks it with gabo's lines; 3 is an ask
# that <label> cannot show (probe item (d) is no for claude and codex), so it
# blocks as 1 and says so; any other exit, a usage error or a signal, blocks and
# names the failure. A block returns 2 and leaves its text in gabo_block_text.
gabo_hook_judge_relay() {
  local kind="$1" engine="$2" label="$3" out code note nl=$'\n'
  out="$(cd "$gabo_root" && printf '%s' "$gabo_input" | GABO_ENGINE="$engine" gabo check "--pre-$kind" 2>&1)"
  code=$?
  case "$code" in
    0) return 0 ;;
    1)
      note="gabo plugin: gabo check --pre-$kind blocked the call without a message"
      gabo_block_text="${out:-$note}"
      ;;
    3)
      gabo_block_text="gabo plugin: $label shows no prompt for an ask, so the call is blocked (intent 0028 S12)"
      [[ -n "$out" ]] && gabo_block_text="$out$nl$gabo_block_text"
      ;;
    *)
      gabo_block_text="gabo plugin: gabo check --pre-$kind failed (exit $code); the call is blocked"
      [[ -n "$out" ]] && gabo_block_text="$gabo_block_text$nl$out"
      ;;
  esac
  return 2
}

# gabo_hook_run <kind> [runtime] judges the hook's tool call with gabo (intent
# 0028 S3, S15) and writes the answer in the runtime's format. A call gabo
# allows, and every silent case, returns 0 with nothing on stdout. Without a
# runtime (Claude Code) a block prints gabo's text on stderr and returns 2.
# With `codex` a block is Codex's PreToolUse deny output on stdout, with gabo's
# text as its reason, and returns 0, since Codex denies the call from that JSON.
gabo_hook_run() {
  local kind="$1" runtime="${2:-}" engine=claude label="Claude Code"
  if [[ "$runtime" == codex ]]; then
    engine=codex
    label=Codex
  fi
  gabo_hook_judge_input || return 0
  gabo_hook_judge_relay "$kind" "$engine" "$label" && return 0
  if [[ "$runtime" != codex ]]; then
    printf '%s\n' "$gabo_block_text" >&2
    return 2
  fi
  printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":%s}}\n' "$(gabo_json_string "$gabo_block_text")"
  return 0
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
