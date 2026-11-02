---
date: 2026-11-01
branch: feat/limpieza
type: feat
author: Marcos Ramos Jacome
agents: claude
pr: pending
version: v0.3.0
---

# Limpiar y liberar espacio (-clean)

## Qué cambia

- **`mapadisco -clean`**: mide lo que se puede liberar, lo lista numerado con su tamaño y qué implica, pregunta
  `¿Qué limpio? [t = todo · 1,3 = solo esos · Enter = nada]` y lo borra. Informa lo liberado por categoría y el
  espacio libre real del disco antes y después. `-yes` limpia todo sin preguntar; `-only shaders,temp` limita las
  categorías; `-clean -admin` lo hace elevado (con `-admin` se relanza con las mismas opciones).
- **Categorías automáticas** (`internal/cleaner`, `cleaner.Auto`): caché de shaders (NVIDIA, D3DSCache, AMD, Intel),
  temporales del usuario (`%TEMP%`), temporales de Windows (Administrador), Papelera (API `SHEmptyRecycleBinW`),
  volcados e informes de errores (CrashDumps, WER, Minidump, MEMORY.DMP) y cachés de npm, pip y yarn. Solo cosas que se
  regeneran o que el usuario ya descartó.
- **Si Windows se opone**, no se detiene: salta el archivo, sigue y al final dice cuántos quedaron y por qué (en uso
  por otro programa, sin permiso, otro error), hasta 3 rutas de ejemplo, qué hacer (cerrar programas o reiniciar;
  `-clean -admin`), a dónde ir a mano (Configuración, Liberador de espacio…) y el `explorer "…"` de la carpeta.
- **Guías manuales** (`cleaner.Manual`): al terminar mide y muestra, con sus pasos, lo que mapadisco no borra por su
  cuenta: Windows Update y Windows.old, WinSxS (DISM), puntos de restauración, hibernación, disco de WSL (compactar),
  Docker, juegos, Descargas y caché de navegadores.
- **Seguridad del borrado** (`cleaner.Clean`): borra el contenido de cada carpeta, nunca la carpeta de la categoría;
  no sigue enlaces simbólicos ni junctions (Go los marca `ModeSymlink`/`ModeIrregular`); quita el atributo de solo
  lectura y reintenta; rechaza rutas relativas y raíces de unidad (`checkSafe`); `Resolve` descarta una ruta si su
  variable de entorno está vacía, para que `$TEMP` vacío nunca se convierta en una ruta relativa.
- **`winapi`**: `RecycleBinUsage`/`EmptyRecycleBin` (`SHQueryRecycleBinW`/`SHEmptyRecycleBinW`; `E_UNEXPECTED` con la
  Papelera vacía no es error) e `IsInUse` (`ERROR_SHARING_VIOLATION`/`ERROR_LOCK_VIOLATION`). `cleaner` es portable:
  recibe `IsInUse` por `Options` y se prueba en Linux.
- **Análisis**: el cierre del reporte pasa a «Este análisis no borra nada. Para liberar espacio: mapadisco -clean» y
  los sospechosos que ya se limpian solos sugieren `mapadisco -clean`.
- **Decisiones:** Windows Update y WinSxS van a la guía manual y no a `Auto`, porque Windows tiene herramientas propias
  que además paran servicios o validan componentes; pnpm queda fuera de `dev` porque su almacén está enlazado con los
  `node_modules` (se sugiere `pnpm store prune`); la Papelera se vacía con la API y no borrando `$Recycle.Bin`, que es
  del sistema. Selección por defecto con Enter = nada: en una acción que borra, no hacer nada es lo seguro.
- **Descartado por ahora:** enviar a la Papelera en vez de borrar (las cachés y temporales se regeneran; duplicaría el
  espacio hasta vaciarla) y la interfaz TUI (como la de draco): queda para más adelante, a pedido del autor.

## Por qué

### Solicitud

El autor preguntó si mapadisco ya limpiaba algo (no: era solo diagnóstico) y pidió «la opción de limpiar o liberar lo
que se pueda, y en caso de que el sistema ponga oposición, que me indique a dónde tengo que ir para limpiarlo
manualmente». La TUI al estilo draco queda para más adelante.

### Motivación

| Problema | Daño real |
|---|---|
| mapadisco encontraba el espacio pero no lo liberaba | En el equipo de prueba, 58 GB recuperables (55 GB de caché de shaders) quedaban como una sugerencia que había que ejecutar a mano en varias herramientas |
| Borrar cachés a mano falla con archivos en uso | El Explorador se detiene en el primer archivo bloqueado y no dice qué quedó ni por qué |
| Lo grande que no se debe borrar a mano (WSL, WinSxS, Windows Update) no tenía pasos concretos | Riesgo de borrar lo que no se debe, o de no saber qué herramienta usar |

## Archivos y áreas

- `internal/cleaner/catalog.go` — nuevo: `Auto`, `Manual`, `IDs`, `Select`.
- `internal/cleaner/clean.go` — nuevo: `Resolve`, `Measure`, `Clean`, `ParseSelection`, `checkSafe`.
- `internal/cleaner/clean_test.go` — nuevo: catálogo, rutas, borrado, enlaces, solo lectura, oposición, en uso, rutas peligrosas, selección.
- `internal/winapi/recycle_windows.go` — nuevo: Papelera e `IsInUse`.
- `cmd/mapadisco/clean_windows.go` — nuevo: flujo de `-clean`, informe de oposición y guías manuales.
- `cmd/mapadisco/main_windows.go` — `-clean`, `-yes`, `-only`; mensaje final del análisis.
- `internal/hotspots/catalog.go` — sugerencias que apuntan a `mapadisco -clean`.
- `docs/guia-de-usuario.md` — sección 10 «Liberar espacio», opciones, códigos de salida, ejemplos, preguntas frecuentes, solución de problemas.
- `README.md`, `CLAUDE.md`, `docs/arquitectura.md` — limpieza; reglas para ampliar `cleaner.Auto`.
- `scripts/check.sh` — el catálogo de limpieza cuenta como cambio visible para la guía.

## Cómo probarlo

```bash
make test lint build
make check
./dist/mapadisco-windows-amd64.exe -clean        # desde WSL (sin consola): solo mide, sugiere -yes y muestra las guías
```

Probado en Windows real:

| Prueba | Resultado |
|---|---|
| `-clean` sin consola interactiva | Midió 58,1 GB en 5 categorías (Papelera con la API incluida), no borró nada, sugirió `-clean -yes` y mostró 8 guías manuales con su tamaño |
| Borrado en una carpeta aislada (`LOCALAPPDATA` y `TEMP` apuntando a ella), `-clean -yes -only shaders,temp,dev` | Borró 7 de 8 archivos y las subcarpetas; las carpetas raíz se conservaron |
| Un archivo abierto por otro proceso sin compartir | Se saltó y se informó como «en uso por otro programa (512 KB)», con ejemplo, qué hacer, ruta manual y `explorer` de la carpeta |
| Archivo de solo lectura | Borrado |
| Junction dentro de `Temp` hacia un archivo fuera | No se siguió: el archivo de afuera quedó intacto |
| Espacio libre antes/después | Medido en el disco real (+4 MB) |

## Notas y pendientes

- **Sin probar:** la pregunta interactiva (desde WSL stdin no es una consola), vaciar la Papelera real (no se ejecutó
  para no borrar la del autor; solo se midió), `temp-sistema` y `volcados` reales, y `-clean -admin`. Verificar en
  Windows con `mapadisco -clean` después del release.
- Lo que muestra el «espacio libre antes/después» puede diferir de lo liberado: Windows puede reservar o liberar espacio
  por su cuenta mientras tanto.
- Futuro: interfaz TUI como la de draco (pedido del autor); `mapadisco update`; guardia de `develop` en `make pr`.
