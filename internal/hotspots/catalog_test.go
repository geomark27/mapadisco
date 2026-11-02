package hotspots

import (
	"path"
	"strings"
	"testing"
)

// El catálogo es datos editados a mano: estas pruebas evitan errores al ampliarlo.
func TestCatalogoValido(t *testing.T) {
	seen := map[string]bool{}
	for _, h := range Catalog {
		if strings.TrimSpace(h.Label) == "" {
			t.Errorf("sospechoso sin nombre: %+v", h)
		}
		if seen[h.Label] {
			t.Errorf("nombre duplicado: %q", h.Label)
		}
		seen[h.Label] = true
		if len(h.Patterns) == 0 {
			t.Errorf("%q no tiene rutas", h.Label)
		}
		for _, p := range h.Patterns {
			switch {
			case strings.Contains(p, "/"):
				t.Errorf("%q: usa \\ en lugar de / en %q", h.Label, p)
			case strings.HasPrefix(p, `\`) || (len(p) > 1 && p[1] == ':'):
				t.Errorf("%q: la ruta debe ser relativa a la unidad: %q", h.Label, p)
			}
			// path.Match usa la misma sintaxis de comodines que filepath.Glob.
			if _, err := path.Match(strings.ReplaceAll(p, `\`, "/"), ""); err != nil {
				t.Errorf("%q: patrón inválido %q: %v", h.Label, p, err)
			}
		}
	}
}
