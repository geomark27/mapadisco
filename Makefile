# Atajos de mapadisco. `make` (o `make help`) lista todo; las opciones van como
# variables: make start name=export-json, make test pkg=./internal/report run=Tree v=1.

SHELL := /bin/bash
.DEFAULT_GOAL := help

REMOTE ?= origin
BASE   ?= develop
type   ?= feat
pkg    ?= ./...
BRANCH := $(shell git branch --show-current 2>/dev/null)
# Plataformas del release: mapadisco solo funciona en Windows (por ahora x64).
PLATFORMS := windows/amd64
LDFLAGS = -s -w -X main.version=$(1)
GOFILES = $(shell git ls-files --cached --others --exclude-standard '*.go')
# Versión de las compilaciones locales: último tag + commits + hash (ej. v1.1.0-2-gabc1234-dirty).
DEV_VERSION = $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: help sync start push pr merge finish push-pr push-pr-merge merge-finish ship prune build run test vet fmt lint bitacora bitacora-check check ci package release-build release-prep promote release
DEVLOG := .claude/skills/devlog/scripts

##@ Git

sync: ## Trae origin y actualiza develop solo con fast-forward, estés en la rama que estés
	@git fetch $(REMOTE)
	@if [ "$(BRANCH)" = "$(BASE)" ]; then git pull --ff-only $(REMOTE) $(BASE); \
	else git fetch $(REMOTE) $(BASE):$(BASE) && echo "$(BASE) al día (sigues en $(BRANCH))"; fi

start: ## Nueva actividad desde develop al día: make start name=<tema> [type=feat|fix|docs|chore|refactor|perf|test]
	@[[ "$(name)" =~ ^[a-z0-9]+(-[a-z0-9]+)*$$ ]] || { echo "name=<tema> en minúsculas y con guiones (ej: name=export-json)"; exit 1; }
	@[[ "$(type)" =~ ^(feat|fix|docs|chore|refactor|perf|test)$$ ]] || { echo "type debe ser feat, fix, docs, chore, refactor, perf o test"; exit 1; }
	@[ -z "$$(git status --porcelain)" ] || { echo "hay cambios sin commitear: guárdalos o descártalos antes"; exit 1; }
	@$(MAKE) --no-print-directory sync
	git switch -c $(type)/$(name) $(BASE)
	@git branch --unset-upstream 2>/dev/null || true

push: ## Sube la rama actual y fija su upstream; force=1 reescribe con --force-with-lease
	@case "$(BRANCH)" in main|$(BASE)|"") echo "no se sube '$(BRANCH)' directo: trabaja en feat/<tema>"; exit 1;; esac
	git push -u $(REMOTE) $(BRANCH) $(if $(force),--force-with-lease,)

pr: ## Crea el PR de la rama actual hacia develop (gh), o muestra el que ya está abierto; sin gh, el enlace
	@case "$(BRANCH)" in main|$(BASE)|"") echo "no hay PR desde '$(BRANCH)'"; exit 1;; esac
	@if command -v gh >/dev/null && gh auth status >/dev/null 2>&1; then \
	open=$$(gh pr list --head $(BRANCH) --state open --json url --jq '.[0].url // empty'); \
	if [ -n "$$open" ]; then echo "✓ PR ya abierto: $$open"; \
	else gh pr create --base $(BASE) --head $(BRANCH) --fill $(if $(web),--web,); fi; \
	else url=$$(git remote get-url $(REMOTE) | sed -E 's#^git@([^:]+):#https://\1/#; s#\.git$$##'); \
	echo "$$url/compare/$(BASE)...$(BRANCH)?expand=1"; fi

merge: ## Fusiona en develop el PR de la rama actual, con merge commit; pregunta antes (yes=1 no pregunta)
	@MERGE_YES=$(yes) bash scripts/merge.sh

finish: ## Tras fusionar el PR: vuelve a develop, la actualiza y borra las ramas ya fusionadas
	git switch $(BASE)
	@$(MAKE) --no-print-directory sync prune

push-pr: ## push + pr
	@$(MAKE) --no-print-directory push
	@$(MAKE) --no-print-directory pr

push-pr-merge: ## push + pr + merge
	@$(MAKE) --no-print-directory push-pr
	@$(MAKE) --no-print-directory merge

merge-finish: ## merge + finish
	@$(MAKE) --no-print-directory merge
	@$(MAKE) --no-print-directory finish

ship: ## Todo: push + pr + merge + finish; se detiene en el primer paso que falle
	@$(MAKE) --no-print-directory push-pr-merge
	@$(MAKE) --no-print-directory finish

prune: ## Borra las ramas locales ya fusionadas en develop (ni main, ni develop, ni la actual, ni las recién creadas)
	@tip=$$(git rev-parse $(BASE)); \
	for b in $$(git branch --merged $(BASE) --format='%(refname:short)'); do \
		case "$$b" in main|$(BASE)|$(BRANCH)) continue;; esac; \
		[ "$$(git rev-parse "$$b")" = "$$tip" ] && continue; \
		git branch -d "$$b"; \
	done

##@ Go

build: ## Compila todo, para Linux (pruebas) y para Windows (el programa real)
	go build ./...
	GOOS=windows go build ./...

run: ## Compila el .exe y lo ejecuta en Windows desde WSL: make run args="-depth 4 C:\\Users"
	@mkdir -p dist/dev
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.version=$(DEV_VERSION)" -o dist/dev/mapadisco.exe ./cmd/mapadisco
	./dist/dev/mapadisco.exe -no-install $(args)

test: ## Pruebas: make test [pkg=./internal/report] [run=Tree] [v=1]
	go test $(if $(v),-v,) $(if $(run),-run '$(run)',) $(pkg)

vet: ## Análisis estático para Linux y para Windows
	go vet ./...
	GOOS=windows go vet ./...

fmt: ## Formatea el código Go
	gofmt -w $(GOFILES)

lint: ## gofmt sin diferencias + vet (Linux y Windows)
	@out=$$(gofmt -l $(GOFILES)); [ -z "$$out" ] || { echo "sin formatear (corre make fmt):"; echo "$$out"; exit 1; }
	@$(MAKE) --no-print-directory vet

##@ Actividad

bitacora: ## Crea (o muestra) la entrada de bitácora de la rama: make bitacora title="Título corto"
	@$(DEVLOG)/new-entry.sh "$(title)"

bitacora-check: ## Valida la entrada de bitácora de la rama (devlog verify.sh)
	@$(DEVLOG)/verify.sh

check: ## Verificaciones antes del commit: rama, Go, datos sensibles, bitácora y guía
	@bash scripts/check.sh

ci: lint test ## Lo que debe pasar en CI: lint + test

##@ Release

package: ## El .exe del release en dist/ con la versión local (git describe); ver=vX.Y.Z para fijarla
	@rm -rf dist && mkdir dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; out=dist/mapadisco-$$os-$$arch.exe; \
		echo "  $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(call LDFLAGS,$(or $(ver),$(DEV_VERSION)))" -o $$out ./cmd/mapadisco || exit 1; \
	done

release-build: ## Ejecutable(s) del release y checksums.txt en dist/: make release-build ver=v0.2.0
	@[[ "$(ver)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$$ ]] || { echo "ver=vMAJOR.MINOR.PATCH (ej: ver=v0.2.0)"; exit 1; }
	@$(MAKE) --no-print-directory package ver=$(ver)
	@cd dist && { if command -v sha256sum >/dev/null; then sha256sum mapadisco-*.exe; else shasum -a 256 mapadisco-*.exe; fi; } > checksums.txt
	@echo "  dist/checksums.txt"

release-prep: ## Cierra la bitácora de la versión en una rama chore/release-…: make release-prep bump=minor | ver=vX.Y.Z
	@bash scripts/release-prep.sh "$(bump)" "$(ver)"

promote: ## Lleva develop a main (lo corre el autor): PR con merge commit, fusión confirmada, main y develop al día
	@bash scripts/promote.sh

release: ## Publica un release desde main (lo corre el autor; exige la bitácora cerrada): make release ver=vX.Y.Z
	@bash scripts/release.sh "$(bump)" "$(ver)"

help: ## Muestra esta ayuda
	@awk 'BEGIN { FS = ":.*## " } \
	/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } \
	/^[a-z-]+:.*## / { printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo
	@echo "Opciones: name= type=feat|fix|docs|chore|…  title=  pkg= run= v=1  force=1  web=1  yes=1  args=  ver=  bump="
