package cleaner

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func write(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogoValido(t *testing.T) {
	ids := map[string]bool{}
	for _, c := range Auto {
		if c.ID == "" || c.Label == "" || c.Note == "" || c.Manual == "" {
			t.Errorf("categoría incompleta: %+v", c)
		}
		if ids[c.ID] {
			t.Errorf("ID repetido: %s", c.ID)
		}
		ids[c.ID] = true
		if c.Kind == Contents && len(c.Paths) == 0 {
			t.Errorf("%s no tiene rutas", c.ID)
		}
		checkPaths(t, c.ID, c.Paths)
	}
	for _, g := range Manual {
		if g.Label == "" || len(g.Steps) == 0 {
			t.Errorf("guía incompleta: %+v", g)
		}
		checkPaths(t, g.Label, g.Paths)
	}
}

// Las rutas deben partir de una variable de entorno: nunca una unidad ni un usuario escritos a mano.
func checkPaths(t *testing.T, owner string, paths []string) {
	for _, p := range paths {
		if !strings.HasPrefix(p, "$") {
			t.Errorf("%s: la ruta debe empezar con una variable de entorno: %q", owner, p)
		}
		if strings.Contains(p, "/") {
			t.Errorf("%s: usa \\ en %q", owner, p)
		}
	}
}

func TestResolve(t *testing.T) {
	base := t.TempDir()
	write(t, filepath.Join(base, "a", "x.vhdx"), 1)
	write(t, filepath.Join(base, "b", "y.vhdx"), 1)
	os.MkdirAll(filepath.Join(base, "cache"), 0o755)
	env := map[string]string{"BASE": base}
	getenv := func(k string) string { return env[k] }

	got := Resolve([]string{"$BASE/*/*.vhdx", "$BASE/cache", "$BASE/no-existe", "$VACIA/cache", "$BASE/cache"}, getenv)
	want := []string{filepath.Join(base, "a", "x.vhdx"), filepath.Join(base, "b", "y.vhdx"), filepath.Join(base, "cache")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Resolve =\n %v\nquiero\n %v", got, want)
	}
}

func TestCleanBorraElContenidoYConservaLaCarpeta(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Temp")
	write(t, filepath.Join(root, "a.tmp"), 100)
	write(t, filepath.Join(root, "sub", "b.tmp"), 50)
	write(t, filepath.Join(root, "sub", "hondo", "c.tmp"), 25)

	if u := Measure([]string{root}); u.Bytes != 175 || u.Files != 3 {
		t.Fatalf("Measure = %+v", u)
	}
	r, err := Clean([]string{root}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Freed != 175 || r.Deleted != 3 || r.Opposed() {
		t.Errorf("resultado = %+v", r)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("la carpeta raíz debe seguir existiendo: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("quedó contenido: %v", entries)
	}
}

func TestCleanNoSigueEnlaces(t *testing.T) {
	tmp := t.TempDir()
	outside := filepath.Join(tmp, "importante")
	write(t, filepath.Join(outside, "no-borrar.txt"), 10)
	root := filepath.Join(tmp, "Cache")
	write(t, filepath.Join(root, "x.bin"), 5)
	if err := os.Symlink(outside, filepath.Join(root, "enlace")); err != nil {
		t.Skip("no se pueden crear enlaces simbólicos aquí:", err)
	}

	if u := Measure([]string{root}); u.Bytes != 5 {
		t.Errorf("Measure siguió el enlace: %+v", u)
	}
	r, err := Clean([]string{root}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outside, "no-borrar.txt")); err != nil {
		t.Fatal("Clean borró un archivo fuera de la carpeta a través de un enlace")
	}
	if r.Links != 1 || r.Deleted != 1 {
		t.Errorf("resultado = %+v", r)
	}
}

func TestCleanArchivoSuelto(t *testing.T) {
	f := filepath.Join(t.TempDir(), "MEMORY.DMP")
	write(t, f, 42)
	r, _ := Clean([]string{f}, Options{})
	if r.Freed != 42 {
		t.Errorf("resultado = %+v", r)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("el archivo sigue ahí")
	}
}

func TestCleanSoloLectura(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Temp")
	f := filepath.Join(root, "ro.tmp")
	write(t, f, 7)
	os.Chmod(f, 0o444)
	r, _ := Clean([]string{root}, Options{})
	if r.Deleted != 1 {
		t.Errorf("no borró el archivo de solo lectura: %+v", r)
	}
}

func TestCleanCuentaLaOposicionYSigue(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("como root los permisos no se aplican")
	}
	root := filepath.Join(t.TempDir(), "Temp")
	write(t, filepath.Join(root, "ok.tmp"), 3)
	locked := filepath.Join(root, "bloqueada")
	write(t, filepath.Join(locked, "dentro.tmp"), 9)
	os.Chmod(locked, 0o000)
	defer os.Chmod(locked, 0o755)

	r, err := Clean([]string{root}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Deleted != 1 || r.Denied == 0 || !r.Opposed() || len(r.Examples) == 0 {
		t.Errorf("resultado = %+v", r)
	}
}

func TestCleanReconoceArchivoEnUso(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("como root los permisos no se aplican")
	}
	root := filepath.Join(t.TempDir(), "Temp")
	locked := filepath.Join(root, "abierto")
	write(t, filepath.Join(locked, "x.tmp"), 4)
	os.Chmod(locked, 0o500) // no se puede borrar lo de dentro
	defer os.Chmod(locked, 0o755)

	inUse := func(err error) bool { return true } // simula ERROR_SHARING_VIOLATION
	r, _ := Clean([]string{root}, Options{InUse: inUse})
	if r.InUse != 1 || r.InUseBytes != 4 || r.Denied != 0 {
		t.Errorf("resultado = %+v", r)
	}
}

func TestCleanRechazaRutasPeligrosas(t *testing.T) {
	for _, p := range []string{`C:\`, `C:`, `D:/`, "/", "relativa/carpeta", `Temp`} {
		if _, err := Clean([]string{p}, Options{}); err == nil {
			t.Errorf("Clean(%q) debió negarse", p)
		}
	}
	if err := checkSafe(`C:\Users\Usuario\AppData\Local\Temp`); err != nil {
		t.Errorf("ruta válida rechazada: %v", err)
	}
}

func TestParseSelection(t *testing.T) {
	cases := []struct {
		in   string
		want []int
		err  bool
	}{
		{"", nil, false},
		{"n", nil, false},
		{"t", []int{0, 1, 2}, false},
		{"TODO", []int{0, 1, 2}, false},
		{"3,1", []int{0, 2}, false},
		{"1 1 2", []int{0, 1}, false},
		{"4", nil, true},
		{"0", nil, true},
		{"uno", nil, true},
	}
	for _, c := range cases {
		got, err := ParseSelection(c.in, 3)
		if (err != nil) != c.err || !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseSelection(%q) = %v, %v; quiero %v (error=%v)", c.in, got, err, c.want, c.err)
		}
	}
}

func TestSelect(t *testing.T) {
	all, err := Select("")
	if err != nil || len(all) != len(Auto) {
		t.Fatalf("Select(\"\") = %d, %v", len(all), err)
	}
	got, err := Select(" Temp , shaders,")
	if err != nil || len(got) != 2 || got[0].ID != "shaders" || got[1].ID != "temp" {
		t.Fatalf("Select = %v, %v", got, err)
	}
	if _, err := Select("shaders,windows"); err == nil {
		t.Error("una categoría desconocida debe dar error")
	}
}
