# Bitácora — mapadisco

Una entrada por actividad: cada rama `feature/<tema>` (o `fix/`, `docs/`) que llega a `develop` por PR. La entrada
viaja en el mismo commit que el cambio que describe. Formato y reglas en
[flujo-de-trabajo.md](../flujo-de-trabajo.md#bitácora).

La historia y el porqué de cada decisión van aquí, no en comentarios del código.

## Índice

La numeración es global y correlativa: las entradas se citan por número.

El commit inicial (`inital commit`, en `main`) se hizo antes de que existiera la bitácora: escaneo paralelo con
`FindFirstFileExW`, árbol, sospechosos habituales, archivos grandes, peso por tipo, CSV, modo Administrador,
instalador por usuario, guía de usuario y pruebas portables.

| # | Actividad | Rama | Fecha | Resumen |
|---|---|---|---|---|
| [001](./001-flujo-de-trabajo.md) | Flujo de trabajo y release | `feature/flujo-de-trabajo` | 2026-11-01 | Ramas `main → develop → feature/<tema>` con Makefile de draco-hq; versión desde el tag de git; release con zip y `checksums.txt`; `make check` y bitácora |
