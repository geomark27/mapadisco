package cleaner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Resolve convierte rutas del catálogo en rutas reales existentes: expande las variables de
// entorno con getenv y los comodines. Una ruta con una variable vacía se descarta (nunca se
// convierte en una ruta relativa ni en la raíz del disco).
func Resolve(patterns []string, getenv func(string) string) []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range patterns {
		missing := false
		expanded := os.Expand(p, func(k string) string {
			v := getenv(k)
			if v == "" {
				missing = true
			}
			return v
		})
		if missing {
			continue
		}
		matches, _ := filepath.Glob(expanded)
		if len(matches) == 0 && !strings.ContainsAny(expanded, "*?[") {
			if _, err := os.Lstat(expanded); err == nil {
				matches = []string{expanded}
			}
		}
		for _, m := range matches {
			if key := strings.ToLower(m); !seen[key] {
				seen[key] = true
				out = append(out, m)
			}
		}
	}
	sort.Strings(out)
	return out
}

// Usage es lo que ocupa un conjunto de rutas.
type Usage struct {
	Bytes  int64
	Files  int64
	Denied bool // alguna carpeta no se pudo leer: el tamaño real puede ser mayor
}

// Measure suma lo que ocupan las rutas sin seguir enlaces ni junctions.
func Measure(paths []string) Usage {
	var u Usage
	for _, p := range paths {
		info, err := os.Lstat(p)
		if err != nil {
			u.Denied = u.Denied || errors.Is(err, fs.ErrPermission)
			continue
		}
		if !info.IsDir() {
			u.Bytes += info.Size()
			u.Files++
			continue
		}
		measureDir(p, &u)
	}
	return u
}

func measureDir(dir string, u *Usage) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		u.Denied = true
		return
	}
	for _, e := range entries {
		if isLink(e.Type()) {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			measureDir(p, u)
			continue
		}
		if info, err := e.Info(); err == nil {
			u.Bytes += info.Size()
			u.Files++
		}
	}
}

// Result es lo que pasó al limpiar.
type Result struct {
	Freed      int64 // bytes de archivos borrados
	Deleted    int64 // archivos borrados
	InUse      int64 // archivos que Windows no dejó borrar porque un programa los tiene abiertos
	InUseBytes int64
	Denied     int64 // archivos o carpetas sin permiso
	Failed     int64 // otros errores
	Links      int64 // enlaces o junctions omitidos (nunca se siguen)
	Examples   []string
}

// Opposed indica si Windows se opuso a borrar algo.
func (r *Result) Opposed() bool { return r.InUse+r.Denied+r.Failed > 0 }

const maxExamples = 3

func (r *Result) example(path string) {
	if len(r.Examples) < maxExamples {
		r.Examples = append(r.Examples, path)
	}
}

// Options ajusta la limpieza a la plataforma.
type Options struct {
	// InUse reconoce el error de "archivo en uso por otro proceso" (en Windows, ERROR_SHARING_VIOLATION).
	InUse func(error) bool
}

// Clean borra el contenido de cada carpeta de paths (no la carpeta) o el archivo indicado.
// Sigue adelante ante cada error y lo cuenta: un archivo en uso no detiene la limpieza.
func Clean(paths []string, opt Options) (*Result, error) {
	for _, p := range paths {
		if err := checkSafe(p); err != nil {
			return nil, err
		}
	}
	r := &Result{}
	for _, p := range paths {
		info, err := os.Lstat(p)
		switch {
		case err != nil:
			continue
		case isLink(info.Mode().Type()):
			r.Links++
		case info.IsDir():
			cleanDir(p, r, opt)
		default:
			removeFile(p, info.Size(), r, opt)
		}
	}
	return r, nil
}

func cleanDir(dir string, r *Result, opt Options) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		r.record(dir, 0, err, opt)
		return
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		switch {
		case isLink(e.Type()):
			r.Links++
		case e.IsDir():
			cleanDir(p, r, opt)
			os.Remove(p) // si quedó algo en uso dentro, la carpeta sigue; ya se contó el archivo
		default:
			var size int64
			if info, err := e.Info(); err == nil {
				size = info.Size()
			}
			removeFile(p, size, r, opt)
		}
	}
}

func removeFile(p string, size int64, r *Result, opt Options) {
	err := os.Remove(p)
	if err != nil && errors.Is(err, fs.ErrPermission) {
		// En Windows un archivo de solo lectura da "acceso denegado": se le quita el atributo y se reintenta.
		if os.Chmod(p, 0o666) == nil {
			err = os.Remove(p)
		}
	}
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		if err == nil {
			r.Freed += size
			r.Deleted++
		}
		return
	}
	r.record(p, size, err, opt)
}

func (r *Result) record(p string, size int64, err error, opt Options) {
	switch {
	case opt.InUse != nil && opt.InUse(err):
		r.InUse++
		r.InUseBytes += size
	case errors.Is(err, fs.ErrPermission):
		r.Denied++
	default:
		r.Failed++
	}
	r.example(p)
}

// isLink: enlaces simbólicos y junctions (Go marca estas últimas como ModeIrregular en Windows).
func isLink(m fs.FileMode) bool { return m&(fs.ModeSymlink|fs.ModeIrregular) != 0 }

// checkSafe rechaza rutas que nunca deben limpiarse en bloque: relativas o la raíz de una unidad.
func checkSafe(p string) error {
	clean := strings.TrimRight(p, `\/`)
	if !filepath.IsAbs(p) && !(len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/')) {
		return fmt.Errorf("ruta no absoluta, no se limpia: %q", p)
	}
	if len(clean) <= 2 || strings.Count(strings.ReplaceAll(clean, `\`, "/"), "/") < 1 {
		return fmt.Errorf("raíz de unidad, no se limpia: %q", p)
	}
	return nil
}

// ParseSelection interpreta la respuesta a «¿Qué limpio?» para n categorías:
// "t"/"todo" = todas; "1,3" o "1 3" = esas; vacío o "n" = ninguna. Devuelve índices base 0.
func ParseSelection(answer string, n int) ([]int, error) {
	a := strings.ToLower(strings.TrimSpace(answer))
	switch a {
	case "", "n", "no", "nada":
		return nil, nil
	case "t", "todo", "todas", "a", "all":
		all := make([]int, n)
		for i := range all {
			all[i] = i
		}
		return all, nil
	}
	var out []int
	seen := map[int]bool{}
	for _, f := range strings.FieldsFunc(a, func(r rune) bool { return r == ',' || r == ' ' || r == ';' }) {
		i, err := strconv.Atoi(f)
		if err != nil || i < 1 || i > n {
			return nil, fmt.Errorf("opción no válida: %q (usa números del 1 al %d, «t» o Enter)", f, n)
		}
		if !seen[i-1] {
			seen[i-1] = true
			out = append(out, i-1)
		}
	}
	sort.Ints(out)
	return out, nil
}
