// Package scanner recorre un árbol de carpetas en paralelo y construye un modelo
// en memoria con tamaños agregados, ranking de archivos grandes y peso por extensión.
//
// El modelo (Node, Result) es portable; el motor de escaneo solo compila en Windows.
package scanner

import (
	"strings"
	"time"
)

// Node representa una carpeta. Solo las carpetas se guardan en memoria; los archivos
// se suman a su carpeta y, si son grandes, entran al ranking de Result.TopFiles.
type Node struct {
	Name       string // nombre de la carpeta; en la raíz, la ruta completa (ej. "C:\")
	Parent     *Node
	Children   []*Node
	Size       int64 // bytes de archivos directamente en esta carpeta
	Files      int64
	Total      int64 // bytes incluyendo subcarpetas (tras SumTotals)
	TotalFiles int64
	Denied     bool // no se pudo leer (permisos)
}

// Path reconstruye la ruta absoluta del nodo.
func (n *Node) Path() string {
	if n.Parent == nil {
		return n.Name
	}
	return JoinPath(n.Parent.Path(), n.Name)
}

// Child busca una subcarpeta por nombre sin distinguir mayúsculas (como NTFS).
func (n *Node) Child(name string) *Node {
	for _, c := range n.Children {
		if strings.EqualFold(c.Name, name) {
			return c
		}
	}
	return nil
}

// SumTotals calcula Total y TotalFiles de todo el subárbol.
func SumTotals(n *Node) {
	n.Total, n.TotalFiles = n.Size, n.Files
	for _, c := range n.Children {
		SumTotals(c)
		n.Total += c.Total
		n.TotalFiles += c.TotalFiles
	}
}

// Lookup localiza una carpeta del árbol por ruta absoluta de Windows.
// Devuelve nil si la ruta está fuera de la raíz o no fue escaneada.
func Lookup(root *Node, target string) *Node {
	base := strings.TrimSuffix(root.Name, `\`)
	if len(target) < len(base) || !strings.EqualFold(target[:len(base)], base) {
		return nil
	}
	rel := strings.Trim(target[len(base):], `\`)
	if rel == "" {
		return root
	}
	if len(target) > len(base) && target[len(base)] != '\\' {
		return nil // "C:\Users2" no está dentro de "C:\Users"
	}
	cur := root
	for seg := range strings.SplitSeq(rel, `\`) {
		if cur = cur.Child(seg); cur == nil {
			return nil
		}
	}
	return cur
}

// JoinPath une rutas de Windows sin duplicar la barra final.
func JoinPath(dir, name string) string {
	if strings.HasSuffix(dir, `\`) {
		return dir + name
	}
	return dir + `\` + name
}

// BigFile es una entrada del ranking de archivos más grandes.
type BigFile struct {
	Size int64
	Dir  *Node
	Name string
}

// Path devuelve la ruta completa del archivo.
func (f BigFile) Path() string { return JoinPath(f.Dir.Path(), f.Name) }

// ExtStat acumula bytes y cantidad de archivos de una extensión.
type ExtStat struct {
	Ext   string
	Bytes int64
	Count int64
}

// ScanError registra una carpeta que no se pudo leer.
type ScanError struct {
	Path string
	Err  error
}

// Result es el resultado completo de un escaneo.
type Result struct {
	Root       *Node
	Files      int64
	Dirs       int64
	Links      int64 // junctions/enlaces omitidos
	Denied     int64 // carpetas sin acceso
	CloudBytes int64 // bytes en archivos "solo en la nube" (no ocupan disco)
	CloudFiles int64
	TopFiles   []BigFile   // ordenados de mayor a menor
	Exts       []ExtStat   // ordenados de mayor a menor
	Errors     []ScanError // limitado a los primeros MaxErrors
	Elapsed    time.Duration
}
