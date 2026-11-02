// Package format contiene utilidades de presentación independientes de la plataforma:
// tamaños legibles, porcentajes, barras y separadores de miles (estilo es-EC).
package format

import (
	"strconv"
	"strings"
)

// Unidades binarias (1 GB = 1024³ bytes), igual que el Explorador de Windows.
const (
	KB int64 = 1 << 10
	MB int64 = 1 << 20
	GB int64 = 1 << 30
	TB int64 = 1 << 40
)

// Size convierte bytes a texto legible con coma decimal: 1536 → "1,5 KB".
func Size(b int64) string {
	neg := b < 0
	if neg {
		b = -b
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	f := float64(b)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	var s string
	if i == 0 {
		s = strconv.FormatInt(b, 10)
	} else {
		s = strings.Replace(strconv.FormatFloat(f, 'f', 1, 64), ".", ",", 1)
	}
	s += " " + units[i]
	if neg {
		s = "-" + s
	}
	return s
}

// Percent devuelve a/b como porcentaje entero ("42%"), "<1%" si es menor a 1, o "-" si b <= 0.
func Percent(a, b int64) string {
	if b <= 0 {
		return "-"
	}
	p := float64(a) * 100 / float64(b)
	if p < 1 {
		return "<1%"
	}
	return strconv.Itoa(int(p+0.5)) + "%"
}

// Bar dibuja una barra proporcional a/b de ancho fijo.
func Bar(a, b int64, width int) string {
	if b <= 0 || width <= 0 {
		return ""
	}
	n := min(max(int(float64(a)/float64(b)*float64(width)), 0), width)
	return strings.Repeat("█", n) + strings.Repeat("░", width-n)
}

// Thousands agrega separador de miles con punto: 1234567 → "1.234.567".
func Thousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign, s = "-", s[1:]
	}
	var out []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, s[i])
	}
	return sign + string(out)
}

// ParseSize interpreta textos como "500MB", "1,5GB", "2 tb" o "1024" (bytes).
func ParseSize(s string) (int64, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	mult := int64(1)
	for _, u := range []struct {
		suf string
		m   int64
	}{{"TB", TB}, {"GB", GB}, {"MB", MB}, {"KB", KB}, {"B", 1}} {
		if strings.HasSuffix(s, u.suf) {
			mult = u.m
			s = strings.TrimSuffix(s, u.suf)
			break
		}
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(s), ",", "."), 64)
	if err != nil {
		return 0, err
	}
	return int64(v * float64(mult)), nil
}
