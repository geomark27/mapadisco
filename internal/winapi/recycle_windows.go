package winapi

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	procSHQueryRecycleBinW = shell32.NewProc("SHQueryRecycleBinW")
	procSHEmptyRecycleBinW = shell32.NewProc("SHEmptyRecycleBinW")
	procGetConsoleWindow   = kernel32.NewProc("GetConsoleWindow")
)

// shQueryRBInfo replica SHQUERYRBINFO (en 64 bits, con alineación natural: 24 bytes).
type shQueryRBInfo struct {
	cbSize   uint32
	size     int64
	numItems int64
}

// RecycleBinUsage devuelve lo que ocupa la Papelera de todas las unidades.
func RecycleBinUsage() (bytes, items int64, err error) {
	info := shQueryRBInfo{cbSize: uint32(unsafe.Sizeof(shQueryRBInfo{}))}
	r, _, _ := procSHQueryRecycleBinW.Call(0, uintptr(unsafe.Pointer(&info)))
	if r != 0 {
		return 0, 0, fmt.Errorf("SHQueryRecycleBin: HRESULT 0x%08X", uint32(r))
	}
	return info.size, info.numItems, nil
}

// ErrCanceled indica que el usuario canceló una ventana de Windows (p. ej. la que pide permiso de
// Administrador) mientras se vaciaba la Papelera.
var ErrCanceled = errors.New("cancelado en la ventana de Windows")

// EmptyRecycleBin vacía la Papelera de todas las unidades, sin confirmación ni sonido. Bloquea hasta
// terminar. Si un elemento necesita permiso de Administrador, Windows abre su propia ventana para
// pedirlo: se la cuelga de la consola para que aparezca delante de ella y no escondida detrás.
func EmptyRecycleBin() error {
	const (
		noConfirmation = 0x1
		noProgressUI   = 0x2
		noSound        = 0x4
		eUnexpected    = 0x8000FFFF // la devuelve cuando la Papelera ya estaba vacía
		eCanceled      = 0x800704C7 // HRESULT_FROM_WIN32(ERROR_CANCELLED)
	)
	// La ventana de permiso es modal y corre en este hilo: que no cambie mientras tanto.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	owner, _, _ := procGetConsoleWindow.Call()
	r, _, _ := procSHEmptyRecycleBinW.Call(owner, 0, noConfirmation|noProgressUI|noSound)
	switch uint32(r) {
	case 0, eUnexpected:
		return nil
	case eCanceled:
		return ErrCanceled
	}
	return fmt.Errorf("SHEmptyRecycleBin: HRESULT 0x%08X", uint32(r))
}

// IsInUse reconoce el error de Windows cuando otro programa tiene el archivo abierto.
func IsInUse(err error) bool {
	const sharingViolation, lockViolation = syscall.Errno(32), syscall.Errno(33)
	return errors.Is(err, sharingViolation) || errors.Is(err, lockViolation)
}
