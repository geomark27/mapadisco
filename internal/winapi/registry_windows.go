package winapi

import (
	"errors"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

// UserPath lee el PATH del usuario (HKCU\Environment\Path) sin expandir las %VAR%.
// expand indica si el valor es REG_EXPAND_SZ; si no existe, devuelve "" y expand=true.
func UserPath() (value string, expand bool, err error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		return "", false, err
	}
	defer key.Close()
	value, kind, err := key.GetStringValue("Path")
	if errors.Is(err, registry.ErrNotExist) {
		return "", true, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, kind == registry.EXPAND_SZ, nil
}

// SetUserPath escribe el PATH del usuario conservando su tipo: reescribir un REG_EXPAND_SZ
// como REG_SZ congelaría como texto literal las entradas %VAR% de otros programas.
// Después avisa a Windows para que las terminales nuevas vean el cambio.
func SetUserPath(value string, expand bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if expand {
		err = key.SetExpandStringValue("Path", value)
	} else {
		err = key.SetStringValue("Path", value)
	}
	if err != nil {
		return err
	}
	BroadcastEnvironmentChange()
	return nil
}

// StartHidden lanza una línea de comandos en un proceso sin ventana que sigue vivo
// cuando mapadisco termina. cmdLine se pasa tal cual (sin reescapar).
func StartHidden(exe, cmdLine string) error {
	const createNoWindow = 0x08000000
	cmd := exec.Command(exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow, CmdLine: cmdLine}
	return cmd.Start()
}
