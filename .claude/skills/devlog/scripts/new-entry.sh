#!/usr/bin/env bash
# Create the log entry for the current branch, or return the existing one.
#
#   new-entry.sh ["Title"]
#
# Always prints the entry path on stdout (new or existing), so an agent can do
#   f=$(new-entry.sh "Title")  and then edit "$f".
# Idempotent and safe with several agents at once: creation is atomic
# (noclobber), so if two arrive together only one creates the file and the
# other reuses it.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

[ -d "$DEVLOG_DIR" ] || fail "$DEVLOG_DIR does not exist. Set it up with: $SKILL_DIR/scripts/init.sh"

BRANCH=$(current_branch)
[ -n "$BRANCH" ] || fail "Detached HEAD: create a branch first (git switch -c feat/…)."
is_protected "$BRANCH" && fail "You are on $BRANCH. Entries are written on the change's branch (feat/…, fix/…)."

EXISTING=$(entries_for_branch "$BRANCH" | head -n1)
if [ -n "$EXISTING" ]; then
  warn "Branch $BRANCH already has an entry: update it instead of creating another."
  echo "$EXISTING"
  exit 0
fi

TEMPLATE="$DEVLOG_DIR/entry-template.md"
[ -f "$TEMPLATE" ] || TEMPLATE="$TEMPLATES/$DEVLOG_LANG/entry.md"

TYPE=${BRANCH%%/*}
case "$TYPE" in feat|fix|chore|docs|refactor|perf|test) ;; *) TYPE=chore ;; esac
TOPIC=$(tr '[:upper:]' '[:lower:]' <<< "${BRANCH#*/}" | sed -E 's/[^a-z0-9]+/-/g; s/^-+|-+$//g')
DATE=$(date +%F)
AUTHOR=$(git config user.name 2>/dev/null || echo "unknown")
AGENTS=${DEVLOG_AGENT:-}
TITLE=${1:-$(sed 's/-/ /g' <<< "$TOPIC")}

mkdir -p "$ENTRIES"
TARGET="$ENTRIES/$DATE-$TYPE-$TOPIC.md"

render() {
  awk -v d="$DATE" -v b="$BRANCH" -v t="$TYPE" -v a="$AUTHOR" -v ag="$AGENTS" -v ti="$TITLE" '{
    gsub(/\{\{DATE\}\}/, d); gsub(/\{\{BRANCH\}\}/, b); gsub(/\{\{TYPE\}\}/, t)
    gsub(/\{\{AUTHOR\}\}/, a); gsub(/\{\{AGENTS\}\}/, ag); gsub(/\{\{TITLE\}\}/, ti)
    sub(/[ \t]+$/, ""); print
  }' "$TEMPLATE"
}

# set -C (noclobber): «>» fails if the file exists. That is the lock between agents.
if ( set -C; render > "$TARGET" ) 2>/dev/null; then
  ok "Entry created. Fill in the required sections (${L_REQUIRED[*]}) before opening the PR."
else
  warn "Another process created it at the same time: reusing it."
fi
echo "$TARGET"
