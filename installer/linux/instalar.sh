#!/bin/bash
# instalar.sh — instalador simple para Linux.
#
# Importante: Inno Setup ES UNA HERRAMIENTA SOLO PARA WINDOWS (genera
# instaladores .exe, y corre nativamente solo en Windows). No existe un
# "Inno Setup para Linux". Las alternativas típicas en Linux son:
#   1. Un script shell simple como este (rápido, sin dependencias).
#   2. Empaquetar un .deb (para Debian/Ubuntu) con dpkg-deb.
#   3. Empaquetar un AppImage (un solo archivo ejecutable, sin instalar).
# Para un programa chico como CobrApp, un script como este alcanza y sobra.
#
# Si en algún momento quisieran generar el instalador .exe de Windows
# DESDE Linux (por ejemplo, en un pipeline de CI), se puede correr el
# compilador de Inno Setup (ISCC.exe) bajo Wine: es la única forma de usar
# Inno Setup fuera de Windows.

set -e

APP=cobrapp
ORIGEN="$(dirname "$0")/../../dist/linux"
DESTINO="$HOME/.local/share/CobrApp"

echo "Instalando CobrApp en $DESTINO..."

mkdir -p "$DESTINO"
cp "$ORIGEN/$APP" "$DESTINO/$APP"
chmod +x "$DESTINO/$APP"

# Accesorio de escritorio (opcional): así aparece en el menú de
# aplicaciones como cualquier otro programa instalado.
mkdir -p "$HOME/.local/share/applications"
cat > "$HOME/.local/share/applications/cobrapp.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=CobrApp
Exec=$DESTINO/$APP
Terminal=false
Categories=Office;
EOF

echo "Listo. Podés abrir CobrApp desde el menú de aplicaciones,"
echo "o ejecutando: $DESTINO/$APP"
