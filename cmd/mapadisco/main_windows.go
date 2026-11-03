// Comando mapadisco: mapa de uso de disco para Windows (solo lectura).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"mapadisco/internal/cleaner"
	"mapadisco/internal/format"
	"mapadisco/internal/hotspots"
	"mapadisco/internal/installer"
	"mapadisco/internal/report"
	"mapadisco/internal/scanner"
	"mapadisco/internal/winapi"
)

// version se inyecta al compilar desde el tag de git: -ldflags "-X main.version=v1.2.3" (ver Makefile).
var version = "dev"

// hotspotMinSize: los sospechosos más pequeños no se muestran.
const hotspotMinSize = 100 * format.MB

type config struct {
	target     string
	tree       report.TreeOptions
	topFiles   int
	topExts    int
	csvPath    string
	workers    int
	noColor    bool
	noProgress bool
	admin      bool
	pause      bool
	install    bool
	uninstall  bool
	noInstall  bool
	clean      bool
	yes        bool
	only       string
	update     bool
	check      bool
	ram        bool
	wslconfig  bool
}

func main() { os.Exit(run()) }

func run() int {
	// Antes que nada: también -version y -h deben limpiar lo que dejó una reinstalación.
	if exe, err := installer.Executable(); err == nil {
		installer.CleanOld(exe)
	}
	cfg, code, ok := parseFlags()
	if !ok {
		return code
	}
	// Sin argumentos y en una consola propia es un doble clic: la ventana se cerraría al terminar.
	alone := len(os.Args) == 1
	if alone && winapi.OwnsConsole() {
		cfg.pause = true
	}
	if cfg.pause {
		defer pauseBeforeExit()
	}

	colors := winapi.EnableConsole() && !cfg.noColor && os.Getenv("NO_COLOR") == ""
	palette := report.NewPalette(colors)
	u := ui{palette}
	switch {
	case cfg.update:
		return runUpdate(u, cfg.check)
	case cfg.uninstall:
		return runUninstall(u)
	case cfg.install:
		return runInstall(u)
	case !cfg.noInstall && winapi.StdinIsConsole():
		if done, code := offerInstall(u, alone); done {
			return code
		}
	}

	if cfg.admin && !winapi.IsAdmin() {
		args := append([]string{"-pause", "-no-install"}, without(os.Args[1:], "-admin", "--admin")...)
		if err := winapi.RelaunchAsAdmin(args); err != nil {
			fmt.Fprintln(os.Stderr, "No se pudo elevar:", err)
			return 1
		}
		fmt.Println("Se abrió una nueva ventana como Administrador.")
		return 0
	}

	out := report.New(os.Stdout, palette)
	elevated := winapi.IsAdmin()
	if cfg.clean {
		return runClean(u, out, cfg.yes, cfg.only, elevated, !cfg.noProgress && winapi.StderrIsConsole())
	}
	if cfg.ram {
		return runRAM(u, out, cfg.wslconfig, cfg.yes)
	}
	isDriveRoot := len(cfg.target) == 3 && cfg.target[1] == ':' && cfg.target[2] == '\\'

	out.Header(version, cfg.target, cfg.workers, elevated)

	s := scanner.New(scanner.Options{Workers: cfg.workers, TopFiles: cfg.topFiles})
	showProgress := !cfg.noProgress && winapi.StderrIsConsole()
	done := make(chan struct{})
	if showProgress {
		go progress(&s.Progress, done)
	}
	res := s.Run(cfg.target)
	close(done)
	if showProgress {
		fmt.Fprint(os.Stderr, "\r\x1b[2K")
	}

	out.ScanStats(res)
	if isDriveRoot {
		if total, free, ok := winapi.DiskSpace(cfg.target); ok {
			out.DiskSummary(cfg.target, total, free, res, elevated)
		}
	}
	out.Cloud(res)
	out.Tree(res.Root, cfg.tree)
	if isDriveRoot {
		out.Hotspots(hotspots.Evaluate(res.Root, hotspots.Catalog, hotspotMinSize), elevated)
	}
	out.TopFiles(res)
	out.Extensions(res, cfg.topExts)
	out.Denied(res, elevated)

	if cfg.csvPath != "" {
		if err := exportCSV(cfg.csvPath, res.Root); err != nil {
			fmt.Fprintln(os.Stderr, "Error exportando CSV:", err)
			return 1
		}
		abs, _ := filepath.Abs(cfg.csvPath)
		out.Success("CSV exportado:", abs)
	}
	out.Note("Este análisis no borra nada. Para liberar espacio: mapadisco -clean")
	return 0
}

func parseFlags() (cfg config, code int, ok bool) {
	t := report.DefaultTreeOptions
	flag.IntVar(&t.Depth, "depth", t.Depth, "profundidad del árbol de carpetas")
	minStr := flag.String("min", "1GB", "tamaño mínimo para mostrar una carpeta en el árbol (ej. 500MB, 2GB)")
	flag.IntVar(&t.PerLevel, "top", t.PerLevel, "máximo de subcarpetas mostradas por nivel")
	noExpand := flag.Bool("no-expand", false, "no bajar más allá de -depth aunque una carpeta pese mucho")
	flag.IntVar(&cfg.topFiles, "files", 25, "cantidad de archivos más grandes a listar")
	flag.IntVar(&cfg.topExts, "ext", 15, "cantidad de extensiones a listar")
	flag.StringVar(&cfg.csvPath, "csv", "", "exporta las carpetas >= 10MB a este archivo CSV (separador ;)")
	flag.IntVar(&cfg.workers, "workers", 0, "hilos de escaneo (0 = automático)")
	flag.BoolVar(&cfg.noColor, "no-color", false, "desactiva colores")
	flag.BoolVar(&cfg.noProgress, "no-progress", false, "oculta el progreso")
	flag.BoolVar(&cfg.admin, "admin", false, "relanza como Administrador (para ver carpetas protegidas)")
	flag.BoolVar(&cfg.pause, "pause", false, "espera Enter al terminar")
	flag.BoolVar(&cfg.install, "install", false, "instala mapadisco para tu usuario (sin preguntar) y sale")
	flag.BoolVar(&cfg.uninstall, "uninstall", false, "desinstala mapadisco: lo quita del PATH y borra su carpeta")
	flag.BoolVar(&cfg.noInstall, "no-install", false, "no ofrece instalarse (uso portable)")
	flag.BoolVar(&cfg.update, "update", false, "actualiza mapadisco a la última versión publicada (verifica su SHA-256)")
	flag.BoolVar(&cfg.check, "check", false, "con -update, solo consulta si hay una versión nueva")
	flag.BoolVar(&cfg.ram, "ram", false, "diagnóstico de memoria RAM: en qué se va, si falta de verdad y qué hacer")
	flag.BoolVar(&cfg.wslconfig, "wslconfig", false, "con -ram, propone limitar la memoria de WSL en .wslconfig (pregunta antes de guardar)")
	flag.BoolVar(&cfg.clean, "clean", false, "libera espacio: muestra lo que se puede limpiar y pregunta qué borrar")
	flag.BoolVar(&cfg.yes, "yes", false, "con -clean, limpia todas las categorías sin preguntar; con -ram -wslconfig, guarda sin preguntar")
	flag.StringVar(&cfg.only, "only", "", "con -clean, solo estas categorías: "+strings.Join(cleaner.IDs(), ","))
	showVersion := flag.Bool("version", false, "muestra la versión")
	flag.Usage = usage
	flag.Parse()

	if cfg.check && !cfg.update {
		fmt.Fprintln(os.Stderr, "-check solo se usa junto con -update: mapadisco -update -check")
		return cfg, 2, false
	}
	if cfg.wslconfig && !cfg.ram {
		fmt.Fprintln(os.Stderr, "-wslconfig solo se usa junto con -ram: mapadisco -ram -wslconfig")
		return cfg, 2, false
	}
	if cfg.only != "" && !cfg.clean {
		fmt.Fprintln(os.Stderr, "-only solo se usa junto con -clean: mapadisco -clean -only shaders,temp")
		return cfg, 2, false
	}
	if cfg.yes && !cfg.clean && !cfg.wslconfig {
		fmt.Fprintln(os.Stderr, "-yes solo se usa con -clean o con -ram -wslconfig")
		return cfg, 2, false
	}
	if _, err := cleaner.Select(cfg.only); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cfg, 2, false
	}
	if *showVersion {
		fmt.Println("mapadisco", version)
		return cfg, 0, false
	}
	minSize, err := format.ParseSize(*minStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Valor inválido en -min:", *minStr)
		return cfg, 2, false
	}
	t.MinSize = minSize
	if *noExpand {
		t.AutoExpand = 0
	}
	cfg.tree = t

	cfg.target, err = resolveTarget(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cfg, 2, false
	}
	if cfg.workers <= 0 {
		cfg.workers = min(max(runtime.NumCPU()*3, 8), 96) // I/O: más hilos que núcleos
	}
	return cfg, 0, true
}

func usage() {
	w := flag.CommandLine.Output()
	fmt.Fprintf(w, "mapadisco %s — mapa de uso de disco para Windows (solo lectura)\n\n", version)
	fmt.Fprintf(w, "Uso: mapadisco [opciones] [ruta]   (por defecto C:\\)\n\n")
	flag.PrintDefaults()
	fmt.Fprintf(w, "\nEjemplos:\n  mapadisco\n  mapadisco -admin\n  mapadisco -depth 4 -min 500MB C:\\Users\n  mapadisco -csv reporte.csv\n  mapadisco -clean\n  mapadisco -clean -admin\n  mapadisco -ram\n  mapadisco -update\n  mapadisco -uninstall\n")
}

// resolveTarget normaliza la ruta a escanear ("C:" → "C:\") y valida que sea una carpeta.
func resolveTarget(arg string) (string, error) {
	if arg == "" {
		arg = `C:\`
	}
	if len(arg) == 2 && arg[1] == ':' {
		arg += `\`
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", fmt.Errorf("ruta inválida: %w", err)
	}
	if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("no existe o no es una carpeta: %s", abs)
	}
	return abs, nil
}

func progress(p *scanner.Progress, done <-chan struct{}) {
	const spin = `|/-\`
	start := time.Now()
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()
	for i := 0; ; i++ {
		select {
		case <-done:
			return
		case <-t.C:
			fmt.Fprintf(os.Stderr, "\r\x1b[2K  %c Escaneando... %s archivos · %s carpetas · %s · %ds",
				spin[i%len(spin)], format.Thousands(p.Files.Load()), format.Thousands(p.Dirs.Load()),
				format.Size(p.Bytes.Load()), int(time.Since(start).Seconds()))
		}
	}
}

func exportCSV(path string, root *scanner.Node) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := report.WriteCSV(f, root); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func without(args []string, drop ...string) []string {
	out := make([]string, 0, len(args))
next:
	for _, a := range args {
		for _, d := range drop {
			if a == d {
				continue next
			}
		}
		out = append(out, a)
	}
	return out
}
