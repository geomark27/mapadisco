package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"mapadisco/internal/installer"
	"mapadisco/internal/updater"
)

// runUpdate consulta la última versión publicada y, si es más nueva, la descarga, verifica su
// SHA-256 contra checksums.txt y reemplaza el ejecutable en uso. Es el único momento en que
// mapadisco se conecta a internet. checkOnly solo informa.
func runUpdate(u ui, checkOnly bool) int {
	fmt.Printf("  Versión actual: %s%s%s\n", u.c.Cyan, version, u.c.Reset)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	c := updater.New(version)
	latest, err := c.Latest(ctx)
	switch {
	case errors.Is(err, updater.ErrNoReleases):
		u.hint(fmt.Sprintf("%s en github.com/%s", err, updater.Repo))
		return 0
	case err != nil:
		u.fail("No se pudo consultar la última versión: " + err.Error())
		u.hint("Revisa tu conexión a internet (o el proxy de la red). También puedes actualizar a mano: ver la guía, «Actualizar».")
		return 1
	}

	if !updater.IsNewer(latest, version) {
		if !updater.IsRelease(version) {
			u.hint("Es una versión de desarrollo: no se actualiza sola. La última publicada es " + latest + ".")
			return 0
		}
		if updater.IsNewer(version, latest) {
			u.ok("Tu versión es más nueva que la última publicada (" + latest + ")")
			return 0
		}
		u.ok("Ya tienes la versión más reciente (" + latest + ")")
		return 0
	}
	fmt.Printf("  Nueva versión: %s%s%s%s\n", u.c.Green, u.c.Bold, latest, u.c.Reset)
	u.hint("Novedades: " + updater.ReleaseURL(latest))
	if checkOnly {
		u.hint("Actualiza con: mapadisco -update")
		return 0
	}

	exe, err := installer.Executable()
	if err != nil {
		u.fail("No se pudo ubicar el ejecutable: " + err.Error())
		return 1
	}
	fmt.Printf("\n  %s[1/2]%s Descargando %s y verificando su SHA-256…\n", u.c.Dim, u.c.Reset, latest)
	data, err := c.Fetch(ctx, latest)
	if err != nil {
		u.fail(err.Error())
		return 1
	}
	fmt.Printf("  %s[2/2]%s Reemplazando %s…\n", u.c.Dim, u.c.Reset, exe)
	if err := updater.Replace(exe, data); err != nil {
		u.fail(err.Error())
		u.hint("Si mapadisco está instalado en otra carpeta o sin permisos, reinstala: ver la guía, «Actualizar».")
		return 1
	}
	u.ok("Actualizado a " + latest)
	return 0
}
