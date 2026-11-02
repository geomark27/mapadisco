package winapi

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	procSHQueryRecycleBinW = shell32.NewProc("SHQueryRecycleBinW")
	procSHEmptyRecycleBinW = shell32.NewProc("SHEmptyRecycleBinW")
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

// EmptyRecycleBin vacía la Papelera de todas las unidades, sin diálogos ni sonido.
func EmptyRecycleBin() error {
	const (
		noConfirmation = 0x1
		noProgressUI   = 0x2
		noSound        = 0x4
		eUnexpected    = 0x8000FFFF // la devuelve cuando la Papelera ya estaba vacía
	)
	r, _, _ := procSHEmptyRecycleBinW.Call(0, 0, noConfirmation|noProgressUI|noSound)
	if r != 0 && uint32(r) != eUnexpected {
		return fmt.Errorf("SHEmptyRecycleBin: HRESULT 0x%08X", uint32(r))
	}
	return nil
}

// IsInUse reconoce el error de Windows cuando otro programa tiene el archivo abierto.
func IsInUse(err error) bool {
	const sharingViolation, lockViolation = syscall.Errno(32), syscall.Errno(33)
	return errors.Is(err, sharingViolation) || errors.Is(err, lockViolation)
}
