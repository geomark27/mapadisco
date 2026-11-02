# Flujo de trabajo

Cómo se desarrolla en mapadisco: de dónde nace cada cambio, cómo llega a `develop` y cuándo pasa a `main`.
El mismo esquema que draco-hq, con los mismos atajos de `make`.

## Ramas

```
main ─────────────────●─────────────────────●──────   solo releases (tags vX.Y.Z)
                     ╱ merge commit         ╱
develop ──●────●────●────────●────●────────●────────   integración
           ╲  ╱ ╲  ╱          ╲  ╱
            ●    ●             ●        feature/<tema>, fix/<tema>, docs/<tema>
```

| Rama | Para qué | Quién escribe |
|---|---|---|
| `main` | Lo publicado. Cada release es un tag sobre `main`. | Solo llega por `make promote` (PR desde `develop`). |
| `develop` | Integración: lo terminado y revisado, aún sin publicar. | Solo llega por PR desde una rama de actividad. |
| `feature/<tema>` | Algo nuevo: una opción, una sección, un sospechoso, un exportador. | Una actividad. |
| `fix/<tema>` | Una corrección. | Una actividad. |
| `docs/<tema>` | Solo documentación. | Una actividad. |

Reglas:

- **Nadie trabaja directo en `main` ni en `develop`.** `make push` se niega a subirlas.
- **Toda actividad nace de `develop` al día.** `make start` actualiza `develop` solo con *fast-forward*:
  si tu `develop` local se desvió de `origin/develop`, falla en vez de mezclar.
- **Una actividad = una rama = un commit = un PR hacia `develop`.**
- **Las fusiones siempre son con *merge commit*** (`gh pr merge --merge`), nunca *squash* ni *rebase and merge*.
  Esos copian los commits con otros hashes, `main` y `develop` dejan de compartir historia y el siguiente PR choca
  en cada archivo (le pasó a draco: su bitácora 026).
- `<tema>` va en minúsculas y con guiones (`export-json`, `cache-shaders-amd`) y es el mismo que el de la entrada de bitácora.

## Ciclo de una actividad

```
make start name=<tema> [type=feature|fix|docs]
      │
      ▼
 desarrollar + pruebas + guía + bitácora
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
make start name=export-json             # feature/export-json desde develop al día
make start name=progreso-hdd type=fix   # fix/progreso-hdd
```

Exige el árbol limpio, actualiza `develop`, crea la rama y le quita el upstream para que un `git push` suelto no termine en `develop`.

### 2. Desarrollar

```bash
make test                       # pruebas (portables, corren en Linux/WSL)
make test pkg=./internal/report run=Tree v=1
make lint                       # gofmt + vet para Linux y Windows
make run args="-depth 4 C:\\"   # compila el .exe y lo ejecuta en Windows desde WSL
make package                    # zip instalable en dist/ con versión local (git describe)
```

En el mismo commit van:

- **Pruebas** de lo que cambia, cuando la lógica es portable (`format`, `scanner/node.go`, `report`, catálogo de `hotspots`).
- **Guía de usuario** (`docs/guia-de-usuario.md`) si cambia algo que el usuario ve: una opción, una sección del reporte,
  un sospechoso, un mensaje, la instalación o el contenido del zip. `make check` avisa si falta.
- **Entrada de bitácora** (ver abajo).

### 3. Verificar

```bash
make check
```

Es de solo lectura y revisa:

| Sección | Qué revisa |
|---|---|
| Rama | Prefijo `feature/`/`fix/`/`docs/`, que no siga a `main`/`develop`, que nazca de `develop`, un solo commit. |
| Go | `build` y `vet` para Linux y Windows, `gofmt`, `go test`. |
| Datos sensibles | Correos y tokens en archivos del repo; y los términos de `.claude/terminos-privados.txt` si existe (no se versiona). |
| Bitácora | Que exista la entrada de la rama, esté en el índice y tenga sus 5 secciones. |
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

Cada actividad tiene **una** entrada en `docs/bitacora/NNN-<tema>.md` (NNN: número global correlativo) y su fila
en el índice `docs/bitacora/README.md`. Viaja en el mismo commit que el cambio. Las secciones obligatorias son:

1. **Solicitud** — lo que se pidió, en términos de quien lo pidió.
2. **Motivación** — tabla *problema → daño real*.
3. **Diseño técnico** — decisiones con su porqué y lo descartado. La historia va aquí, no en comentarios del código.
4. **Archivos** — una línea por archivo.
5. **Cómo verificar** — comandos que de verdad se corrieron.

Opcional: **6. Pendientes** — lo que quedó fuera o sin probar.

Las entradas de actividades ya fusionadas no se reescriben; solo admiten una nota aclaratoria con fecha.

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
