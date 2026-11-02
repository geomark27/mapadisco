package winapi

import (
	"syscall"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procSendMessageTimeoutW = user32.NewProc("SendMessageTimeoutW")
	procGetConsoleProcList  = kernel32.NewProc("GetConsoleProcessList")
)

// BroadcastEnvironmentChange avisa a Windows (Explorador incluido) que cambiaron las
// variables de entorno del usuario, para que las terminales nuevas vean el PATH actualizado.
func BroadcastEnvironmentChange() {
	const (
		hwndBroadcast   = 0xFFFF
		wmSettingChange = 0x001A
		smtoAbortIfHung = 0x0002
	)
	env, _ := syscall.UTF16PtrFromString("Environment")
	var result uintptr
	procSendMessageTimeoutW.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(env)),
		smtoAbortIfHung, 2000, uintptr(unsafe.Pointer(&result)))
}

// StdinIsConsole indica si se puede preguntar al usuario (stdin es una consola, no un archivo ni una tubería).
func StdinIsConsole() bool {
	h, err := syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
	if err != nil {
		return false
	}
	var mode uint32
	return syscall.GetConsoleMode(h, &mode) == nil
}

// OwnsConsole indica si la consola se creó solo para este proceso, como al abrir el .exe
// con doble clic: entonces la ventana se cierra en cuanto el programa termina.
func OwnsConsole() bool {
	var pids [2]uint32
	n, _, _ := procGetConsoleProcList.Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	return n == 1
}
