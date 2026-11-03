package cleaner

import (
	"testing"
	"time"
)

func TestDrainProgress(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := NewDrain(100, t0)
	if d.Done() != 0 || d.Total() != 100 {
		t.Fatalf("al inicio: hecho %d de %d", d.Done(), d.Total())
	}
	d.Update(40, t0.Add(time.Second))
	if d.Done() != 60 {
		t.Errorf("hecho = %d, quiero 60", d.Done())
	}
	// Si se borra algo más mientras tanto, lo hecho no se vuelve negativo ni supera el total.
	d.Update(150, t0.Add(2*time.Second))
	if d.Done() != 0 {
		t.Errorf("con más que el total: hecho = %d, quiero 0", d.Done())
	}
	d.Update(-5, t0.Add(3*time.Second))
	if d.Done() != 100 {
		t.Errorf("con negativo: hecho = %d, quiero 100", d.Done())
	}
}

func TestDrainStalled(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := NewDrain(10, t0)
	if d.Stalled(t0.Add(StallAfter - time.Millisecond)) {
		t.Error("detenido antes de StallAfter")
	}
	if !d.Stalled(t0.Add(StallAfter)) {
		t.Error("sin avance durante StallAfter debería estar detenido")
	}
	// Un cambio reinicia la cuenta.
	d.Update(4, t0.Add(StallAfter))
	if d.Stalled(t0.Add(StallAfter + time.Second)) {
		t.Error("detenido justo después de avanzar")
	}
	// El mismo valor no cuenta como avance.
	d.Update(4, t0.Add(StallAfter+3*time.Second))
	if !d.Stalled(t0.Add(2 * StallAfter)) {
		t.Error("repetir el mismo valor no debería contar como avance")
	}
	// Vacío del todo nunca está detenido.
	d.Update(0, t0.Add(2*StallAfter))
	if d.Stalled(t0.Add(10 * StallAfter)) {
		t.Error("terminado no puede estar detenido")
	}
}
