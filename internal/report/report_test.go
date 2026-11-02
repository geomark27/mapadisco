package report

import (
	"bytes"
	"strings"
	"testing"

	"mapadisco/internal/format"
	"mapadisco/internal/scanner"
)

func add(parent *scanner.Node, name string, size int64) *scanner.Node {
	n := &scanner.Node{Name: name, Parent: parent, Size: size, Files: 1}
	parent.Children = append(parent.Children, n)
	return n
}

// sample: 100 GB en C:\Users\Marcos\AppData\Local\NVIDIA (profundo) + 20 GB en Windows.
func sample() *scanner.Node {
	root := &scanner.Node{Name: `C:\`}
	local := add(add(add(add(root, "Users", 0), "Marcos", 0), "AppData", 0), "Local", 0)
	add(local, "NVIDIA", 100*format.GB)
	add(root, "Windows", 20*format.GB)
	add(root, "Pequeña", 5*format.MB)
	scanner.SumTotals(root)
	return root
}

func render(o TreeOptions) string {
	var buf bytes.Buffer
	New(&buf, NewPalette(false)).Tree(sample(), o)
	return buf.String()
}

func TestTreeAutoExpandeAlCulpable(t *testing.T) {
	out := render(DefaultTreeOptions) // profundidad 3, pero NVIDIA está en el nivel 5
	if !strings.Contains(out, "NVIDIA") {
		t.Fatalf("la auto-expansión debió llegar hasta NVIDIA:\n%s", out)
	}
	if strings.Contains(out, "Pequeña") {
		t.Errorf("no debe mostrar carpetas bajo el mínimo:\n%s", out)
	}
}

func TestTreeSinAutoExpand(t *testing.T) {
	o := DefaultTreeOptions
	o.AutoExpand = 0
	out := render(o)
	if strings.Contains(out, "Local") {
		t.Errorf("sin auto-expansión no debe pasar de la profundidad 3:\n%s", out)
	}
	if !strings.Contains(out, "AppData") {
		t.Errorf("debe mostrar hasta la profundidad 3:\n%s", out)
	}
}

func TestTreeSinLineaDeRestoEnHojas(t *testing.T) {
	out := render(DefaultTreeOptions)
	if strings.Contains(out, "archivos y carpetas menores") {
		t.Errorf("no hay resto >= 1 GB en este árbol; no debe aparecer la línea:\n%s", out)
	}
}

func TestWriteCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, sample()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "\uFEFFRuta;Bytes;GB") {
		t.Errorf("falta BOM o cabecera: %q", out[:min(40, len(out))])
	}
	if !strings.Contains(out, `"C:\Users\Marcos\AppData\Local\NVIDIA";107374182400;100,000;1;5;no`) {
		t.Errorf("fila de NVIDIA incorrecta:\n%s", out)
	}
	if strings.Contains(out, "Pequeña") {
		t.Errorf("no debe exportar carpetas < 10 MB:\n%s", out)
	}
}
