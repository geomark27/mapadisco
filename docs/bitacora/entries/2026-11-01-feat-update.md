---
date: 2026-11-01
branch: feat/update
type: feat
author: Marcos Ramos Jacome
agents: claude
pr: pending
version: v0.4.0
---

# Actualizar con mapadisco -update

## Qué cambia

- **`mapadisco -update`**: consulta la última versión publicada en GitHub y, si es más nueva, descarga
  `mapadisco-windows-amd64.exe`, **lo instala solo si su SHA-256 coincide** con el `checksums.txt` del mismo release
  y reemplaza el ejecutable en uso. **`-update -check`** solo informa. Muestra el enlace a las novedades de la versión.
- Casos: «Ya tienes la versión más reciente»; versión de desarrollo (`dev`, `abc1234-dirty`): no se actualiza sola;
  versión más nueva que la publicada: lo dice en lugar de afirmar que es «la más reciente»; sin conexión o con un
  SHA-256 distinto: error (código 1) sin tocar nada. `-check` sin `-update` es un error de uso (código 2).
- **`internal/updater`** (portable, mismo diseño que el de draco-hq): `Latest` (API de GitHub, `User-Agent`
  `mapadisco/<versión>`), `Fetch` (descarga con límite de tamaño y verificación SHA-256), `IsNewer`/`IsRelease`
  y `Replace` (escribe al lado, aparta el actual como `.old`, renombra; si falla, restaura). El `.old` lo borra
  `installer.CleanOld` al arrancar la versión nueva, igual que tras una reinstalación.
- **Sin aviso automático de versión nueva**, a diferencia de draco: la guía promete que mapadisco no se conecta a
  internet; ahora la única excepción es `-update`, cuando el usuario lo ejecuta. Quedó como regla en `CLAUDE.md`.
- `-update` corre antes de la pregunta de instalación: actualizar un `.exe` portable no ofrece instalarlo.
- **Quien tenga v0.3.0 o anterior** no tiene `-update` («flag provided but not defined»): debe actualizar una vez
  reinstalando; la guía lo explica sin nombrar versiones.

## Por qué

### Solicitud

El autor preguntó cómo actualizar la versión instalada (tenía v0.2.0 con la v0.3.0 publicada); la respuesta era
reinstalar. Pidió «haz el mapadisco update», como `draco update`.

### Motivación

| Problema | Daño real |
|---|---|
| Actualizar era repetir la instalación (línea de PowerShell o descargar el `.exe`) | Hay que recordar la URL o volver a GitHub en cada versión; la versión instalada se queda atrás |
| No había forma de saber desde la terminal si hay una versión nueva | Se usan versiones viejas sin saberlo (la v0.2.0 instalada no tenía `-clean`) |

## Archivos y áreas

- `internal/updater/updater.go`, `updater_test.go` — nuevos: cliente de releases, verificación y reemplazo; pruebas contra un GitHub simulado (`httptest`).
- `cmd/mapadisco/update_windows.go` — nuevo: flujo de `-update` / `-update -check`.
- `cmd/mapadisco/main_windows.go` — opciones `-update` y `-check`; ejemplo en la ayuda.
- `docs/guia-de-usuario.md` — «Actualizar a una versión nueva», conexión a internet, opciones, códigos de salida, solución de problemas.
- `README.md`, `CLAUDE.md`, `docs/arquitectura.md` — actualización y regla de red.

## Cómo probarlo

```bash
make test lint build
make check
```

Probado en Windows contra el **GitHub real** (release v0.3.0), en una carpeta aislada (la instalación del autor no se tocó):

| Prueba | Resultado |
|---|---|
| Copia compilada como `v0.2.0`: `-update -check` | «Nueva versión: v0.3.0», enlace a novedades, sin descargar |
| `-update` | Descargó v0.3.0, verificó y reemplazó; SHA-256 del resultado = el de `checksums.txt` del release; quedó `.old` |
| Arrancar el reemplazado (`-version`) | `mapadisco v0.3.0`; el `.old` se borró |
| Compilada como `v0.3.0`: `-update` | «Ya tienes la versión más reciente (v0.3.0)» |
| Compilada como `v9.0.0` | «Tu versión es más nueva que la última publicada (v0.3.0)» (antes decía «la más reciente»: corregido) |
| Compilación de desarrollo | «Es una versión de desarrollo: no se actualiza sola» |
| `-check` sin `-update` | Mensaje de uso, código 2 |
| La v0.3.0 real con `-update` | «flag provided but not defined»: por eso la guía explica actualizar una vez a mano |

## Notas y pendientes

- **Sin probar:** detrás de un proxy corporativo (Go usa `HTTPS_PROXY` del entorno, no la configuración de proxy de
  Windows) y el reemplazo dentro de `%LOCALAPPDATA%\Programs\mapadisco` (se probó en otra carpeta; el mecanismo es el
  mismo que ya usa `-install`).
- Si más adelante se quiere un aviso de versión nueva, debe ser opcional y documentado (rompe la promesa de no usar red).
