#!/usr/bin/env bash
# Set up the devlog in the current project. Never overwrites existing files.
#
#   init.sh [--lang en|es|…] [--dir docs/devlog] [--project "Name"] [--vendor]
#
# --lang     language of the log content (templates/<lang>/). Default: en.
# --dir      log folder. Default: docs/devlog.
# --project  name shown in the log README. Default: repo folder name.
# --vendor   also copy this skill into .claude/skills/devlog/ so teammates and
#            CI get the same skill and scripts without depending on ~/.claude.
#
# Writes .devlog.conf at the repo root with the chosen language and folder.
set -euo pipefail

VENDOR=0; PROJECT=""
while [ $# -gt 0 ]; do
  case "$1" in
    --lang) export DEVLOG_LANG=${2:?}; shift 2 ;;
    --dir) export DEVLOG_DIR=${2:?}; shift 2 ;;
    --project) PROJECT=${2:?}; shift 2 ;;
    --vendor) VENDOR=1; shift ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
PROJECT=${PROJECT:-$(basename "$ROOT")}

copy_new() { # copy_new <src> <dst>: only if dst does not exist
  if [ -e "$2" ]; then warn "$2 exists: left untouched."; else cp -R "$1" "$2"; ok "Created $2"; fi
}

if [ -e .devlog.conf ]; then
  warn ".devlog.conf exists: left untouched."
else
  printf '# devlog settings (parsed, never executed)\nDEVLOG_DIR=%s\nDEVLOG_LANG=%s\n' "$DEVLOG_DIR" "$DEVLOG_LANG" > .devlog.conf
  ok "Created .devlog.conf (dir: $DEVLOG_DIR, lang: $DEVLOG_LANG)"
fi

mkdir -p "$ENTRIES" "$RELEASES"
[ -e "$ENTRIES/.gitkeep" ] || : > "$ENTRIES/.gitkeep"
[ -e "$RELEASES/.gitkeep" ] || : > "$RELEASES/.gitkeep"

copy_new "$TEMPLATES/$DEVLOG_LANG/entry.md" "$DEVLOG_DIR/entry-template.md"

if [ -e "$DEVLOG_DIR/README.md" ]; then
  warn "$DEVLOG_DIR/README.md exists: left untouched."
else
  awk -v p="$PROJECT" -v d="$DEVLOG_DIR" '{ gsub(/\{\{PROJECT\}\}/, p); gsub(/\{\{DIR\}\}/, d); print }' \
    "$TEMPLATES/$DEVLOG_LANG/README.md" > "$DEVLOG_DIR/README.md"
  ok "Created $DEVLOG_DIR/README.md"
fi

if [ "$VENDOR" = 1 ]; then
  DEST=.claude/skills/devlog
  if [ "$(cd "$SKILL_DIR" && pwd -P)" = "$(mkdir -p "$DEST" && cd "$DEST" && pwd -P)" ]; then
    warn "Already running from $DEST: nothing to vendor."
  else
    mkdir -p "$DEST"
    for item in SKILL.md HOW-IT-WORKS.md scripts templates; do copy_new "$SKILL_DIR/$item" "$DEST/$item"; done
  fi
fi

echo "" >&2
echo "Next: add the devlog section to the project's CLAUDE.md" >&2
echo "(see «Adopting it in a project» in $SKILL_DIR/HOW-IT-WORKS.md)." >&2
