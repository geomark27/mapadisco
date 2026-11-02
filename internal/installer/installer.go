// Package installer instala mapadisco.exe para el usuario actual: lo copia a
// %LOCALAPPDATA%\Programs\mapadisco y agrega esa carpeta al PATH del usuario.
// No necesita permisos de Administrador.
//
// La lógica de rutas es portable (con pruebas en Linux); el registro de Windows
// vive en path_windows.go.
package installer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExeName es el nombre del ejecutable instalado.
const ExeName = "mapadisco.exe"

// dirName es la carpeta propia de mapadisco; Uninstall solo borra una carpeta con este nombre.
const dirName = "mapadisco"

// ErrNotInstalled indica que no había nada que desinstalar.
var ErrNotInstalled = errors.New("mapadisco no está instalado")

// Dir es la carpeta de instalación: %LOCALAPPDATA%\Programs\mapadisco.
func Dir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(base, "Programs", dirName), nil
}

// Executable es la ruta del ejecutable en curso.
func Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// IsInstalled indica si exe ya corre desde dir, o si es una compilación de
// `go run` / `go test`, que no se instala.
func IsInstalled(exe, dir string) bool {
	if strings.Contains(strings.ReplaceAll(exe, `\`, "/"), "/go-build") {
		return true
	}
	return canonEntry(parentDir(exe)) == canonEntry(dir)
}

// Result describe lo que cambió Install.
type Result struct {
	Binary      string // ejecutable instalado
	PathChanged bool   // se agregó la carpeta al PATH: hace falta una terminal nueva
}

// Install copia exe a dir como mapadisco.exe y agrega dir al PATH del usuario si no está.
// Si ya había una versión instalada, la reemplaza aunque esté en uso.
func Install(exe, dir string) (*Result, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("no se pudo crear %s: %w", dir, err)
	}
	r := &Result{Binary: filepath.Join(dir, ExeName)}
	if err := copyFile(exe, r.Binary); err != nil {
		return nil, fmt.Errorf("no se pudo copiar el ejecutable: %w", err)
	}
	changed, err := addToPath(dir)
	if err != nil {
		return r, fmt.Errorf("mapadisco quedó en %s pero no se pudo agregar al PATH: %w", dir, err)
	}
	r.PathChanged = changed
	return r, nil
}

// Removal describe lo que hizo Uninstall.
type Removal struct {
	PathChanged bool // se quitó la carpeta del PATH
	Deferred    bool // la carpeta se borra al cerrar mapadisco (el ejecutable en uso era el instalado)
}

// Uninstall quita dir del PATH del usuario y borra la carpeta. Si exe (el programa en curso)
// está dentro de dir, Windows no deja borrarlo mientras corre: el borrado queda programado
// para cuando el proceso termine.
func Uninstall(exe, dir string) (*Removal, error) {
	if filepath.Base(dir) != dirName {
		return nil, fmt.Errorf("carpeta de instalación inesperada, no se borra: %s", dir)
	}
	r := &Removal{}
	changed, err := removeFromPath(dir)
	if err != nil {
		return nil, fmt.Errorf("no se pudo quitar del PATH: %w", err)
	}
	r.PathChanged = changed

	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if !changed {
			return nil, ErrNotInstalled
		}
		return r, nil
	}
	if IsInstalled(exe, dir) {
		if err := scheduleRemoval(dir); err != nil {
			return r, fmt.Errorf("no se pudo programar el borrado de %s: %w", dir, err)
		}
		r.Deferred = true
		return r, nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return r, fmt.Errorf("no se pudo borrar %s (¿mapadisco está abierto en otra ventana?): %w", dir, err)
	}
	return r, nil
}

// CleanOld borra el ejecutable anterior que Install dejó apartado como .old.
func CleanOld(exe string) {
	os.Remove(exe + ".old")
}

// copyFile escribe una copia temporal junto a dst y la renombra. Si dst existe y está en uso,
// Windows no deja sobrescribirlo pero sí renombrarlo: se aparta como .old (CleanOld lo borra después).
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".mapadisco-install-*")
	if err != nil {
		return err
	}
	_, err = io.Copy(tmp, in)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		os.Remove(dst + ".old")
		if rerr := os.Rename(dst, dst+".old"); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
			err = rerr
		}
	}
	if err == nil {
		err = os.Rename(tmp.Name(), dst)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return err
}

// inPath indica si dir es una de las entradas de un valor de PATH. Se comparan entradas
// completas: buscar el texto dentro del valor confundiría con carpetas como "mapadisco-viejo".
func inPath(pathValue, dir string) bool {
	want := canonEntry(dir)
	if want == "" {
		return false
	}
	for e := range strings.SplitSeq(pathValue, ";") {
		if canonEntry(e) == want {
			return true
		}
	}
	return false
}

// appendEntry agrega dir al final de un valor de PATH respetando su estilo: si el valor
// terminaba en ";", sigue terminando así. withoutEntry deshace exactamente este cambio.
func appendEntry(pathValue, dir string) string {
	switch {
	case pathValue == "":
		return dir
	case strings.HasSuffix(pathValue, ";"):
		return pathValue + dir + ";"
	default:
		return pathValue + ";" + dir
	}
}

// withoutEntry quita las apariciones de dir de un valor de PATH y deja el resto tal cual,
// incluidas las entradas vacías y el ";" final.
func withoutEntry(pathValue, dir string) string {
	want := canonEntry(dir)
	parts := strings.Split(pathValue, ";")
	keep := parts[:0]
	for _, e := range parts {
		if e != "" && canonEntry(e) == want {
			continue
		}
		keep = append(keep, e)
	}
	return strings.Join(keep, ";")
}

// canonEntry normaliza una ruta de Windows para compararla: sin comillas, espacios ni barra
// final, y en minúsculas (NTFS no distingue mayúsculas).
func canonEntry(e string) string {
	e = strings.Trim(strings.TrimSpace(e), `"`)
	return strings.ToLower(strings.TrimRight(e, `\/`))
}

// parentDir devuelve la carpeta de una ruta, con \ o / como separador.
func parentDir(p string) string {
	if i := strings.LastIndexAny(p, `\/`); i >= 0 {
		return p[:i]
	}
	return ""
}
