// Package memory diagnostica el uso de la memoria RAM: agrupa los procesos por aplicación, mide
// si hay presión real (Windows leyendo memoria del disco) y propone acciones concretas.
//
// No "libera RAM" vaciando la memoria de los procesos: eso solo manda datos al disco y vuelve el
// equipo más lento. Es portable; la lectura de los datos de Windows vive en internal/winapi.
package memory

import (
	"sort"
	"strings"

	"mapadisco/internal/format"
)

// Snapshot es una foto de la memoria del sistema. Todos los valores en bytes.
type Snapshot struct {
	Total       int64 // RAM física
	Available   int64 // caché + libre: lo que se puede dar a un programa ya
	Free        int64 // libre y en ceros
	Cache       int64 // en caché (standby): se libera al instante si hace falta
	Modified    int64 // pendiente de escribir al disco antes de reutilizarse
	Compressed  int64 // memoria comprimida (proceso "Memory Compression")
	CommitTotal int64 // confirmada: lo que los programas reservaron (RAM + archivo de paginación)
	CommitLimit int64 // máximo confirmable (RAM + paginación)
	KernelPaged int64
	KernelNonp  int64 // no paginada: drivers y kernel, siempre en RAM
	PagesInSec  float64

	Procs   []Proc
	Startup []StartupItem
	WSL     WSLConfig
}

// InUse es la memoria en uso, como la muestra el Administrador de tareas.
func (s Snapshot) InUse() int64 { return max(s.Total-s.Available, 0) }

// Proc es un proceso con su memoria.
type Proc struct {
	Name       string // nombre del ejecutable sin .exe
	WorkingSet int64  // en RAM ahora
	Private    int64  // reservada (confirmada) por el proceso
}

// StartupItem es un programa configurado para arrancar con Windows.
type StartupItem struct {
	Name    string
	Command string
	Enabled bool
}

// Group agrupa los procesos de una misma aplicación.
type Group struct {
	Name       string
	Kind       Kind
	Count      int
	WorkingSet int64
	Private    int64
	Processes  []string // nombres de proceso distintos, para explicar qué incluye
}

// Kind clasifica un grupo para las sugerencias.
type Kind int

const (
	KindOther Kind = iota
	KindBrowser
	KindLauncher
	KindWSL
	KindDocker
	KindSystem
)

// family describe una aplicación conocida y los procesos que la forman.
type family struct {
	name  string
	kind  Kind
	procs []string // nombres en minúsculas, sin .exe
}

var families = []family{
	{"WSL (Linux de Windows)", KindWSL, []string{"vmmemwsl", "vmmem", "wslservice", "wslhost", "wslrelay"}},
	{"Docker Desktop", KindDocker, []string{"docker desktop", "com.docker.backend", "com.docker.build", "com.docker.service", "docker-agent", "com.docker.proxy"}},
	{"Microsoft Edge", KindBrowser, []string{"msedge"}},
	{"Google Chrome", KindBrowser, []string{"chrome"}},
	{"Brave", KindBrowser, []string{"brave"}},
	{"Firefox", KindBrowser, []string{"firefox"}},
	{"Opera", KindBrowser, []string{"opera", "opera_gx"}},
	{"Steam", KindLauncher, []string{"steam", "steamwebhelper", "steamservice"}},
	{"Epic Games", KindLauncher, []string{"epicgameslauncher", "epicwebhelper", "epiconlineservices", "epiconlineserviceshost", "epiconlineservicesinstallhelper", "epiconlineservicesuserhelper"}},
	{"EA app", KindLauncher, []string{"eadesktop", "eabackgroundservice", "eacefsubprocess", "eaconnect_microsoft", "ealocalhostsvc"}},
	{"Battle.net", KindLauncher, []string{"battle.net", "battle.net helper"}},
	{"Ubisoft Connect", KindLauncher, []string{"upc", "ubisoftconnect", "uplaywebcore", "upcbrowser"}},
	{"Riot Client", KindLauncher, []string{"riotclientservices", "riotclientux", "riotclientuxrender", "riotclientcrashhandler"}},
	{"App de Xbox", KindLauncher, []string{"xboxpcapp", "xboxpctray"}},
	{"Gaming Services (servicio de Windows para juegos de la Store)", KindSystem, []string{"gamingservices", "gamingservicesnet"}},
	{"WebView2 (componentes web de otras apps)", KindOther, []string{"msedgewebview2"}},
	{"Visual Studio Code", KindOther, []string{"code"}},
	{"Cursor", KindOther, []string{"cursor"}},
	{"JetBrains", KindOther, []string{"idea64", "datagrip64", "goland64", "pycharm64", "webstorm64", "phpstorm64", "rider64", "clion64", "rustrover64", "jetbrains-toolbox"}},
	{"Claude", KindOther, []string{"claude"}},
	{"Discord", KindOther, []string{"discord"}},
	{"Microsoft Teams", KindOther, []string{"ms-teams", "teams"}},
	{"Spotify", KindOther, []string{"spotify"}},
	{"Servicios de Windows (svchost)", KindSystem, []string{"svchost"}},
	{"Escritorio de Windows (dwm, explorer)", KindSystem, []string{"dwm", "explorer"}},
	{"Memoria comprimida de Windows", KindSystem, []string{"memory compression"}},
}

var byProc = func() map[string]*family {
	m := map[string]*family{}
	for i := range families {
		for _, p := range families[i].procs {
			m[p] = &families[i]
		}
	}
	return m
}()

// GroupProcesses agrupa los procesos por aplicación y los ordena por memoria reservada.
func GroupProcesses(procs []Proc) []Group {
	idx := map[string]*Group{}
	var order []string
	seen := map[string]map[string]bool{}
	for _, p := range procs {
		key := strings.ToLower(strings.TrimSuffix(p.Name, ".exe"))
		name, kind := p.Name, KindOther
		if f := byProc[key]; f != nil {
			name, kind = f.name, f.kind
		}
		g := idx[name]
		if g == nil {
			g = &Group{Name: name, Kind: kind}
			idx[name] = g
			order = append(order, name)
			seen[name] = map[string]bool{}
		}
		g.Count++
		g.WorkingSet += p.WorkingSet
		g.Private += p.Private
		if !seen[name][key] {
			seen[name][key] = true
			g.Processes = append(g.Processes, p.Name)
		}
	}
	out := make([]Group, 0, len(order))
	for _, n := range order {
		out = append(out, *idx[n])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Private > out[j].Private })
	return out
}

// Level es el nivel de presión de memoria.
type Level int

const (
	PressureLow Level = iota
	PressureMedium
	PressureHigh
)

// Pressure resume si la falta de memoria afecta de verdad: lo que pesa es cuánto lee Windows del
// disco para recuperar memoria (páginas de entrada por segundo), no cuánta queda "disponible".
func Pressure(s Snapshot) (Level, string) {
	availPct := 0.0
	if s.Total > 0 {
		availPct = float64(s.Available) / float64(s.Total) * 100
	}
	commitPct := 0.0
	if s.CommitLimit > 0 {
		commitPct = float64(s.CommitTotal) / float64(s.CommitLimit) * 100
	}
	switch {
	case s.PagesInSec >= 300 || (availPct < 5 && s.PagesInSec >= 50) || commitPct >= 90:
		return PressureHigh, "Windows está sacando memoria del disco a menudo o está cerca del límite: el equipo se siente lento"
	case availPct < 15 || s.PagesInSec >= 50 || commitPct >= 75:
		return PressureMedium, "la memoria está justa: hoy alcanza, pero abrir algo grande hará que Windows use el disco"
	default:
		return PressureLow, "hay margen: la memoria se está aprovechando, no falta"
	}
}

// Tip es una sugerencia concreta.
type Tip struct {
	Title   string
	Detail  string
	Steps   []string
	Savings int64 // estimación de memoria recuperable; 0 si no se puede estimar
}

// Lanzadores de juegos que suelen arrancar con Windows, por fragmento del nombre o del comando en minúsculas.
var launcherStartup = []string{"steam", "epicgames", "epic games", "eadesktop", "ea desktop", "electronic arts", "battle.net", "ubisoft", "upc.exe", "riotclient", "xboxpcapp"}

// Advice genera las sugerencias a partir de la foto y los grupos.
func Advice(s Snapshot, groups []Group) []Tip {
	var tips []Tip

	// 1. WSL: sin límite, toma hasta la mitad de la RAM y no la devuelve.
	if g := find(groups, KindWSL); g != nil && g.Private > 0 {
		target := SuggestedWSLMemory(s.Total)
		var steps []string
		switch {
		case s.WSL.Memory == 0:
			steps = append(steps, "Limita su memoria y haz que devuelva la que deja de usar: mapadisco -ram -wslconfig "+
				"(muestra el cambio y pregunta antes de escribir C:\\Users\\<tu usuario>\\.wslconfig)")
		case s.WSL.AutoReclaim == "" || strings.EqualFold(s.WSL.AutoReclaim, "disabled"):
			steps = append(steps, "Haz que devuelva la memoria que deja de usar: mapadisco -ram -wslconfig")
		}
		steps = append(steps,
			"Se aplica al reiniciar WSL: wsl --shutdown (cierra las terminales de Linux y Docker; hazlo cuando no los uses)",
			"Si no estás usando Linux ni Docker ahora, wsl --shutdown libera toda esa memoria al instante")
		detail := "Linux usa la memoria libre como caché de archivos y Windows la ve como ocupada."
		if s.WSL.Memory == 0 {
			detail = "Sin límite configurado, WSL puede tomar hasta la mitad de tu RAM (" + format.Size(s.Total/2) +
				") y no la devuelve sola. " + detail
		} else {
			detail = "Límite actual en .wslconfig: " + format.Size(s.WSL.Memory) + ". " + detail
		}
		if s.WSL.Memory == 0 || s.WSL.AutoReclaim == "" || strings.EqualFold(s.WSL.AutoReclaim, "disabled") {
			tips = append(tips, Tip{
				Title:   "WSL (Linux de Windows) reserva " + format.Size(g.Private),
				Detail:  detail + " Límite sugerido para tu equipo: " + format.Size(target) + ".",
				Steps:   steps,
				Savings: max(g.Private-target, 0),
			})
		}
	}

	// 2. Lanzadores de juegos abiertos y que arrancan con Windows.
	var launchers []Group
	var launcherMem int64
	for _, g := range groups {
		if g.Kind == KindLauncher {
			launchers = append(launchers, g)
			launcherMem += g.Private
		}
	}
	if len(launchers) > 0 {
		names := make([]string, len(launchers))
		for i, l := range launchers {
			names[i] = l.Name
		}
		steps := []string{"Ciérralos desde el icono junto al reloj cuando no estés jugando (cerrar la ventana no basta: quedan en segundo plano)"}
		enabled, disabled := startupLaunchers(s.Startup)
		if len(enabled) > 0 {
			steps = append(steps, "Arrancan con Windows: "+strings.Join(enabled, ", ")+". Desactívalos en Administrador de tareas > "+
				"Aplicaciones de arranque, o en Configuración > Aplicaciones > Inicio (start ms-settings:startupapps)")
		}
		if len(disabled) > 0 {
			steps = append(steps, "Ya no arrancan con Windows: "+strings.Join(disabled, ", ")+" (se abrieron después, al usarlos o desde otro programa)")
		}
		tips = append(tips, Tip{
			Title:   "Lanzadores de juegos abiertos: " + strings.Join(names, ", "),
			Detail:  "Ocupan memoria aunque no estés jugando.",
			Steps:   steps,
			Savings: launcherMem,
		})
	}

	// 3. Varios navegadores a la vez.
	var browsers []Group
	for _, g := range groups {
		if g.Kind == KindBrowser {
			browsers = append(browsers, g)
		}
	}
	if len(browsers) >= 2 {
		names := make([]string, len(browsers))
		var smaller int64
		for i, b := range browsers {
			names[i] = b.Name
			if i > 0 {
				smaller += b.Private
			}
		}
		tips = append(tips, Tip{
			Title:  "Varios navegadores abiertos: " + strings.Join(names, ", "),
			Detail: "Cada navegador carga su propio motor y sus extensiones.",
			Steps: []string{
				"Usa uno solo a la vez",
				"En Edge activa las pestañas en suspensión: edge://settings/system/managePerformance",
				"En Chrome, «Ahorro de memoria»: chrome://settings/performance",
			},
			Savings: smaller,
		})
	}

	// 4. Docker Desktop abierto.
	if g := find(groups, KindDocker); g != nil {
		tips = append(tips, Tip{
			Title:  "Docker Desktop está abierto",
			Detail: "Además de sus procesos, sus contenedores corren dentro de WSL y suman a su memoria.",
			Steps:  []string{"Si no lo estás usando, ciérralo desde el icono junto al reloj (Quit Docker Desktop)"},
		})
	}

	// 5. Memoria no paginada alta (drivers).
	if s.KernelNonp > 1<<30 {
		tips = append(tips, Tip{
			Title:  "La memoria de drivers (no paginada) es alta: " + format.Size(s.KernelNonp),
			Detail: "Lo normal suele ser menos de 1 GB. Si crece día a día sin reiniciar, algún driver o el antivirus pierde memoria.",
			Steps: []string{
				"Actualiza los drivers (tarjeta gráfica, red) y el antivirus",
				"Si sigue creciendo, el área de Tecnología puede identificarlo con poolmon (Windows Driver Kit)",
			},
		})
	}

	// 6. Cerca del límite de memoria confirmada.
	if s.CommitLimit > 0 && float64(s.CommitTotal) >= 0.9*float64(s.CommitLimit) {
		tips = append(tips, Tip{
			Title:  "Casi sin memoria confirmable",
			Detail: "Si se llega al límite, Windows avisará «memoria insuficiente» y cerrará programas.",
			Steps:  []string{"Cierra programas que no uses", "No reduzcas ni desactives el archivo de paginación"},
		})
	}

	sort.SliceStable(tips, func(i, j int) bool { return tips[i].Savings > tips[j].Savings })
	return tips
}

// startupLaunchers separa los programas de arranque que son lanzadores en habilitados y deshabilitados.
func startupLaunchers(items []StartupItem) (enabled, disabled []string) {
	for _, it := range items {
		text := strings.ToLower(it.Name + " " + it.Command)
		for _, l := range launcherStartup {
			if strings.Contains(text, l) {
				if it.Enabled {
					enabled = append(enabled, it.Name)
				} else {
					disabled = append(disabled, it.Name)
				}
				break
			}
		}
	}
	return enabled, disabled
}

func find(groups []Group, k Kind) *Group {
	for i := range groups {
		if groups[i].Kind == k {
			return &groups[i]
		}
	}
	return nil
}

// SuggestedWSLMemory propone el límite de WSL: ~3/8 de la RAM, entre 4 y 8 GB, en GB enteros.
func SuggestedWSLMemory(total int64) int64 {
	const gb = 1 << 30
	s := (total*3/8 + gb/2) / gb
	return min(max(s, 4), 8) * gb
}
