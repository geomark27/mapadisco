package winapi

import (
	"syscall"
	"unsafe"
)

// Atributos de archivo Win32.
const (
	AttrDirectory  = 0x10
	AttrSparse     = 0x200
	AttrReparse    = 0x400
	AttrCompressed = 0x800
	AttrOffline    = 0x1000
	AttrRecallOpen = 0x40000
	AttrRecallData = 0x400000
)

// Etiquetas de reparse que apuntan a otra carpeta (no se recorren para no contar doble).
const (
	TagMountPoint = 0xA0000003 // junctions: "Archivos de programa", "Documents and Settings"...
	TagSymlink    = 0xA000000C
)

// Errores Win32 relevantes.
const (
	ErrFileNotFound = syscall.Errno(2)
	ErrPathNotFound = syscall.Errno(3)
)

const (
	findExInfoBasic       = 1
	findExSearchNameMatch = 0
	findFirstExLargeFetch = 2
)

var (
	procFindFirstFileExW       = kernel32.NewProc("FindFirstFileExW")
	procFindNextFileW          = kernel32.NewProc("FindNextFileW")
	procGetCompressedFileSizeW = kernel32.NewProc("GetCompressedFileSizeW")
)

// FindData replica WIN32_FIND_DATAW con el tamaño correcto
// (syscall.Win32finddata declara cFileName con 259 elementos en lugar de 260).
type FindData struct {
	FileAttributes    uint32
	CreationTime      syscall.Filetime
	LastAccessTime    syscall.Filetime
	LastWriteTime     syscall.Filetime
	FileSizeHigh      uint32
	FileSizeLow       uint32
	Reserved0         uint32 // etiqueta de reparse cuando FILE_ATTRIBUTE_REPARSE_POINT
	Reserved1         uint32
	FileName          [260]uint16
	AlternateFileName [14]uint16
}

// Name devuelve el nombre de la entrada.
func (fd *FindData) Name() string { return syscall.UTF16ToString(fd.FileName[:]) }

// Size devuelve el tamaño lógico del archivo.
func (fd *FindData) Size() int64 { return int64(fd.FileSizeHigh)<<32 | int64(fd.FileSizeLow) }

// IsDir indica si la entrada es una carpeta.
func (fd *FindData) IsDir() bool { return fd.FileAttributes&AttrDirectory != 0 }

// IsLink indica si es una junction o enlace simbólico (apunta a otro lugar).
func (fd *FindData) IsLink() bool {
	return fd.FileAttributes&AttrReparse != 0 && (fd.Reserved0 == TagMountPoint || fd.Reserved0 == TagSymlink)
}

// IsCloudOnly indica si el contenido no está en disco local (OneDrive "solo en línea", etc.).
func (fd *FindData) IsCloudOnly() bool {
	return fd.FileAttributes&(AttrRecallData|AttrRecallOpen|AttrOffline) != 0
}

// IsCompressedOrSparse indica si el tamaño real en disco puede diferir del lógico.
func (fd *FindData) IsCompressedOrSparse() bool {
	return fd.FileAttributes&(AttrSparse|AttrCompressed) != 0
}

// FindFirst inicia una enumeración con FindFirstFileExW (info básica + large fetch).
func FindFirst(pattern string, fd *FindData) (syscall.Handle, error) {
	p, err := syscall.UTF16PtrFromString(pattern)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	r, _, e := procFindFirstFileExW.Call(
		uintptr(unsafe.Pointer(p)), findExInfoBasic, uintptr(unsafe.Pointer(fd)),
		findExSearchNameMatch, 0, findFirstExLargeFetch)
	h := syscall.Handle(r)
	if h == syscall.InvalidHandle {
		return h, e
	}
	return h, nil
}

// FindNext avanza la enumeración; devuelve error al terminar.
func FindNext(h syscall.Handle, fd *FindData) error {
	r, _, e := procFindNextFileW.Call(uintptr(h), uintptr(unsafe.Pointer(fd)))
	if r == 0 {
		return e
	}
	return nil
}

// FindClose libera el handle de enumeración.
func FindClose(h syscall.Handle) { syscall.FindClose(h) }

// Stat obtiene los datos de una sola ruta. Funciona con archivos bloqueados
// como pagefile.sys, a diferencia de os.Stat.
func Stat(path string) (FindData, bool) {
	var fd FindData
	h, err := FindFirst(LongPath(path), &fd)
	if err != nil {
		return fd, false
	}
	FindClose(h)
	return fd, true
}

// AllocatedSize devuelve el tamaño real en disco de un archivo comprimido o disperso.
func AllocatedSize(path string) (int64, bool) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, false
	}
	var high uint32
	low, _, e := procGetCompressedFileSizeW.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&high)))
	if uint32(low) == 0xFFFFFFFF && e != syscall.Errno(0) {
		return 0, false
	}
	return int64(high)<<32 | int64(uint32(low)), true
}
