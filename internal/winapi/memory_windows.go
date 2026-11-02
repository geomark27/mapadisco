package winapi

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

var (
	pdh                          = syscall.NewLazyDLL("pdh.dll")
	procGetPerformanceInfo       = kernel32.NewProc("K32GetPerformanceInfo")
	procPdhOpenQueryW            = pdh.NewProc("PdhOpenQueryW")
	procPdhAddEnglishCounterW    = pdh.NewProc("PdhAddEnglishCounterW")
	procPdhCollectQueryData      = pdh.NewProc("PdhCollectQueryData")
	procPdhGetFormattedCounterVa = pdh.NewProc("PdhGetFormattedCounterValue")
	procPdhCloseQuery            = pdh.NewProc("PdhCloseQuery")
)

// SystemMemory son los totales de memoria del sistema, en bytes.
type SystemMemory struct {
	Total, Available            int64
	CommitTotal, CommitLimit    int64
	KernelPaged, KernelNonpaged int64
	SystemCache                 int64
}

// performanceInformation replica PERFORMANCE_INFORMATION (valores en páginas salvo PageSize).
type performanceInformation struct {
	cb                uint32
	commitTotal       uintptr
	commitLimit       uintptr
	commitPeak        uintptr
	physicalTotal     uintptr
	physicalAvailable uintptr
	systemCache       uintptr
	kernelTotal       uintptr
	kernelPaged       uintptr
	kernelNonpaged    uintptr
	pageSize          uintptr
	handleCount       uint32
	processCount      uint32
	threadCount       uint32
}

// ReadSystemMemory lee los totales con GetPerformanceInfo.
func ReadSystemMemory() (SystemMemory, error) {
	var pi performanceInformation
	pi.cb = uint32(unsafe.Sizeof(pi))
	r, _, e := procGetPerformanceInfo.Call(uintptr(unsafe.Pointer(&pi)), uintptr(pi.cb))
	if r == 0 {
		return SystemMemory{}, fmt.Errorf("GetPerformanceInfo: %w", e)
	}
	p := int64(pi.pageSize)
	return SystemMemory{
		Total:          int64(pi.physicalTotal) * p,
		Available:      int64(pi.physicalAvailable) * p,
		CommitTotal:    int64(pi.commitTotal) * p,
		CommitLimit:    int64(pi.commitLimit) * p,
		KernelPaged:    int64(pi.kernelPaged) * p,
		KernelNonpaged: int64(pi.kernelNonpaged) * p,
		SystemCache:    int64(pi.systemCache) * p,
	}, nil
}

// pdhCounterValue replica PDH_FMT_COUNTERVALUE con PDH_FMT_DOUBLE.
type pdhCounterValue struct {
	status uint32
	_      uint32
	value  float64
}

// MemoryCounters lee contadores de rendimiento de memoria durante sample (los que son "por
// segundo" necesitan dos muestras). Usa los nombres en inglés, que funcionan en Windows en
// cualquier idioma. Devuelve los que pudo leer, por nombre de contador.
func MemoryCounters(names []string, sample time.Duration) (map[string]float64, error) {
	var query uintptr
	if r, _, _ := procPdhOpenQueryW.Call(0, 0, uintptr(unsafe.Pointer(&query))); r != 0 {
		return nil, fmt.Errorf("PdhOpenQuery: 0x%08X", uint32(r))
	}
	defer procPdhCloseQuery.Call(query)

	counters := map[string]uintptr{}
	for _, n := range names {
		path, _ := syscall.UTF16PtrFromString(n)
		var c uintptr
		if r, _, _ := procPdhAddEnglishCounterW.Call(query, uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&c))); r == 0 {
			counters[n] = c
		}
	}
	if len(counters) == 0 {
		return nil, errors.New("no se pudo leer ningún contador de memoria")
	}
	procPdhCollectQueryData.Call(query)
	time.Sleep(sample)
	procPdhCollectQueryData.Call(query)

	const pdhFmtDouble = 0x00000200
	out := map[string]float64{}
	for n, c := range counters {
		var v pdhCounterValue
		if r, _, _ := procPdhGetFormattedCounterVa.Call(c, pdhFmtDouble, 0, uintptr(unsafe.Pointer(&v))); r == 0 {
			out[n] = v.value
		}
	}
	return out, nil
}

// ProcessMemory es la memoria de un proceso.
type ProcessMemory struct {
	Name       string
	PID        uint32
	WorkingSet int64
	Private    int64
}

// systemProcessInformation replica SYSTEM_PROCESS_INFORMATION (64 bits) hasta PrivatePageCount.
type systemProcessInformation struct {
	nextEntryOffset              uint32
	numberOfThreads              uint32
	workingSetPrivateSize        int64
	hardFaultCount               uint32
	numberOfThreadsHighWatermark uint32
	cycleTime                    uint64
	createTime                   int64
	userTime                     int64
	kernelTime                   int64
	imageNameLength              uint16
	imageNameMaximumLength       uint16
	_                            uint32
	imageNameBuffer              uintptr
	basePriority                 int32
	_                            uint32
	uniqueProcessID              uintptr
	inheritedFromUniqueProcessID uintptr
	handleCount                  uint32
	sessionID                    uint32
	uniqueProcessKey             uintptr
	peakVirtualSize              uintptr
	virtualSize                  uintptr
	pageFaultCount               uint32
	_                            uint32
	peakWorkingSetSize           uintptr
	workingSetSize               uintptr
	quotaPeakPagedPoolUsage      uintptr
	quotaPagedPoolUsage          uintptr
	quotaPeakNonPagedPoolUsage   uintptr
	quotaNonPagedPoolUsage       uintptr
	pagefileUsage                uintptr
	peakPagefileUsage            uintptr
	privatePageCount             uintptr
}

var procNtQuerySystemInformation = syscall.NewLazyDLL("ntdll.dll").NewProc("NtQuerySystemInformation")

// Processes lista todos los procesos con su memoria usando NtQuerySystemInformation: a diferencia
// de abrir cada proceso, incluye los protegidos (vmmemWSL, Memory Compression) y no necesita permisos.
func Processes() ([]ProcessMemory, error) {
	const (
		systemProcessInformationClass = 5
		statusInfoLengthMismatch      = 0xC0000004
	)
	size := uint32(1 << 20)
	for range 8 {
		buf := make([]uint64, size/8) // alineado a 8 bytes, como exige la estructura
		var needed uint32
		r, _, _ := procNtQuerySystemInformation.Call(systemProcessInformationClass,
			uintptr(unsafe.Pointer(&buf[0])), uintptr(size), uintptr(unsafe.Pointer(&needed)))
		if uint32(r) == statusInfoLengthMismatch {
			size = max(needed+64<<10, size*2)
			continue
		}
		if r != 0 {
			return nil, fmt.Errorf("NtQuerySystemInformation: 0x%08X", uint32(r))
		}
		return parseProcesses(unsafe.Slice((*byte)(unsafe.Pointer(&buf[0])), len(buf)*8)), nil
	}
	return nil, errors.New("NtQuerySystemInformation: la lista de procesos no deja de crecer")
}

func parseProcesses(b []byte) []ProcessMemory {
	base := uintptr(unsafe.Pointer(&b[0]))
	var out []ProcessMemory
	for off := uintptr(0); off+unsafe.Sizeof(systemProcessInformation{}) <= uintptr(len(b)); {
		pi := (*systemProcessInformation)(unsafe.Pointer(&b[off]))
		if pi.uniqueProcessID != 0 { // el 0 es el proceso inactivo del sistema
			name := ""
			if n := uintptr(pi.imageNameLength); n > 0 && pi.imageNameBuffer >= base && pi.imageNameBuffer-base+n <= uintptr(len(b)) {
				start := pi.imageNameBuffer - base
				u16 := make([]uint16, n/2)
				for i := range u16 {
					u16[i] = uint16(b[start+uintptr(i)*2]) | uint16(b[start+uintptr(i)*2+1])<<8
				}
				name = strings.TrimSuffix(syscall.UTF16ToString(u16), ".exe")
			}
			out = append(out, ProcessMemory{
				Name: name, PID: uint32(pi.uniqueProcessID),
				WorkingSet: int64(pi.workingSetSize), Private: int64(pi.privatePageCount),
			})
		}
		if pi.nextEntryOffset == 0 {
			break
		}
		off += uintptr(pi.nextEntryOffset)
	}
	return out
}

// StartupEntry es un programa configurado para arrancar con Windows (claves Run del registro).
type StartupEntry struct {
	Name    string
	Command string
	Enabled bool // según StartupApproved (lo que muestra Administrador de tareas > Aplicaciones de arranque)
}

// StartupEntries lee las claves Run del usuario y de la máquina, con su estado habilitado/deshabilitado.
func StartupEntries() []StartupEntry {
	const run = `Software\Microsoft\Windows\CurrentVersion\Run`
	const approved = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
	sources := []struct {
		root          registry.Key
		path, approve string
	}{
		{registry.CURRENT_USER, run, approved},
		{registry.LOCAL_MACHINE, run, approved},
		{registry.LOCAL_MACHINE, `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Run`, `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run32`},
	}
	var out []StartupEntry
	for _, s := range sources {
		k, err := registry.OpenKey(s.root, s.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		names, _ := k.ReadValueNames(-1)
		for _, n := range names {
			cmd, _, _ := k.GetStringValue(n)
			out = append(out, StartupEntry{Name: n, Command: cmd, Enabled: startupEnabled(s.root, s.approve, n)})
		}
		k.Close()
	}
	return out
}

// startupEnabled: en StartupApproved, el primer byte par (02, 06) es habilitado e impar (03, 07)
// es deshabilitado. Sin entrada, está habilitado.
func startupEnabled(root registry.Key, path, name string) bool {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return true
	}
	defer k.Close()
	b, _, err := k.GetBinaryValue(name)
	if err != nil || len(b) == 0 {
		return true
	}
	return b[0]%2 == 0
}
