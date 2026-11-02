#!/usr/bin/env bash
# Check that the current branch has a valid log entry.
#
#   verify.sh [--base <branch>]
#
# With --base it also requires the entry to be in the diff against that branch
# (what a PR check does). Exits 0 when everything is fine and 1 otherwise,
# listing every problem. Read-only: for reviewer agents and CI.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

BASE=""
while [ $# -gt 0 ]; do
  case "$1" in
    --base) BASE=${2:?Missing branch after --base}; shift 2 ;;
    *) fail "Unknown argument: $1" ;;
  esac
done

BRANCH=${DEVLOG_BRANCH:-$(current_branch)}
[ -n "$BRANCH" ] || fail "Detached HEAD: set the branch with DEVLOG_BRANCH=…"

PROBLEMS=()
mapfile -t FOUND < <(entries_for_branch "$BRANCH")

if [ "${#FOUND[@]}" -eq 0 ]; then
  PROBLEMS+=("Branch $BRANCH has no entry in $ENTRIES.")
elif [ "${#FOUND[@]}" -gt 1 ]; then
  PROBLEMS+=("Branch $BRANCH has ${#FOUND[@]} entries; there must be one: ${FOUND[*]}")
fi

# Lines with real content in a section: HTML comments and blank lines do not count.
content_of() {
  awk -v s="## $2" '
    $0 == s { inside = 1; next }
    inside && /^## / { exit }
    inside {
      gsub(/<!--.*-->/, "")
      if ($0 ~ /<!--/) { com = 1; next }
      if (com) { if ($0 ~ /-->/) com = 0; next }
      if ($0 ~ /[^[:space:]]/) n++
    }
    END { print n + 0 }
  ' "$1"
}

SECRETS='-----BEGIN [A-Z ]*PRIVATE KEY|gh[pousr]_[A-Za-z0-9]{20,}|sk-[A-Za-z0-9_-]{20,}|AKIA[0-9A-Z]{16}|xox[baprs]-|(password|passwd|secret|token|api[_-]?key)[[:space:]]*[:=][[:space:]]*[^[:space:]<]{6,}|rediss?://[^[:space:]]*@|api\.vercel\.com/v1/integrations/deploy'

for f in "${FOUND[@]}"; do
  for key in date branch type author version; do
    [ -n "$(field_of "$f" "$key")" ] || PROBLEMS+=("$f: missing «$key» in front matter.")
  done
  v=$(field_of "$f" version)
  [ "$v" = "unreleased" ] || PROBLEMS+=("$f: version is «$v»; on an unreleased branch it must be «unreleased».")
  t=$(field_of "$f" type)
  case "$t" in feat|fix|chore|docs|refactor|perf|test) ;; *) PROBLEMS+=("$f: invalid type «$t».") ;; esac
  for s in "${L_REQUIRED[@]}"; do
    grep -qx "## $s" "$f" || { PROBLEMS+=("$f: missing required section «## $s»."); continue; }
    [ "$(content_of "$f" "$s")" -gt 0 ] || PROBLEMS+=("$f: section «$s» is empty.")
  done
  if grep -q '{{' "$f"; then PROBLEMS+=("$f: template placeholders {{…}} left unfilled."); fi
  if grep -qEi -- "$SECRETS" "$f"; then PROBLEMS+=("$f: looks like it contains a secret or a private URL. Review it."); fi
done

if [ -n "$BASE" ]; then
  git rev-parse -q --verify "$BASE" >/dev/null || BASE="origin/$BASE"
  if ! git diff --name-only --diff-filter=AM "$BASE"...HEAD -- "$ENTRIES" | grep -q .; then
    PROBLEMS+=("No new or modified entry in $ENTRIES compared to $BASE.")
  fi
fi

if [ "${#PROBLEMS[@]}" -gt 0 ]; then
  echo "${R}Log for $BRANCH: ${#PROBLEMS[@]} problem(s)${N}" >&2
  printf '  - %s\n' "${PROBLEMS[@]}" >&2
  exit 1
fi
ok "Log for $BRANCH is valid: ${FOUND[0]}"
