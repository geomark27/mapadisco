# Flujo de trabajo

Cómo se desarrolla en mapadisco: de dónde nace cada cambio, cómo llega a `develop` y cuándo pasa a `main`.
El mismo esquema que draco-hq, con los mismos atajos de `make`; la bitácora usa **devlog**.

## Ramas

```
main ─────────────────●─────────────────────●──────   solo releases (tags vX.Y.Z)
                     ╱ merge commit         ╱
develop ──●────●────●────────●────●────────●────────   integración
           ╲  ╱ ╲  ╱          ╲  ╱
            ●    ●             ●        feat/<tema>, fix/<tema>, docs/<tema>, chore/<tema>…
```

| Rama | Para qué | Quién escribe |
|---|---|---|
| `main` | Lo publicado. Cada release es un tag sobre `main`. | Solo llega por `make promote` (PR desde `develop`). |
| `develop` | Integración: lo terminado y revisado, aún sin publicar. | Solo llega por PR desde una rama de actividad. |
| `feat/<tema>` | Algo nuevo: una opción, una sección, un sospechoso, un exportador. | Una actividad. |
| `fix/<tema>` | Una corrección. | Una actividad. |
| `docs/<tema>` | Solo documentación. | Una actividad. |
| `chore/<tema>` | Mantenimiento sin cambio funcional; también el cierre de versión (`chore/release-X-Y-Z`). | Una actividad. |
| `refactor/`, `perf/`, `test/` | Reestructurar sin cambiar el comportamiento, rendimiento, solo pruebas. | Una actividad. |

Los prefijos son los tipos de devlog: la entrada de bitácora toma su `type` del prefijo de la rama.

Reglas:

- **Nadie trabaja directo en `main` ni en `develop`.** `make push` se niega a subirlas.
- **Toda actividad nace de `develop` al día.** `make start` actualiza `develop` solo con *fast-forward*:
  si tu `develop` local se desvió de `origin/develop`, falla en vez de mezclar.
- **Una actividad = una rama = un commit = un PR hacia `develop`.**
- **Las fusiones siempre son con *merge commit*** (`gh pr merge --merge`), nunca *squash* ni *rebase and merge*.
  Esos copian los commits con otros hashes, `main` y `develop` dejan de compartir historia y el siguiente PR choca
  en cada archivo (le pasó a draco: su bitácora 026).
- `<tema>` va en minúsculas y con guiones (`export-json`, `cache-shaders-amd`); la entrada de bitácora se llama
  `AAAA-MM-DD-<tipo>-<tema>.md`.

## Ciclo de una actividad

```
make start name=<tema> [type=feat|fix|docs|chore|…]
      │
      ▼
make bitacora title="…"  →  desarrollar + pruebas + guía + completar la entrada
      │
      ▼
make check ──✗──► corregir
      │ ✓
      ▼
 git add … && git commit          ← lo ejecuta el autor
      │
      ▼
make push ─► make pr ─► make merge ─► make finish     (o todo junto: make ship)
```

### 1. Empezar

```bash
make start name=export-json             # feat/export-json desde develop al día
make start name=progreso-hdd type=fix   # fix/progreso-hdd
make bitacora title="Exportar a JSON"   # crea la entrada de la rama (o muestra la existente)
```

Exige el árbol limpio, actualiza `develop`, crea la rama y le quita el upstream para que un `git push` suelto no termine en `develop`.

### 2. Desarrollar

```bash
make test                       # pruebas (portables, corren en Linux/WSL)
make test pkg=./internal/report run=Tree v=1
make lint                       # gofmt + vet para Linux y Windows
make run args="-depth 4 C:\\"   # compila el .exe y lo ejecuta en Windows desde WSL
make package                    # el .exe del release en dist/ con versión local (git describe)
```

En el mismo commit van:

- **Pruebas** de lo que cambia, cuando la lógica es portable (`format`, `scanner/node.go`, `report`, catálogo de `hotspots`).
- **Guía de usuario** (`docs/guia-de-usuario.md`) si cambia algo que el usuario ve: una opción, una sección del reporte,
  un sospechoso, un mensaje o la instalación. `make check` avisa si falta.
- **Entrada de bitácora** (ver abajo).

### 3. Verificar

```bash
make check
```

Es de solo lectura y revisa:

| Sección | Qué revisa |
|---|---|
| Rama | Prefijo de devlog (`feat/`, `fix/`, `docs/`, `chore/`…), que no siga a `main`/`develop`, que nazca de `develop`, un solo commit. |
| Go | `build` y `vet` para Linux y Windows, `gofmt`, `go test`. |
| Datos sensibles | Correos y tokens en archivos del repo; y los términos de `.claude/terminos-privados.txt` si existe (no se versiona). |
| Bitácora | `verify.sh` de devlog (una sola entrada, front matter, *Qué cambia* y *Por qué* con contenido, sin secretos) y que la entrada viaje en el cambio. En `chore/release-…`, la ficha de la versión. |
| Guía de usuario | Que se haya actualizado si cambió algo visible. |

`✓` en orden · `⚠` revisar · `✗` bloquea el commit.

### 4. Commit (lo hace el autor)

Un solo commit por actividad. Mensaje en español: título corto (sustantivo o imperativo) y cuerpo con viñetas.

```bash
git add <archivos de la actividad>     # revisar git status; no -A a ciegas
git commit -m "Título" -m "- punto 1
- punto 2"
```

Si ya hay un commit y **no** se ha subido, se corrige con `git commit --amend`. Si ya se subió:
`git commit --amend` + `make push force=1` (usa `--force-with-lease`).

### 5. Subir, PR y fusionar

| Comando | Qué hace |
|---|---|
| `make push` | Sube la rama y fija su upstream. |
| `make pr` | Crea el PR hacia `develop` con `gh` (o muestra el ya abierto). `web=1` lo abre en el navegador. |
| `make merge` | Fusiona el PR en `develop` con *merge commit*. Pregunta antes (`yes=1` no pregunta). |
| `make finish` | Vuelve a `develop`, la actualiza y borra las ramas locales ya fusionadas. |
| `make ship` | `push` + `pr` + `merge` + `finish`; se detiene en el primer paso que falle. |

## Bitácora

La bitácora usa la skill **devlog** (incluida en `.claude/skills/devlog/`, configurada en `.devlog.conf`; guía
completa en `.claude/skills/devlog/HOW-IT-WORKS.md`). Índice y reglas: `docs/bitacora/README.md`.

- **Una entrada por rama** en `docs/bitacora/entries/AAAA-MM-DD-<tipo>-<tema>.md`, creada con
  `make bitacora title="…"`. Si la rama ya tiene entrada, devuelve esa: nunca hay dos.
- Front matter generado (`date`, `branch`, `type`, `author`, `agents`, `pr`, `version: unreleased`). `version` solo
  lo cambia el cierre de versión; `pr` se completa con `"#N"` al abrir el PR.
- Secciones: **Qué cambia** y **Por qué** (obligatorias), *Archivos y áreas*, *Cómo probarlo*, *Notas y pendientes*.
  En este proyecto se escriben con detalle: dentro de *Qué cambia* van las decisiones técnicas y lo descartado; en
  *Por qué*, la solicitud y la tabla *problema → daño real*. La historia va aquí, no en comentarios del código.
- Viaja en el mismo commit que el cambio. `make bitacora-check` la valida (también `make check`).
- Las entradas de versiones publicadas no se reescriben (salvo erratas); si algo cambia, va en una entrada nueva.
- **Una ficha por versión** en `docs/bitacora/releases/vX.Y.Z.md`, generada por `make release-prep`
  (ver `docs/release.md`).

## Trabajo con Claude Code

Claude Code sigue este flujo (ver `CLAUDE.md`), con una diferencia: **no hace commits, push ni PR.** Al terminar una
actividad entrega un resumen de lo hecho, el resultado de `make check` y el comando de commit listo para ejecutar.

## Puesta en marcha (una sola vez)

`develop` debe existir en GitHub antes del primer `make start`:

```bash
git switch develop          # o: git branch develop main
git push -u origin develop
```

Opcional en GitHub (*Settings → General → Pull Requests*): dejar habilitado solo **Allow merge commits**, para que
nadie fusione con *squash* o *rebase* por error desde la web.
