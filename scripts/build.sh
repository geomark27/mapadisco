#!/usr/bin/env bash
# Compila mapadisco.exe (Windows x64) y arma el paquete instalable en dist/.
# Uso: scripts/build.sh            (versión tomada del archivo VERSION)
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="$(tr -d '[:space:]' < VERSION)"
OUT="dist/mapadisco"

echo "→ Pruebas"
go test ./...

echo "→ Análisis estático (GOOS=windows)"
GOOS=windows go vet ./...

echo "→ Compilando mapadisco $VERSION"
rm -rf dist && mkdir -p "$OUT"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$OUT/mapadisco.exe" ./cmd/mapadisco

cp installer/* README.md "$OUT/"
cp docs/guia-de-usuario.md "$OUT/GUIA-DE-USUARIO.md"
sed -i 's/\r\?$/\r/' "$OUT"/*.cmd "$OUT"/*.ps1   # finales de línea CRLF para Windows
# En el paquete, el enlace del README a la guía apunta a la copia local.
sed -i 's#(docs/guia-de-usuario.md)#(GUIA-DE-USUARIO.md)#; s#\[docs/arquitectura.md\](docs/arquitectura.md)#docs/arquitectura.md (en el código fuente)#' "$OUT/README.md"

ZIP="mapadisco-$VERSION-windows-x64.zip"
(cd dist && python3 -c "import shutil,sys; shutil.make_archive(sys.argv[1][:-4],'zip','.','mapadisco')" "$ZIP")
echo "✓ dist/$ZIP"
