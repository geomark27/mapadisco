#!/usr/bin/env bash
# Verificaciones de solo lectura antes del commit de una actividad (make check).
# Se ejecuta desde la raíz del repo. Código 0 = listo para commit.
set -u

ok()   { printf '  \033[32m✓\033[0m %s\n' "$1"; }
warn() { printf '  \033[33m⚠\033[0m %s\n' "$1"; warnings=$((warnings + 1)); }
fail() { printf '  \033[31m✗\033[0m %s\n' "$1"; failures=$((failures + 1)); }
section() { printf '\n\033[1m%s\033[0m\n' "$1"; }

warnings=0
failures=0

if [ ! -f go.mod ] || ! grep -q '^module mapadisco' go.mod; then
	echo "Ejecuta el script desde la raíz de mapadisco." >&2
	exit 2
fi

base=develop
git rev-parse -q --verify origin/develop >/dev/null && base=origin/develop

section "1. Rama"
branch=$(git branch --show-current)
case "$branch" in
	main | develop | "") fail "estás en '${branch:-HEAD separado}': las actividades van en feat/<tema>, fix/<tema>, docs/<tema>…" ;;
	feat/* | fix/* | docs/* | chore/* | refactor/* | perf/* | test/*) ok "rama $branch" ;;
	*) warn "rama '$branch' sin prefijo de devlog (feat/, fix/, docs/, chore/, refactor/, perf/, test/)" ;;
esac

upstream=$(git rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null || true)
case "$upstream" in
	"") ok "sin upstream (se fija con el primer make push)" ;;
	origin/main | origin/develop) fail "la rama sigue a $upstream: un git push iría ahí. Corre: git branch --unset-upstream" ;;
	*) ok "upstream $upstream" ;;
esac

if git merge-base --is-ancestor "$base" HEAD; then
	ok "nace de $base actualizada"
else
	warn "$base tiene commits que esta rama no incluye: actualiza la rama antes del PR"
fi

commits=$(git rev-list --count "$base"..HEAD)
case "$commits" in
	0) ok "sin commits todavía" ;;
	1) ok "1 commit sobre $base" ;;
	*) warn "$commits commits sobre $base: la regla es uno por actividad" ;;
esac

if [ -n "$(git status --porcelain)" ]; then
	ok "cambios sin commitear: $(git status --porcelain | wc -l | tr -d ' ') archivo(s)"
fi
[ -n "$(git config user.email)" ] || fail "git no tiene user.email configurado"

section "2. Go"
gofiles=$(git ls-files -co --exclude-standard '*.go')
if out=$(go build ./... 2>&1 && GOOS=windows go build ./... 2>&1); then ok "go build (Linux y Windows)"; else fail "go build:"; echo "$out" | sed 's/^/      /'; fi
if out=$(go vet ./... 2>&1 && GOOS=windows go vet ./... 2>&1); then ok "go vet (Linux y Windows)"; else fail "go vet:"; echo "$out" | sed 's/^/      /'; fi

unformatted=$(gofmt -l $gofiles)
if [ -z "$unformatted" ]; then ok "gofmt"; else fail "gofmt: $(echo "$unformatted" | tr '\n' ' ')"; fi

if out=$(go test ./... 2>&1); then
	ok "go test ($(echo "$out" | grep -c '^ok') paquetes)"
else
	fail "go test:"
	echo "$out" | grep -v '^ok' | sed 's/^/      /'
fi

section "3. Datos sensibles"
files=$(git ls-files -co --exclude-standard | grep -v '^scripts/check\.sh$')
hits=$(echo "$files" | xargs grep -nIE \
	-e '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.(com|net|org|ec|io)\b' \
	-e '(ghp|gho|github_pat)_[A-Za-z0-9_]{20,}' \
	-e 'AKIA[0-9A-Z]{16}' \
	2>/dev/null | grep -vE '@example\.(com|org|test)|tu@correo\.com|noreply@anthropic\.com' || true)
if [ -z "$hits" ]; then ok "sin correos ni tokens"; else warn "revisar:"; echo "$hits" | sed 's/^/      /'; fi

terms=.claude/terminos-privados.txt
if [ -f "$terms" ]; then
	found=$(grep -vE '^\s*(#|$)' "$terms" | while IFS= read -r term; do
		echo "$files" | xargs grep -nIiF -- "$term" 2>/dev/null
	done)
	if [ -z "$found" ]; then ok "ningún término de $terms"; else fail "términos privados en el repo:"; echo "$found" | sed 's/^/      /'; fi
else
	printf '  · sin %s (opcional, ignorado por git)\n' "$terms"
fi

section "4. Bitácora (devlog)"
devlog=.claude/skills/devlog/scripts
logdir=$(sed -nE 's/^[[:space:]]*DEVLOG_DIR[[:space:]]*=[[:space:]]*"?([^"#[:space:]]+)"?.*/\1/p' .devlog.conf 2>/dev/null | tail -n1)
if [ ! -x "$devlog/verify.sh" ] || [ -z "$logdir" ]; then
	fail "falta devlog (.devlog.conf y $devlog/)"
else
	changed_log=$( (git diff --name-only "$base" -- "$logdir"; git ls-files -o --exclude-standard "$logdir") | sort -u)
	card=$(echo "$changed_log" | grep -E "^$logdir/releases/v[0-9]+\.[0-9]+\.[0-9]+\.md$" || true)
	if [[ "$branch" == chore/release-* ]]; then
		# Cierra una versión: lleva la ficha nueva en lugar de una entrada propia.
		if [ -n "$card" ]; then ok "cierre de versión: $card"; else fail "rama de cierre sin ficha nueva: corre make release-prep"; fi
		grep -qx 'version: unreleased' "$logdir"/entries/*.md 2>/dev/null &&
			warn "quedan entradas «unreleased»: ¿se fusionó algo después de make release-prep?"
	else
		if out=$("$devlog/verify.sh" 2>&1); then
			ok "$(echo "$out" | tail -n1 | sed 's/^✓ //')"
		else
			fail "la entrada de la rama no es válida (make bitacora title=\"…\" para crearla):"
			echo "$out" | sed 's/^/      /'
		fi
		if echo "$changed_log" | grep -q "^$logdir/entries/"; then
			ok "la entrada viaja en este cambio"
		else
			fail "la entrada de la rama no está entre los cambios"
		fi
		for f in $(git diff --name-only --diff-filter=M "$base" -- "$logdir/entries/"); do
			v=$(grep -m1 '^version:' "$f" || true)
			[ "$v" = "version: unreleased" ] || warn "$f ya se publicó ($v): no se reescribe, salvo erratas"
		done
	fi
fi

section "5. Guía de usuario"
changed=$( (git diff --name-only "$base"; git ls-files -o --exclude-standard) | sort -u)
visible=$(echo "$changed" | grep -E '^(cmd/|internal/report/|internal/hotspots/catalog\.go|internal/cleaner/catalog\.go|internal/installer/|scripts/install\.ps1)' | grep -v '_test\.go$' || true)
if [ -z "$visible" ]; then
	ok "sin cambios en lo que ve el usuario"
elif echo "$changed" | grep -qx 'docs/guia-de-usuario.md'; then
	ok "docs/guia-de-usuario.md actualizada"
else
	warn "cambian $(echo "$visible" | wc -l | tr -d ' ') archivo(s) que el usuario ve y no docs/guia-de-usuario.md: actualízala, o explica en la bitácora por qué no hace falta"
fi

section "Resultado"
if [ "$failures" -gt 0 ]; then
	printf '  \033[31m%d bloqueante(s)\033[0m, %d advertencia(s)\n' "$failures" "$warnings"
	exit 1
fi
printf '  \033[32mlisto para commit\033[0m (%d advertencia(s))\n' "$warnings"
