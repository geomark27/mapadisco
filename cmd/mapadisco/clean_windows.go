package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
// live indica si se puede mostrar el avance en la consola (stderr).
func runClean(u ui, out *report.Printer, yes bool, only string, elevated, live bool) int {
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
		freed += cleanOne(u, items[i], elevated, live)
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
// live indica si se puede mostrar el avance en la consola (stderr).
func cleanOne(u ui, it cleanItem, elevated, live bool) int64 {
	t := it.target
	if t.Kind == cleaner.RecycleBin {
		return cleanRecycleBin(u, it, elevated, live)
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

// cleanRecycleBin vacía la Papelera y cuenta lo que de verdad se fue: si Windows pidió permiso y se
// canceló, o algo falló a medias, informa lo que quedó y qué hacer.
func cleanRecycleBin(u ui, it cleanItem, elevated, live bool) int64 {
	t := it.target
	before := it.usage
	after, err := emptyRecycleBin(u, before, live)
	freed := max(before.Bytes-after.Bytes, 0)
	items := max(before.Files-after.Files, 0)
	canceled := errors.Is(err, winapi.ErrCanceled)

	if err != nil && !canceled && freed == 0 {
		u.fail(fmt.Sprintf("%s: Windows no la vació (%v)", t.Label, err))
		u.hint("→ A mano: " + t.Manual)
		return 0
	}
	summary := fmt.Sprintf("%s: %s liberados (%s)", t.Label, format.Size(freed), count(items, "elemento", "elementos"))
	if err == nil && after.Files == 0 {
		u.ok(summary)
		return freed
	}

	fmt.Printf("  %s⚠%s %s; quedaron %s (%s)\n", u.c.Yellow, u.c.Reset, summary,
		count(after.Files, "elemento", "elementos"), format.Size(after.Bytes))
	switch {
	case canceled:
		u.hint("· se canceló la ventana de Windows que pedía permiso")
	case err != nil:
		u.hint(fmt.Sprintf("· Windows no terminó: %v", err))
	}
	if !elevated {
		u.hint("→ Los que piden permiso de Administrador: mapadisco -clean -admin -only papelera")
	}
	u.hint("→ A mano: " + t.Manual)
	return freed
}

// emptyRecycleBin vacía la Papelera y devuelve lo que quedó en ella. Windows no informa el avance y,
// si un elemento necesita permiso de Administrador, abre una ventana y espera en silencio; por eso,
// con live, se muestra cuánto falta y se avisa si deja de avanzar.
func emptyRecycleBin(u ui, before cleaner.Usage, live bool) (after cleaner.Usage, err error) {
	done := make(chan error, 1)
	go func() { done <- winapi.EmptyRecycleBin() }()

	if live {
		err = watchRecycleBin(u, before.Files, done)
	} else {
		fmt.Printf("  %sVaciando la Papelera (%s)… si Windows pide permiso, su ventana puede quedar detrás de esta.%s\n",
			u.c.Dim, count(before.Files, "elemento", "elementos"), u.c.Reset)
		err = <-done
	}

	b, n, qerr := winapi.RecycleBinUsage()
	switch {
	case qerr == nil:
		return cleaner.Usage{Bytes: b, Files: n}, err
	case err != nil:
		return before, err // no se sabe cuánto se fue: se asume que nada
	default:
		return cleaner.Usage{}, nil
	}
}

// watchRecycleBin muestra en stderr cuántos elementos van mientras Windows vacía la Papelera, hasta
// que done entrega el resultado. Si deja de avanzar, avisa una vez dónde buscar la ventana de Windows.
func watchRecycleBin(u ui, total int64, done <-chan error) error {
	const spin = `|/-\`
	const clear = "\r\x1b[2K"
	start := time.Now()
	drain := cleaner.NewDrain(total, start)

	// Consultar la Papelera tarda si tiene mucho: se hace aparte, una consulta a la vez.
	var left atomic.Int64
	left.Store(-1)
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if _, n, err := winapi.RecycleBinUsage(); err == nil {
					left.Store(n)
				}
			}
		}
	}()

	warned := false
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for i := 0; ; i++ {
		select {
		case err := <-done:
			fmt.Fprint(os.Stderr, clear)
			return err
		case now := <-tick.C:
			if n := left.Load(); n >= 0 {
				drain.Update(n, now)
			}
			if !warned && drain.Stalled(now) {
				warned = true
				fmt.Fprintf(os.Stderr, "%s  %s⚠%s La Papelera no avanza hace unos segundos. Si Windows pide permiso de Administrador,\n"+
					"    su ventana puede estar detrás de esta terminal o parpadeando en la barra de tareas: respóndela para seguir.\n",
					clear, u.c.Yellow, u.c.Reset)
			}
			fmt.Fprintf(os.Stderr, "%s  %c Vaciando la Papelera… %s de %s elementos · %ds", clear, spin[i%len(spin)],
				format.Thousands(drain.Done()), format.Thousands(drain.Total()), int(now.Sub(start).Seconds()))
		}
	}
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
