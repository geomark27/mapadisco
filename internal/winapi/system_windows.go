package winapi

import (
	"os"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	procGetDiskFreeSpaceExW = kernel32.NewProc("GetDiskFreeSpaceExW")
	procSetConsoleMode      = kernel32.NewProc("SetConsoleMode")
	procSetConsoleOutputCP  = kernel32.NewProc("SetConsoleOutputCP")
	procIsUserAnAdmin       = shell32.NewProc("IsUserAnAdmin")
	procShellExecuteW       = shell32.NewProc("ShellExecuteW")
)

// DiskSpace devuelve capacidad y espacio libre de la unidad que contiene root.
func DiskSpace(root string) (total, free int64, ok bool) {
	p, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return 0, 0, false
	}
	var avail, tot, totFree uint64
	r, _, _ := procGetDiskFreeSpaceExW.Call(uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&avail)), uintptr(unsafe.Pointer(&tot)), uintptr(unsafe.Pointer(&totFree)))
	if r == 0 {
		return 0, 0, false
	}
	return int64(tot), int64(totFree), true
}

// IsAdmin indica si el proceso corre elevado.
func IsAdmin() bool {
	r, _, _ := procIsUserAnAdmin.Call()
	return r != 0
}

// RelaunchAsAdmin vuelve a abrir el ejecutable actual elevado (UAC) con los argumentos dados.
func RelaunchAsAdmin(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	quoted := make([]string, 0, len(args))
	for _, a := range args {
		quoted = append(quoted, syscall.EscapeArg(a))
	}
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	params, _ := syscall.UTF16PtrFromString(strings.Join(quoted, " "))
	r, _, e := procShellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)), 0, 1)
	if r <= 32 {
		return e
	}
	return nil
}

// EnableConsole activa UTF-8 y secuencias ANSI en la consola.
// Devuelve false si la salida está redirigida o la consola no soporta colores.
func EnableConsole() bool {
	procSetConsoleOutputCP.Call(65001)
	h, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return false
	}
	var mode uint32
	if syscall.GetConsoleMode(h, &mode) != nil {
		return false
	}
	const enableVirtualTerminalProcessing = 0x0004
	r, _, _ := procSetConsoleMode.Call(uintptr(h), uintptr(mode|enableVirtualTerminalProcessing))
	return r != 0
}

// StderrIsConsole indica si stderr es una consola (para mostrar progreso).
func StderrIsConsole() bool {
	h, err := syscall.GetStdHandle(syscall.STD_ERROR_HANDLE)
	if err != nil {
		return false
	}
	var mode uint32
	return syscall.GetConsoleMode(h, &mode) == nil
}

// LongPath añade el prefijo \\?\ para soportar rutas de más de 260 caracteres.
func LongPath(p string) string {
	switch {
	case strings.HasPrefix(p, `\\?\`):
		return p
	case strings.HasPrefix(p, `\\`):
		return `\\?\UNC\` + p[2:]
	default:
		return `\\?\` + p
	}
}
