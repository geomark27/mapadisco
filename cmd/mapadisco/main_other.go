//go:build !windows

// Comando mapadisco: en sistemas que no son Windows solo informa que no está soportado,
// para que `go build ./...` y `go test ./...` funcionen en Linux/WSL.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "mapadisco por ahora solo funciona en Windows. Compila con: GOOS=windows go build ./cmd/mapadisco")
	os.Exit(1)
}
