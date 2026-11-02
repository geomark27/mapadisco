package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"mapadisco/internal/cleaner"
	"mapadisco/internal/format"
	"mapadisco/internal/report"
	"mapadisco/internal/winapi"
)

type cleanItem struct {
	target cleaner.Target
	paths  []string
	usage  cleaner.Usage
}

// runClean mide las categorías que se pueden limpiar, pregunta cuáles (o todas con -yes), las limpia
// y explica a dónde ir con lo que Windows no dejó borrar y con lo que solo se libera a mano.
func runClean(u ui, out *report.Printer, yes bool, only string, elevated bool) int {
	drive := os.Getenv("SystemDrive")
	if drive == "" {
		drive = "C:"
	}
	drive += `\`
	_, freeBefore, okBefore := winapi.DiskSpace(drive)

	out.Section("LIBERAR ESPACIO")
	fmt.Printf("  %sMidiendo lo que se puede liberar…%s\n", u.c.Dim, u.c.Reset)
	targets, _ := cleaner.Select(only) // ya validado en parseFlags
	items := measureTargets(targets)
	if len(items) == 0 {
		u.ok("No hay nada que limpiar en las categorías automáticas.")
		printManualGuides(u, out)
		return 0
	}

	var total int64
	for i, it := range items {
		total += it.usage.Bytes
		fmt.Printf("  %s%3d%s  %10s  %s%s%s\n", u.c.Cyan, i+1, u.c.Reset, format.Size(it.usage.Bytes), u.c.Bold, it.target.Label, u.c.Reset)
		note := it.target.Note
		if it.target.Admin && !elevated {
			note += " · " + u.c.Yellow + "parte requiere Administrador" + u.c.Dim
		}
		fmt.Printf("                   %s%s%s\n", u.c.Dim, note, u.c.Reset)
	}
	fmt.Printf("       %s%10s  en total%s\n\n", u.c.Bold, format.Size(total), u.c.Reset)

	selected, proceed := chooseTargets(u, len(items), yes)
	if !proceed {
		printManualGuides(u, out)
		return 0
	}
	if len(selected) == 0 {
		u.hint("No se borró nada.")
		printManualGuides(u, out)
		return 0
	}

	fmt.Println()
	var freed int64
	for _, i := range selected {
		freed += cleanOne(u, items[i], elevated)
	}

	fmt.Println()
	if _, freeAfter, ok := winapi.DiskSpace(drive); ok && okBefore {
		fmt.Printf("  %sEspacio libre en %s%s %s → %s (%s+%s%s)\n", u.c.Bold, drive, u.c.Reset,
			format.Size(freeBefore), format.Size(freeAfter), u.c.Green, format.Size(max(freeAfter-freeBefore, 0)), u.c.Reset)
	} else {
		fmt.Printf("  %sLiberado:%s %s\n", u.c.Bold, u.c.Reset, format.Size(freed))
	}
	printManualGuides(u, out)
	return 0
}

// measureTargets mide en paralelo las categorías automáticas y devuelve las que tienen algo.
func measureTargets(targets []cleaner.Target) []cleanItem {
	all := make([]cleanItem, len(targets))
	var wg sync.WaitGroup
	for i, t := range targets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			it := cleanItem{target: t}
			if t.Kind == cleaner.RecycleBin {
				if b, n, err := winapi.RecycleBinUsage(); err == nil {
					it.usage = cleaner.Usage{Bytes: b, Files: n}
				}
			} else {
				it.paths = cleaner.Resolve(t.Paths, os.Getenv)
				it.usage = cleaner.Measure(it.paths)
			}
			all[i] = it
		}()
	}
	wg.Wait()
	var out []cleanItem
	for _, it := range all {
		if it.usage.Bytes > 0 || it.usage.Files > 0 {
			out = append(out, it)
		}
	}
	return out
}

// chooseTargets devuelve los índices a limpiar. proceed=false cuando no se puede preguntar.
func chooseTargets(u ui, n int, yes bool) (selected []int, proceed bool) {
	if yes {
		sel, _ := cleaner.ParseSelection("t", n)
		return sel, true
	}
	if !winapi.StdinIsConsole() {
		u.hint("Para limpiar todo sin preguntar: mapadisco -clean -yes")
		return nil, false
	}
	for {
		fmt.Print("¿Qué limpio? [t = todo · 1,3 = solo esos · Enter = nada]: ")
		answer, err := stdin.ReadString('\n')
		if err != nil {
			return nil, true
		}
		sel, err := cleaner.ParseSelection(answer, n)
		if err == nil {
			return sel, true
		}
		u.fail(err.Error())
	}
}

// cleanOne limpia una categoría, informa el resultado y, si Windows se opuso, a dónde ir.
func cleanOne(u ui, it cleanItem, elevated bool) int64 {
	t := it.target
	if t.Kind == cleaner.RecycleBin {
		if err := winapi.EmptyRecycleBin(); err != nil {
			u.fail(fmt.Sprintf("%s: Windows no la vació (%v)", t.Label, err))
			u.hint("→ A mano: " + t.Manual)
			return 0
		}
		u.ok(fmt.Sprintf("%s: %s liberados (%s)", t.Label, format.Size(it.usage.Bytes), count(it.usage.Files, "elemento", "elementos")))
		return it.usage.Bytes
	}

	r, err := cleaner.Clean(it.paths, cleaner.Options{InUse: winapi.IsInUse})
	if err != nil {
		u.fail(fmt.Sprintf("%s: %v", t.Label, err))
		return 0
	}
	summary := fmt.Sprintf("%s: %s liberados (%s)", t.Label, format.Size(r.Freed), count(r.Deleted, "archivo", "archivos"))
	if !r.Opposed() {
		u.ok(summary)
		return r.Freed
	}

	fmt.Printf("  %s⚠%s %s; Windows no dejó borrar %s:\n", u.c.Yellow, u.c.Reset, summary, format.Thousands(r.InUse+r.Denied+r.Failed))
	if r.InUse > 0 {
		u.hint(fmt.Sprintf("· %s en uso por otro programa (%s) → cierra los programas abiertos o reinicia, y vuelve a ejecutar mapadisco -clean",
			format.Thousands(r.InUse), format.Size(r.InUseBytes)))
	}
	if r.Denied > 0 {
		msg := fmt.Sprintf("· %s sin permiso", format.Thousands(r.Denied))
		if !elevated {
			msg += " → mapadisco -clean -admin"
		}
		u.hint(msg)
	}
	if r.Failed > 0 {
		u.hint(fmt.Sprintf("· %s con otros errores", format.Thousands(r.Failed)))
	}
	for _, e := range r.Examples {
		u.hint(u.c.Dim + "  ej.: " + e + u.c.Reset)
	}
	u.hint("→ A mano: " + t.Manual)
	if folder := folderOf(r.Examples, it.paths); folder != "" {
		u.hint(`→ Abrir la carpeta: explorer "` + folder + `"`)
	}
	return r.Freed
}

func count(n int64, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return format.Thousands(n) + " " + many
}

// folderOf elige qué carpeta abrir: la categoría que contiene el primer ejemplo.
func folderOf(examples, paths []string) string {
	for _, e := range examples {
		for _, p := range paths {
			if strings.HasPrefix(strings.ToLower(e), strings.ToLower(p)) {
				return p
			}
		}
	}
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}

// printManualGuides mide y muestra lo que no se borra automáticamente, con los pasos para hacerlo a mano.
func printManualGuides(u ui, out *report.Printer) {
	type found struct {
		g     cleaner.Guide
		paths []string
		usage cleaner.Usage
	}
	all := make([]found, len(cleaner.Manual))
	var wg sync.WaitGroup
	for i, g := range cleaner.Manual {
		wg.Add(1)
		go func() {
			defer wg.Done()
			paths := cleaner.Resolve(g.Paths, os.Getenv)
			all[i] = found{g, paths, cleaner.Measure(paths)}
		}()
	}
	wg.Wait()

	var shown []found
	for _, f := range all {
		if f.usage.Bytes >= 100*format.MB || (len(f.paths) > 0 && f.usage.Denied && f.usage.Bytes == 0) {
			shown = append(shown, f)
		}
	}
	if len(shown) == 0 {
		return
	}
	sort.SliceStable(shown, func(i, j int) bool { return shown[i].usage.Bytes > shown[j].usage.Bytes })

	out.Section("PARA LIBERAR MÁS (A MANO)")
	fmt.Printf("  %smapadisco no borra esto por su cuenta: o es del sistema, o son datos tuyos.%s\n", u.c.Dim, u.c.Reset)
	for _, f := range shown {
		size := format.Size(f.usage.Bytes)
		if f.usage.Bytes == 0 {
			size = "?"
		}
		fmt.Printf("  %10s  %s%s%s\n", size, u.c.Bold, f.g.Label, u.c.Reset)
		for i, p := range f.paths {
			if i == 2 {
				fmt.Printf("              %s… y %d más%s\n", u.c.Dim, len(f.paths)-2, u.c.Reset)
				break
			}
			fmt.Printf("              %s%s%s\n", u.c.Dim, p, u.c.Reset)
		}
		for _, s := range f.g.Steps {
			fmt.Printf("              %s→ %s%s\n", u.c.Green, s, u.c.Reset)
		}
	}
}
