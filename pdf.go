package main

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/phpdave11/gofpdf"
)

func generarPDF(cliente *Cliente, ventas []Venta, cobros []Cobro) ([]byte, error) {

	pdf := gofpdf.New("P", "mm", "A4", "")

	pdf.SetTitle("Reporte de Cliente", false)

	pdf.AddPage()


	// ===== TITULO =====

	pdf.SetFont("Arial", "B", 20)
	pdf.CellFormat(
		0,
		12,
		"Reporte de Cliente",
		"",
		1,
		"C",
		false,
		0,
		"",
	)

	pdf.Ln(8)


	// ===== INFORMACION CLIENTE =====

	pdf.SetFont("Arial", "B", 13)
	pdf.Cell(0, 10, "Datos del cliente")
	pdf.Ln(8)


	pdf.SetFont("Arial", "", 11)

	pdf.Cell(40, 8, "Nombre:")
	pdf.Cell(0, 8, cliente.Nombre+" "+cliente.Apellido)
	pdf.Ln(7)

	pdf.Cell(40, 8, "Telefono:")
	pdf.Cell(0, 8, cliente.Telefono)
	pdf.Ln(7)

	pdf.Cell(40, 8, "Email:")
	pdf.Cell(0, 8, cliente.Email)
	pdf.Ln(12)



	// ===== SALDO =====

	pdf.SetFont("Arial", "B", 13)

	pdf.Cell(40, 10, "Saldo actual:")

	pdf.SetFont("Arial", "B", 13)

	saldo := strconv.FormatFloat(cliente.Saldo, 'f', 2, 64)

	pdf.CellFormat(
		0,
		10,
		"$ "+saldo,
		"",
		1,
		"",
		false,
		0,
		"",
	)

	pdf.Ln(10)



	// ===== TABLA VENTAS =====

	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10, "Historial de ventas")
	pdf.Ln(8)


	// encabezado tabla

	pdf.SetFillColor(230,230,230)

	pdf.SetFont("Arial", "B", 10)

	pdf.CellFormat(35,8,"Fecha","1",0,"C",true,0,"")
	pdf.CellFormat(75,8,"Descripcion","1",0,"C",true,0,"")
	pdf.CellFormat(40,8,"Monto","1",1,"C",true,0,"")


	pdf.SetFont("Arial", "", 10)


	for _, venta := range ventas {

		pdf.CellFormat(
			35,
			8,
			venta.Fecha,
			"1",
			0,
			"C",
			false,
			0,
			"",
		)

		pdf.CellFormat(
			75,
			8,
			venta.Descripcion,
			"1",
			0,
			"L",
			false,
			0,
			"",
		)

		pdf.CellFormat(
			40,
			8,
			"$ "+strconv.FormatFloat(venta.Total,'f',2,64),
			"1",
			1,
			"R",
			false,
			0,
			"",
		)
	}



	pdf.Ln(12)



	// ===== TABLA COBROS =====

	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0,10,"Historial de cobros")
	pdf.Ln(8)


	pdf.SetFont("Arial","B",10)

	pdf.CellFormat(35,8,"Fecha","1",0,"C",true,0,"")
	pdf.CellFormat(75,8,"Observacion","1",0,"C",true,0,"")
	pdf.CellFormat(40,8,"Monto","1",1,"C",true,0,"")


	pdf.SetFont("Arial","",10)


	for _, cobro := range cobros {


		pdf.CellFormat(
			35,
			8,
			cobro.Fecha,
			"1",
			0,
			"C",
			false,
			0,
			"",
		)


		pdf.CellFormat(
			75,
			8,
			cobro.Observacion,
			"1",
			0,
			"L",
			false,
			0,
			"",
		)


		pdf.CellFormat(
			40,
			8,
			"$ "+strconv.FormatFloat(cobro.Monto,'f',2,64),
			"1",
			1,
			"R",
			false,
			0,
			"",
		)
	}



	// ===== PIE =====

	pdf.Ln(15)

	pdf.SetFont("Arial","I",9)

	pdf.CellFormat(
		0,
		10,
		fmt.Sprintf(
			"Generado el %s",
			time.Now().Format("02/01/2006"),
		),
		"",
		0,
		"C",
		false,
		0,
		"",
	)



	var buffer bytes.Buffer

	err := pdf.Output(&buffer)

	if err != nil {
		return nil, err
	}


	return buffer.Bytes(), nil
}