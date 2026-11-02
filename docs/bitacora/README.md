# Bitácora de cambios — mapadisco

Registro de **qué** cambiamos, **por qué** y **en qué versión salió**.
Los commits dicen *cómo*; la bitácora dice *por qué*.

## Estructura

```
docs/bitacora/
├── README.md            ← este índice
├── entry-template.md    ← plantilla de entrada
├── entries/             ← una por rama / PR
│   └── AAAA-MM-DD-<tipo>-<tema>.md
└── releases/            ← una por versión publicada (la genera el release)
    └── vX.Y.Z.md
```

## Reglas

1. Toda rama que se integre lleva **una** entrada. Sin entrada, no hay PR.
2. Una rama = una entrada = un dueño. Si la rama crece, se actualiza la misma entrada.
3. `version` empieza en `unreleased` y **nunca** se escribe a mano: la pone el release.
4. Las entradas publicadas no se reescriben (solo erratas). Si algo cambia, entrada nueva.
5. Sin secretos, tokens, URLs privadas ni datos personales reales.

Tipos (los prefijos de los commits): `feat`, `fix`, `chore`, `docs`, `refactor`, `perf`, `test`.

## Versiones publicadas

<!-- releases:start -->
| Versión | Fecha | Tipo | Entradas |
|---|---|---|---|
| [v0.5.0](releases/v0.5.0.md) | 2026-11-01 | minor | 1 |
| [v0.4.0](releases/v0.4.0.md) | 2026-11-01 | minor | 1 |
| [v0.3.0](releases/v0.3.0.md) | 2026-11-01 | minor | 1 |
| [v0.2.0](releases/v0.2.0.md) | 2026-11-01 | minor | 1 |
| [v0.1.0](releases/v0.1.0.md) | 2026-11-01 | minor | 1 |
<!-- releases:end -->

## En este proyecto

- Crear o ver la entrada de la rama: `make bitacora title="Título corto"`. Validarla: `make bitacora-check`
  (también la revisa `make check`).
- Cerrar una versión: en una rama `chore/release-X-Y-Z`, `make release-prep bump=…` (ver `docs/release.md`).
- Hasta la actividad `feat/exe-unico` la bitácora usaba el formato de draco-hq (`NNN-<tema>.md`); esa entrada se
  migró a `entries/` y la ficha de v0.1.0 está reconstruida (`reconstructed: true`).

