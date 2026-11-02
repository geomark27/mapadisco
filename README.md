# mapadisco

Mapa de uso de disco para Windows desde la terminal: te muestra en qué se va el espacio y, con `mapadisco -clean`,
libera lo que se puede borrar con seguridad (cachés, temporales, Papelera) y te dice a dónde ir con el resto.
El análisis es de **solo lectura**.

📘 **[Guía de usuario completa](docs/guia-de-usuario.md)**: instalación, cómo leer el reporte, todas las opciones, qué hacer con lo que encuentres y solución de problemas.

## Instalar

En PowerShell:

```powershell
irm https://raw.githubusercontent.com/geomark27/mapadisco/main/scripts/install.ps1 | iex
```

Descarga `mapadisco-windows-amd64.exe` del último release, verifica su SHA-256 contra `checksums.txt` y lo instala para
tu usuario en `%LOCALAPPDATA%\Programs\mapadisco`, agregando esa carpeta al PATH (sin permisos de Administrador).

También puedes descargar el `.exe` desde **Releases** y abrirlo: la primera vez ofrece instalarse. Abre una terminal
nueva y escribe `mapadisco`. Para desinstalar: `mapadisco -uninstall`.

## Uso

```
mapadisco                         # analiza C:\
mapadisco -admin                  # relanza como Administrador (ve System Volume Information, WindowsApps, etc.)
mapadisco -depth 4 -min 500MB     # árbol más profundo y con carpetas más pequeñas
mapadisco -no-expand              # no bajar más allá de -depth
mapadisco -csv reporte.csv        # exporta las carpetas >= 10 MB para Excel (separador ;)
mapadisco -clean                  # libera espacio: muestra qué se puede limpiar y pregunta qué borrar
mapadisco -clean -admin           # lo mismo, incluyendo los temporales de Windows
mapadisco D:\                     # otra unidad o carpeta
mapadisco -h                      # todas las opciones
```

## Qué muestra

- **Resumen del disco**: capacidad, usado, libre, lo que encontró el escaneo y el espacio **no visible** (lo que Windows marca como usado pero el escaneo no pudo leer: puntos de restauración, carpetas protegidas, metadatos NTFS).
- **Árbol de carpetas más pesadas**, con porcentaje y barra (rojo ≥ 50 GB, amarillo ≥ 10 GB). Las carpetas que pesan ≥ 5 % del total se siguen desglosando aunque superen `-depth`, para no esconder al culpable.
- **Sospechosos habituales**: caché de shaders de la GPU, discos de WSL/Docker (`.vhdx`), hiberfil.sys, pagefile, Windows.old, caché de Windows Update, puntos de restauración, Steam/Xbox/Epic, cachés de npm/pip/Gradle, Android SDK, navegadores, etc., cada uno con una sugerencia.
- **Archivos más grandes** y **peso por tipo de archivo**.
- **Carpetas sin acceso**: sin permisos de Administrador, su contenido no se cuenta.
- **Limpieza (`-clean`)**: caché de shaders, temporales, Papelera, volcados de errores y cachés de npm/pip/yarn, con
  confirmación. Si Windows no deja borrar algo (archivo en uso o sin permiso) lo salta y te dice qué hacer; y lista con
  sus pasos lo que solo se libera a mano (Windows Update, WinSxS, WSL, Docker, juegos, hibernación…).

## Notas de precisión

- No sigue junctions ni enlaces simbólicos (`Archivos de programa`, `Documents and Settings`…), así que nada se cuenta dos veces.
- Los archivos comprimidos o dispersos se miden por su tamaño real en disco; los de OneDrive "solo en la nube" no se cuentan.
- Los enlaces duros (p. ej. `WinSxS` ↔ `System32`) se cuentan en ambos lugares, por eso `C:\Windows` puede verse algo mayor de lo real.
- Usa unidades binarias (1 GB = 1024³ bytes), igual que el Explorador de Windows.

## Desarrollo

Requiere Go 1.26+. Se puede desarrollar desde Linux/WSL: el código de Windows está aislado y las pruebas corren en cualquier sistema.

```
make                     # lista todos los atajos
make test                # pruebas (portables)
make lint                # gofmt + vet para Linux y Windows
make run args="-depth 4" # compila el .exe y lo ejecuta en Windows desde WSL
make package             # el .exe del release en dist/
make check               # verificaciones antes del commit
```

- Estructura y cómo extender el programa: [docs/arquitectura.md](docs/arquitectura.md).
- Ramas (`main` ← `develop` ← `feat/<tema>`), commits y PR: [docs/flujo-de-trabajo.md](docs/flujo-de-trabajo.md).
- Versionado (tag de git, SemVer) y publicación: [docs/release.md](docs/release.md).
- Historia de cada cambio (devlog): [docs/bitacora/](docs/bitacora/README.md).
