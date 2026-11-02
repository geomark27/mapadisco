# Bitácora de cambios — {{PROJECT}}

Registro de **qué** cambiamos, **por qué** y **en qué versión salió**.
Los commits dicen *cómo*; la bitácora dice *por qué*.

## Estructura

```
{{DIR}}/
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
<!-- releases:end -->
