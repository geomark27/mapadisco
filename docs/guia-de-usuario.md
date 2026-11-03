# Guía de usuario de mapadisco

Windows 10 / 11 (64 bits) · Para saber qué versión tienes: `mapadisco -version`

---

## Contenido

1. [¿Qué es mapadisco?](#1-qué-es-mapadisco)
2. [Requisitos](#2-requisitos)
3. [Instalación](#3-instalación)
4. [Primer uso](#4-primer-uso)
5. [Cómo leer el reporte](#5-cómo-leer-el-reporte)
6. [Opciones](#6-opciones)
7. [Ejemplos por caso de uso](#7-ejemplos-por-caso-de-uso)
8. [Exportar a Excel (CSV)](#8-exportar-a-excel-csv)
9. [Modo Administrador](#9-modo-administrador)
10. [Liberar espacio](#10-liberar-espacio)
11. [Memoria RAM](#11-memoria-ram)
12. [Preguntas frecuentes](#12-preguntas-frecuentes)
13. [Solución de problemas](#13-solución-de-problemas)
14. [Desinstalación](#14-desinstalación)

---

## 1. ¿Qué es mapadisco?

mapadisco es una herramienta de terminal que responde a una sola pregunta:
**¿dónde se está yendo el espacio de mi disco?**

Recorre una unidad completa (o una carpeta) en pocos segundos y te muestra:

| Sección | Qué te dice |
|---|---|
| **Resumen del disco** | Capacidad, usado y libre según Windows, comparado con lo que el escaneo encontró. La diferencia es espacio oculto o protegido. |
| **Carpetas más pesadas** | Un árbol con las carpetas que más ocupan, con porcentaje y barra. Baja automáticamente hasta el culpable aunque esté muy adentro (p. ej. dentro de `AppData`). |
| **Sospechosos habituales** | Lugares conocidos que suelen crecer sin que te des cuenta (caché de la tarjeta gráfica, discos de WSL o Docker, archivo de hibernación, restos de juegos, cachés de programación…), con una sugerencia de qué hacer. |
| **Archivos más grandes** | Los 25 archivos individuales que más ocupan. |
| **Peso por tipo de archivo** | Cuánto suman los `.vhdx`, `.iso`, `.mp4`, `.dll`, etc. |
| **Carpetas sin acceso** | Lo que no se pudo leer por falta de permisos, para que sepas qué quedó sin contar. |

Con **`mapadisco -ram`** explica en qué se va la memoria RAM, si falta de verdad y qué cerrar o configurar
(ver la [sección 11](#11-memoria-ram)).

Y con **`mapadisco -clean`** libera espacio: borra cachés y temporales que se regeneran solos, después de mostrarte
cuánto ocupa cada uno y preguntarte qué limpiar. Lo que no se puede borrar con seguridad te lo explica paso a paso
(ver la [sección 10](#10-liberar-espacio)).

### Lo que NO hace

- **El análisis (`mapadisco`) no borra, mueve ni modifica nada.** Es de solo lectura. Las sugerencias son eso: sugerencias que tú decides si aplicar.
- **Solo `mapadisco -clean` borra**, y solo las categorías que aparecen en su lista, después de preguntarte (o con `-yes`).
  Nunca toca tus documentos, fotos, descargas, juegos ni archivos del sistema.
- `mapadisco -ram` solo lee. Su única acción, `-ram -wslconfig`, muestra el cambio y escribe `.wslconfig` solo si
  confirmas (guardando una copia del anterior).
- Fuera de eso, solo escribe su propia instalación (si aceptas instalarlo) y el CSV que pidas con `-csv`.
- No se conecta a internet ni envía información a ningún lado. La única excepción es `mapadisco -update`, cuando tú lo
  ejecutas: consulta en GitHub cuál es la última versión y la descarga.
- No corre en segundo plano ni queda residente: solo funciona mientras lo ejecutas.

### Por qué es rápido

Usa directamente la API nativa de Windows y lee muchas carpetas en paralelo. En un disco SSD con unos 700 mil archivos
tarda del orden de 30 segundos la primera vez y 3 a 5 segundos las siguientes (Windows guarda en memoria la estructura del disco).

---

## 2. Requisitos

- Windows 10 u 11 de 64 bits.
- Una terminal: **Windows Terminal** (recomendada, muestra colores), PowerShell o Símbolo del sistema (cmd).
- No necesita permisos de Administrador para instalarse ni para el uso normal.
- No necesita instalar nada adicional (no requiere .NET, Python ni otros programas).

---

## 3. Instalación

mapadisco es **un solo archivo**: `mapadisco-windows-amd64.exe`. No hay zip ni instalador aparte: el mismo `.exe`
se instala solo la primera vez que lo abres. La instalación es **solo para tu usuario** y no pide permisos de
Administrador: copia el programa a `%LOCALAPPDATA%\Programs\mapadisco` (normalmente
`C:\Users\<tu usuario>\AppData\Local\Programs\mapadisco`) y agrega esa carpeta a tu `PATH`, para que puedas
escribir `mapadisco` desde cualquier terminal.

### Opción A — Una línea en PowerShell (recomendada)

Abre PowerShell y pega:

```powershell
irm https://raw.githubusercontent.com/geomark27/mapadisco/main/scripts/install.ps1 | iex
```

El script descarga el `.exe` del último release, **verifica su SHA-256** contra `checksums.txt` (si no coincide, no
instala nada), lo instala y muestra la versión instalada. Para instalar una versión concreta:

```powershell
$env:MAPADISCO_VERSION = "v0.2.0"; irm https://raw.githubusercontent.com/geomark27/mapadisco/main/scripts/install.ps1 | iex
```

### Opción B — Descargar el .exe y abrirlo

1. En la página **Releases** del repositorio (`https://github.com/geomark27/mapadisco/releases`), descarga de la
   última versión `mapadisco-windows-amd64.exe` y, si quieres verificarlo, `checksums.txt`.
2. *(Recomendado)* Verifica la descarga en PowerShell, en la carpeta de descargas:

   ```powershell
   Get-FileHash .\mapadisco-windows-amd64.exe
   ```

   El valor `Hash` debe ser igual al que aparece junto a ese nombre en `checksums.txt` (mayúsculas y minúsculas dan
   igual). Si no coincide, no lo abras: vuelve a descargarlo.
3. **Haz doble clic en `mapadisco-windows-amd64.exe`** (o ejecútalo desde una terminal). Verás:

   ```
   mapadisco no está instalado
     Se copiará a C:\Users\Usuario\AppData\Local\Programs\mapadisco y esa carpeta se agregará a tu PATH.
     Solo para tu usuario; no necesita permisos de Administrador.

   ¿Instalar mapadisco ahora? [S/n]:
   ```

   Presiona **Enter** (o `s`). Verás:

   ```
     ✓ Instalado en C:\Users\Usuario\AppData\Local\Programs\mapadisco\mapadisco.exe
     ✓ Carpeta agregada al PATH de tu usuario
       Abre una terminal nueva y escribe: mapadisco
   ```

   Si lo abriste con doble clic, la ventana espera a que presiones Enter para cerrarse.
4. Ya puedes borrar el archivo descargado.

Si respondes `n`, no se instala nada y mapadisco hace el análisis igual (ver la opción C).

### Comprobar la instalación

Abre una **terminal nueva** (el `PATH` nuevo solo lo ven las terminales abiertas después de instalar) y escribe:

```
mapadisco -version
```

Debe responder con la versión instalada, por ejemplo `mapadisco v0.2.0`.

### Opción C — Sin instalar (portable)

Si prefieres no instalarlo, ejecuta el `.exe` desde su carpeta con `-no-install`: no pregunta nada y analiza directamente.

```
.\mapadisco-windows-amd64.exe -no-install
```

En el Explorador, puedes escribir `cmd` o `powershell` en la barra de direcciones de esa carpeta para abrir una
terminal ahí. Sin instalar, el comando `mapadisco` no existe en otras carpetas.

### Instalar desde un script

`mapadisco-windows-amd64.exe -install` instala sin preguntar y termina (código de salida 0 si todo fue bien).

### Actualizar a una versión nueva

```
mapadisco -update            # descarga e instala la última versión
mapadisco -update -check     # solo consulta si hay una versión nueva
```

```
  Versión actual: v0.3.0
  Nueva versión: v0.4.0
    Novedades: https://github.com/geomark27/mapadisco/releases/tag/v0.4.0

  [1/2] Descargando v0.4.0 y verificando su SHA-256…
  [2/2] Reemplazando C:\Users\Usuario\AppData\Local\Programs\mapadisco\mapadisco.exe…
  ✓ Actualizado a v0.4.0
```

- Consulta la última versión publicada en GitHub y, si es más nueva que la tuya, descarga el `.exe` y **solo lo
  instala si su SHA-256 coincide** con el `checksums.txt` de esa versión. Si no coincide, no cambia nada.
- Reemplaza el `mapadisco.exe` que estás usando, aunque esté abierto en otra ventana. Para comprobarlo, ejecuta
  `mapadisco -version`.
- Si ya tienes la última versión, solo lo dice. Una versión de desarrollo (compilada a mano) no se actualiza sola.
- Es el único momento en que mapadisco se conecta a internet, y solo cuando tú lo ejecutas: no busca versiones nuevas
  por su cuenta.

Si tu versión no conoce `-update` (responde *"flag provided but not defined: -update"*), es anterior a esta función:
actualiza **esa vez** repitiendo la instalación (la línea de PowerShell de la opción A, o descarga el `.exe` nuevo y
ábrelo). Desde entonces basta con `mapadisco -update`.

### Si Windows muestra una advertencia

Como el ejecutable no tiene firma digital comercial, Windows SmartScreen o el antivirus pueden avisar la primera vez:

- **"Windows protegió su PC"** → *Más información* → *Ejecutar de todas formas*.
- Si el antivirus lo pone en cuarentena, agrega una excepción para la carpeta de instalación o consulta con el área de Tecnología.

---

## 4. Primer uso

Abre una terminal nueva y escribe:

```
mapadisco
```

Por defecto analiza `C:\`. Mientras trabaja verás una línea de progreso:

```
  / Escaneando... 412.380 archivos · 98.211 carpetas · 201,4 GB · 12s
```

Al terminar aparece el reporte completo. Si la salida es larga, sube con la rueda del ratón o usa
`mapadisco | more` para verla página por página.

### Ejemplo de reporte (datos ficticios)

```
mapadisco v1.2.0  ·  analizando C:\  ·  24 hilos  ·  usuario normal
Escaneo completo en 4.2s: 650.120 archivos, 140.300 carpetas, 60 enlaces/junctions omitidos (evita contar doble)

── RESUMEN DEL DISCO C:\ ──────────────────────────────────────────────
  Capacidad          476,3 GB
  Usado              301,8 GB  ███████████████████░░░░░░░░░░░
  Libre              174,5 GB
  Encontrado         290,2 GB  (lo que este escaneo pudo leer)
  No visible          11,6 GB  ← espacio usado que el escaneo no pudo ver

── CARPETAS MÁS PESADAS (profundidad 3, >= 1,0 GB) ────────────────────
    290,2 GB   100%  C:\
    150,0 GB    52%  ├─ Users  ██████████░░░░░░░░░░
    150,0 GB    52%  │  └─ Usuario  ██████████░░░░░░░░░░
    120,4 GB    41%  │     ├─ AppData  ████████░░░░░░░░░░░░
    112,9 GB    39%  │     │  └─ Local  ███████░░░░░░░░░░░░░
     40,2 GB    14%  │     │     ├─ wsl  ██░░░░░░░░░░░░░░░░░░
     35,7 GB    12%  │     │     ├─ NVIDIA  ██░░░░░░░░░░░░░░░░░░
    ...

── SOSPECHOSOS HABITUALES ─────────────────────────────────────────────
     40,2 GB  Discos virtuales de WSL
              C:\Users\Usuario\AppData\Local\wsl\{...}\ext4.vhdx
              → No se reducen solos: wsl --shutdown y compactar con Optimize-VHD o diskpart
     35,7 GB  Caché de shaders (GPU / DirectX)
              ...
```

---

## 5. Cómo leer el reporte

### 5.1 Resumen del disco

Solo aparece cuando analizas una unidad completa (`C:\`, `D:\`…).

| Línea | Significado |
|---|---|
| **Capacidad / Usado / Libre** | Lo mismo que muestra Windows en *Este equipo*. |
| **Encontrado** | Lo que mapadisco pudo sumar recorriendo carpetas. |
| **No visible** | `Usado − Encontrado`. Espacio que Windows da por ocupado pero el escaneo no pudo ver. Solo se muestra si supera 1 GB. |

El espacio **no visible** suele estar en:

- `System Volume Information`: puntos de restauración e instantáneas de volumen (*shadow copies*). Es lo más común cuando la diferencia es grande.
- Carpetas protegidas o de otros usuarios del equipo.
- Metadatos del sistema de archivos NTFS (`$MFT`), normalmente de 1 a 3 GB.

Para reducir esa diferencia, ejecuta `mapadisco -admin` (ver [Modo Administrador](#9-modo-administrador)).

Si aparece *"El escaneo suma más que lo usado"*, no es un error: ver [Preguntas frecuentes](#12-preguntas-frecuentes).

También puede aparecer una línea de **archivos solo en la nube** (OneDrive con *Archivos a petición*): esos archivos se ven en el Explorador pero
no ocupan espacio local, por eso no se cuentan.

### 5.2 Carpetas más pesadas

```
    120,4 GB    41%  │     ├─ AppData  ████████░░░░░░░░░░░░
    ───┬────   ──┬─         ───┬───   ─────────┬──────────
     tamaño   % del         nombre    barra proporcional
     total    total                   al total escaneado
```

- **Colores:** rojo = 50 GB o más; amarillo = 10 GB o más.
- Solo se muestran carpetas de **1 GB o más** (ajustable con `-min`) y como máximo **12 por nivel** (ajustable con `-top`).
- **(archivos y carpetas menores):** lo que suman los archivos sueltos de esa carpeta y las subcarpetas que no se mostraron por ser pequeñas.
- **(sin acceso):** no se pudo leer por permisos; su tamaño real es desconocido.
- **Auto-expansión:** se muestran 3 niveles (`-depth`), pero cualquier carpeta que pese **5 % o más del total** se sigue abriendo
  (hasta 5 niveles extra). Así el árbol llega solo hasta el culpable aunque esté en algo como
  `Users\Usuario\AppData\Local\NVIDIA\DXCache`. Desactívalo con `-no-expand`.

### 5.3 Sospechosos habituales

Solo aparece al analizar una unidad completa. Revisa ubicaciones conocidas y muestra las que ocupan **100 MB o más**, ordenadas de mayor a menor.
Cada una indica las rutas encontradas y, en verde, una sugerencia (`→`).

| Sospechoso | Qué es |
|---|---|
| Caché de shaders (GPU / DirectX) | Programas de sombreado que la tarjeta gráfica compila al jugar. Puede crecer a decenas de GB. Se regenera sola. |
| Discos virtuales de WSL | El disco de tu Linux en Windows (`ext4.vhdx`). Crece al usarlo y **no se reduce solo** aunque borres archivos dentro de Linux. |
| Docker Desktop | Imágenes, contenedores y volúmenes de Docker, guardados en un `.vhdx`. |
| Hyper-V / VMs | Discos de máquinas virtuales. |
| Hibernación (`hiberfil.sys`) | Copia de la memoria RAM para hibernar y para el *inicio rápido*. |
| Archivo de paginación | `pagefile.sys` / `swapfile.sys`: memoria virtual de Windows. **No borrar.** |
| Puntos de restauración | Copias de seguridad del sistema en `System Volume Information`. |
| Windows.old | Instalación anterior de Windows tras una actualización grande. |
| Caché de Windows Update | Actualizaciones ya descargadas. |
| Almacén WinSxS | Componentes de Windows. Su tamaño aparece **inflado** (ver preguntas frecuentes). |
| Windows Installer (caché) | Necesario para desinstalar o reparar programas. **No borrar a mano.** |
| Optimización de distribución | Actualizaciones compartidas con otros equipos de la red. |
| Temporales | Del sistema y de tu usuario. |
| Volcados de memoria | Archivos de diagnóstico que quedan cuando un programa o Windows falla. |
| Papelera de reciclaje | Lo borrado que aún no vaciaste. |
| Steam / Epic / Xbox | Juegos instalados y restos de juegos desinstalados. |
| Android SDK / emuladores | SDK e imágenes de emuladores Android. |
| Cachés de desarrollo | npm, pnpm, yarn, pip, conda, NuGet, Gradle, Maven, Cargo, Composer. |
| Navegadores | Perfiles y caché de Chrome, Edge y Brave. |
| OneDrive | Archivos de OneDrive descargados en el equipo. |
| Descargas | Tu carpeta de Descargas: instaladores, `.zip` e `.iso` olvidados. |

Si un sospechoso aparece con **`?`** y *(sin acceso — usa -admin)*, existe pero hacen falta permisos de Administrador para medirlo.

### 5.4 Archivos más grandes

Los 25 archivos más grandes de todo el escaneo (ajustable con `-files`), con su ruta completa.
Es la forma más rápida de encontrar una `.iso`, un video o un disco virtual olvidado.

### 5.5 Peso por tipo de archivo

Suma por extensión (`.vhdx`, `.mp4`, `.dll`…) y cuántos archivos hay de cada una. Muestra 15 (ajustable con `-ext`).
Es útil para detectar patrones, por ejemplo cientos de GB en `.mp4` repartidos en muchas carpetas.

### 5.6 Carpetas sin acceso

Las carpetas de primeros niveles que no se pudieron leer (máximo 15 en pantalla; el total aparece en el título).
Su contenido **no se sumó** en ninguna sección. Con `-admin` la mayoría se vuelven legibles.

---

## 6. Opciones

```
mapadisco [opciones] [ruta]
```

> **Importante:** las opciones van **antes** de la ruta.
> ✔ `mapadisco -depth 4 D:\`  ✘ `mapadisco D:\ -depth 4` (la opción se ignora).

| Opción | Valor por defecto | Descripción |
|---|---|---|
| `ruta` | `C:\` | Unidad o carpeta a analizar. `D:` equivale a `D:\`. Acepta rutas relativas (`.` = carpeta actual). |
| `-depth N` | `3` | Niveles garantizados del árbol de carpetas. |
| `-min TAMAÑO` | `1GB` | Tamaño mínimo para que una carpeta aparezca en el árbol. Acepta `B`, `KB`, `MB`, `GB`, `TB` y decimales con coma o punto (`500MB`, `1,5GB`). |
| `-top N` | `12` | Máximo de subcarpetas por nivel del árbol. |
| `-no-expand` | desactivado | No bajar más allá de `-depth` aunque una carpeta pese mucho. |
| `-files N` | `25` | Cantidad de archivos más grandes a listar. |
| `-ext N` | `15` | Cantidad de tipos de archivo a listar. |
| `-csv ARCHIVO` | — | Exporta todas las carpetas de 10 MB o más a un CSV para Excel. Ver [sección 8](#8-exportar-a-excel-csv). |
| `-admin` | desactivado | Abre una ventana nueva como Administrador para ver carpetas protegidas. Ver [sección 9](#9-modo-administrador). |
| `-workers N` | automático | Hilos de lectura en paralelo. Automático = 3 × núcleos del procesador (entre 8 y 96). Bájalo (p. ej. `-workers 4`) en discos mecánicos (HDD) o de red. |
| `-no-color` | desactivado | Sin colores. También se respeta la variable de entorno `NO_COLOR`. |
| `-no-progress` | desactivado | Oculta la línea de progreso (del escaneo y, con `-clean`, la de vaciar la Papelera). |
| `-pause` | desactivado | Espera Enter antes de cerrar. Útil desde un acceso directo. Al abrirlo con doble clic se activa solo. |
| `-install` | — | Instala mapadisco para tu usuario sin preguntar y sale. Ver [sección 3](#3-instalación). |
| `-uninstall` | — | Desinstala mapadisco: lo quita del `PATH` y borra su carpeta. Ver [sección 14](#14-desinstalación). |
| `-no-install` | desactivado | No ofrece instalarse; para usarlo portable sin que pregunte cada vez. |
| `-update` | — | Actualiza a la última versión publicada, verificando su SHA-256. Ver [Actualizar](#actualizar-a-una-versión-nueva). |
| `-check` | desactivado | Con `-update`: solo consulta si hay una versión nueva, sin instalarla. |
| `-ram` | — | Diagnóstico de memoria RAM: en qué se va, si falta de verdad y qué hacer. Ver [sección 11](#11-memoria-ram). |
| `-wslconfig` | desactivado | Con `-ram`: propone limitar la memoria de WSL en `.wslconfig` y lo guarda si confirmas. |
| `-clean` | — | Libera espacio: mide lo que se puede limpiar, pregunta qué borrar y lo borra. Ver [sección 10](#10-liberar-espacio). |
| `-yes` | desactivado | Con `-clean`: limpia todas las categorías sin preguntar. Con `-ram -wslconfig`: guarda sin preguntar. |
| `-only LISTA` | todas | Con `-clean`: solo esas categorías, separadas por coma: `shaders`, `temp`, `temp-sistema`, `papelera`, `volcados`, `dev`. |
| `-version` | — | Muestra la versión y sale. |
| `-h` | — | Muestra la ayuda. |

Las opciones también se pueden escribir con dos guiones (`--depth 4`) y con `=` (`-min=500MB`).

### Códigos de salida

Útiles si usas mapadisco desde un script:

| Código | Significado |
|---|---|
| `0` | Terminó correctamente. |
| `1` | Error al exportar el CSV, al pedir permisos de Administrador, al instalar o desinstalar, o al actualizar (sin conexión, SHA-256 distinto…). |
| `2` | Opción inválida (p. ej. `-min mucho`, una categoría desconocida en `-only`, `-yes` sin `-clean` ni `-wslconfig`, `-check` sin `-update`, `-wslconfig` sin `-ram`) o la ruta no existe. |

Con `-clean`, que Windows no deje borrar algunos archivos **no** es un error: se informa y el código sigue siendo `0`.

---

## 7. Ejemplos por caso de uso

**"Mi disco C: está lleno y no sé por qué"**

```
mapadisco
```

Mira primero *Sospechosos habituales* y luego el árbol.

**Ver todo, incluidas las carpetas protegidas**

```
mapadisco -admin
```

**Más detalle: carpetas desde 500 MB y un nivel más**

```
mapadisco -depth 4 -min 500MB
```

**Analizar solo tu carpeta de usuario**

```
mapadisco %USERPROFILE%          (en cmd)
mapadisco $env:USERPROFILE       (en PowerShell)
```

**Analizar otro disco**

```
mapadisco D:
```

**Buscar los 100 archivos más grandes**

```
mapadisco -files 100
```

**Guardar el reporte en un archivo de texto**

```
mapadisco -no-color -no-progress > reporte.txt
```

**Exportar para analizar en Excel**

```
mapadisco -csv "%USERPROFILE%\Desktop\disco.csv"
```

**"Mi equipo está lento / tengo poca RAM disponible"**

```
mapadisco -ram
mapadisco -ram -wslconfig      (si usas WSL o Docker)
```

**Liberar espacio**

```
mapadisco -clean
mapadisco -clean -admin        (incluye los temporales de Windows)
```

**Disco mecánico (HDD) o unidad de red**

```
mapadisco -workers 4 E:\
```

---

## 8. Exportar a Excel (CSV)

```
mapadisco -csv reporte.csv
```

Crea un archivo con **todas** las carpetas de 10 MB o más (no solo las que caben en pantalla). Está pensado para Excel en español:
separador `;`, coma decimal y codificación UTF-8 con BOM, para que las tildes se vean bien al abrirlo con doble clic.

| Columna | Contenido |
|---|---|
| `Ruta` | Ruta completa de la carpeta. |
| `Bytes` | Tamaño total en bytes (incluye subcarpetas). |
| `GB` | El mismo tamaño en GB, con 3 decimales. |
| `Archivos` | Cantidad de archivos dentro (incluye subcarpetas). |
| `Nivel` | Profundidad: 0 = raíz, 1 = primer nivel, etc. |
| `SinAcceso` | `sí` si la carpeta no se pudo leer. |

Ideas en Excel: filtrar por `Nivel = 2` para ver las carpetas de segundo nivel, ordenar por `GB` de mayor a menor,
o buscar `node_modules` en `Ruta` para ver cuánto ocupan en total.

---

## 9. Modo Administrador

Sin permisos de Administrador, Windows no deja leer algunas carpetas (`System Volume Information`, `Program Files\WindowsApps`,
perfiles de otros usuarios…). Su contenido queda como **no visible** en el resumen.

```
mapadisco -admin
```

1. Windows mostrará el aviso de Control de cuentas de usuario (UAC): acepta.
2. Se abre **una ventana nueva** con el análisis completo. La ventana original solo dice *"Se abrió una nueva ventana como Administrador."*
3. Al terminar, la ventana nueva espera a que presiones Enter para cerrarse.

Todas las demás opciones que escribas se pasan a la ventana nueva (por ejemplo `mapadisco -admin -depth 4`).

Aun como Administrador, unas pocas carpetas del sistema siguen bloqueadas y los metadatos de NTFS no se pueden medir, así que
un pequeño espacio *no visible* (1 a 3 GB) es normal.

---

## 10. Liberar espacio

### Automático: `mapadisco -clean`

```
mapadisco -clean
```

1. **Mide** lo que se puede liberar y lo muestra numerado (datos de ejemplo):

   ```
   ── LIBERAR ESPACIO ────────────────────────────────────────────────
       1     35,7 GB  Caché de shaders (GPU / DirectX)
                      se regenera sola; los primeros minutos de cada juego pueden ir algo más lentos
       2      2,1 GB  Temporales de tu usuario
                      archivos temporales que dejan los programas
       3    700,0 MB  Papelera de reciclaje
                      lo que ya borraste; después no se puede recuperar
             38,5 GB  en total

   ¿Qué limpio? [t = todo · 1,3 = solo esos · Enter = nada]:
   ```

2. **Tú eliges:** `t` limpia todo, `1,3` solo esas categorías y Enter no borra nada.
3. **Limpia** y te dice cuánto liberó en cada una, y el espacio libre del disco antes y después:

   ```
     ✓ Caché de shaders (GPU / DirectX): 35,7 GB liberados (2.140 archivos)
     ✓ Papelera de reciclaje: 700,0 MB liberados (52 elementos)

     Espacio libre en C:\ 174,5 GB → 210,9 GB (+36,4 GB)
   ```

4. Al final muestra **lo que se puede liberar a mano** (ver abajo).

Qué categorías limpia:

| Categoría (`-only`) | Qué borra | ¿Seguro? |
|---|---|---|
| Caché de shaders (`shaders`) | Contenido de `NVIDIA\DXCache`, `NVIDIA\GLCache`, `D3DSCache`, `AMD\DxCache`, `AMD\DxcCache`, `Intel\ShaderCache` en `%LOCALAPPDATA%` | Sí: se regenera al jugar. |
| Temporales de tu usuario (`temp`) | Contenido de `%TEMP%` | Sí: los que estén en uso se saltan. |
| Temporales de Windows (`temp-sistema`) | Contenido de `C:\Windows\Temp` | Sí. Requiere Administrador. |
| Papelera de reciclaje (`papelera`) | Vacía la Papelera de todas las unidades | Sí, pero **no se puede deshacer**. |
| Volcados e informes de errores (`volcados`) | `CrashDumps`, informes de errores de Windows (WER), `Minidump`, `MEMORY.DMP` | Sí. Parte requiere Administrador. |
| Cachés de desarrollo (`dev`) | Contenido de las cachés de npm, pip y yarn | Sí: se vuelven a descargar cuando hacen falta. |

Variantes:

```
mapadisco -clean -admin                 # como Administrador: incluye Temporales de Windows y volcados del sistema
mapadisco -clean -yes                   # todo, sin preguntar (para scripts)
mapadisco -clean -only shaders,temp     # solo esas categorías
```

### Si Windows no deja borrar algo

mapadisco **no se detiene**: salta lo que no puede borrar, sigue con el resto y al final te dice por qué y a dónde ir:

```
  ⚠ Caché de shaders (GPU / DirectX): 34,9 GB liberados (2.100 archivos); Windows no dejó borrar 40:
    · 40 en uso por otro programa (820,0 MB) → cierra los programas abiertos o reinicia, y vuelve a ejecutar mapadisco -clean
      ej.: C:\Users\Usuario\AppData\Local\NVIDIA\DXCache\fc52a9389f221040.nvph
    → A mano: cierra los juegos y el panel de la tarjeta gráfica y vuelve a intentar; o Liberador de espacio (cleanmgr) > «Caché de sombreador DirectX»
    → Abrir la carpeta: explorer "C:\Users\Usuario\AppData\Local\NVIDIA\DXCache"
```

| Motivo | Qué hacer |
|---|---|
| **En uso por otro programa** | Cierra el programa (juego, navegador, el panel de la tarjeta gráfica…) o reinicia el equipo, y vuelve a ejecutar `mapadisco -clean`. |
| **Sin permiso** | Ejecuta `mapadisco -clean -admin`. |
| **Sigue sin poder** | Usa la ruta **A mano** que indica cada categoría, o abre la carpeta con el comando `explorer "…"` que muestra y borra su contenido tú mismo. |

### Al vaciar la Papelera

Mientras Windows vacía la Papelera, mapadisco muestra cuántos elementos van:

```
  / Vaciando la Papelera… 1.240 de 5.300 elementos · 12s
```

Si algún elemento necesita permiso de Administrador (por ejemplo, algo que borraste de `C:\Program Files`),
**Windows abre su propia ventana para pedirlo y espera tu respuesta**. Esa ventana puede quedar detrás de la
terminal o solo parpadear en la barra de tareas; si la Papelera deja de avanzar unos segundos, mapadisco te avisa:

```
  ⚠ La Papelera no avanza hace unos segundos. Si Windows pide permiso de Administrador,
    su ventana puede estar detrás de esta terminal o parpadeando en la barra de tareas: respóndela para seguir.
```

Si cancelas esa ventana, mapadisco te dice cuánto se liberó y cuánto quedó:

```
  ⚠ Papelera de reciclaje: 650,0 MB liberados (48 elementos); quedaron 4 elementos (50,0 MB)
    · se canceló la ventana de Windows que pedía permiso
    → Los que piden permiso de Administrador: mapadisco -clean -admin -only papelera
    → A mano: clic derecho en la Papelera del escritorio > Vaciar Papelera de reciclaje
```

### A mano: lo que mapadisco no borra por su cuenta

Al terminar, `mapadisco -clean` mide estas ubicaciones y muestra las que ocupan algo, con sus pasos. No las borra
porque son del sistema (y deben limpiarse con las herramientas de Windows) o porque son datos tuyos.

Antes de borrar, asegúrate de que no necesitas esos datos. En un equipo de la empresa, consulta con el área de
Tecnología antes de cambiar configuraciones del sistema. Las acciones son referenciales:

| Hallazgo | Acción sugerida | ¿Seguro? |
|---|---|---|
| Caché de shaders | *Liberador de espacio en disco* → marcar *Caché de sombreador DirectX*. En NVIDIA: Panel de control → Configuración 3D → *Tamaño de caché del sombreador*. | Sí, se regenera (los primeros juegos pueden ir algo más lentos al inicio). |
| Disco de WSL grande | Borrar lo que no uses dentro de Linux, luego en PowerShell: `wsl --shutdown` y compactar el `.vhdx` (`Optimize-VHD` con Hyper-V, o `diskpart` → `select vdisk file="…"` → `compact vdisk`). | Sí, si sigues los pasos con WSL apagado. Haz una copia antes si tienes datos importantes. |
| Docker | `docker system prune -a` (borra imágenes y contenedores sin uso) y luego compactar su `.vhdx`. | Revisa antes qué imágenes necesitas. |
| Juegos (Steam, Epic, Xbox) | Desinstalar desde su propio lanzador. Revisa que no queden carpetas en `steamapps\common`. | Sí. |
| Hibernación | Como Administrador: `powercfg /h off`. Desactiva la hibernación y el *inicio rápido*. | Sí, si no los usas. Se reactiva con `powercfg /h on`. |
| Puntos de restauración | *Configuración* → *Sistema* → *Información* → *Protección del sistema* → *Configurar* → limitar el uso máximo o eliminar puntos antiguos. | Pierdes la opción de volver a esos puntos. |
| Windows.old, Windows Update, temporales, optimización de distribución | *Configuración* → *Sistema* → *Almacenamiento* → *Archivos temporales*, o *Liberador de espacio en disco* → *Limpiar archivos del sistema*. | Sí. |
| WinSxS | Como Administrador: `Dism /Online /Cleanup-Image /StartComponentCleanup`. **Nunca** borrar a mano. | Sí con DISM; tarda varios minutos. |
| Papelera | Vaciar la papelera. | Sí, si no necesitas lo que hay. |
| Cachés de desarrollo | `npm cache clean --force`, `pnpm store prune`, `pip cache purge`, `conda clean -a`, `composer clear-cache`. | Sí, se descargan de nuevo cuando hacen falta. |
| Volcados de memoria | Borrar desde *Liberador de espacio* o manualmente. | Sí. |
| Descargas e `.iso` | Revisar y borrar lo que ya no necesites. | Tú decides. |
| `pagefile.sys`, `swapfile.sys`, `Windows\Installer` | No tocar. | — |

Después de liberar espacio, vuelve a ejecutar `mapadisco` para comprobar el resultado.

---

## 11. Memoria RAM

```
mapadisco -ram
```

Responde lo que el Administrador de tareas no explica: **en qué se va la memoria, si de verdad falta y qué hacer**.
Mide durante 2 segundos y muestra tres partes (datos de ejemplo):

```
── MEMORIA RAM ──────────────────────────────────────────────────────
  Total                15,7 GB
  En uso               13,6 GB  ████████████████████░░░░  (de eso, comprimida: 918,4 MB)
  Disponible            2,1 GB  = en caché 2,0 GB + libre 42,6 MB
  Confirmada           28,4 GB  de 41,7 GB (RAM + archivo de paginación)
  Kernel y drivers      2,3 GB  no paginada 1,5 GB · paginada 799,4 MB
  Presión                MEDIA  la memoria está justa: hoy alcanza, pero abrir algo grande hará que Windows use el disco
                                lecturas de memoria desde el disco: 196 páginas/s

── QUIÉN USA LA MEMORIA ─────────────────────────────────────────────
   Reservada      En RAM  Proc.  Aplicación
      5,5 GB      2,2 GB     22  WSL (Linux de Windows)
      2,7 GB      1,8 GB      9  Epic Games
      1,5 GB    977,4 MB     18  Microsoft Edge
      …

── QUÉ HACER ────────────────────────────────────────────────────────
  1. Lanzadores de juegos abiertos: Epic Games, Steam, EA app  (hasta ~5,4 GB)
     → Ciérralos desde el icono junto al reloj cuando no estés jugando …
  2. WSL (Linux de Windows) reserva 5,5 GB
     → Limita su memoria y haz que devuelva la que deja de usar: mapadisco -ram -wslconfig …
```

### Cómo leer los números

Son los mismos que muestra el Administrador de tareas (pestaña *Rendimiento > Memoria*):

| Número | Qué es |
|---|---|
| **En uso** | Lo que ocupan los programas y Windows. Entre paréntesis, cuánto de eso está **comprimido**: memoria poco usada que Windows comprime en vez de mandarla al disco. En el Administrador de tareas aparece como *"13,6 GB (918 MB)"*. |
| **En caché** | Archivos leídos hace poco, guardados por si se vuelven a pedir. **Se liberan al instante** si un programa necesita memoria: no es memoria perdida. |
| **Disponible** | Caché + libre: lo que se le puede dar a un programa ya. |
| **Confirmada** | Lo que los programas **reservaron**. Puede superar la RAM porque lo poco usado vive comprimido o en el archivo de paginación. |
| **Kernel y drivers** | Memoria de Windows y de los controladores. La *no paginada* siempre ocupa RAM; más de ~1 GB es alto. |
| **Presión** | Lo que de verdad importa: si Windows tiene que **leer memoria desde el disco** (páginas/s). |

Niveles de presión:

| Nivel | Cuándo | Qué significa |
|---|---|---|
| **BAJA** | Hay memoria disponible y casi no se lee del disco | La memoria se está aprovechando; no falta. |
| **MEDIA** | Disponible menor al 15 %, o reservada mayor al 75 % del límite, o lecturas del disco frecuentes | Hoy alcanza, pero abrir algo grande va a hacer que Windows use el disco. |
| **ALTA** | Muchas lecturas del disco (300+ páginas/s), o casi sin memoria y leyendo del disco, o reservada al 90 % del límite | El equipo se siente lento. Conviene cerrar algo ya. |

Un "Disponible" bajo **no es un problema por sí solo**: Windows usa la RAM libre como caché a propósito. Es un
problema cuando la presión sube.

### Quién usa la memoria

Agrupa los procesos por aplicación (los 18 procesos de Edge cuentan como "Microsoft Edge"):

- **Reservada:** lo que pidió el programa, esté en RAM, comprimido o en el disco. Es lo que se libera al cerrarlo.
- **En RAM:** lo que ocupa ahora mismo en la memoria física.

Aplicaciones que reconoce: WSL, Docker Desktop, navegadores (Edge, Chrome, Brave, Firefox, Opera), lanzadores de juegos
(Steam, Epic, EA, Battle.net, Ubisoft, Riot, Xbox), editores y otras apps comunes, y los servicios de Windows.

### Qué hacer

Las sugerencias aparecen según lo que encuentra, ordenadas por cuánta memoria liberarían:

| Si encuentra… | Sugiere |
|---|---|
| **WSL** sin límite de memoria | Limitarlo y que devuelva la memoria que deja de usar: `mapadisco -ram -wslconfig`. Si no estás usando Linux ni Docker, `wsl --shutdown` libera todo al instante. |
| **Lanzadores de juegos** abiertos | Cerrarlos desde el icono junto al reloj; si arrancan con Windows, desactivarlos en *Aplicaciones de arranque*. |
| **Varios navegadores** a la vez | Usar uno solo y activar las pestañas en suspensión / ahorro de memoria. |
| **Docker Desktop** abierto | Cerrarlo si no lo usas (sus contenedores corren dentro de WSL). |
| Memoria de **drivers** alta | Actualizar drivers y antivirus; si crece día a día, avisar al área de Tecnología. |
| Cerca del **límite** de memoria | Cerrar programas; no reducir el archivo de paginación. |

### Limitar la memoria de WSL: `mapadisco -ram -wslconfig`

Sin configuración, WSL (el Linux de Windows, que también usa Docker Desktop) puede tomar **hasta la mitad de tu RAM**
y no la devuelve sola. Este comando:

1. Muestra el `.wslconfig` que propone (en `C:\Users\<tu usuario>\.wslconfig`):

   ```ini
   [wsl2]
   memory=6GB

   [experimental]
   autoMemoryReclaim=gradual
   ```

   - `memory`: el máximo de RAM para WSL (~3/8 de tu RAM, entre 4 y 8 GB).
   - `autoMemoryReclaim=gradual`: Linux devuelve a Windows la memoria que deja de usar.
2. Pregunta `¿Guardar este .wslconfig? [s/N]` (Enter = no). Con `-yes` guarda sin preguntar.
3. Si ya tenías un `.wslconfig`, **solo agrega o cambia esas dos claves** y deja el resto igual; guarda una copia en
   `.wslconfig.bak`. Si ya tenías un límite propio, lo respeta.
4. Se aplica la próxima vez que arranque WSL. Para aplicarlo ya: `wsl --shutdown`. **Ojo:** cierra las terminales de
   Linux y Docker; hazlo cuando no los estés usando.

### Lo que `-ram` no hace (y por qué)

No tiene un botón de "liberar RAM". Los "liberadores de RAM" vacían la memoria de los programas: Windows manda esos
datos al disco y, en cuanto el programa los vuelve a usar, los lee de nuevo, así que **el equipo queda más lento**.
Vaciar la caché tampoco ayuda: es memoria disponible que acelera lo que abres. La memoria se libera de verdad
**cerrando lo que no usas** o **limitando lo que reserva de más** (como WSL), que es lo que sugiere mapadisco.

---

## 12. Preguntas frecuentes

**¿Por qué el Explorador muestra otro tamaño para una carpeta?**
El Explorador muestra el tamaño lógico. mapadisco usa el espacio real en disco para los archivos comprimidos o dispersos, no cuenta los
archivos de OneDrive que solo están en la nube y no sigue accesos de tipo *junction*. Las diferencias pequeñas son normales.

**¿Por qué `C:\Windows` aparece más grande de lo que debería?**
Windows usa *enlaces duros*: el mismo archivo aparece en `WinSxS` y en `System32` pero ocupa espacio una sola vez. mapadisco lo cuenta en
ambos lugares, así que `Windows` (y sobre todo `WinSxS`) se ve inflado. Por la misma razón, a veces el resumen indica que el escaneo suma más que lo usado.

**¿Qué son los "enlaces/junctions omitidos"?**
Accesos que apuntan a otra carpeta, como `C:\Archivos de programa` → `C:\Program Files` o `Documents and Settings` → `Users`.
mapadisco no entra en ellos para no contar lo mismo dos veces.

**Borré archivos dentro de WSL (o Docker) pero el disco de Windows no recuperó espacio.**
Es normal: el archivo `.vhdx` crece pero no se encoge solo. Hay que compactarlo (ver la [sección 10](#10-liberar-espacio)).

**Desinstalé juegos y el espacio no volvió.**
Revisa *Sospechosos habituales*: la caché de shaders de la tarjeta gráfica, los restos en `steamapps` y la papelera suelen quedar.

**¿Las unidades son GB o GiB?**
Binarias (1 GB = 1024³ bytes), igual que el Explorador de Windows, para que los números coincidan.

**¿Puede dañar algo?**
El análisis, no: solo lee la estructura de carpetas, no abre el contenido de los archivos ni escribe nada (salvo el CSV
si lo pides). `-clean` sí borra, pero solo cachés, temporales, la Papelera y volcados de errores, que Windows y los
programas regeneran o no necesitan; nunca sigue accesos directos de carpeta (*junctions*) hacia otros lugares y nunca
borra la carpeta de la categoría, solo su contenido. Lo único sin vuelta atrás es la Papelera: revísala antes si dudas.

**¿Qué pasa si un juego o programa está abierto mientras limpio?**
Windows no deja borrar los archivos que ese programa tiene abiertos. mapadisco los salta, borra el resto y te dice
cuántos quedaron y qué hacer: cerrar el programa (o reiniciar) y volver a ejecutar `mapadisco -clean`.

**¿Funciona en discos externos, USB o de red?**
Sí: `mapadisco E:\` o `mapadisco \\servidor\carpeta`. En unidades lentas usa `-workers 4`. El resumen del disco y los sospechosos
habituales solo aparecen al analizar la raíz de una unidad con letra.

---

## 13. Solución de problemas

| Problema | Solución |
|---|---|
| `"mapadisco" no se reconoce como un comando…` | Cierra y abre una terminal nueva después de instalar. Si sigue igual, vuelve a instalar (ver [sección 3](#3-instalación)) o usa la ruta completa: `%LOCALAPPDATA%\Programs\mapadisco\mapadisco.exe`. |
| Pregunta *"¿Instalar mapadisco ahora?"* cada vez | Lo estás ejecutando desde la carpeta de descargas, no desde la instalación. Instálalo (Enter) y usa `mapadisco`, o agrega `-no-install` si lo quieres portable. |
| La línea de PowerShell dice *"el SHA-256 no coincide"* | La descarga llegó incompleta o alterada y no se instaló nada. Vuelve a intentarlo; si se repite, avisa al área de Tecnología. |
| *"La ejecución de scripts está deshabilitada"* | La línea `irm … \| iex` no ejecuta un archivo de script, así que no debería pasar. Si tu equipo lo bloquea igual, usa la [opción B](#opción-b--descargar-el-exe-y-abrirlo). |
| La ventana se cierra sola al abrir el `.exe` | No debería pasar: con doble clic espera Enter antes de cerrarse. Si pasa, ábrelo desde una terminal para ver el mensaje. |
| Aparecen símbolos raros como `←[33m` | Tu terminal no interpreta colores (cmd antiguo). Usa Windows Terminal o agrega `-no-color`. |
| Las barras `█` o las líneas `├─` se ven como `?` | Usa Windows Terminal, o una fuente como Cascadia Mono o Consolas. |
| `-admin` no abre nada | Acepta el aviso de UAC; si lo rechazaste se muestra *"No se pudo elevar"*. Tu cuenta debe tener permisos de Administrador. |
| `no existe o no es una carpeta` | Revisa la ruta. Si tiene espacios, ponla entre comillas: `mapadisco "D:\Mis juegos"`. |
| Una opción no hace efecto | Ponla **antes** de la ruta (ver [sección 6](#6-opciones)). |
| Muy lento en un disco mecánico | Usa `-workers 4`. |
| `-ram -wslconfig` guardó el archivo pero WSL sigue usando lo mismo | El cambio se aplica al reiniciar WSL: `wsl --shutdown` (cierra Linux y Docker) y vuelve a abrirlos. |
| Tras `-ram -wslconfig`, algo en Linux o Docker va lento o se queda sin memoria | El límite quedó corto para lo que haces: sube `memory=` en `.wslconfig` (o restaura `.wslconfig.bak`) y `wsl --shutdown`. |
| `-update` dice *"No se pudo consultar la última versión"* | Revisa la conexión a internet. En redes con proxy, GitHub puede estar bloqueado: consulta con el área de Tecnología o actualiza descargando el `.exe` a mano (opción B de la [sección 3](#3-instalación)). |
| *"flag provided but not defined: -update"* | Tu versión es anterior a `-update`: actualiza esa vez reinstalando (ver [Actualizar](#actualizar-a-una-versión-nueva)). |
| `-update` dice *"el SHA-256 … no coincide"* | La descarga llegó incompleta o alterada y no se instaló nada. Vuelve a intentarlo; si se repite, avisa al área de Tecnología. |
| `-clean` dice *"Windows no dejó borrar…"* | Ver [Si Windows no deja borrar algo](#si-windows-no-deja-borrar-algo). |
| `-clean` no muestra *Temporales de Windows* o los deja a medias | Requieren Administrador: `mapadisco -clean -admin`. |
| `-clean` se queda en *"Vaciando la Papelera…"* sin avanzar | Windows está esperando que respondas su ventana de permiso: búscala detrás de la terminal o en la barra de tareas. Ver [Al vaciar la Papelera](#al-vaciar-la-papelera). |
| El antivirus lo bloquea | Ver [Si Windows muestra una advertencia](#si-windows-muestra-una-advertencia). |

---

## 14. Desinstalación

En cualquier terminal:

```
mapadisco -uninstall
```

```
  ✓ Carpeta quitada del PATH de tu usuario
  ✓ La carpeta C:\Users\Usuario\AppData\Local\Programs\mapadisco se borrará en unos segundos, al cerrar mapadisco
```

Quita la carpeta de tu `PATH` (el resto del `PATH` queda exactamente como estaba) y borra
`%LOCALAPPDATA%\Programs\mapadisco`. Como Windows no deja borrar un programa mientras se ejecuta, la carpeta se borra
unos segundos después de que mapadisco termina. Si mapadisco está abierto en otra ventana, ciérrala antes.

mapadisco no deja otros archivos, servicios ni entradas en el registro. Los CSV que hayas exportado quedan donde los guardaste.
