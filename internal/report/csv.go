package report

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"mapadisco/internal/scanner"
)

// CSVMinSize es el tamaño mínimo de carpeta que se exporta (la raíz siempre se incluye).
const CSVMinSize = 10 << 20

// WriteCSV exporta las carpetas >= CSVMinSize en formato compatible con Excel en español:
// separador ";", coma decimal y BOM UTF-8 para que se lean bien las tildes.
func WriteCSV(w io.Writer, root *scanner.Node) error {
	bw := bufio.NewWriterSize(w, 1<<20)
	bw.WriteString("\uFEFFRuta;Bytes;GB;Archivos;Nivel;SinAcceso\n")
	var walk func(n *scanner.Node, path string, level int)
	walk = func(n *scanner.Node, path string, level int) {
		if level > 0 && n.Total < CSVMinSize {
			return
		}
		gb := strings.Replace(strconv.FormatFloat(float64(n.Total)/(1<<30), 'f', 3, 64), ".", ",", 1)
		fmt.Fprintf(bw, "\"%s\";%d;%s;%d;%d;%s\n", strings.ReplaceAll(path, `"`, `""`),
			n.Total, gb, n.TotalFiles, level, map[bool]string{true: "sí", false: "no"}[n.Denied])
		for _, c := range n.Children {
			walk(c, scanner.JoinPath(path, c.Name), level+1)
		}
	}
	walk(root, root.Name, 0)
	return bw.Flush()
}
