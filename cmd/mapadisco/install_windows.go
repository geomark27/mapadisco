package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"mapadisco/internal/installer"
	"mapadisco/internal/report"
)

// stdin es compartido: la pregunta de instalación y la pausa final leen del mismo búfer.
var stdin = bufio.NewReader(os.Stdin)

func pauseBeforeExit() {
	fmt.Print("\nPresiona Enter para cerrar...")
	stdin.ReadString('\n')
}

func confirm(question string, def bool) bool {
	fmt.Print(question)
	answer, err := stdin.ReadString('\n')
	if err != nil {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "":
		return def
	case "s", "si", "sí", "y", "yes":
		return true
	default:
		return false
	}
}

type ui struct{ c report.Palette }

func (u ui) ok(msg string)   { fmt.Printf("  %s✓%s %s\n", u.c.Green, u.c.Reset, msg) }
func (u ui) fail(msg string) { fmt.Fprintf(os.Stderr, "  %s✗%s %s\n", u.c.Red, u.c.Reset, msg) }
func (u ui) hint(msg string) { fmt.Printf("    %s\n", msg) }

func locate() (exe, dir string, err error) {
	if exe, err = installer.Executable(); err != nil {
		return "", "", err
	}
	dir, err = installer.Dir()
	return exe, dir, err
}

// offerInstall pregunta si instalar cuando el programa no corre desde su carpeta de instalación.
// done indica que el programa termina ahí: tras instalar sin argumentos (doble clic o
// el .exe recién descargado), o si la instalación falla. Si no, sigue con el análisis.
func offerInstall(u ui, alone bool) (done bool, code int) {
	exe, dir, err := locate()
	if err != nil || installer.IsInstalled(exe, dir) {
		return false, 0
	}
	fmt.Printf("%smapadisco no está instalado%s\n", u.c.Bold, u.c.Reset)
	fmt.Printf("  Se copiará a %s y esa carpeta se agregará a tu PATH.\n", dir)
	fmt.Printf("  %sSolo para tu usuario; no necesita permisos de Administrador.%s\n\n", u.c.Dim, u.c.Reset)
	if !confirm("¿Instalar mapadisco ahora? [S/n]: ", true) {
		u.hint("No se instaló. Para no ver esta pregunta: mapadisco -no-install")
		fmt.Println()
		return false, 0
	}
	if code := install(u, exe, dir); code != 0 || alone {
		return true, code
	}
	fmt.Println()
	return false, 0
}

func runInstall(u ui) int {
	exe, dir, err := locate()
	if err != nil {
		u.fail(err.Error())
		return 1
	}
	if installer.IsInstalled(exe, dir) {
		u.ok("mapadisco ya está instalado en " + exe)
		return 0
	}
	return install(u, exe, dir)
}

func install(u ui, exe, dir string) int {
	r, err := installer.Install(exe, dir)
	if err != nil {
		u.fail(err.Error())
		return 1
	}
	u.ok("Instalado en " + r.Binary)
	if r.PathChanged {
		u.ok("Carpeta agregada al PATH de tu usuario")
		u.hint("Abre una terminal nueva y escribe: mapadisco")
	} else {
		u.hint("Ya puedes usarlo desde cualquier terminal: mapadisco")
	}
	return 0
}

func runUninstall(u ui) int {
	exe, dir, err := locate()
	if err != nil {
		u.fail(err.Error())
		return 1
	}
	r, err := installer.Uninstall(exe, dir)
	switch {
	case errors.Is(err, installer.ErrNotInstalled):
		u.ok("mapadisco no está instalado: no hay nada que desinstalar")
		return 0
	case err != nil:
		u.fail(err.Error())
		return 1
	}
	if r.PathChanged {
		u.ok("Carpeta quitada del PATH de tu usuario")
	}
	if r.Deferred {
		u.ok("La carpeta " + dir + " se borrará en unos segundos, al cerrar mapadisco")
	} else {
		u.ok("Carpeta borrada: " + dir)
	}
	return 0
}
