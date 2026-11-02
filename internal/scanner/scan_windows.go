package scanner

import (
	"container/heap"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mapadisco/internal/winapi"
)

// MaxErrors limita cuántas carpetas sin acceso se guardan con detalle.
const MaxErrors = 2000

// Options configura el escaneo.
type Options struct {
	Workers  int // goroutines concurrentes (I/O); <= 0 usa 8
	TopFiles int // tamaño del ranking de archivos grandes
}

// Progress expone contadores en vivo para mostrar avance mientras se escanea.
type Progress struct {
	Files, Dirs, Bytes atomic.Int64
}

// Scanner recorre carpetas en paralelo con FindFirstFileExW.
type Scanner struct {
	opts     Options
	Progress Progress

	sem chan struct{}
	wg  sync.WaitGroup

	links, denied        atomic.Int64
	cloudBytes, cloudCnt atomic.Int64

	threshold atomic.Int64 // tamaño mínimo para entrar al ranking (lectura sin lock)
	fmu       sync.Mutex
	top       fileHeap

	emu  sync.Mutex
	exts map[string]*ExtStat

	errMu  sync.Mutex
	errors []ScanError
}

// New crea un Scanner.
func New(opts Options) *Scanner {
	if opts.Workers <= 0 {
		opts.Workers = 8
	}
	return &Scanner{
		opts: opts,
		sem:  make(chan struct{}, opts.Workers),
		exts: make(map[string]*ExtStat, 512),
	}
}

// Run escanea rootPath (ruta absoluta) y devuelve el resultado. Un Scanner se usa una sola vez.
func (s *Scanner) Run(rootPath string) *Result {
	start := time.Now()
	root := &Node{Name: rootPath}
	s.walk(root, winapi.LongPath(rootPath))
	s.wg.Wait()
	SumTotals(root)

	top := append([]BigFile(nil), s.top...)
	sort.Slice(top, func(i, j int) bool { return top[i].Size > top[j].Size })
	exts := make([]ExtStat, 0, len(s.exts))
	for _, e := range s.exts {
		exts = append(exts, *e)
	}
	sort.Slice(exts, func(i, j int) bool { return exts[i].Bytes > exts[j].Bytes })

	return &Result{
		Root:       root,
		Files:      s.Progress.Files.Load(),
		Dirs:       s.Progress.Dirs.Load(),
		Links:      s.links.Load(),
		Denied:     s.denied.Load(),
		CloudBytes: s.cloudBytes.Load(),
		CloudFiles: s.cloudCnt.Load(),
		TopFiles:   top,
		Exts:       exts,
		Errors:     s.errors,
		Elapsed:    time.Since(start),
	}
}

type pending struct {
	n    *Node
	path string
}

func (s *Scanner) walk(n *Node, path string) {
	var fd winapi.FindData
	h, err := winapi.FindFirst(JoinPath(path, "*"), &fd)
	if err != nil {
		if err != winapi.ErrFileNotFound && err != winapi.ErrPathNotFound {
			s.recordDenied(n, path, err)
		}
		return
	}

	var subs []pending
	var local map[string]*ExtStat

	for {
		name := fd.Name()
		if name != "." && name != ".." {
			switch {
			case fd.IsDir() && fd.IsLink():
				s.links.Add(1) // apunta a una carpeta que ya se cuenta en otro lugar
			case fd.IsDir():
				c := &Node{Name: name, Parent: n}
				n.Children = append(n.Children, c)
				subs = append(subs, pending{c, JoinPath(path, name)})
			case fd.IsCloudOnly():
				s.cloudBytes.Add(fd.Size())
				s.cloudCnt.Add(1)
			default:
				size := fd.Size()
				if fd.IsCompressedOrSparse() {
					if a, ok := winapi.AllocatedSize(JoinPath(path, name)); ok {
						size = a
					}
				}
				n.Size += size
				n.Files++
				if local == nil {
					local = make(map[string]*ExtStat, 8)
				}
				addExt(local, name, size)
				if size >= s.threshold.Load() {
					s.considerFile(BigFile{size, n, name})
				}
			}
		}
		if winapi.FindNext(h, &fd) != nil {
			break
		}
	}
	winapi.FindClose(h)

	s.Progress.Files.Add(n.Files)
	s.Progress.Bytes.Add(n.Size)
	s.Progress.Dirs.Add(1)
	s.mergeExts(local)

	for _, c := range subs {
		select {
		case s.sem <- struct{}{}:
			s.wg.Add(1)
			go func(c pending) {
				defer func() { <-s.sem; s.wg.Done() }()
				s.walk(c.n, c.path)
			}(c)
		default:
			s.walk(c.n, c.path) // sin hilos libres: se procesa en el actual
		}
	}
}

func (s *Scanner) recordDenied(n *Node, path string, err error) {
	n.Denied = true
	s.denied.Add(1)
	s.errMu.Lock()
	if len(s.errors) < MaxErrors {
		s.errors = append(s.errors, ScanError{strings.TrimPrefix(path, `\\?\`), err})
	}
	s.errMu.Unlock()
}

func addExt(m map[string]*ExtStat, name string, size int64) {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" || len(ext) > 12 {
		ext = "(sin extensión)"
	}
	e := m[ext]
	if e == nil {
		e = &ExtStat{Ext: ext}
		m[ext] = e
	}
	e.Bytes += size
	e.Count++
}

func (s *Scanner) mergeExts(local map[string]*ExtStat) {
	if local == nil {
		return
	}
	s.emu.Lock()
	defer s.emu.Unlock()
	for k, v := range local {
		if g := s.exts[k]; g != nil {
			g.Bytes += v.Bytes
			g.Count += v.Count
		} else {
			s.exts[k] = v
		}
	}
}

func (s *Scanner) considerFile(f BigFile) {
	if s.opts.TopFiles <= 0 {
		return
	}
	s.fmu.Lock()
	defer s.fmu.Unlock()
	switch {
	case len(s.top) < s.opts.TopFiles:
		heap.Push(&s.top, f)
	case f.Size > s.top[0].Size:
		s.top[0] = f
		heap.Fix(&s.top, 0)
	default:
		return
	}
	if len(s.top) == s.opts.TopFiles {
		s.threshold.Store(s.top[0].Size)
	}
}

// fileHeap es un min-heap: la raíz es el archivo más pequeño del ranking.
type fileHeap []BigFile

func (h fileHeap) Len() int           { return len(h) }
func (h fileHeap) Less(i, j int) bool { return h[i].Size < h[j].Size }
func (h fileHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *fileHeap) Push(x any)        { *h = append(*h, x.(BigFile)) }
func (h *fileHeap) Pop() any {
	old := *h
	x := old[len(old)-1]
	*h = old[:len(old)-1]
	return x
}
