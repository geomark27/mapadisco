package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mapadisco/internal/format"
	"mapadisco/internal/memory"
	"mapadisco/internal/report"
	"mapadisco/internal/winapi"
)

// Contadores de rendimiento (nombres en inglés: PdhAddEnglishCounter los traduce).
const (
	ctrPagesIn   = `\Memory\Pages Input/sec`
	ctrFree      = `\Memory\Free & Zero Page List Bytes`
	ctrModified  = `\Memory\Modified Page List Bytes`
	ctrStandby1  = `\Memory\Standby Cache Core Bytes`
	ctrStandby2  = `\Memory\Standby Cache Normal Priority Bytes`
	ctrStandby3  = `\Memory\Standby Cache Reserve Bytes`
	ramTopGroups = 15
)

// runRAM muestra en qué se va la memoria y qué hacer. Con wslconfig, ofrece limitar WSL.
func runRAM(u ui, out *report.Printer, wslconfig, yes bool) int {
	fmt.Printf("  %sMidiendo la memoria (2 segundos)…%s\n", u.c.Dim, u.c.Reset)
	s, err := snapshot()
	if err != nil {
		u.fail("No se pudo leer la memoria: " + err.Error())
		return 1
	}
	if wslconfig {
		return runWSLConfig(u, out, s, yes)
	}

	printMemorySummary(u, out, s)
	groups := memory.GroupProcesses(s.Procs)
	printGroups(u, out, groups)
	printTips(u, out, memory.Advice(s, groups))
	return 0
}

func snapshot() (memory.Snapshot, error) {
	sys, err := winapi.ReadSystemMemory()
	if err != nil {
		return memory.Snapshot{}, err
	}
	s := memory.Snapshot{
		Total: sys.Total, Available: sys.Available, CommitTotal: sys.CommitTotal, CommitLimit: sys.CommitLimit,
		KernelPaged: sys.KernelPaged, KernelNonp: sys.KernelNonpaged,
	}
	if c, err := winapi.MemoryCounters([]string{ctrPagesIn, ctrFree, ctrModified, ctrStandby1, ctrStandby2, ctrStandby3}, 2*time.Second); err == nil {
		s.PagesInSec = c[ctrPagesIn]
		s.Free = int64(c[ctrFree])
		s.Modified = int64(c[ctrModified])
		s.Cache = int64(c[ctrStandby1] + c[ctrStandby2] + c[ctrStandby3])
	}
	if s.Cache == 0 && s.Free > 0 {
		s.Cache = max(s.Available-s.Free, 0)
	}

	procs, err := winapi.Processes()
	if err != nil {
		return s, err
	}
	for _, p := range procs {
		s.Procs = append(s.Procs, memory.Proc{Name: p.Name, WorkingSet: p.WorkingSet, Private: p.Private})
		if strings.EqualFold(p.Name, "Memory Compression") {
			s.Compressed = p.WorkingSet
		}
	}
	for _, e := range winapi.StartupEntries() {
		s.Startup = append(s.Startup, memory.StartupItem{Name: e.Name, Command: e.Command, Enabled: e.Enabled})
	}
	s.WSL = readWSLConfig()
	return s, nil
}

func wslConfigPath() string {
	home := os.Getenv("USERPROFILE")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".wslconfig")
}

func readWSLConfig() memory.WSLConfig {
	path := wslConfigPath()
	b, err := os.ReadFile(path)
	c := memory.ParseWSLConfig(string(b))
	c.Path, c.Exists = path, err == nil
	return c
}

func printMemorySummary(u ui, out *report.Printer, s memory.Snapshot) {
	out.Section("MEMORIA RAM")
	row := func(label, value, extra string) {
		fmt.Printf("  %-17s %10s  %s\n", label, value, extra)
	}
	row("Total", format.Size(s.Total), "")
	inUse := s.InUse()
	extra := format.Bar(inUse, s.Total, 24)
	if s.Compressed > 0 {
		extra += fmt.Sprintf("  %s(de eso, comprimida: %s)%s", u.c.Dim, format.Size(s.Compressed), u.c.Reset)
	}
	row("En uso", format.Size(inUse), extra)
	avail := ""
	if s.Cache > 0 || s.Free > 0 {
		avail = fmt.Sprintf("= en caché %s + libre %s", format.Size(s.Cache), format.Size(s.Free))
	}
	row("Disponible", format.Size(s.Available), avail)
	row("Confirmada", format.Size(s.CommitTotal), fmt.Sprintf("de %s (RAM + archivo de paginación)", format.Size(s.CommitLimit)))
	row("Kernel y drivers", format.Size(s.KernelNonp+s.KernelPaged),
		fmt.Sprintf("no paginada %s · paginada %s", format.Size(s.KernelNonp), format.Size(s.KernelPaged)))

	level, why := memory.Pressure(s)
	label, color := "BAJA", u.c.Green
	switch level {
	case memory.PressureMedium:
		label, color = "MEDIA", u.c.Yellow
	case memory.PressureHigh:
		label, color = "ALTA", u.c.Red
	}
	fmt.Printf("  %-17s %s%10s%s  %s\n", "Presión", color+u.c.Bold, label, u.c.Reset, why)
	fmt.Printf("  %-17s %10s  %slecturas de memoria desde el disco: %.0f páginas/s%s\n", "", "", u.c.Dim, s.PagesInSec, u.c.Reset)

	fmt.Printf("\n  %sCómo leerlo (los mismos nombres que el Administrador de tareas):%s\n", u.c.Bold, u.c.Reset)
	for _, l := range []string{
		"En uso: lo que ocupan programas y Windows. La parte «comprimida» es memoria poco usada que Windows comprime en vez de mandarla al disco.",
		"En caché: archivos leídos hace poco, guardados por si se vuelven a pedir. Se liberan al instante si un programa lo necesita: no es memoria perdida.",
		"Disponible = caché + libre. Que sea bajo no es un problema por sí solo; lo es si la «Presión» sube (Windows leyendo memoria del disco).",
		"Confirmada: lo que los programas reservaron. Puede superar la RAM porque lo poco usado vive comprimido o en el archivo de paginación.",
	} {
		fmt.Printf("  %s· %s%s\n", u.c.Dim, l, u.c.Reset)
	}
}

func printGroups(u ui, out *report.Printer, groups []memory.Group) {
	out.Section("QUIÉN USA LA MEMORIA")
	fmt.Printf("  %s%10s  %10s  %5s  %s%s\n", u.c.Dim, "Reservada", "En RAM", "Proc.", "Aplicación", u.c.Reset)
	for i, g := range groups {
		if i == ramTopGroups {
			break
		}
		color := ""
		switch {
		case g.Private >= 4<<30:
			color = u.c.Red
		case g.Private >= 1<<30:
			color = u.c.Yellow
		}
		fmt.Printf("  %s%10s%s  %10s  %5d  %s\n", color, format.Size(g.Private), u.c.Reset, format.Size(g.WorkingSet), g.Count, g.Name)
	}
	fmt.Printf("  %sReservada: lo que pidió el programa (en RAM, comprimido o en el disco). En RAM: lo que ocupa ahora mismo.%s\n", u.c.Dim, u.c.Reset)
}

func printTips(u ui, out *report.Printer, tips []memory.Tip) {
	out.Section("QUÉ HACER")
	if len(tips) == 0 {
		u.ok("No hay nada evidente que cerrar o configurar.")
	}
	for i, t := range tips {
		saving := ""
		if t.Savings > 0 {
			saving = fmt.Sprintf("  %s(hasta ~%s)%s", u.c.Green, format.Size(t.Savings), u.c.Reset)
		}
		fmt.Printf("  %s%d. %s%s%s\n", u.c.Bold, i+1, t.Title, u.c.Reset, saving)
		if t.Detail != "" {
			fmt.Printf("     %s%s%s\n", u.c.Dim, t.Detail, u.c.Reset)
		}
		for _, s := range t.Steps {
			fmt.Printf("     %s→ %s%s\n", u.c.Green, s, u.c.Reset)
		}
	}
	fmt.Printf("\n  %sNo uses «liberadores de RAM» (vaciar la memoria con un botón): mandan todo al disco y los programas\n"+
		"  lo vuelven a pedir enseguida, así que el equipo queda más lento. mapadisco solo explica y sugiere.%s\n", u.c.Dim, u.c.Reset)
}

// runWSLConfig propone el .wslconfig con límite de memoria y devolución gradual, lo muestra y lo
// escribe solo si el usuario confirma (o con -yes), guardando una copia del anterior.
func runWSLConfig(u ui, out *report.Printer, s memory.Snapshot, yes bool) int {
	out.Section("CONFIGURAR LA MEMORIA DE WSL")
	path := s.WSL.Path
	current := ""
	if b, err := os.ReadFile(path); err == nil {
		current = string(b)
	} else if !errors.Is(err, fs.ErrNotExist) {
		u.fail("No se pudo leer " + path + ": " + err.Error())
		return 1
	}
	proposed := memory.ProposeWSLConfig(current, s.Total)
	if proposed == current {
		u.ok(path + " ya limita la memoria de WSL y la devuelve: no hace falta cambiar nada")
		return 0
	}

	if s.WSL.Exists {
		fmt.Printf("  Se modificará %s (lo demás que tenga queda igual):\n\n", path)
	} else {
		fmt.Printf("  Se creará %s con:\n\n", path)
	}
	for line := range strings.Lines(strings.ReplaceAll(proposed, "\r\n", "\n")) {
		fmt.Printf("    %s%s%s", u.c.Cyan, line, u.c.Reset)
	}
	fmt.Printf("\n  %s· memory: WSL no usará más de esa RAM (tu equipo tiene %s).%s\n", u.c.Dim, format.Size(s.Total), u.c.Reset)
	fmt.Printf("  %s· autoMemoryReclaim=gradual: Linux devuelve a Windows la memoria que deja de usar.%s\n\n", u.c.Dim, u.c.Reset)

	if !yes {
		if !winapi.StdinIsConsole() {
			u.hint("Para escribirlo sin preguntar: mapadisco -ram -wslconfig -yes")
			return 0
		}
		if !confirm("¿Guardar este .wslconfig? [s/N]: ", false) {
			u.hint("No se cambió nada.")
			return 0
		}
	}
	if s.WSL.Exists {
		if err := os.WriteFile(path+".bak", []byte(current), 0o644); err != nil {
			u.fail("No se pudo guardar la copia " + path + ".bak: " + err.Error())
			return 1
		}
		u.ok("Copia del anterior: " + path + ".bak")
	}
	if err := os.WriteFile(path, []byte(proposed), 0o644); err != nil {
		u.fail("No se pudo escribir " + path + ": " + err.Error())
		return 1
	}
	u.ok("Guardado: " + path)
	u.hint("Se aplica la próxima vez que arranque WSL. Para aplicarlo ya: wsl --shutdown")
	u.hint("(cierra las terminales de Linux y Docker; hazlo cuando no los estés usando)")
	return 0
}
