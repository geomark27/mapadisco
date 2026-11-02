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
│   └── format/             Tamaños, porcentajes, barras, miles; sin dependencias
├── installer/              install.ps1 + instalar.cmd / desinstalar.cmd
├── scripts/                package.sh (zip), check.sh (antes del commit), merge/promote/release.sh
├── Makefile                Atajos del día a día: make help
├── CLAUDE.md               Reglas del proyecto para Claude Code
└── docs/
    ├── guia-de-usuario.md    Guía para quien usa la herramienta (va dentro del zip)
    ├── arquitectura.md       Este documento
    ├── flujo-de-trabajo.md   Ramas, actividades, commits y PR
    ├── release.md            Versionado y publicación
    └── bitacora/             Una entrada por actividad
```

No hay archivo `VERSION`: la versión es el tag de git (`vX.Y.Z`) y se inyecta al compilar
(ver [release.md](release.md)).

## Dependencias entre paquetes

```
cmd/mapadisco ──► report ──► scanner ──► winapi
      │             │           ▲
      │             └──► format │
      └──► hotspots ────────────┘ (y winapi)
```

- `winapi` no depende de nadie del proyecto. Nada fuera de él usa `syscall`.
- `format`, el modelo de `scanner` (`node.go`), el catálogo de `hotspots` y todo `report`
  son **portables** y tienen pruebas que corren en Linux/WSL.
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
