# Versionado y publicación

Un release de mapadisco es un **tag `vMAJOR.MINOR.PATCH` sobre `main`** y un **GitHub Release** con **un solo
ejecutable**, `mapadisco-windows-amd64.exe`, y su `checksums.txt`. Lo publica el autor con `make promote` y `make release`, igual que en draco-hq, después de cerrar la bitácora de la
versión con devlog (`make release-prep`).

## Versionado

### De dónde sale la versión

**El tag de git es la única fuente de verdad.** No hay archivo `VERSION` ni números escritos en el código:

| Compilación | Versión que muestra `mapadisco -version` |
|---|---|
| `make release` / `make release-build ver=v1.2.0` | `v1.2.0` |
| `make package` o `make run` en desarrollo | `git describe --tags --always --dirty`: en `main`, `v0.2.0-3-gabc1234` (3 commits después del tag); en `develop` y sus ramas, el hash corto (`abc1234-dirty`), porque el tag vive en el *merge commit* de `main`, que `develop` no contiene |
| `go build` a mano sin `-ldflags` | `dev` |

Se inyecta con `-ldflags "-X main.version=<versión>"` (objetivo `package` del Makefile).

### Qué número subir

[SemVer](https://semver.org/lang/es/) aplicado a lo que el usuario usa: opciones, salida, CSV, códigos de salida
e instalación. El `bump` lo decide el autor en cada release; esta tabla es la guía:

| Bump | Cuándo | Ejemplos |
|---|---|---|
| `patch` (v1.2.**3**) | Correcciones sin cambiar cómo se usa. | Un tamaño mal calculado, un texto, más velocidad, un fallo con rutas largas. |
| `minor` (v1.**3**.0) | Algo nuevo y compatible. | Una opción nueva, una sección del reporte, un sospechoso habitual, exportar a JSON. |
| `major` (v**2**.0.0) | Rompe lo que alguien ya usa. | Quitar o renombrar una opción, cambiar columnas o separador del CSV, cambiar códigos de salida. |

Si un release junta varias actividades, manda la mayor: un `minor` y dos `patch` → `minor`.

#### Mientras la versión sea `v0.x`

El proyecto empezó en `v0.1.0` (decisión del autor). Mientras la versión mayor sea `0`, la herramienta todavía no se
considera estable y se sigue la práctica habitual de SemVer para esa etapa:

| Bump | En `v0.x` | Ejemplo |
|---|---|---|
| `patch` | Correcciones | v0.2.0 → v0.2.1 |
| `minor` | Algo nuevo, **o algo que rompe** lo que ya se usa | v0.2.1 → v0.3.0 |
| `major` | Solo cuando el autor declare la herramienta estable | → v1.0.0 |

## Requisitos

- `gh` autenticado (`gh auth login`) con permiso de push sobre `origin`.
- GNU Make, Go 1.26, `sha256sum` (o `shasum`).

## Pasos

### 0. Cerrar la bitácora de la versión

Como `main` solo recibe cambios por PR, el cierre de la bitácora es una actividad más, en su propia rama:

```bash
make start name=release-0-2-0 type=chore    # chore/release-0-2-0 desde develop al día
make release-prep bump=minor                # o ver=v0.2.0
```

`scripts/release-prep.sh`:

- solo corre en una rama `chore/release-…`;
- calcula la versión desde el último tag (igual que `make release`) y se detiene si su ficha ya existe;
- muestra qué entradas `unreleased` entran (`close-release.sh --dry-run` de devlog) y pregunta;
- con `s`, crea `docs/bitacora/releases/vX.Y.Z.md`, marca esas entradas con la versión y agrega la fila al índice.

No hace commits. Después: `make check`, commit `release: vX.Y.Z` y `make ship` (PR a `develop`).

### 1. Llevar `develop` a `main`

```bash
make promote
```

`scripts/promote.sh`:

- se detiene si `gh` no está autenticado, si hay cambios sin commitear o si `develop` local tiene commits que no están
  en `origin/develop`; si `main` ya tiene todo `develop`, no hace nada;
- muestra las actividades que llegan a `main` y crea el PR `develop → main` (o reutiliza el abierto) con esa lista;
- pregunta antes de fusionar; con `N` el PR queda abierto para revisarlo en GitHub;
- fusiona con *merge commit* (`gh pr merge --merge`), sin borrar `develop`;
- deja `main` y `develop` locales al día (solo *fast-forward*) y te deja en `main`.

### 2. Publicar desde `main`

```bash
make release ver=v0.2.0        # la versión que se cerró en el paso 0 (o bump=…, que da la misma)
```

`scripts/release.sh`:

- se detiene si `gh` no está autenticado, si no estás en `main`, si hay cambios sin commitear o si `main` no es igual
  a `origin/main`;
- calcula la versión desde el último tag (sin tags parte de `v0.0.0`) y exige que sea mayor y que el tag no exista;
- exige que la ficha `docs/bitacora/releases/<versión>.md` esté en `main` (paso 0);
- corre `make lint test` y `make release-build ver=<versión>`, que deja en `dist/`:

  ```
  dist/mapadisco-windows-amd64.exe    el programa, que se instala solo al abrirlo
  dist/checksums.txt                  SHA-256 de cada ejecutable
  ```

- pregunta antes de publicar; recién entonces crea el tag anotado, lo sube y crea el release con
  `gh release create --generate-notes --verify-tag`: las notas enlazan la ficha de la bitácora y suman los PR
  fusionados desde el tag anterior.

No crea commits ni sube ramas.

### 3. Volver a `develop`

```bash
git switch develop
```

El nombre del ejecutable **no lleva la versión** a propósito: así
`https://github.com/geomark27/mapadisco/releases/latest/download/mapadisco-windows-amd64.exe` apunta siempre al último
release, y `scripts/install.ps1` (la instalación de una línea) no cambia entre versiones. La versión está dentro del
programa (`mapadisco -version`) y en el tag.

## Verificar un release

1. En PowerShell, instala con la línea de la guía de usuario:
   `irm https://raw.githubusercontent.com/geomark27/mapadisco/main/scripts/install.ps1 | iex`.
   Debe terminar con `OK: SHA-256 verificado` y mostrar la versión publicada.
2. O a mano: descarga `mapadisco-windows-amd64.exe` y `checksums.txt` desde *Releases*, compara
   `Get-FileHash .\mapadisco-windows-amd64.exe` con la línea de `checksums.txt`, ábrelo y acepta instalarlo.
3. En una terminal nueva, `mapadisco -version` debe mostrar la versión publicada.

## Si algo falla

- **Antes de la pregunta de publicar** no se creó nada: corrige y vuelve a correr.
- **El tag se subió pero el release no se creó**: el script imprime el `gh release create` para reintentar con los
  mismos archivos de `dist/`.
- **Un release con un ejecutable malo**: no se reemplazan archivos a mano; se publica una versión nueva (`bump=patch`).

## Lo que no se hace

- Etiquetar fuera de `main` o desde un `main` que no es igual a `origin/main`.
- Subir ejecutables compilados fuera de `make release-build`, o editar `checksums.txt`.
- Marcar el release como *pre-release*: GitHub lo excluye de *latest*.
