---
date: 2026-11-01
branch: feature/flujo-de-trabajo
type: feat
author: Marcos Ramos Jacome
agents: claude
pr: "#1"
version: v0.1.0
---

# Flujo de trabajo y release

## Qué cambia

- **Mismo flujo y atajos que draco-hq.** `main ← develop ← feature|fix|docs/<tema>`, una actividad = un commit = un
  PR, fusiones siempre con *merge commit*. Los objetivos de git del Makefile (`sync`, `start`, `push`, `pr`, `merge`,
  `finish`, `prune` y los combinados hasta `ship`) son los de draco sin cambios, y `scripts/merge.sh` y
  `scripts/promote.sh` se copiaron tal cual: son genéricos. Así el autor usa los mismos comandos en ambos proyectos.
- **Objetivos de Go propios de mapadisco.** El programa es solo para Windows y se desarrolla en WSL, así que:
  - `build`, `vet` y `lint` corren para Linux **y** para `GOOS=windows`, porque la mayor parte del código está en
    archivos `_windows.go` y un `go vet` normal no los ve;
  - `run` compila el `.exe` y lo ejecuta en Windows por la interoperabilidad de WSL (draco usa `go run .`, que aquí
    solo mostraría el aviso de `main_other.go`). A diferencia de draco, mapadisco no tiene un instalador automático
    que se dispare al ejecutar un binario recién compilado, así que ejecutarlo es seguro;
  - no hay `golden` ni `fuzz`: mapadisco no tiene goldens ni un conversor que lo justifique.
- **La versión es el tag de git.** Se eliminó `VERSION` y `scripts/build.sh`. `scripts/package.sh <versión>` inyecta
  `-X main.version`; `make package` y `make run` usan `git describe --tags --always --dirty` y el release usa el tag.
  Igual que draco, con un formato `vX.Y.Z` (antes el `.exe` mostraba `1.1.0`; ahora `v1.1.0`).
- **Artefactos del release.** draco publica binarios sueltos de 5 plataformas; mapadisco publica **un zip por
  plataforma** (hoy solo `windows/amd64`) con el `.exe`, el instalador, el README y la guía, porque `instalar.cmd`
  necesita el `.exe` al lado. `checksums.txt` cubre los zips. `PLATFORMS` queda en el Makefile para sumar `windows/arm64`
  sin tocar los scripts; `package.sh` rechaza plataformas que no sean Windows.
- **`scripts/release.sh`** es el de draco con los artefactos de mapadisco: mismos guardias (solo desde `main` limpia e
  igual a `origin/main`, versión mayor que el último tag, tag inexistente), `make lint test` y `release-build` antes de
  preguntar, tag anotado y `gh release create --generate-notes --verify-tag`. No crea commits ni sube ramas.
- **Criterio de `bump`** (en `docs/release.md`): SemVer sobre lo que el usuario usa (opciones, salida, CSV, códigos de
  salida, instalación). draco dejó su regla pendiente; aquí se escribe una guía, pero el `bump` lo sigue decidiendo el
  autor en cada release.
- **`make check`** (`scripts/check.sh`), adaptado del `check.sh` de draco: rama, Go en ambas plataformas, correos y
  tokens, términos privados opcionales, entrada de bitácora y guía de usuario. Diferencias:
  - la bitácora es plana, `docs/bitacora/NNN-<tema>.md`, porque mapadisco no tiene roadmap por fases;
  - lo «visible para el usuario» es `cmd/`, `internal/report/`, el catálogo de `hotspots`, `installer/` y
    `package.sh`;
  - vive en `scripts/` y no en `.claude/skills/`, porque el proyecto no tiene skills propias.
- **Claude no commitea** (decisión del autor). Quedó escrito en `CLAUDE.md` y en `docs/flujo-de-trabajo.md`, junto
  con lo que entrega al final de cada actividad.
- **Guía de usuario:** instalación desde GitHub Releases, verificación con `Get-FileHash` contra `checksums.txt`, nuevo
  nombre del zip (`mapadisco-vX.Y.Z-windows-amd64.zip`, que se descomprime en una carpeta propia) y versión con `v`.
  Se quitó el número de versión fijo del encabezado: la guía no debe nombrar una versión que el tag puede contradecir.
- **Datos de prueba:** el usuario de Windows de las rutas de prueba pasó a `Usuario`.
- **Descartado:**
  - mantener `VERSION` y validar contra el tag: dos fuentes de verdad para el mismo dato;
  - skills de proyecto como las de draco: por ahora `CLAUDE.md` y `make check` alcanzan.

## Por qué

### Solicitud

El autor subió mapadisco a GitHub y pidió que, desde ahora, el trabajo no vaya directo a `main`: cada tarea nace de
`develop` en su propia rama, se integra a `develop` y de ahí pasa a release. Pidió tomar del Makefile de draco-hq lo
que aplica a mapadisco en dos puntos:

1. cómo se hace cada desarrollo en su rama y cómo llega a `develop` y luego al release;
2. cómo se hace el release y cómo se maneja el versionamiento.

También pidió que Claude no haga commits: al terminar cada desarrollo entrega un resumen, el comando de commit que
ejecutará el autor y lo necesario para seguir.

### Motivación

| Problema | Daño real |
|---|---|
| Todo iba a `main` en un solo commit, sin rama de integración | Un cambio a medias queda publicado; no hay dónde revisar antes de publicar |
| `scripts/build.sh` leía un archivo `VERSION` que no llegó al commit inicial | En el repo, `scripts/build.sh` falla en la primera línea útil: nadie puede compilar el paquete |
| La versión era un archivo editado a mano | El número del `.exe`, el del zip y el de la guía se desincronizan; nada ata una versión a un commit |
| Sin release en GitHub ni huella de los archivos | Quien descarga no tiene de dónde bajar una versión concreta ni cómo verificar que el zip está completo |
| Sin verificación antes del commit ni registro de decisiones | Se sube código sin formatear o sin probar para Windows, y el porqué de cada cambio se pierde |
| Las pruebas usaban el nombre real del usuario de Windows del autor en rutas | Dato personal en un repo público, contrario a la regla de datos ficticios |

## Archivos y áreas

- `Makefile` — nuevo: git (de draco), Go para Linux y Windows, `check`, `package`, `release-build`, `promote`, `release`.
- `scripts/merge.sh`, `scripts/promote.sh` — copiados de draco-hq sin cambios.
- `scripts/release.sh` — de draco-hq, con el zip y el título de mapadisco.
- `scripts/package.sh` — nuevo: compila con la versión dada y arma el zip por plataforma.
- `scripts/check.sh` — nuevo, adaptado de draco-hq.
- `scripts/build.sh` — eliminado (lo reemplazan `package.sh` y el Makefile).
- `CLAUDE.md` — nuevo: reglas del proyecto para Claude Code.
- `docs/flujo-de-trabajo.md` — nuevo: ramas, ciclo de una actividad, `make check`, commit, PR y bitácora.
- `docs/release.md` — nuevo: versión desde el tag, criterio de `bump`, `promote` y `release`.
- `docs/bitacora/README.md`, `docs/bitacora/001-flujo-de-trabajo.md` — nuevos.
- `docs/guia-de-usuario.md` — descarga desde Releases, verificación, nombre del zip, formato de versión.
- `docs/arquitectura.md` — estructura con `Makefile`, `scripts/`, `CLAUDE.md` y los documentos nuevos.
- `README.md` — instalación desde Releases; sección Desarrollo con `make` y enlaces a los documentos.
- `cmd/mapadisco/main_windows.go` — solo el comentario de `version` (ya no menciona `build.sh`).
- `internal/scanner/node_test.go`, `internal/report/report_test.go` — usuario ficticio en las rutas.
- `.gitignore` — `CLAUDE.local.md` y `.claude/terminos-privados.txt`.

## Cómo probarlo

```bash
make                                  # ayuda agrupada: Git, Go, Actividad, Release
make lint test build
make package                          # dist/mapadisco-<git describe>-windows-amd64.zip con 7 archivos
make release-build ver=v9.9.9         # zip + checksums.txt
(cd dist && sha256sum -c checksums.txt)   # OK
make run args="-version"              # mapadisco <git describe> desde Windows
make release bump=patch               # fuera de main: "✗ los releases salen de main"
make release-build ver=1.2            # rechazada: "ver=vMAJOR.MINOR.PATCH"
make start name=Mal                   # rechazado: minúsculas y guiones
make check
```

Todos se corrieron en esta actividad con los resultados indicados.

## Notas y pendientes

- **Sin probar contra GitHub:** `make push`, `pr`, `merge`, `finish`, `promote` y `release`. Los corre el autor; la
  primera vez, `develop` debe existir en `origin` (`git push -u origin develop`).
- Primer release: no hay tags. Para seguir la numeración del `.exe` que ya se usó (1.1.0), la recomendación es
  `make release ver=v1.1.0`; con `bump=` partiría de `v0.0.0`.
- Opcional en GitHub: dejar solo *Allow merge commits* en los ajustes del repo.
- CI (GitHub Actions con `make ci`): no se agregó; queda para una actividad propia si el autor lo quiere.

**Nota aclaratoria (2026-11-01, actividad 002):** el autor decidió que el primer release fuera `v0.1.0` y no `v1.1.0`
como se recomendaba en Pendientes; se publicó con `bump=minor` desde `v0.0.0`. La regla de versionado para `v0.x`
quedó en `docs/release.md`. El zip y `scripts/package.sh` descritos aquí se reemplazaron en la 002 por un solo `.exe`.

**Migración a devlog (2026-11-01, rama `feat/exe-unico`):** esta entrada se escribió como
`docs/bitacora/001-flujo-de-trabajo.md` con el formato de draco-hq y se trasladó a `entries/` sin cambiar su
contenido: «Diseño técnico» pasó a *Qué cambia*, «Solicitud» y «Motivación» a *Por qué*. La rama se llamaba
`feature/` (antes de adoptar los prefijos de devlog), por eso el campo `branch` conserva ese nombre.
