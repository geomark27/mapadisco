#!/usr/bin/env bash
# Close the log for a version. Called by the project's release process BEFORE
# its release commit; this script makes no commits or tags.
#
#   close-release.sh <X.Y.Z> [--type patch|minor|major] [--previous vA.B.C]
#                    [--notes file.md] [--repo owner/name] [--dry-run]
#
# 1. Finds entries with «version: unreleased».
# 2. Writes releases/vX.Y.Z.md with links to them, the notes and a compare link.
# 3. Sets «version: vX.Y.Z» on those entries.
# 4. Adds the version row to the README index.
# --dry-run only lists the entries that would ship. Prints the release card path on stdout.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

VERSION=""; TYPE=""; PREVIOUS=""; NOTES_FILE=""; REPO=""; DRY=0
while [ $# -gt 0 ]; do
  case "$1" in
    --type) TYPE=${2:?}; shift 2 ;;
    --previous) PREVIOUS=${2:?}; shift 2 ;;
    --notes) NOTES_FILE=${2:?}; shift 2 ;;
    --repo) REPO=${2:?}; shift 2 ;;
    --dry-run) DRY=1; shift ;;
    -*) fail "Unknown option: $1" ;;
    *) VERSION=${1#v}; shift ;;
  esac
done
[[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+([-+].+)?$ ]] || fail "Give the version as X.Y.Z (got: «${VERSION:-nothing}»)."
[ -d "$DEVLOG_DIR" ] || fail "$DEVLOG_DIR does not exist."

# GitHub repo from the origin remote unless given. «|| true»: a repo may have no origin.
if [ -z "$REPO" ]; then
  REPO=$( (git remote get-url origin 2>/dev/null || true) | sed -nE 's#.*github\.com[:/]([^/]+/[^/]+)$#\1#p' | sed 's/\.git$//')
fi

mapfile -t PENDING < <(unreleased_entries)

if [ "$DRY" = 1 ]; then
  echo "Entries that would ship in v$VERSION (${#PENDING[@]}):" >&2
  for e in "${PENDING[@]}"; do echo "  · $(title_of "$e")  ($(basename "$e"))" >&2; done
  exit 0
fi

CARD="$RELEASES/v$VERSION.md"
[ -e "$CARD" ] && fail "$CARD already exists: that version is already closed."
[ "${#PENDING[@]}" -gt 0 ] || warn "No unreleased entries: the card will only carry the notes."

TODAY=$(date +%F)
mkdir -p "$RELEASES"
{
  printf -- '---\nversion: v%s\ndate: %s\ntype: %s\nprevious: %s\n---\n\n# v%s\n\n' \
    "$VERSION" "$TODAY" "${TYPE:-unspecified}" "${PREVIOUS:-none}" "$VERSION"
  printf '## %s\n\n' "$L_ENTRIES_HEADING"
  if [ "${#PENDING[@]}" -gt 0 ]; then
    for e in "${PENDING[@]}"; do
      printf -- '- [%s](../entries/%s)\n' "$(title_of "$e")" "$(basename "$e")"
    done
  else
    printf '%s\n' "$L_NO_ENTRIES"
  fi
  if [ -n "$NOTES_FILE" ]; then
    printf '\n## %s\n\n' "$L_NOTES_HEADING"
    cat "$NOTES_FILE"
  fi
  if [ -n "$PREVIOUS" ] && [ -n "$REPO" ]; then
    printf '\n**%s:** https://github.com/%s/compare/%s...v%s\n' "$L_COMPARE" "$REPO" "$PREVIOUS" "$VERSION"
  fi
} > "$CARD"

for e in "${PENDING[@]}"; do
  replace_in "s/^version: unreleased\$/version: v$VERSION/" "$e"
done

README="$DEVLOG_DIR/README.md"
if grep -q '<!-- releases:start -->' "$README" 2>/dev/null; then
  ROW="| [v$VERSION](releases/v$VERSION.md) | $TODAY | ${TYPE:-unspecified} | ${#PENDING[@]} |"
  tmp=$(mktemp)
  awk -v row="$ROW" '
    /<!-- releases:start -->/ { inside = 1 }
    { print }
    inside && /^\|---/ { print row; inside = 0 }
  ' "$README" > "$tmp" && mv "$tmp" "$README"
else
  warn "README has no <!-- releases:start/end --> markers: index not updated."
fi

ok "Log for v$VERSION closed (${#PENDING[@]} entries). Add $DEVLOG_DIR to the release commit."
echo "$CARD"
