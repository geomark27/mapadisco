package memory

import (
	"strconv"
	"strings"
)

// WSLConfig es lo que importa de %USERPROFILE%\.wslconfig para la memoria.
type WSLConfig struct {
	Path        string
	Exists      bool
	Memory      int64  // [wsl2] memory; 0 = sin límite configurado (WSL usa hasta la mitad de la RAM)
	AutoReclaim string // [experimental] autoMemoryReclaim: "", "disabled", "gradual" o "dropcache"
}

// ParseWSLConfig lee el contenido de un .wslconfig.
func ParseWSLConfig(content string) WSLConfig {
	var c WSLConfig
	section := ""
	for line := range strings.Lines(content) {
		l := strings.TrimSpace(line)
		switch {
		case l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, ";"):
			continue
		case strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]"):
			section = strings.ToLower(strings.TrimSpace(l[1 : len(l)-1]))
			continue
		}
		k, v, ok := strings.Cut(l, "=")
		if !ok {
			continue
		}
		k, v = strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(stripComment(v))
		switch {
		case section == "wsl2" && k == "memory":
			c.Memory = ParseSize(v)
		case (section == "experimental" || section == "wsl2") && k == "automemoryreclaim":
			c.AutoReclaim = strings.ToLower(v)
		}
	}
	return c
}

func stripComment(v string) string {
	if i := strings.IndexAny(v, "#;"); i >= 0 {
		v = v[:i]
	}
	return strings.Trim(strings.TrimSpace(v), `"`)
}

// ParseSize interpreta tamaños de .wslconfig: "6GB", "4096MB", "8G" (sin unidad, bytes).
func ParseSize(v string) int64 {
	v = strings.ToUpper(strings.TrimSpace(v))
	mult := int64(1)
	for _, u := range []struct {
		suf string
		m   int64
	}{{"TB", 1 << 40}, {"GB", 1 << 30}, {"MB", 1 << 20}, {"KB", 1 << 10}, {"T", 1 << 40}, {"G", 1 << 30}, {"M", 1 << 20}, {"K", 1 << 10}, {"B", 1}} {
		if strings.HasSuffix(v, u.suf) {
			mult = u.m
			v = strings.TrimSuffix(v, u.suf)
			break
		}
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil || n <= 0 {
		return 0
	}
	return int64(n * float64(mult))
}

// SetKey fija key=value dentro de [section] y deja el resto del archivo tal cual: comentarios,
// otras secciones y otras claves. Si la clave existe la reemplaza; si no, la agrega al final de
// la sección; si la sección no existe, la crea al final. Respeta los finales de línea (CRLF o LF).
func SetKey(content, section, key, value string) string {
	nl := "\n"
	if strings.Contains(content, "\r\n") {
		nl = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	trailing := len(lines) > 0 && lines[len(lines)-1] == ""
	if trailing {
		lines = lines[:len(lines)-1]
	}

	want := strings.ToLower(section)
	cur, start, end := "", -1, -1 // [start, end): líneas de la sección buscada
	for i, line := range lines {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]") {
			if cur == want && end < 0 {
				end = i
			}
			cur = strings.ToLower(strings.TrimSpace(l[1 : len(l)-1]))
			if cur == want && start < 0 {
				start = i + 1
			}
			continue
		}
		if cur != want {
			continue
		}
		k, _, ok := strings.Cut(l, "=")
		if ok && !strings.HasPrefix(l, "#") && !strings.HasPrefix(l, ";") && strings.EqualFold(strings.TrimSpace(k), key) {
			lines[i] = key + "=" + value
			return join(lines, nl, trailing)
		}
	}
	entry := key + "=" + value
	switch {
	case start < 0:
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "["+section+"]", entry)
	default:
		if end < 0 {
			end = len(lines)
		}
		// después de la última línea no vacía de la sección
		at := end
		for at > start && strings.TrimSpace(lines[at-1]) == "" {
			at--
		}
		lines = append(lines[:at], append([]string{entry}, lines[at:]...)...)
	}
	return join(lines, nl, true)
}

func join(lines []string, nl string, trailing bool) string {
	s := strings.Join(lines, nl)
	if trailing {
		s += nl
	}
	return s
}

// ProposeWSLConfig devuelve el .wslconfig con el límite de memoria y la devolución gradual de
// memoria, partiendo del contenido actual (vacío si no existe). Si un valor ya está, lo conserva
// salvo que no haya límite.
func ProposeWSLConfig(current string, total int64) string {
	c := ParseWSLConfig(current)
	out := current
	if c.Memory == 0 {
		out = SetKey(out, "wsl2", "memory", strconv.FormatInt(SuggestedWSLMemory(total)>>30, 10)+"GB")
	}
	if c.AutoReclaim == "" || c.AutoReclaim == "disabled" {
		out = SetKey(out, "experimental", "autoMemoryReclaim", "gradual")
	}
	return out
}
