---
date: 2026-11-03
branch: fix/papelera-progreso
type: fix
author: Marcos Ramos
agents: claude
pr: pending
version: v0.5.1
---

# Avance y aviso de permisos al vaciar la Papelera

## Qué cambia

- **Avance en vivo** al vaciar la Papelera con `-clean`: `Vaciando la Papelera… 1.240 de 5.300 elementos · 12s`
  (en stderr, como el del escaneo; `-no-progress` lo oculta). Se consulta lo que queda cada segundo con
  `SHQueryRecycleBin` en otra goroutine, porque `SHEmptyRecycleBin` no informa nada mientras trabaja.
- **Aviso si no avanza** durante 5 s (`cleaner.StallAfter`): se explica que Windows puede estar esperando respuesta en
  su ventana de permiso, y que puede estar detrás de la terminal o parpadeando en la barra de tareas. Se avisa una vez.
- **La ventana de permiso se cuelga de la consola**: `SHEmptyRecycleBin` recibe `GetConsoleWindow()` como ventana
  dueña en vez de `0`, para que aparezca delante de la terminal. La llamada fija su hilo (`runtime.LockOSThread`)
  porque esa ventana es modal y corre en él.
- **Resultado real**: lo liberado se calcula midiendo la Papelera antes y después, no se da por vaciada entera. Si
  quedó algo (se canceló la ventana de permiso, `ERROR_CANCELLED` → `winapi.ErrCanceled`, o Windows falló a medias),
  se muestra `⚠ … quedaron N elementos (X)` con el motivo, `mapadisco -clean -admin -only papelera` (si no es
  Administrador) y la ruta a mano.
- Sin consola para el avance (salida redirigida o `-no-progress`), se imprime una línea fija antes de empezar con el
  mismo aviso sobre la ventana de permiso.
- **`internal/cleaner`**: `Drain` (portable, con pruebas) sigue lo que queda, lo hecho y si está detenido.
- Guía de usuario: nueva sección *Al vaciar la Papelera*, `-no-progress` y una fila en problemas frecuentes.

## Por qué

### Solicitud

El autor vació la Papelera con `-clean` y tenía elementos que necesitaban permiso de Administrador. La herramienta se
quedó mucho tiempo sin mostrar nada; solo al cambiar de ventana vio el cuadro de Windows que pedía el permiso.

### Motivación

| Problema | Daño real |
|---|---|
| `SHEmptyRecycleBin` se llamaba sin ventana dueña (`hwnd = 0`) | La ventana de permiso de Windows salía sin dueño y, como el proceso no tenía el foco, quedaba detrás de la terminal |
| Sin avance mientras se vacía | No se distinguía «está trabajando» de «está esperando que respondas» |
| Se informaba como liberado el tamaño inicial completo | Si se cancelaba el permiso, el resultado decía que se vació todo |

## Archivos y áreas

- `internal/winapi/recycle_windows.go`: ventana dueña, hilo fijo, `ErrCanceled`.
- `internal/cleaner/drain.go`, `drain_test.go`: seguimiento del avance y detección de pausa.
- `cmd/mapadisco/clean_windows.go`: `cleanRecycleBin`, `emptyRecycleBin`, `watchRecycleBin`; `runClean` y `cleanOne`
  reciben `live`.
- `cmd/mapadisco/main_windows.go`: pasa `live` (`!-no-progress` y stderr en consola).
- `docs/guia-de-usuario.md`.

## Cómo probarlo

- `make test pkg=./internal/cleaner run=Drain v=1` y `make lint`.
- A mano en Windows (sin Administrador):
  1. Borrar a la Papelera algunos archivos propios y uno de una carpeta protegida (p. ej. una copia en
     `C:\Program Files\Prueba`, creada como Administrador).
  2. `make run args="-clean -only papelera"` y elegir `t`.
  3. Comprobar la línea de avance, que la ventana de permiso aparezca delante de la terminal y, si no se responde, el
     aviso a los 5 s.
  4. Cancelar la ventana y comprobar el `⚠ … quedaron 1 elemento` con sus sugerencias.
  5. Repetir con `-no-progress` y con la salida redirigida (`> salida.txt`).

## Notas y pendientes

- **Sin verificar a mano en Windows** todavía: el comportamiento de la ventana de permiso depende de la terminal. Con
  Windows Terminal, `GetConsoleWindow()` devuelve la ventana de la pseudoconsola, que en versiones recientes es
  propiedad de la ventana de Windows Terminal; en versiones antiguas la ventana de permiso podría seguir quedando
  detrás. Para eso queda el aviso por pausa.
- Qué devuelve `SHEmptyRecycleBin` al cancelar la ventana de permiso no está documentado: se trata `ERROR_CANCELLED`
  aparte, pero el resultado se basa en medir antes y después, así que es correcto aunque devuelva otro código.
- El aviso por pausa puede salir también si Windows tarda en enumerar una Papelera muy grande antes de empezar a
  borrar; el texto dice «si Windows pide permiso», no afirma que lo esté pidiendo.
- Las demás categorías (`cleaner.Clean`) tampoco muestran avance; no se tocó en esta rama.
