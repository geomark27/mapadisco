#!/usr/bin/env bash
# Cierra la bitácora de la próxima versión (devlog close-release.sh) en una rama
# chore/release-…: crea docs/bitacora/releases/vX.Y.Z.md y marca con esa versión las
# entradas «unreleased». No hace commits: el autor commitea «release: vX.Y.Z» y la rama
# llega a develop por PR, como cualquier actividad. Después: make promote y make release.
#   make release-prep bump=patch|minor|major
#   make release-prep ver=vX.Y.Z
set -euo pipefail
cd "$(dirname "$0")/.."
source scripts/version.sh

bump=${1:-}
ver=${2:-}
devlog=.claude/skills/devlog/scripts

branch=$(git branch --show-current)
case "$branch" in
chore/release-*) ;;
*) fail "el cierre de versión va en su propia rama: make start name=release-X-Y-Z type=chore (estás en '$branch')" ;;
esac
git fetch --quiet --tags origin 2>/dev/null || true

last=$(last_version)
ver=$(next_version "$last" "$bump" "$ver")
type=$(bump_type "$last" "$ver")
dir=$(devlog_dir)
[ ! -e "$dir/releases/$ver.md" ] || fail "$dir/releases/$ver.md ya existe: esa versión ya está cerrada"
ok "versión $last → $ver ($type)"

"$devlog/close-release.sh" "$ver" --dry-run
read -r -p "¿Cerrar la bitácora de $ver con esas entradas? [s/N]: " answer
[[ "$answer" =~ ^[sSyY] ]] || { echo "Cancelado: no se cambió nada."; exit 0; }

previous=()
[ "$last" = v0.0.0 ] || previous=(--previous "$last")
card=$("$devlog/close-release.sh" "$ver" --type "$type" "${previous[@]}")
ok "ficha creada: $card"
echo
echo "Siguiente:"
echo "  make check"
echo "  git add $dir && git commit -m \"release: $ver\""
echo "  make ship                 # PR a develop"
echo "  make promote && make release ver=$ver"
