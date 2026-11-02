package installer

import (
	"os"
	"path/filepath"
	"testing"
)

const dir = `C:\Users\Usuario\AppData\Local\Programs\mapadisco`

func TestInPath(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{`C:\Windows;` + dir, true},
		{`C:\WINDOWS;"c:\users\usuario\appdata\local\programs\mapadisco\";C:\otro`, true},
		{dir + `-viejo;C:\Windows`, false}, // carpeta hermana: no cuenta
		{``, false},
		{`;;`, false},
	}
	for _, c := range cases {
		if got := inPath(c.value, dir); got != c.want {
			t.Errorf("inPath(%q) = %v, quiero %v", c.value, got, c.want)
		}
	}
}

func TestAppendEntry(t *testing.T) {
	cases := map[string]string{
		``:           dir,
		`C:\a;C:\b`:  `C:\a;C:\b;` + dir,
		`C:\a;C:\b;`: `C:\a;C:\b;` + dir + `;`,
	}
	for in, want := range cases {
		if got := appendEntry(in, dir); got != want {
			t.Errorf("appendEntry(%q) = %q, quiero %q", in, got, want)
		}
	}
}

func TestWithoutEntry(t *testing.T) {
	value := `%USERPROFILE%\bin;` + dir + `;C:\Windows;;` + dir + `\;` + dir + `-viejo;`
	want := `%USERPROFILE%\bin;C:\Windows;;` + dir + `-viejo;`
	if got := withoutEntry(value, dir); got != want {
		t.Errorf("withoutEntry =\n %q\nquiero\n %q", got, want)
	}
}

// Instalar y desinstalar debe dejar el PATH del usuario exactamente como estaba.
func TestAgregarYQuitarDejaElPathIntacto(t *testing.T) {
	for _, original := range []string{
		``,
		`C:\a`,
		`C:\a;C:\b;`,
		`%USERPROFILE%\bin;;C:\Windows;`,
	} {
		if got := withoutEntry(appendEntry(original, dir), dir); got != original {
			t.Errorf("PATH %q quedó como %q", original, got)
		}
	}
}

func TestIsInstalled(t *testing.T) {
	cases := []struct {
		exe  string
		want bool
	}{
		{dir + `\mapadisco.exe`, true},
		{`c:\users\usuario\appdata\local\programs\MAPADISCO\mapadisco.exe`, true},
		{`C:\Users\Usuario\Downloads\mapadisco-windows-amd64.exe`, false},
		{`C:\Users\Usuario\AppData\Local\Temp\go-build123\b001\exe\mapadisco.exe`, true},
	}
	for _, c := range cases {
		if got := IsInstalled(c.exe, dir); got != c.want {
			t.Errorf("IsInstalled(%q) = %v, quiero %v", c.exe, got, c.want)
		}
	}
}

func TestDir(t *testing.T) {
	t.Setenv("LOCALAPPDATA", filepath.Join("base", "Local"))
	got, err := Dir()
	if err != nil || got != filepath.Join("base", "Local", "Programs", "mapadisco") {
		t.Fatalf("Dir() = %q, %v", got, err)
	}
}

func TestCopyFileReemplazaYApartaElAnterior(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "nuevo.exe")
	dst := filepath.Join(tmp, ExeName)
	os.WriteFile(src, []byte("v2"), 0o644)
	os.WriteFile(dst, []byte("v1"), 0o644)

	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(dst); string(b) != "v2" {
		t.Errorf("instalado = %q, quiero v2", b)
	}
	if b, _ := os.ReadFile(dst + ".old"); string(b) != "v1" {
		t.Errorf(".old = %q, quiero v1", b)
	}
	CleanOld(dst)
	if _, err := os.Stat(dst + ".old"); !os.IsNotExist(err) {
		t.Error("CleanOld no borró el .old")
	}
	if left, _ := filepath.Glob(filepath.Join(tmp, ".mapadisco-install-*")); len(left) > 0 {
		t.Errorf("quedaron temporales: %v", left)
	}
}

func TestCopyFileSinInstalacionPrevia(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "nuevo.exe")
	os.WriteFile(src, []byte("v1"), 0o644)
	if err := copyFile(src, filepath.Join(tmp, ExeName)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ExeName+".old")); !os.IsNotExist(err) {
		t.Error("no debe crear .old si no había instalación")
	}
}

func TestUninstallSoloBorraSuCarpeta(t *testing.T) {
	if _, err := Uninstall(`C:\x.exe`, `C:\Windows`); err == nil {
		t.Fatal("Uninstall debe negarse a borrar una carpeta que no se llama mapadisco")
	}
}
