package memory

import (
	"strings"
	"testing"
)

const gb = int64(1) << 30

func TestGroupProcesses(t *testing.T) {
	procs := []Proc{
		{Name: "msedge", WorkingSet: 100, Private: 300},
		{Name: "msedge", WorkingSet: 50, Private: 200},
		{Name: "steamwebhelper", WorkingSet: 40, Private: 400},
		{Name: "steam", WorkingSet: 10, Private: 100},
		{Name: "vmmemWSL", WorkingSet: 2000, Private: 5000},
		{Name: "miapp", WorkingSet: 5, Private: 7},
	}
	g := GroupProcesses(procs)
	if g[0].Name != "WSL (Linux de Windows)" || g[0].Kind != KindWSL || g[0].Private != 5000 {
		t.Fatalf("primer grupo = %+v", g[0])
	}
	var steam, edge *Group
	for i := range g {
		switch g[i].Name {
		case "Steam":
			steam = &g[i]
		case "Microsoft Edge":
			edge = &g[i]
		}
	}
	if steam == nil || steam.Count != 2 || steam.Private != 500 || steam.Kind != KindLauncher || len(steam.Processes) != 2 {
		t.Errorf("Steam = %+v", steam)
	}
	if edge == nil || edge.Count != 2 || edge.WorkingSet != 150 || len(edge.Processes) != 1 {
		t.Errorf("Edge = %+v", edge)
	}
	if g[len(g)-1].Name != "miapp" || g[len(g)-1].Kind != KindOther {
		t.Errorf("proceso desconocido = %+v", g[len(g)-1])
	}
}

func TestPressure(t *testing.T) {
	base := Snapshot{Total: 16 * gb, Available: 8 * gb, CommitTotal: 10 * gb, CommitLimit: 40 * gb}
	if l, _ := Pressure(base); l != PressureLow {
		t.Errorf("con margen = %v", l)
	}
	justa := base
	justa.Available = 2 * gb // 12,5 %
	if l, _ := Pressure(justa); l != PressureMedium {
		t.Errorf("disponible bajo sin lecturas del disco = %v, quiero media", l)
	}
	lenta := justa
	lenta.PagesInSec = 500
	if l, _ := Pressure(lenta); l != PressureHigh {
		t.Errorf("muchas lecturas del disco = %v, quiero alta", l)
	}
	limite := base
	limite.CommitTotal = 37 * gb
	if l, _ := Pressure(limite); l != PressureHigh {
		t.Errorf("cerca del límite confirmable = %v, quiero alta", l)
	}
}

func TestSuggestedWSLMemory(t *testing.T) {
	cases := map[int64]int64{8 * gb: 4 * gb, 16 * gb: 6 * gb, 32 * gb: 8 * gb, 64 * gb: 8 * gb}
	for total, want := range cases {
		if got := SuggestedWSLMemory(total); got != want {
			t.Errorf("SuggestedWSLMemory(%d GB) = %d GB, quiero %d", total/gb, got/gb, want/gb)
		}
	}
}

func TestAdvice(t *testing.T) {
	s := Snapshot{
		Total: 16 * gb, Available: 2 * gb, CommitTotal: 28 * gb, CommitLimit: 41 * gb, KernelNonp: 3 * gb / 2,
		Startup: []StartupItem{
			{Name: "Steam", Command: `"C:\Program Files (x86)\Steam\steam.exe" -silent`, Enabled: true},
			{Name: "EpicGamesLauncher", Command: `C:\x\EpicGamesLauncher.exe`, Enabled: false},
			{Name: "OneDrive", Command: `OneDrive.exe /background`, Enabled: true},
			{Name: "Spotify", Command: `C:\Users\Usuario\AppData\Roaming\Spotify\Spotify.exe --autostart`, Enabled: false},
		},
	}
	groups := GroupProcesses([]Proc{
		{Name: "vmmemWSL", Private: 5 * gb},
		{Name: "steamwebhelper", Private: gb},
		{Name: "EpicWebHelper", Private: gb},
		{Name: "msedge", Private: gb},
		{Name: "chrome", Private: gb / 2},
		{Name: "com.docker.backend", Private: gb / 10},
	})
	tips := Advice(s, groups)
	text := ""
	for _, tp := range tips {
		text += tp.Title + "|" + tp.Detail + "|" + strings.Join(tp.Steps, "|") + "\n"
	}
	for _, want := range []string{"WSL", "Lanzadores", "Varios navegadores", "Docker", "drivers", "mapadisco -ram -wslconfig", "Arrancan con Windows: Steam"} {
		if !strings.Contains(text, want) {
			t.Errorf("falta %q en las sugerencias:\n%s", want, text)
		}
	}
	if !strings.Contains(text, "Ya no arrancan con Windows: EpicGamesLauncher") {
		t.Errorf("debe aclarar que Epic ya está deshabilitado al arrancar:\n%s", text)
	}
	if strings.Contains(text, "Arrancan con Windows: Steam, EpicGamesLauncher") || strings.Contains(text, "OneDrive") || strings.Contains(text, "Spotify") {
		t.Errorf("no debe mezclar arranques deshabilitados ni programas que no son lanzadores:\n%s", text)
	}
	// Ordenadas por ahorro inmediato: los lanzadores liberan 2 GB al cerrarlos; limitar WSL a 6 GB
	// cuando usa 5 GB no libera nada ya (evita que crezca), así que no va primero.
	if !strings.HasPrefix(tips[0].Title, "Lanzadores") || tips[0].Savings != 2*gb {
		t.Errorf("primera sugerencia = %q (%d), quiero los lanzadores con 2 GB", tips[0].Title, tips[0].Savings)
	}
}

func TestAdviceWSLYaConfigurado(t *testing.T) {
	s := Snapshot{Total: 16 * gb, WSL: WSLConfig{Exists: true, Memory: 6 * gb, AutoReclaim: "gradual"}}
	tips := Advice(s, GroupProcesses([]Proc{{Name: "vmmemWSL", Private: 3 * gb}}))
	for _, tp := range tips {
		if strings.HasPrefix(tp.Title, "WSL") {
			t.Errorf("con límite y devolución configurados no debe sugerir WSL: %+v", tp)
		}
	}
}

func TestParseWSLConfig(t *testing.T) {
	c := ParseWSLConfig("# mío\r\n[wsl2]\r\nmemory = 6GB # límite\r\nprocessors=4\r\n\r\n[experimental]\r\nautoMemoryReclaim=Gradual\r\n")
	if c.Memory != 6*gb || c.AutoReclaim != "gradual" {
		t.Errorf("ParseWSLConfig = %+v", c)
	}
	if c := ParseWSLConfig("[wsl2]\nmemory=4096MB\n"); c.Memory != 4*gb {
		t.Errorf("4096MB = %d", c.Memory)
	}
	if c := ParseWSLConfig("[otra]\nmemory=2GB\n"); c.Memory != 0 {
		t.Error("memory fuera de [wsl2] no cuenta")
	}
}

func TestSetKeyConservaElResto(t *testing.T) {
	in := "# config de Usuario\n[wsl2]\nprocessors=4\nmemory=12GB\n\n[otra]\nx=1\n"
	got := SetKey(in, "wsl2", "memory", "6GB")
	want := "# config de Usuario\n[wsl2]\nprocessors=4\nmemory=6GB\n\n[otra]\nx=1\n"
	if got != want {
		t.Errorf("reemplazar:\n%q\nquiero\n%q", got, want)
	}
	got = SetKey("[wsl2]\nprocessors=4\n\n[otra]\nx=1\n", "wsl2", "memory", "6GB")
	if got != "[wsl2]\nprocessors=4\nmemory=6GB\n\n[otra]\nx=1\n" {
		t.Errorf("agregar en la sección:\n%q", got)
	}
	got = SetKey("[wsl2]\r\nprocessors=4\r\n", "experimental", "autoMemoryReclaim", "gradual")
	if got != "[wsl2]\r\nprocessors=4\r\n\r\n[experimental]\r\nautoMemoryReclaim=gradual\r\n" {
		t.Errorf("sección nueva con CRLF:\n%q", got)
	}
	if got := SetKey("", "wsl2", "memory", "6GB"); got != "[wsl2]\nmemory=6GB\n" {
		t.Errorf("archivo vacío:\n%q", got)
	}
}

func TestProposeWSLConfig(t *testing.T) {
	got := ProposeWSLConfig("", 16*gb)
	if c := ParseWSLConfig(got); c.Memory != 6*gb || c.AutoReclaim != "gradual" {
		t.Errorf("propuesta = %q", got)
	}
	// Un límite que el usuario ya puso se respeta.
	got = ProposeWSLConfig("[wsl2]\nmemory=10GB\n", 16*gb)
	if c := ParseWSLConfig(got); c.Memory != 10*gb || c.AutoReclaim != "gradual" {
		t.Errorf("respetar el límite existente: %q", got)
	}
	// Nada que cambiar: idéntico.
	same := "[wsl2]\nmemory=4GB\n[experimental]\nautoMemoryReclaim=dropcache\n"
	if got := ProposeWSLConfig(same, 16*gb); got != same {
		t.Errorf("sin cambios debería quedar igual: %q", got)
	}
}
