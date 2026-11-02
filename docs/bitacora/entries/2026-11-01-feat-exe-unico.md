---
date: 2026-11-01
branch: feat/exe-unico
type: feat
author: Marcos Ramos Jacome
agents: claude
pr: pending
version: v0.2.0
---

# Un solo .exe que se instala solo y bitácora con devlog

## Qué cambia

- **Mismo modelo que draco** (`internal/installer` y `cmd/startup.go` de draco-hq): un `.exe` por plataforma con
  nombre fijo (`mapadisco-windows-amd64.exe`), `checksums.txt`, autoinstalación al primer uso e instalación de una
  línea con `irm … | iex`.
- **`internal/installer`:** `Dir`, `IsInstalled`, `Install`, `Uninstall` y `CleanOld`. Las rutas se comparan como
  Windows (sin distinguir mayúsculas, sin barra final ni comillas) con funciones de texto, no `filepath`, para que
  las pruebas corran en Linux con rutas `C:\…`. `Uninstall` se niega a borrar una carpeta que no se llame
  `mapadisco`.
- **Reemplazo de un `.exe` en uso:** Windows no deja sobrescribirlo pero sí renombrarlo. `Install` escribe un
  temporal, aparta el instalado como `.old` y renombra el temporal; el programa borra el `.old` al arrancar. Mismo
  mecanismo que el `install.ps1` de draco.
- **PATH en el registro (`internal/winapi`):** `UserPath`/`SetUserPath` con `golang.org/x/sys/windows/registry`,
  conservando el tipo `REG_EXPAND_SZ`. Es la primera dependencia de mapadisco y se justifica: escribir el registro a
  mano con `syscall` duplicaría lo que esa librería oficial ya resuelve, y draco usa la misma versión (v0.49.0). Se
  mantiene la regla de que solo `winapi` toca Win32: `installer` decide qué escribir y `winapi` lo escribe.
- **Diferencias con draco:**
  - tras cambiar el PATH se envía `WM_SETTINGCHANGE` (`BroadcastEnvironmentChange`). Sin ese aviso, las terminales
    que se abren desde el Explorador o el menú Inicio no ven el PATH nuevo hasta cerrar sesión. draco escribe el
    registro sin avisar; convendría revisarlo allá;
  - hay `-uninstall`. Un `.exe` en uso no se puede borrar, así que si corre el instalado, un `cmd.exe` oculto
    (`StartHidden`) borra la carpeta unos segundos después de salir;
  - `scripts/install.ps1` no reimplementa la instalación: descarga, verifica el SHA-256 y ejecuta
    `mapadisco-windows-amd64.exe -install`. Así la lógica (registro, `.old`, aviso) vive en un solo lugar.
- **PATH idéntico al desinstalar.** La primera prueba real en Windows mostró que agregar y quitar la entrada borraba
  el `;` final del PATH: `appendEntry` recortaba separadores y `withoutEntry` descartaba entradas vacías. Ahora
  `appendEntry` respeta el `;` final, `withoutEntry` solo quita la entrada de mapadisco, y
  `TestAgregarYQuitarDejaElPathIntacto` exige el ida y vuelta exacto. El PATH del equipo de prueba se restauró desde
  la copia tomada antes de probar y se comprobó idéntico (tipo y valor).
- **Cuándo pregunta:** solo si stdin es una consola (nunca en una tubería, un script ni desde WSL), si no corre desde
  la carpeta instalada y sin `-no-install`. `-version` y `-h` no preguntan. Sin argumentos (doble clic o el `.exe`
  recién descargado) termina tras instalar; con argumentos sigue con el análisis. Si se rechaza, analiza igual.
  `-admin` relanza con `-no-install` para no repetir la pregunta en la ventana elevada.
- **Doble clic:** si el proceso es el único en su consola (`GetConsoleProcessList` = 1) y no tiene argumentos, se
  activa `-pause`, para que la ventana no se cierre antes de leer el resultado. Se exige "sin argumentos" porque
  `make run` desde WSL también puede tener consola propia.
- **Makefile:** `package` y `release-build` generan el `.exe` (como `release-build` de draco); se eliminan
  `scripts/package.sh`, la carpeta `installer/` y la dependencia de `zip`. `make run` pasa `-no-install`.
  `release.sh` sube `dist/mapadisco-*.exe`.
- **Versionado `v0.x`** (en `docs/release.md`): el autor decidió empezar en `v0.1.0`; mientras la versión mayor sea
  0, `minor` cubre también lo que rompe. Este cambio altera la instalación, así que corresponde `bump=minor` → v0.2.0.
- **Descartado:** instalador MSI o firmado (otra herramienta y un certificado); mantener el zip además del `.exe`
  (dos formas de instalar que documentar y probar); `mapadisco update` como en draco (actividad propia, ver
  Pendientes).

### Bitácora con devlog

A mitad de la actividad se instaló la skill **devlog** en el proyecto (`.devlog.conf`, `.claude/skills/devlog/`,
`entries/`, `releases/`, `entry-template.md`) y el autor decidió adoptarla en esta misma rama, en lugar del formato
`NNN-<tema>.md` de draco-hq que se había montado en la actividad anterior.

- **Entradas:** una por rama en `docs/bitacora/entries/AAAA-MM-DD-<tipo>-<tema>.md`, con front matter
  (`branch`, `type`, `version: unreleased`…) y las secciones de la plantilla en español; *Qué cambia* y *Por qué*
  son obligatorias. Se crean con `make bitacora title="…"` (`new-entry.sh`) y se validan con `make bitacora-check`
  (`verify.sh`).
- **Migración:** la 001 pasó a `entries/2026-11-01-feat-flujo-de-trabajo.md` con `version: v0.1.0` y `pr: "#1"`, sin
  cambiar su contenido (nota al final). Se reconstruyó `releases/v0.1.0.md` con `reconstructed: true`, como pide la
  guía de devlog para proyectos con historia. La 002 es esta entrada. El índice pasó a la plantilla de devlog, con su
  tabla de versiones.
- **Prefijos de rama:** devlog toma el tipo del prefijo (`feat`, `fix`, `docs`, `chore`, `refactor`, `perf`,
  `test`); una rama `feature/x` quedaría como `chore`. `make start` usa ahora esos tipos (por defecto `feat`) y esta
  rama, que aún no estaba subida, se renombró de `feature/exe-unico` a `feat/exe-unico`.
- **Cierre de versión compatible con PR:** `close-release.sh` modifica archivos (ficha de la versión y `version:` de
  las entradas), y aquí `main` solo recibe cambios por PR y `release.sh` no commitea. Por eso hay un paso nuevo,
  `make release-prep bump=…|ver=…`, que corre en una rama `chore/release-X-Y-Z`, muestra con `--dry-run` qué entradas
  entran, pregunta y cierra la bitácora; esa rama llega a `develop` por PR. `make release` exige que
  `releases/<versión>.md` exista en `main` y enlaza la ficha en las notas del GitHub Release.
- **`make check`:** la sección de bitácora usa `verify.sh` (entrada única, front matter, secciones con contenido,
  sin secretos) y exige que la entrada viaje en el cambio. En una rama de cierre de versión acepta la ficha nueva en
  lugar de una entrada. `verify.sh --base` no se usa antes del commit porque solo mira cambios ya commiteados.
- **Versión en un solo lugar:** `scripts/version.sh` (último tag, siguiente versión, tipo de bump) lo usan
  `release-prep.sh` y `release.sh`. El último tag se toma con `git tag --sort=-v:refname` y no con `git describe`:
  al probar, `describe` devolvía `v0.0.0` desde esta rama aunque existe `v0.1.0`, porque el tag está en el *merge
  commit* de `main` y `develop` no lo contiene. `release-prep` habría propuesto `v0.1.0` otra vez. (El `release.sh`
  de draco usa `describe`, pero corre en `main`, donde sí ve el tag.)
- **`make check` en ramas de cierre:** el modo «cierre de versión» depende del nombre `chore/release-…`. La primera
  versión lo deducía de que hubiera una ficha nueva, y en esta rama (que trae la ficha reconstruida de v0.1.0)
  dejaba de validar la entrada.

## Por qué

### Solicitud

Tras publicar la v0.1.0 (un zip con el `.exe`, `instalar.cmd`, `desinstalar.cmd`, `install.ps1`, README y guía), el
autor pidió algo más sencillo: «un .exe con todo directo, como lo hace draco».

Durante la actividad se instaló devlog en el proyecto y el autor eligió adoptarlo ya, migrando la bitácora existente.

### Motivación

| Problema | Daño real |
|---|---|
| El release era un zip con 7 archivos | Hay que descargar, descomprimir, entender qué abrir y no ejecutar desde dentro del zip; varias filas de la guía existían solo para esos errores |
| La instalación dependía de `install.ps1` y de la política de ejecución de PowerShell | En equipos con scripts bloqueados, `instalar.cmd` podía fallar |
| Desinstalar requería conservar el zip (`desinstalar.cmd`) | Quien lo borró no tenía cómo desinstalar salvo a mano |
| El nombre del zip llevaba la versión | No había una URL fija al último release: imposible una instalación de una línea |
| El instalador anterior leía el PATH con `GetEnvironmentVariable` (que expande las `%VAR%`) y lo guardaba con `SetEnvironmentVariable` | Podía dejar el PATH del usuario con las `%VAR%` de otros programas convertidas en texto fijo (no se probó en esta actividad; el instalador nuevo lo evita por diseño) |
| Dos formatos de bitácora en el repo (draco y devlog) | `make check` fallaba con la plantilla de devlog y cada actividad tendría que elegir formato |
| El cierre de versión de devlog necesita un commit y `main` solo acepta PR | Sin un paso propio, la ficha de la versión no tenía cómo llegar a `main` antes del tag |

## Archivos y áreas

- `internal/installer/installer.go` — nuevo: lógica portable de instalación y rutas.
- `internal/installer/path_windows.go`, `path_other.go` — PATH y borrado programado vía `winapi`; stubs fuera de Windows.
- `internal/installer/installer_test.go` — nuevo: PATH (incluido el ida y vuelta exacto), `IsInstalled`, `Dir`,
  reemplazo con `.old`, guardia de `Uninstall`.
- `internal/winapi/registry_windows.go` — nuevo: `UserPath`, `SetUserPath`, `StartHidden`.
- `internal/winapi/env_windows.go` — nuevo: `BroadcastEnvironmentChange`, `StdinIsConsole`, `OwnsConsole`.
- `cmd/mapadisco/install_windows.go` — nuevo: pregunta de instalación, `-install`, `-uninstall`, pausa final.
- `cmd/mapadisco/main_windows.go` — flags `-install`, `-uninstall`, `-no-install`; limpieza del `.old`; pausa con doble clic.
- `go.mod`, `go.sum` — `golang.org/x/sys v0.49.0` (`go 1.26` pasa a `1.26.0`, lo escribe `go get`; es la misma versión).
- `scripts/install.ps1` — nuevo: instalación de una línea con verificación SHA-256.
- `installer/` (`instalar.cmd`, `desinstalar.cmd`, `install.ps1`), `scripts/package.sh` — eliminados.
- `Makefile` — `package`/`release-build` con el `.exe`; `run` con `-no-install`.
- `scripts/release.sh` — sube el `.exe`.
- `scripts/check.sh` — lo «visible» incluye `internal/installer/` y `scripts/install.ps1`.
- `docs/guia-de-usuario.md` — instalación (una línea, `.exe`, portable, script, actualizar), opciones nuevas, códigos de
  salida, solución de problemas, desinstalación.
- `docs/release.md` — artefacto `.exe` de nombre fijo, verificación, regla `v0.x`.
- `docs/arquitectura.md` — paquete `installer`, dependencias y flujo de instalación.
- `README.md`, `CLAUDE.md`, `docs/flujo-de-trabajo.md` — instalación y `make package`; regla de escritura y dependencias.
- `.devlog.conf`, `.claude/skills/devlog/` — devlog en el proyecto (instalado por el autor; no se modificó).
- `docs/bitacora/README.md` — índice con la plantilla de devlog y la fila de v0.1.0.
- `docs/bitacora/entry-template.md`, `entries/`, `releases/` — de devlog.
- `docs/bitacora/entries/2026-11-01-feat-flujo-de-trabajo.md` — la 001 migrada (más la nota de versión inicial).
- `docs/bitacora/entries/2026-11-01-feat-exe-unico.md` — esta entrada.
- `docs/bitacora/releases/v0.1.0.md` — ficha reconstruida.
- `docs/bitacora/001-flujo-de-trabajo.md` — eliminado (migrado).
- `scripts/version.sh`, `scripts/release-prep.sh` — nuevos: versión compartida y cierre de la bitácora.
- `scripts/release.sh` — exige la ficha de la versión y la enlaza en las notas.
- `scripts/check.sh` — prefijos de devlog y `verify.sh`; `scripts/merge.sh` — mensaje con `feat/`.

## Cómo probarlo

```bash
make test lint build
make package                      # dist/mapadisco-windows-amd64.exe
make run args="-depth 1 C:\\"     # el análisis sigue igual (2,3 s, 728 mil archivos)
make check
```

Ciclo real en Windows, con `LOCALAPPDATA` apuntando a una carpeta de prueba y el PATH del usuario respaldado antes:

| Paso | Resultado |
|---|---|
| `mapadisco-windows-amd64.exe -install` desde una carpeta "descarga" | Instalado; PATH sigue `ExpandString` y contiene la carpeta |
| El instalado `-version` | Responde sin preguntar |
| `-install` de nuevo y desde otra copia | Reemplaza; deja `mapadisco.exe.old` |
| El instalado arranca | Borra el `.old` |
| El instalado `-uninstall` | Quita del PATH; la carpeta desaparece a los segundos |
| `-uninstall` otra vez | «no está instalado: no hay nada que desinstalar», código 0 |
| PATH final vs respaldo | Idéntico (tipo y valor) |

Bitácora:

```bash
make bitacora-check                         # verify.sh: entrada válida
make release-prep bump=minor                # en esta rama: «el cierre de versión va en su propia rama…»
make release ver=v0.2.0                     # fuera de main: se detiene antes de nada
```

## Notas y pendientes

- **Sin probar:** la pregunta interactiva y la pausa con doble clic (necesitan una consola real; desde WSL stdin no lo
  es) y `scripts/install.ps1`, que descarga de `releases/latest`: funcionará cuando exista un release con el `.exe`
  (v0.2.0). Verificar ambos al publicar (`docs/release.md`, «Verificar un release»).
- La v0.1.0 sigue publicada como zip; quien la tenga instalada con `instalar.cmd` queda en la misma carpeta y la v0.2.0
  la reemplaza al instalarse.
- `mapadisco update` (como `draco update`): descargar e instalar la última versión verificando `checksums.txt`.
- Guardia de `develop` en `make pr`/`make start` (propuesta tras el primer `make ship`).
- Probar `make release-prep` de punta a punta en la rama `chore/release-0-2-0` (primer cierre real con devlog).
- Opcional: el check de devlog en GitHub Actions (`HOW-IT-WORKS.md`, §10), junto con `make ci`.
