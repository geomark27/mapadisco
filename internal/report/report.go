// Package report presenta el resultado de un escaneo en la terminal y lo exporta a CSV.
// No depende de Windows: escribe sobre un io.Writer, lo que permite probarlo en cualquier SO.
package report

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"mapadisco/internal/format"
	"mapadisco/internal/hotspots"
	"mapadisco/internal/scanner"
)

// Palette agrupa las secuencias ANSI; vacía si los colores están desactivados.
type Palette struct{ Reset, Bold, Dim, Red, Yellow, Green, Cyan string }

// NewPalette devuelve la paleta con colores o una vacía.
func NewPalette(enabled bool) Palette {
	if !enabled {
		return Palette{}
	}
	return Palette{"\x1b[0m", "\x1b[1m", "\x1b[2m", "\x1b[31m", "\x1b[33m", "\x1b[32m", "\x1b[36m"}
}

// Umbrales de color en el árbol.
const (
	warnSize = 10 * format.GB
	critSize = 50 * format.GB
)

// Printer escribe las secciones del reporte.
type Printer struct {
	w io.Writer
	c Palette
}

// New crea un Printer.
func New(w io.Writer, c Palette) *Printer { return &Printer{w: w, c: c} }

func (p *Printer) f(format string, a ...any) { fmt.Fprintf(p.w, format, a...) }

// Section imprime un título de sección.
func (p *Printer) Section(title string) {
	p.f("\n%s%s── %s %s%s\n", p.c.Bold, p.c.Cyan, title, strings.Repeat("─", max(0, 70-len([]rune(title)))), p.c.Reset)
}

// Header imprime la cabecera del programa.
func (p *Printer) Header(version, target string, workers int, elevated bool) {
	mode := p.c.Yellow + "usuario normal" + p.c.Reset
	if elevated {
		mode = p.c.Green + "Administrador" + p.c.Reset
	}
	p.f("%smapadisco %s%s  ·  analizando %s%s%s  ·  %d hilos  ·  %s\n",
		p.c.Bold, version, p.c.Reset, p.c.Cyan, target, p.c.Reset, workers, mode)
}

// ScanStats imprime duración y conteos del escaneo.
func (p *Printer) ScanStats(r *scanner.Result) {
	p.f("%sEscaneo completo en %s%s: %s archivos, %s carpetas, %d enlaces/junctions omitidos (evita contar doble)\n",
		p.c.Dim, r.Elapsed.Round(time.Millisecond), p.c.Reset,
		format.Thousands(r.Files), format.Thousands(r.Dirs), r.Links)
}

// DiskSummary compara lo que Windows reporta como usado con lo que el escaneo encontró.
func (p *Printer) DiskSummary(target string, total, free int64, r *scanner.Result, elevated bool) {
	used := total - free
	found := r.Root.Total
	p.Section("RESUMEN DEL DISCO " + target)
	p.f("  Capacidad        %10s\n", format.Size(total))
	p.f("  Usado            %10s  %s\n", format.Size(used), format.Bar(used, total, 30))
	p.f("  Libre            %10s\n", format.Size(free))
	p.f("  Encontrado       %10s  (lo que este escaneo pudo leer)\n", format.Size(found))

	switch gap := used - found; {
	case gap > format.GB:
		p.f("  %sNo visible       %10s%s  ← espacio usado que el escaneo no pudo ver\n", p.c.Yellow, format.Size(gap), p.c.Reset)
		p.f("  %s  Suele estar en: System Volume Information (puntos de restauración / instantáneas),\n", p.c.Dim)
		p.f("    carpetas sin permiso, perfiles de otros usuarios y metadatos NTFS ($MFT).%s\n", p.c.Reset)
		if elevated {
			p.f("  %s→ Revisa instantáneas con: vssadmin list shadowstorage%s\n", p.c.Yellow, p.c.Reset)
		} else {
			p.f("  %s→ Ejecuta 'mapadisco -admin' para ver más; y como admin: vssadmin list shadowstorage%s\n", p.c.Yellow, p.c.Reset)
		}
	case gap < -format.GB:
		p.f("  %sEl escaneo suma más que lo usado: hay enlaces duros (WinSxS) o compresión que inflan el conteo.%s\n", p.c.Dim, p.c.Reset)
	}
}

// Cloud informa archivos "solo en la nube" que no se contaron.
func (p *Printer) Cloud(r *scanner.Result) {
	if r.CloudFiles == 0 {
		return
	}
	p.f("  %sArchivos solo en la nube (OneDrive, etc.): %s en %s archivos — NO ocupan disco, no se cuentan.%s\n",
		p.c.Dim, format.Size(r.CloudBytes), format.Thousands(r.CloudFiles), p.c.Reset)
}

// TreeOptions controla el árbol de carpetas.
type TreeOptions struct {
	Depth    int   // profundidad garantizada
	MinSize  int64 // tamaño mínimo para mostrar una carpeta
	PerLevel int   // máximo de subcarpetas por nivel
	// AutoExpand: más allá de Depth se sigue bajando por carpetas que pesan
	// al menos esta fracción del total (0 desactiva). Evita esconder al culpable.
	AutoExpand float64
	MaxExtra   int // niveles extra máximos para AutoExpand
}

// DefaultTreeOptions son los valores por defecto del CLI.
var DefaultTreeOptions = TreeOptions{Depth: 3, MinSize: format.GB, PerLevel: 12, AutoExpand: 0.05, MaxExtra: 5}

// Tree imprime el árbol de carpetas más pesadas.
func (p *Printer) Tree(root *scanner.Node, o TreeOptions) {
	p.Section(fmt.Sprintf("CARPETAS MÁS PESADAS (profundidad %d, >= %s)", o.Depth, format.Size(o.MinSize)))
	p.f("  %10s  %5s  %s%s%s\n", format.Size(root.Total), format.Percent(root.Total, root.Total), p.c.Bold, root.Name, p.c.Reset)
	p.tree(root, root.Total, 0, o, "")
}

func (p *Printer) tree(n *scanner.Node, grand int64, level int, o TreeOptions, prefix string) {
	if level >= o.Depth {
		expand := o.AutoExpand > 0 && float64(n.Total) >= float64(grand)*o.AutoExpand && level < o.Depth+o.MaxExtra
		if !expand {
			return
		}
	}
	kids := make([]*scanner.Node, 0, len(n.Children))
	for _, c := range n.Children {
		if c.Total >= o.MinSize {
			kids = append(kids, c)
		}
	}
	if len(kids) == 0 {
		return
	}
	sort.Slice(kids, func(i, j int) bool { return kids[i].Total > kids[j].Total })
	if o.PerLevel > 0 && len(kids) > o.PerLevel {
		kids = kids[:o.PerLevel]
	}
	rest := n.Total // archivos sueltos + subcarpetas no mostradas
	for _, c := range kids {
		rest -= c.Total
	}
	showRest := rest >= o.MinSize

	for i, c := range kids {
		branch, next := "├─ ", "│  "
		if i == len(kids)-1 && !showRest {
			branch, next = "└─ ", "   "
		}
		color := ""
		switch {
		case c.Total >= critSize:
			color = p.c.Red
		case c.Total >= warnSize:
			color = p.c.Yellow
		}
		name := c.Name
		if c.Denied {
			name += p.c.Yellow + " (sin acceso)" + p.c.Reset
		}
		p.f("  %s%10s%s  %5s  %s%s%s%s  %s%s%s\n", color, format.Size(c.Total), p.c.Reset, format.Percent(c.Total, grand),
			p.c.Dim, prefix+branch, p.c.Reset, name, p.c.Cyan, format.Bar(c.Total, grand, 20), p.c.Reset)
		p.tree(c, grand, level+1, o, prefix+next)
	}
	if showRest {
		p.f("  %10s  %5s  %s%s└─ (archivos y carpetas menores)%s\n", format.Size(rest), format.Percent(rest, grand), p.c.Dim, prefix, p.c.Reset)
	}
}

// Hotspots imprime los sospechosos habituales encontrados.
func (p *Printer) Hotspots(ms []hotspots.Match, elevated bool) {
	const maxPaths = 4
	p.Section("SOSPECHOSOS HABITUALES")
	if len(ms) == 0 {
		p.f("  Nada relevante.\n")
		return
	}
	for _, m := range ms {
		if m.Size == 0 {
			hint := " — usa -admin"
			if elevated {
				hint = ""
			}
			p.f("  %10s  %-36s %s(sin acceso%s)%s\n", "?", m.Label, p.c.Yellow, hint, p.c.Reset)
			continue
		}
		color := ""
		if m.Size >= warnSize {
			color = p.c.Yellow
		}
		p.f("  %s%10s%s  %s%s%s\n", color, format.Size(m.Size), p.c.Reset, p.c.Bold, m.Label, p.c.Reset)
		for i, path := range m.Paths {
			if i == maxPaths {
				p.f("              %s… y %d más%s\n", p.c.Dim, len(m.Paths)-maxPaths, p.c.Reset)
				break
			}
			p.f("              %s%s%s\n", p.c.Dim, path, p.c.Reset)
		}
		if m.Hint != "" {
			p.f("              %s→ %s%s\n", p.c.Green, m.Hint, p.c.Reset)
		}
	}
}

// TopFiles imprime el ranking de archivos más grandes.
func (p *Printer) TopFiles(r *scanner.Result) {
	p.Section(fmt.Sprintf("LOS %d ARCHIVOS MÁS GRANDES", len(r.TopFiles)))
	for _, f := range r.TopFiles {
		p.f("  %10s  %s%s%s%s\n", format.Size(f.Size), p.c.Dim, scanner.JoinPath(f.Dir.Path(), ""), p.c.Reset, f.Name)
	}
}

// Extensions imprime el peso por tipo de archivo.
func (p *Printer) Extensions(r *scanner.Result, n int) {
	p.Section("PESO POR TIPO DE ARCHIVO")
	for i, e := range r.Exts {
		if i >= n {
			break
		}
		p.f("  %10s  %-16s %s archivos\n", format.Size(e.Bytes), e.Ext, format.Thousands(e.Count))
	}
}

// Denied lista las carpetas de primeros niveles que no se pudieron leer.
func (p *Printer) Denied(r *scanner.Result, elevated bool) {
	const maxShown, maxDepth = 15, 3
	if r.Denied == 0 {
		return
	}
	p.Section(fmt.Sprintf("CARPETAS SIN ACCESO (%d)", r.Denied))
	shown := 0
	for _, e := range r.Errors {
		if strings.Count(e.Path, `\`) > maxDepth {
			continue
		}
		p.f("  %s%s%s\n", p.c.Dim, e.Path, p.c.Reset)
		if shown++; shown >= maxShown {
			break
		}
	}
	if !elevated {
		p.f("  %s→ Su contenido no se contó. Usa 'mapadisco -admin' para incluirlas.%s\n", p.c.Yellow, p.c.Reset)
	}
}

// Note imprime una línea de cierre atenuada.
func (p *Printer) Note(msg string) { p.f("\n%s%s%s\n", p.c.Dim, msg, p.c.Reset) }

// Success imprime una línea de confirmación.
func (p *Printer) Success(label, detail string) {
	p.f("\n%s%s%s %s\n", p.c.Green, label, p.c.Reset, detail)
}
