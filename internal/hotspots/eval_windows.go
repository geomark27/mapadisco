package hotspots

import (
	"path/filepath"
	"sort"
	"strings"

	"mapadisco/internal/scanner"
	"mapadisco/internal/winapi"
)

// Evaluate revisa el catálogo contra la unidad escaneada (root debe ser la raíz de la unidad).
// Las carpetas se miden con el árbol ya escaneado; los archivos sueltos, con winapi.Stat.
// Devuelve los sospechosos con al menos minSize bytes, o sin acceso, de mayor a menor.
func Evaluate(root *scanner.Node, catalog []Hotspot, minSize int64) []Match {
	drive := root.Name
	var out []Match
	for _, h := range catalog {
		m := Match{Hotspot: h}
		for _, pat := range h.Patterns {
			for _, p := range expand(drive, pat) {
				size, denied := measure(root, p)
				if denied {
					m.Denied = true
				}
				if size > 0 {
					m.Size += size
					m.Paths = append(m.Paths, p)
				}
			}
		}
		if m.Size >= minSize || (m.Denied && m.Size == 0) {
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Size > out[j].Size })
	return out
}

// expand resuelve un patrón a rutas existentes.
func expand(drive, pat string) []string {
	full := filepath.Join(drive, pat)
	matches, _ := filepath.Glob(full)
	if len(matches) == 0 && !strings.ContainsAny(pat, "*?[") {
		// Glob falla con rutas protegidas (System Volume Information);
		// FindFirstFile sobre la ruta exacta sí las ve desde la carpeta padre.
		if _, ok := winapi.Stat(full); ok {
			matches = []string{full}
		}
	}
	return matches
}

// measure devuelve el tamaño de una ruta y si existe pero no se pudo leer.
func measure(root *scanner.Node, path string) (size int64, denied bool) {
	if n := scanner.Lookup(root, path); n != nil {
		return n.Total, n.Denied && n.Total == 0
	}
	fd, ok := winapi.Stat(path)
	if !ok {
		return 0, false
	}
	if fd.IsDir() {
		return 0, true // carpeta que el escaneo no pudo recorrer
	}
	return fd.Size(), false
}
