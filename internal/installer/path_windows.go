package installer

import "mapadisco/internal/winapi"

// addToPath agrega dir al PATH del usuario si no está.
func addToPath(dir string) (bool, error) {
	return editPath(func(current string) (string, bool) {
		if inPath(current, dir) {
			return current, false
		}
		return appendEntry(current, dir), true
	})
}

// removeFromPath quita dir del PATH del usuario si está.
func removeFromPath(dir string) (bool, error) {
	return editPath(func(current string) (string, bool) {
		if !inPath(current, dir) {
			return current, false
		}
		return withoutEntry(current, dir), true
	})
}

func editPath(edit func(current string) (string, bool)) (bool, error) {
	current, expand, err := winapi.UserPath()
	if err != nil {
		return false, err
	}
	value, changed := edit(current)
	if !changed {
		return false, nil
	}
	return true, winapi.SetUserPath(value, expand)
}

// scheduleRemoval borra dir unos segundos después, desde un cmd.exe oculto que sigue
// vivo cuando mapadisco termina (un ejecutable en uso no se puede borrar).
func scheduleRemoval(dir string) error {
	return winapi.StartHidden("cmd.exe", `cmd.exe /d /c ping -n 3 127.0.0.1 >nul & rmdir /s /q "`+dir+`"`)
}
