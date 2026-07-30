; cobrapp.iss — script de Inno Setup para generar el instalador de Windows.
;
; Cómo usarlo:
;   1. Instalá Inno Setup (gratis): https://jrsoftware.org/isinfo.php
;   2. Compilá antes el binario de Windows: ./build.sh (genera dist/windows/cobrapp.exe)
;   3. Abrí este archivo con el "Inno Setup Compiler" y apretá "Compile"
;      (o por línea de comandos: ISCC.exe cobrapp.iss)
;   4. El resultado queda en installer/windows/Output/CobrApp-Setup.exe,
;      un único instalador que le podés pasar a un cliente.
;
; Qué hace este script, en criollo: copia cobrapp.exe a
; "Archivos de programa\CobrApp", crea un acceso directo en el Menú Inicio
; y (opcionalmente) en el Escritorio, y registra un desinstalador prolijo
; en "Agregar o quitar programas" de Windows.

#define MyAppName "CobrApp"
#define MyAppVersion "1.0"
#define MyAppPublisher "CobrApp"
#define MyAppExeName "cobrapp.exe"

[Setup]
AppId={{B3D1F5A0-6E1B-4C2C-9B0B-COBRAPP00001}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
; Instala en Archivos de Programa, pero cada usuario puede requerir admin
; (DefaultDirName usa {autopf}, que resuelve a Program Files correctamente
; tanto en Windows de 32 como de 64 bits).
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
; No pedimos que el usuario elija dónde instalar; se puede habilitar
; DisableDirPage=no si quieren dejarlo elegir.
DisableDirPage=no
OutputDir=Output
OutputBaseFilename=CobrApp-Setup
Compression=lzma2
SolidCompression=yes
; Como la app guarda datos (data/, logs/, config.json) AL LADO del
; ejecutable, es buena idea NO instalar en Archivos de Programa si van a
; correrla sin permisos de administrador (esa carpeta suele ser de solo
; lectura para usuarios normales). Alternativa más simple: instalar en
; {localappdata}\CobrApp en vez de {autopf}\CobrApp. Descomentá la línea
; de abajo y comentá la de arriba si prefieren esto:
; DefaultDirName={localappdata}\{#MyAppName}
PrivilegesRequired=lowest

[Languages]
Name: "spanish"; MessagesFile: "compiler:Languages\Spanish.isl"

[Tasks]
Name: "desktopicon"; Description: "Crear un acceso directo en el Escritorio"; GroupDescription: "Accesos directos:"

[Files]
; La barra "*" trae TODO lo que haya en dist/windows — como templates y
; static ya están embebidos en el .exe (ver embebidos.go), en la práctica
; esto es un solo archivo: cobrapp.exe.
Source: "..\..\dist\windows\cobrapp.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\Desinstalar {#MyAppName}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
; Ofrece abrir la app apenas termina de instalar.
Filename: "{app}\{#MyAppExeName}"; Description: "Abrir {#MyAppName}"; Flags: nowait postinstall skipifsilent
