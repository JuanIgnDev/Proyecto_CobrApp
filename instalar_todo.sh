#!/bin/bash
# instalar_todo.sh — compila e instala CobrApp en Linux con un solo comando.
#
# Uso:
#   ./instalar_todo.sh
#
# Hace, en orden, exactamente lo mismo que hacías a mano:
#   1. Da permiso de ejecución a build.sh (si no lo tenía)
#   2. Corre build.sh -> genera dist/linux/cobrapp (y dist/windows/cobrapp.exe)
#   3. Da permiso de ejecución a installer/linux/instalar.sh (si no lo tenía)
#   4. Corre instalar.sh -> copia el binario + crea el ícono de menú
#
# "set -e" hace que el script se corte apenas un comando falla, en vez de
# seguir de largo y terminar instalando algo a medias. Si algo falla, el
# mensaje de error que tira ESE comando (go, cp, etc.) te dice qué pasó.
set -e

# Nos aseguramos de correr todo relativo a la carpeta donde está este
# script, sin importar desde dónde lo invoques (podrías estar parado en
# otra carpeta y ejecutarlo con una ruta completa).
cd "$(dirname "$0")"

echo "== 1/2: Compilando (build.sh) =="
chmod +x build.sh
./build.sh

echo ""
echo "== 2/2: Instalando (installer/linux/instalar.sh) =="
chmod +x installer/linux/instalar.sh
./installer/linux/instalar.sh

echo ""
echo "Listo. CobrApp instalado. Buscalo en el menú de aplicaciones,"
echo "o corré: ~/.local/share/CobrApp/cobrapp"