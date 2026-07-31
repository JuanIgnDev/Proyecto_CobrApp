#!/bin/bash
# build.sh — compila CobrApp para Windows y Linux, listo para distribuir.
#
# -ldflags="-s -w":
#   -s  quita la tabla de símbolos (los nombres de funciones/variables que
#       usa el debugger). El binario funciona igual, pero si algún día
#       necesitás debuggear con delve vas a perder esa info.
#   -w  quita la información de DWARF (para debuggers tipo gdb/delve).
#   Juntos, en un proyecto como este, suelen bajar el tamaño del binario
#   entre un 20% y un 30%. No afectan la performance en runtime, solo lo
#   que ya no está disponible para debuggear el binario final.
#
# CGO_ENABLED=0:
#   Como usás modernc.org/sqlite (que es sqlite reescrito en Go puro, sin
#   C de por medio), podés compilar con CGO desactivado. Esto es lo que
#   te permite CROSS-COMPILAR: generar el .exe de Windows estando parado
#   en Linux, sin instalar ningún compilador de C ni toolchain de Windows.
#   Si en algún momento usaran github.com/mattn/go-sqlite3 (que sí es CGO,
#   usa el sqlite en C) esto no funcionaría tan fácil: ahí GOOS=windows
#   necesitaría un compilador C cruzado (mingw-w64) instalado.

set -e

APP=cobrapp
VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS="-s -w -X main.version=$VERSION"

echo "Compilando $APP versión $VERSION..."

mkdir -p dist

echo "-> Windows (amd64)"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$LDFLAGS" -o dist/windows/${APP}.exe .

echo "-> Linux (amd64)"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$LDFLAGS" -o dist/linux/${APP} .

echo ""
echo "Listo. Binarios en:"
echo "  dist/windows/${APP}.exe"
echo "  dist/linux/${APP}"
echo ""
echo "Nota: como templates/, static/ y el schema van embebidos (embed.FS),"
echo "estos DOS archivos ya son autocontenidos. Lo único que necesitan al"
echo "lado (y se crea solo si no existe) es config.json, y las carpetas"
echo "data/ y logs/."
