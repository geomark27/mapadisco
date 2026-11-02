//go:build !windows

package installer

import "errors"

// En otros sistemas solo existen para que el paquete compile y sus pruebas corran.

var errOnlyWindows = errors.New("la instalación solo está disponible en Windows")

func addToPath(string) (bool, error)      { return false, errOnlyWindows }
func removeFromPath(string) (bool, error) { return false, errOnlyWindows }
func scheduleRemoval(string) error        { return errOnlyWindows }
