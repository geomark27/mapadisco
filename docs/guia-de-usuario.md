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
10. [Qué hacer con lo que encuentres](#10-qué-hacer-con-lo-que-encuentres)
11. [Preguntas frecuentes](#11-preguntas-frecuentes)
12. [Solución de problemas](#12-solución-de-problemas)
13. [Desinstalación](#13-desinstalación)

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

### Lo que NO hace

- **No borra, mueve ni modifica nada del disco que analiza.** Es de solo lectura. Las sugerencias son eso: sugerencias que tú decides si aplicar.
  Lo único que escribe es su propia instalación (si aceptas instalarlo) y el CSV que pidas con `-csv`.
- No se conecta a internet ni envía información a ningún lado.
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

Repite la instalación con la versión nueva: la línea de PowerShell de la opción A, o descarga el `.exe` nuevo y
ábrelo. Reemplaza el ejecutable anterior aunque mapadisco esté abierto en otra ventana, y el `PATH` no se duplica.
Confírmalo con `mapadisco -version`.

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

Si aparece *"El escaneo suma más que lo usado"*, no es un error: ver [Preguntas frecuentes](#11-preguntas-frecuentes).

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
| `-no-progress` | desactivado | Oculta la línea de progreso. |
| `-pause` | desactivado | Espera Enter antes de cerrar. Útil desde un acceso directo. Al abrirlo con doble clic se activa solo. |
| `-install` | — | Instala mapadisco para tu usuario sin preguntar y sale. Ver [sección 3](#3-instalación). |
| `-uninstall` | — | Desinstala mapadisco: lo quita del `PATH` y borra su carpeta. Ver [sección 13](#13-desinstalación). |
| `-no-install` | desactivado | No ofrece instalarse; para usarlo portable sin que pregunte cada vez. |
| `-version` | — | Muestra la versión y sale. |
| `-h` | — | Muestra la ayuda. |

Las opciones también se pueden escribir con dos guiones (`--depth 4`) y con `=` (`-min=500MB`).

### Códigos de salida

Útiles si usas mapadisco desde un script:

| Código | Significado |
|---|---|
| `0` | Terminó correctamente. |
| `1` | Error al exportar el CSV, al pedir permisos de Administrador, o al instalar o desinstalar. |
| `2` | Opción inválida (p. ej. `-min mucho`) o la ruta no existe. |

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

## 10. Qué hacer con lo que encuentres

mapadisco **no borra nada**. Esta tabla resume acciones habituales; son referenciales. Antes de borrar, asegúrate de que no
necesitas esos datos. En un equipo de la empresa, consulta con el área de Tecnología antes de cambiar configuraciones del sistema.

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

## 11. Preguntas frecuentes

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
Es normal: el archivo `.vhdx` crece pero no se encoge solo. Hay que compactarlo (ver la [sección 10](#10-qué-hacer-con-lo-que-encuentres)).

**Desinstalé juegos y el espacio no volvió.**
Revisa *Sospechosos habituales*: la caché de shaders de la tarjeta gráfica, los restos en `steamapps` y la papelera suelen quedar.

**¿Las unidades son GB o GiB?**
Binarias (1 GB = 1024³ bytes), igual que el Explorador de Windows, para que los números coincidan.

**¿Puede dañar algo?**
No. Solo lee la estructura de carpetas: no abre el contenido de los archivos ni escribe nada, salvo el CSV si lo pides con `-csv`.

**¿Funciona en discos externos, USB o de red?**
Sí: `mapadisco E:\` o `mapadisco \\servidor\carpeta`. En unidades lentas usa `-workers 4`. El resumen del disco y los sospechosos
habituales solo aparecen al analizar la raíz de una unidad con letra.

---

## 12. Solución de problemas

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
| El antivirus lo bloquea | Ver [Si Windows muestra una advertencia](#si-windows-muestra-una-advertencia). |

---

## 13. Desinstalación

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
