# CLAUDE.md

Guía para Claude Code al trabajar en este repositorio.

## Qué es mapadisco

Herramienta de terminal en Go que muestra en qué se va el espacio de un disco en Windows: árbol de carpetas pesadas,
"sospechosos habituales", archivos más grandes, peso por tipo y exportación a CSV. El análisis es de **solo lectura**;
`-clean` libera espacio borrando solo las categorías del catálogo de `internal/cleaner`, con confirmación. Se compila para Windows; el desarrollo se hace desde Linux/WSL.

- Uso y alcance para el usuario: `docs/guia-de-usuario.md`.
- Estructura de paquetes y cómo extender: `docs/arquitectura.md`.
- Ramas, actividades y PR: `docs/flujo-de-trabajo.md`.
- Versionado y release: `docs/release.md`.
- Historia y porqué de cada cambio: `docs/bitacora/`.

## Comandos

`make` lista todo; las opciones van como variables.

```bash
make start name=<tema> [type=feat|fix|docs|chore|refactor|perf|test]   # nueva actividad desde develop al día
make bitacora title="…"   # crea (o muestra) la entrada de bitácora de la rama (devlog)
make test [pkg=./internal/report] [run=Tree] [v=1]
make lint                 # gofmt + vet (Linux y Windows)
make run args="-depth 4"  # compila el .exe y lo ejecuta en Windows desde WSL
make package              # el .exe del release en dist/ con versión local
make check                # verificaciones antes del commit (scripts/check.sh)
make push | pr | merge | finish | ship          # los corre el autor
make promote              # develop → main con merge commit (autor)
make release-prep bump=…  # cierra la bitácora de la versión, en una rama chore/release-X-Y-Z
make release ver=vX.Y.Z   # tag + GitHub Release desde main; exige la ficha de la versión (autor)
```

## Flujo de trabajo (regla)

```
main ── develop ── feat/<tema> | fix/<tema> | docs/<tema> | chore/<tema> …   (PR → develop, nunca a main)
```

- Toda actividad nace de `develop` actualizada con `make start` y llega a `develop` por PR, en **un solo commit**.
  Nunca se trabaja en `main` ni en `develop`.
- `develop → main` solo con `make promote`, siempre con *merge commit* (nunca *squash* ni *rebase*).
- **Claude no hace commits, push, PR, merge ni release.** Al terminar cada actividad entrega:
  1. un resumen breve de lo hecho y lo que quedó sin probar;
  2. el resultado de `make check`;
  3. el comando de commit listo para ejecutar (`git add <archivos>` explícitos + `git commit` con título y viñetas
     en español), y los siguientes pasos (`make push`, `make pr`… o `make ship`).
- **Bitácora obligatoria con devlog** (`.devlog.conf`, skill en `.claude/skills/devlog/`): toda rama lleva **una**
  entrada en `docs/bitacora/entries/` (`make bitacora title="…"`), en español, en el mismo commit que el cambio.
  Sigue la skill `devlog` antes de dar por terminada una actividad: un cambio sin entrada no está terminado. Con
  varios agentes, solo el orquestador escribe la entrada; los subagentes devuelven `DEVLOG CONTRIBUTION`.
  `version:` solo la cambia `make release-prep`; las entradas publicadas no se reescriben.
- **`docs/guia-de-usuario.md` se actualiza en el mismo commit** que cambia lo que el usuario ve: una opción, una
  sección del reporte, un sospechoso, un mensaje, o la instalación. `make check` avisa si falta.
- La versión sale del tag de git (`-X main.version`); no hay archivo `VERSION` ni versiones escritas en el código ni en
  la guía. El `bump` lo decide el autor (criterio en `docs/release.md`).

## Reglas del código

- Solo `internal/winapi` llama a Win32 / `syscall`. El código exclusivo de Windows va en archivos `_windows.go`.
- `format`, el modelo de `scanner` (`node.go`), el catálogo de `hotspots`, `report`, la lógica de rutas de
  `installer` y todo `cleaner` son portables y llevan pruebas que corren en Linux. `report` escribe en un `io.Writer`, nunca directo en la consola.
- El análisis no escribe en el disco. Lo único que borra es `-clean`, y solo lo de `cleaner.Auto` (cachés y temporales
  que se regeneran, Papelera, volcados), después de mostrarlo y preguntar (o con `-yes`). Nunca sigue enlaces ni
  junctions, nunca borra la carpeta raíz de una categoría ni rutas relativas o raíces de unidad (hay pruebas que lo
  exigen). Lo demás va a `cleaner.Manual` como guía: no se borra. Además escribe su propia instalación y el CSV.
- Al ampliar `cleaner.Auto`: solo cosas que se regeneran solas o que el usuario ya descartó; nunca datos del usuario
  ni del sistema que tengan herramienta oficial (Windows Update, WinSxS): esos van a `cleaner.Manual`.
- Los cambios al PATH del usuario conservan el tipo del valor (`REG_EXPAND_SZ`) y el resto de entradas tal cual:
  instalar y desinstalar deja el PATH idéntico (hay una prueba que lo exige).
- Datos de prueba y ejemplos de la documentación: ficticios (`C:\Users\Usuario`), nunca rutas, nombres ni datos de un
  equipo real. Sin correos, tokens ni secretos en el repo (`make check` los busca).
- Dependencias: solo `golang.org/x/sys` (registro de Windows, en `internal/winapi`). Cualquier otra se justifica antes.
- Comentarios y mensajes en español.

## Pruebas

- `go test ./...` corre en Linux/WSL y no toca el disco real.
- El escaneo real se verifica a mano con `make run` (ejecuta el `.exe` en Windows desde WSL) y se anota en la bitácora.
