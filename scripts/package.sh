#!/usr/bin/env bash
# Compila mapadisco.exe y arma un zip instalable por plataforma en dist/:
# el ejecutable, el instalador, el README y la guía de usuario.
#   scripts/package.sh <versión> [os/arch ...]     (por defecto windows/amd64)
# Lo usan `make package` (versión local) y `make release-build` (versión del release).
set -euo pipefail
cd "$(dirname "$0")/.."

ver=${1:?indica la versión: scripts/package.sh v1.2.0}
shift
platforms=("$@")
[ ${#platforms[@]} -gt 0 ] || platforms=(windows/amd64)

command -v zip >/dev/null || { echo "✗ falta 'zip': sudo apt install zip" >&2; exit 1; }

rm -rf dist && mkdir -p dist
for p in "${platforms[@]}"; do
	os=${p%/*}
	arch=${p#*/}
	[ "$os" = windows ] || { echo "✗ mapadisco solo funciona en Windows; plataforma no válida: $p" >&2; exit 1; }

	name="mapadisco-$ver-$os-$arch"
	dir="dist/$name"
	mkdir -p "$dir"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch \
		go build -trimpath -ldflags "-s -w -X main.version=$ver" -o "$dir/mapadisco.exe" ./cmd/mapadisco

	cp installer/* README.md "$dir/"
	cp docs/guia-de-usuario.md "$dir/GUIA-DE-USUARIO.md"
	sed -i 's/\r\?$/\r/' "$dir"/*.cmd "$dir"/*.ps1 # CRLF para Windows
	# Dentro del zip, los enlaces del README apuntan a la copia local de la guía.
	sed -i 's#(docs/guia-de-usuario.md)#(GUIA-DE-USUARIO.md)#g; s#\[\([^]]*\)\](docs/[^)]*)#\1 (en el repositorio)#g' "$dir/README.md"

	(cd dist && zip -qr "$name.zip" "$name")
	echo "  dist/$name.zip"
done
