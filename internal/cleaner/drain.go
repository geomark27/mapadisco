package cleaner

import "time"

// StallAfter es cuánto tiempo sin avance se tolera antes de avisar. Al vaciar la Papelera, una pausa
// así suele ser Windows esperando una respuesta en otra ventana (p. ej. la que pide permiso de
// Administrador), que puede haber quedado detrás de la terminal.
const StallAfter = 5 * time.Second

// Drain sigue cómo baja una cantidad mientras otro la vacía (los elementos de la Papelera mientras
// Windows la vacía): cuánto va hecho y desde cuándo no se mueve.
type Drain struct {
	total   int64
	left    int64
	changed time.Time
}

// NewDrain empieza a seguir un total que debería bajar hasta 0.
func NewDrain(total int64, now time.Time) *Drain {
	return &Drain{total: total, left: total, changed: now}
}

// Update registra lo que queda. Cualquier cambio (también si sube, porque se borró algo más mientras
// tanto) cuenta como actividad.
func (d *Drain) Update(left int64, now time.Time) {
	if left != d.left {
		d.left = left
		d.changed = now
	}
}

// Done devuelve cuánto se vació, entre 0 y el total inicial.
func (d *Drain) Done() int64 { return min(max(d.total-d.left, 0), d.total) }

// Total devuelve el total inicial.
func (d *Drain) Total() int64 { return d.total }

// Stalled indica si lleva StallAfter o más sin cambios y aún queda algo.
func (d *Drain) Stalled(now time.Time) bool {
	return d.left > 0 && now.Sub(d.changed) >= StallAfter
}
