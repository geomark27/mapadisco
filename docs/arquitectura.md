# Arquitectura de mapadisco

## Estructura

```
mapadisco/
├── cmd/mapadisco/          Punto de entrada: opciones de línea de comandos y orquestación
│   ├── main_windows.go       programa real
│   └── main_other.go         en Linux/macOS solo avisa que no está soportado
├── internal/
│   ├── winapi/             Única capa que llama a Win32 (FindFirstFileExW, disco, consola, UAC)
│   ├── scanner/            Modelo del árbol (Node, Result) + motor de escaneo paralelo
│   ├── hotspots/           Catálogo de "sospechosos habituales" + evaluación contra el árbol
│   ├── report/             Presentación en terminal y exportación CSV (escribe a io.Writer)
│   ├── installer/          Autoinstalación: copia a %LOCALAPPDATA%\Programs\mapadisco y PATH del usuario
│   ├── cleaner/            Limpieza (-clean): catálogo de categorías, medir, borrar contenido, guías manuales
│   ├── updater/            Actualización (-update): última versión en GitHub, descarga verificada, reemplazo
│   └── format/             Tamaños, porcentajes, barras, miles; sin dependencias
├── scripts/                install.ps1 (instalación de una línea), check.sh, version.sh, merge/promote/release(-prep).sh
├── .claude/skills/devlog/  Bitácora (devlog), configurada en .devlog.conf
├── Makefile                Atajos del día a día: make help
├── CLAUDE.md               Reglas del proyecto para Claude Code
└── docs/
    ├── guia-de-usuario.md    Guía para quien usa la herramienta
    ├── arquitectura.md       Este documento
    ├── flujo-de-trabajo.md   Ramas, actividades, commits y PR
    ├── release.md            Versionado y publicación
    └── bitacora/             devlog: entries/ (una por rama) y releases/ (una por versión)
```

No hay archivo `VERSION`: la versión es el tag de git (`vX.Y.Z`) y se inyecta al compilar
(ver [release.md](release.md)).

## Dependencias entre paquetes

```
cmd/mapadisco ──► report ──► scanner ──► winapi ──► golang.org/x/sys (registro)
      │             │           ▲          ▲
      │             └──► format │          │
      ├──► hotspots ────────────┘ (y winapi)
      ├──► installer ──────────────────────┘
      ├──► cleaner   (portable; cmd le pasa winapi.IsInUse y vacía la Papelera con winapi)
      └──► updater   (portable; red solo con -update)
```

- `winapi` no depende de nadie del proyecto. Nada fuera de él usa `syscall` ni el registro.
- `format`, el modelo de `scanner` (`node.go`), el catálogo de `hotspots`, todo `report` y la lógica de
  rutas de `installer` son **portables** y tienen pruebas que corren en Linux/WSL.

## Limpieza (`internal/cleaner`, `-clean`)

1. `cleaner.Auto` es el catálogo de lo que se puede borrar; `cleaner.Manual`, lo que solo se explica. Las rutas usan
   variables de entorno (`$LOCALAPPDATA`, `$TEMP`, `${ProgramFiles(x86)}`…) y comodines; `Resolve` descarta una
   ruta si su variable está vacía, para no terminar nunca en una ruta relativa o en la raíz.
2. `cmd` mide todas las categorías en paralelo (`Measure`; la Papelera con `SHQueryRecycleBinW`), las lista y pregunta
   (`ParseSelection`), o toma todas con `-yes` y un subconjunto con `-only`.
3. `Clean` borra el **contenido** de cada carpeta (o el archivo indicado), recursivo, sin seguir enlaces ni junctions
   (`ModeSymlink`/`ModeIrregular`) y sin borrar la carpeta raíz. Quita el atributo de solo lectura si hace falta.
   Ante cada error sigue adelante y lo clasifica: en uso (`ERROR_SHARING_VIOLATION`, que `cmd` reconoce con
   `winapi.IsInUse`), sin permiso, otro; guarda hasta 3 ejemplos. `checkSafe` rechaza rutas relativas y raíces de unidad.
4. La Papelera se vacía con `SHEmptyRecycleBinW` (sin diálogos).
5. `cmd` informa por categoría y, si Windows se opuso, el motivo, qué hacer, la ruta manual y el `explorer "…"` de la
   carpeta; luego el espacio libre real antes y después (`GetDiskFreeSpaceExW`) y las guías manuales con su tamaño.

## Actualización (`internal/updater`, `-update`)

1. `Latest` consulta `api.github.com/repos/<repo>/releases/latest` (sin pre-releases); un 404 es `ErrNoReleases`.
2. `IsNewer` compara `vMAJOR.MINOR.PATCH`; una compilación de desarrollo (`dev`, `abc1234-dirty`) nunca es más nueva
   ni más vieja: no se actualiza sola.
3. `Fetch` baja `checksums.txt` y `mapadisco-windows-amd64.exe` del release (con un límite de tamaño) y devuelve el
   ejecutable solo si su SHA-256 coincide.
4. `Replace` escribe la versión nueva junto al ejecutable en uso, aparta el actual como `.old` (Windows deja
   renombrar un `.exe` en uso pero no sobrescribirlo) y renombra; si algo falla, deja todo como estaba.
   `installer.CleanOld` borra el `.old` al arrancar la versión nueva.

Mismo diseño que `draco update`, sin su aviso automático de versión nueva: mapadisco solo usa la red cuando se pide.

## Instalación (`internal/installer`)

El mismo `.exe` del release se instala solo, como draco:

1. Al arrancar borra el `mapadisco.exe.old` que pudo dejar una reinstalación.
2. Si no corre desde `%LOCALAPPDATA%\Programs\mapadisco` y stdin es una consola, pregunta si instalarse
   (`-no-install` lo evita; `-install` instala sin preguntar). Sin argumentos (doble clic o el `.exe` recién
   descargado) termina tras instalar; con argumentos, sigue con el análisis.
3. `Install` copia el ejecutable a un temporal en esa carpeta, aparta el instalado como `.old` (Windows deja
   renombrar un `.exe` en uso pero no sobrescribirlo) y renombra el temporal. Luego agrega la carpeta al PATH del
   usuario en `HKCU\Environment`, conservando el tipo `REG_EXPAND_SZ`, y avisa con `WM_SETTINGCHANGE` para que
   las terminales nuevas lo vean sin cerrar sesión.
4. `Uninstall` quita la entrada del PATH (dejándolo idéntico a como estaba antes de instalar) y borra la carpeta;
   si el que corre es el instalado, el borrado lo hace un `cmd.exe` oculto segundos después de salir.
5. Con doble clic la consola es propia del proceso (`GetConsoleProcessList` = 1): se espera Enter antes de
   cerrar la ventana.
- El código exclusivo de Windows lleva el sufijo `_windows.go`, que Go aplica como
  restricción de compilación automáticamente.

## Flujo de una ejecución

1. `cmd` lee las opciones y resuelve la ruta (`C:` → `C:\`).
2. `scanner.New(...).Run(ruta)` recorre el disco:
   - cada carpeta se enumera con `FindFirstFileExW` (*large fetch*, sin nombres 8.3);
   - las subcarpetas se reparten entre goroutines limitadas por un semáforo; si no hay
     hilos libres, se procesan en el hilo actual (no hay bloqueo ni cola sin límite);
   - en memoria solo se guardan carpetas; los archivos se suman a su carpeta, entran a un
     *min-heap* de los N más grandes y al acumulado por extensión;
   - al final, `SumTotals` agrega los tamaños de abajo hacia arriba.
3. `hotspots.Evaluate` cruza el catálogo con el árbol (las carpetas se miden con el árbol
   ya escaneado; los archivos sueltos como `hiberfil.sys`, con `winapi.Stat`).
4. `report` imprime las secciones y, si se pidió, exporta el CSV.

## Cómo extender

| Quiero… | Dónde |
|---|---|
| Agregar una categoría de limpieza | Una entrada en `cleaner.Auto` (o en `cleaner.Manual` si no debe borrarse sola). La prueba del catálogo exige rutas con variables de entorno. |
| Agregar un sospechoso habitual | Una entrada en `hotspots.Catalog` (`internal/hotspots/catalog.go`). La prueba del catálogo valida el formato. |
| Agregar una sección al reporte | Un método nuevo en `report.Printer` y su llamada en `cmd/mapadisco/main_windows.go`. |
| Nuevo formato de exportación (JSON, HTML…) | Un archivo nuevo en `internal/report/` con una función `WriteXxx(io.Writer, *scanner.Node)`, igual que `WriteCSV`. |
| Nueva opción de línea de comandos | `parseFlags` en `cmd/mapadisco/main_windows.go`. |
| Nueva llamada a Win32 | `internal/winapi/` en un archivo `_windows.go`; expón una función con tipos de Go, no `uintptr`. |
| Recopilar más datos durante el escaneo | `scanner.walk` y `scanner.Result`. Mantén el bucle de enumeración sin bloqueos por archivo: acumula en local y fusiona una vez por carpeta. |

## Decisiones y limitaciones conocidas

- **Enlaces duros**: no se deduplican, porque saber cuántos enlaces tiene un archivo exige
  abrirlo (un `CreateFile` por archivo, mucho más lento). Se avisa en `WinSxS`.
- **Junctions y enlaces simbólicos**: no se recorren (etiquetas `MOUNT_POINT` y `SYMLINK`).
  Otros puntos de reparse (OneDrive, contenedores de apps) sí se recorren.
- **Tamaño en disco**: es el lógico, salvo en archivos comprimidos o dispersos, donde se usa
  `GetCompressedFileSizeW`. No se calcula el desperdicio por tamaño de clúster.
- **Memoria**: aproximadamente un nodo por carpeta (~160 mil en un C: típico).
