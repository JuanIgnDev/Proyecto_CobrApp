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

set -e

APP=cobrapp
ORIGEN="$(dirname "$0")/../../dist/linux"
ICONO_ORIGEN="$(dirname "$0")/../../static/img/icono.png"
DESTINO="$HOME/.local/share/CobrApp"

echo "Instalando CobrApp en $DESTINO..."

mkdir -p "$DESTINO"
cp "$ORIGEN/$APP" "$DESTINO/$APP"
chmod +x "$DESTINO/$APP"

# En vez de copiar el ícono a la carpeta "oficial" de temas de íconos
# (~/.local/share/icons/hicolor/...) y depender de que el sistema
# refresque su caché para encontrarlo por nombre, lo copiamos DENTRO de
# la propia carpeta de instalación y en el .desktop apuntamos con la
# RUTA ABSOLUTA del archivo. Esto funciona siempre, en cualquier
# escritorio (GNOME, KDE, XFCE...), sin caché de por medio y sin
# necesidad de cerrar sesión para verlo aparecer.
if [ -f "$ICONO_ORIGEN" ]; then
    cp "$ICONO_ORIGEN" "$DESTINO/icono.png"
else
    echo "Aviso: no encontré $ICONO_ORIGEN, sigo sin ícono personalizado."
fi

mkdir -p "$HOME/.local/share/applications"
cat > "$HOME/.local/share/applications/cobrapp.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=CobrApp
Exec=$DESTINO/$APP
Icon=$DESTINO/icono.png
Terminal=false
Categories=Office;
EOF

# Refrescamos la caché de aplicaciones (esto sí es liviano y rápido,
# no hace falta esperar a un logout) para que CobrApp aparezca al
# buscarlo en el menú apenas termine el script.
command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$HOME/.local/share/applications/" || true

echo "Listo. Podés abrir CobrApp desde el menú de aplicaciones,"
echo "o ejecutando: $DESTINO/$APP"