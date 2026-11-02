# Shared config and helpers for the devlog scripts. Sourced, never executed.
#
# Settings, in order of precedence: environment > .devlog.conf (repo root) > defaults.
#   DEVLOG_DIR        log folder                                (docs/devlog)
#   DEVLOG_LANG       language of the generated content         (en)  → templates/<lang>/
#   DEVLOG_PROTECTED  branches where entries are never created  ("main master develop")
#   DEVLOG_REQUIRED   required section headings, "A|B"         (from templates/<lang>/lang.sh)
#   DEVLOG_AGENT      who is writing, when it is an agent       (e.g. "claude:backend")
#
# .devlog.conf is parsed as KEY=value lines; it is never executed.

C=$'\033[0;36m'; G=$'\033[0;32m'; Y=$'\033[0;33m'; R=$'\033[0;31m'; N=$'\033[0m'
# No colors when stderr is not a terminal (agents, CI, pipes).
[ -t 2 ] || { C=""; G=""; Y=""; R=""; N=""; }

# Messages go to stderr so stdout only carries paths that callers can capture.
fail() { echo "${R}✗ $*${N}" >&2; exit 1; }
warn() { echo "${Y}! $*${N}" >&2; }
ok()   { echo "${G}✓ $*${N}" >&2; }

ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || fail "Not inside a git repository."
cd "$ROOT"

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SKILL_DIR=$(cd "$HERE/.." && pwd)
TEMPLATES="$SKILL_DIR/templates"

conf() {
  [ -f .devlog.conf ] || return 0
  sed -nE "s/^[[:space:]]*$1[[:space:]]*=[[:space:]]*\"?([^\"#]*[^\"#[:space:]])\"?[[:space:]]*(#.*)?$/\1/p" .devlog.conf | tail -n1
}

DEVLOG_DIR=${DEVLOG_DIR:-$(conf DEVLOG_DIR)};             DEVLOG_DIR=${DEVLOG_DIR:-docs/devlog}
DEVLOG_LANG=${DEVLOG_LANG:-$(conf DEVLOG_LANG)};          DEVLOG_LANG=${DEVLOG_LANG:-en}
DEVLOG_PROTECTED=${DEVLOG_PROTECTED:-$(conf DEVLOG_PROTECTED)}
DEVLOG_PROTECTED=${DEVLOG_PROTECTED:-"main master develop"}
DEVLOG_REQUIRED=${DEVLOG_REQUIRED:-$(conf DEVLOG_REQUIRED)}
ENTRIES="$DEVLOG_DIR/entries"
RELEASES="$DEVLOG_DIR/releases"

load_lang() {
  local f="$TEMPLATES/$DEVLOG_LANG/lang.sh"
  [ -f "$f" ] || fail "No templates for language '$DEVLOG_LANG'. Available: $(ls "$TEMPLATES" | tr '\n' ' ')"
  # shellcheck source=/dev/null
  source "$f"
  if [ -n "$DEVLOG_REQUIRED" ]; then IFS='|' read -r -a L_REQUIRED <<< "$DEVLOG_REQUIRED"; fi
}
load_lang

current_branch() { git branch --show-current; }

is_protected() {
  local p
  for p in $DEVLOG_PROTECTED; do [ "$1" = "$p" ] && return 0; done
  return 1
}

# Entries whose front matter says «branch: <name>». One per line.
entries_for_branch() { grep -lx "branch: $1" "$ENTRIES"/*.md 2>/dev/null || true; }

unreleased_entries() { grep -lx 'version: unreleased' "$ENTRIES"/*.md 2>/dev/null || true; }

title_of() { sed -n 's/^# //p' "$1" | head -n1; }

# Front matter value: field_of file key
field_of() {
  awk -v k="$2" 'NR==1 && $0=="---" {f=1; next} f && $0=="---" {exit}
    f && index($0, k":")==1 { v=substr($0, length(k)+2); sub(/^[ \t]+/, "", v); print v; exit }' "$1"
}

# In-place edit that behaves the same with GNU and BSD sed.
replace_in() { local tmp; tmp=$(mktemp); sed "$1" "$2" > "$tmp" && mv "$tmp" "$2"; }
