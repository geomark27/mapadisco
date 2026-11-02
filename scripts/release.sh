#!/usr/bin/env bash
# Publica un release de mapadisco: tag en main y el ejecutable con
# checksums.txt en GitHub Releases. Lo corre el autor (necesita gh autenticado
# y permiso de push sobre origin).
# Antes, la bitácora de la versión debe estar cerrada en main (make release-prep).
#   make release bump=patch|minor|major
#   make release ver=vX.Y.Z
set -euo pipefail
cd "$(dirname "$0")/.."
source scripts/version.sh

bump=${1:-}
ver=${2:-}
remote=origin

command -v gh >/dev/null && gh auth status >/dev/null 2>&1 || fail "hace falta gh autenticado: gh auth login"
[ "$(git branch --show-current)" = main ] || fail "los releases salen de main: git switch main && git pull --ff-only"
[ -z "$(git status --porcelain)" ] || fail "hay cambios sin commitear"
git fetch --quiet --tags "$remote" main
[ "$(git rev-parse HEAD)" = "$(git rev-parse "$remote/main")" ] || fail "main no está igual a $remote/main: git pull --ff-only"
ok "main limpia e igual a $remote/main"

last=$(last_version)
ver=$(next_version "$last" "$bump" "$ver")
card="$(devlog_dir)/releases/$ver.md"
[ -f "$card" ] || fail "falta $card: cierra la bitácora antes (make start name=release-${ver//./-} type=chore, make release-prep ver=$ver, PR y make promote)"
ok "bitácora cerrada: $card"
! git ls-remote --exit-code --tags "$remote" "refs/tags/$ver" >/dev/null 2>&1 || fail "el tag $ver ya existe en $remote"
! git rev-parse -q --verify "refs/tags/$ver" >/dev/null || fail "el tag $ver ya existe en local: git tag -d $ver"
ok "versión $last → $ver"

make --no-print-directory lint test
make --no-print-directory release-build ver="$ver"

read -r -p "¿Publicar $ver en GitHub? [s/N]: " answer
[[ "$answer" =~ ^[sSyY] ]] || { echo "Cancelado: no se creó el tag."; exit 0; }

git tag -a "$ver" -m "mapadisco $ver"
git push "$remote" "$ver"
repo=$(git remote get-url "$remote" | sed -nE 's#.*github\.com[:/]([^/]+/[^/]+)$#\1#p' | sed 's/\.git$//')
notes="Bitácora de esta versión: https://github.com/$repo/blob/$ver/$card"
gh release create "$ver" dist/mapadisco-*.exe dist/checksums.txt --title "mapadisco $ver" --notes "$notes" --generate-notes --verify-tag ||
	fail "el tag $ver se subió pero el release no se creó. Reintenta: gh release create $ver dist/mapadisco-*.exe dist/checksums.txt --title \"mapadisco $ver\" --generate-notes --verify-tag"
ok "release $ver publicado: $(gh release view "$ver" --json url --jq .url)"
echo "Siguiente: git switch develop"
