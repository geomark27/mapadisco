# CLAUDE.md

Guía para Claude Code al trabajar en este repositorio.

## Qué es mapadisco

Herramienta de terminal en Go que muestra en qué se va el espacio de un disco en Windows: árbol de carpetas pesadas,
"sospechosos habituales", archivos más grandes, peso por tipo y exportación a CSV. **Solo lectura**: nunca borra ni
modifica nada del disco que analiza. Se compila para Windows; el desarrollo se hace desde Linux/WSL.

- Uso y alcance para el usuario: `docs/guia-de-usuario.md`.
- Estructura de paquetes y cómo extender: `docs/arquitectura.md`.
- Ramas, actividades y PR: `docs/flujo-de-trabajo.md`.
- Versionado y release: `docs/release.md`.
- Historia y porqué de cada cambio: `docs/bitacora/`.

## Comandos

`make` lista todo; las opciones van como variables.

```bash
make start name=<tema> [type=feature|fix|docs]   # nueva actividad desde develop al día
make test [pkg=./internal/report] [run=Tree] [v=1]
make lint                 # gofmt + vet (Linux y Windows)
make run args="-depth 4"  # compila el .exe y lo ejecuta en Windows desde WSL
make package              # zip instalable en dist/ con versión local
make check                # verificaciones antes del commit (scripts/check.sh)
make push | pr | merge | finish | ship          # los corre el autor
make promote              # develop → main con merge commit (autor)
make release bump=patch|minor|major | ver=vX.Y.Z  # tag + GitHub Release desde main (autor)
```

## Flujo de trabajo (regla)

```
main ── develop ── feature/<tema> | fix/<tema> | docs/<tema>   (PR → develop, nunca a main)
```

- Toda actividad nace de `develop` actualizada con `make start` y llega a `develop` por PR, en **un solo commit**.
  Nunca se trabaja en `main` ni en `develop`.
- `develop → main` solo con `make promote`, siempre con *merge commit* (nunca *squash* ni *rebase*).
- **Claude no hace commits, push, PR, merge ni release.** Al terminar cada actividad entrega:
  1. un resumen breve de lo hecho y lo que quedó sin probar;
  2. el resultado de `make check`;
  3. el comando de commit listo para ejecutar (`git add <archivos>` explícitos + `git commit` con título y viñetas
     en español), y los siguientes pasos (`make push`, `make pr`… o `make ship`).
- Toda actividad lleva su entrada **`docs/bitacora/NNN-<tema>.md`** (NNN global correlativo, `<tema>` = el de la rama)
  y su fila en `docs/bitacora/README.md`. Las entradas de actividades ya fusionadas no se reescriben.
- **`docs/guia-de-usuario.md` se actualiza en el mismo commit** que cambia lo que el usuario ve: una opción, una
  sección del reporte, un sospechoso, un mensaje, la instalación o el zip. `make check` avisa si falta.
- La versión sale del tag de git (`-X main.version`); no hay archivo `VERSION` ni versiones escritas en el código ni en
  la guía. El `bump` lo decide el autor (criterio en `docs/release.md`).

## Reglas del código

- Solo `internal/winapi` llama a Win32 / `syscall`. El código exclusivo de Windows va en archivos `_windows.go`.
- `format`, el modelo de `scanner` (`node.go`), el catálogo de `hotspots` y `report` son portables y llevan pruebas
  que corren en Linux. `report` escribe en un `io.Writer`, nunca directo en la consola.
- El programa no escribe en el disco analizado salvo el CSV que pide el usuario con `-csv`.
- Datos de prueba y ejemplos de la documentación: ficticios (`C:\Users\Usuario`), nunca rutas, nombres ni datos de un
  equipo real. Sin correos, tokens ni secretos en el repo (`make check` los busca).
- Sin dependencias externas por ahora (solo la biblioteca estándar); cualquier dependencia se justifica antes.
- Comentarios y mensajes en español.

## Pruebas

- `go test ./...` corre en Linux/WSL y no toca el disco real.
- El escaneo real se verifica a mano con `make run` (ejecuta el `.exe` en Windows desde WSL) y se anota en la bitácora.
