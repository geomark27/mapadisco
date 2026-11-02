# Versionado y publicación

Un release de mapadisco es un **tag `vMAJOR.MINOR.PATCH` sobre `main`** y un **GitHub Release** con el zip
instalable y su `checksums.txt`. Lo publica el autor con `make promote` y `make release`, igual que en draco-hq.

## Versionado

### De dónde sale la versión

**El tag de git es la única fuente de verdad.** No hay archivo `VERSION` ni números escritos en el código:

| Compilación | Versión que muestra `mapadisco -version` |
|---|---|
| `make release` / `make release-build ver=v1.2.0` | `v1.2.0` |
| `make package` o `make run` en desarrollo | `git describe`: `v1.2.0-3-gabc1234` (3 commits después de v1.2.0), con `-dirty` si hay cambios sin commitear; sin tags, el hash corto |
| `go build` a mano sin `-ldflags` | `dev` |

Se inyecta con `-ldflags "-X main.version=<versión>"` (`scripts/package.sh`).

### Qué número subir

[SemVer](https://semver.org/lang/es/) aplicado a lo que el usuario usa: opciones, salida, CSV, códigos de salida
e instalación. El `bump` lo decide el autor en cada release; esta tabla es la guía:

| Bump | Cuándo | Ejemplos |
|---|---|---|
| `patch` (v1.2.**3**) | Correcciones sin cambiar cómo se usa. | Un tamaño mal calculado, un texto, más velocidad, un fallo con rutas largas. |
| `minor` (v1.**3**.0) | Algo nuevo y compatible. | Una opción nueva, una sección del reporte, un sospechoso habitual, exportar a JSON. |
| `major` (v**2**.0.0) | Rompe lo que alguien ya usa. | Quitar o renombrar una opción, cambiar columnas o separador del CSV, cambiar códigos de salida. |

Si un release junta varias actividades, manda la mayor: un `minor` y dos `patch` → `minor`.

## Requisitos

- `gh` autenticado (`gh auth login`) con permiso de push sobre `origin`.
- GNU Make, Go 1.26, `zip`, `sha256sum` (o `shasum`).

## Pasos

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
make release bump=minor        # o patch / major
make release ver=v1.1.0        # o una versión exacta
```

`scripts/release.sh`:

- se detiene si `gh` no está autenticado, si no estás en `main`, si hay cambios sin commitear o si `main` no es igual
  a `origin/main`;
- calcula la versión desde el último tag (sin tags parte de `v0.0.0`) y exige que sea mayor y que el tag no exista;
- corre `make lint test` y `make release-build ver=<versión>`, que deja en `dist/`:

  ```
  dist/mapadisco-v1.2.0-windows-amd64.zip    mapadisco.exe + instalar/desinstalar + README + GUIA-DE-USUARIO.md
  dist/checksums.txt                         SHA-256 de cada zip
  ```

- pregunta antes de publicar; recién entonces crea el tag anotado, lo sube y crea el release con
  `gh release create --generate-notes --verify-tag` (las notas salen de los PR fusionados desde el tag anterior).

No crea commits ni sube ramas.

### 3. Volver a `develop`

```bash
git switch develop
```

## Verificar un release

1. Descarga el zip y `checksums.txt` desde *Releases* en GitHub.
2. En PowerShell: `Get-FileHash .\mapadisco-v1.2.0-windows-amd64.zip` y compara con la línea de `checksums.txt`
   (mayúsculas y minúsculas dan igual).
3. Descomprime, ejecuta `instalar.cmd` y, en una terminal nueva, `mapadisco -version` debe mostrar la versión publicada.

## Si algo falla

- **Antes de la pregunta de publicar** no se creó nada: corrige y vuelve a correr.
- **El tag se subió pero el release no se creó**: el script imprime el `gh release create` para reintentar con los
  mismos archivos de `dist/`.
- **Un release con un zip malo**: no se reemplazan archivos a mano; se publica una versión nueva (`bump=patch`).

## Lo que no se hace

- Etiquetar fuera de `main` o desde un `main` que no es igual a `origin/main`.
- Subir zips compilados fuera de `make release-build`, o editar `checksums.txt`.
- Marcar el release como *pre-release*: GitHub lo excluye de *latest*.
