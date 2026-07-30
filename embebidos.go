package main

import "embed"

// go:embed mete el contenido de esas carpetas/archivos DENTRO del binario
// compilado, como si fueran bytes del propio programa. Esto es clave para
// distribuir la app: hoy, si movés cobrapp.exe a otra carpeta sin llevarte
// también templates/ y static/, el programa no arranca (busca esos
// archivos relativos a "donde se lo ejecuta" y no los encuentra).
//
// Con embed, el .exe (o el binario de Linux) es UN SOLO ARCHIVO
// autocontenido: se lo pasás a un cliente por USB o WhatsApp y funciona,
// sin carpetas sueltas que se puedan perder u olvidar.
//
// Importante: lo que SÍ sigue siendo un archivo aparte es data/cobrapp.db
// (los datos del cliente) y config.json/logs/ (cosas que tienen que poder
// editarse o inspeccionarse sin recompilar). Eso es intencional: se
// embebe el CÓDIGO/DISEÑO de la app (fijo), no los DATOS del usuario
// (variables).

//go:embed templates
var plantillasFS embed.FS

//go:embed static
var staticFS embed.FS

//go:embed data/schema.sql
var esquemaSQL string
