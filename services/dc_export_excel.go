package services

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// GenerateDCExcel creates an Excel file for a Delivery Challan.
func GenerateDCExcel(data *DCExportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "DC Details"
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheetName); err != nil {
		return nil, fmt.Errorf("set sheet name: %w", err)
	}

	// Styles
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16},
	})
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#212529"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#212529", Style: 1},
		},
	})
	labelStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 9, Color: "#666666"},
	})
	valueStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10},
	})
	currencyStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		NumFmt:    4, // #,##0.00
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	totalLabelStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	totalValueStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		NumFmt:    4,
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})

	// Column widths
	colWidths := map[string]float64{
		"A": 6, "B": 35, "C": 12, "D": 10, "E": 10,
		"F": 14, "G": 14, "H": 10, "I": 14, "J": 14,
	}
	for col, w := range colWidths {
		_ = f.SetColWidth(sheetName, col, col, w)
	}

	row := 1

	// Title
	f.SetCellValue(sheetName, cell("A", row), "DELIVERY CHALLAN")
	f.SetCellStyle(sheetName, cell("A", row), cell("A", row), titleStyle)
	row += 2

	// DC metadata
	dcMeta := []struct{ label, value string }{
		{"DC Number", data.DCNumber},
		{"DC Type", data.DCType},
		{"Status", data.Status},
		{"Challan Date", data.ChallanDate},
		{"Company", data.CompanyName},
	}
	for _, m := range dcMeta {
		f.SetCellValue(sheetName, cell("A", row), m.label)
		f.SetCellStyle(sheetName, cell("A", row), cell("A", row), labelStyle)
		f.SetCellValue(sheetName, cell("B", row), m.value)
		f.SetCellStyle(sheetName, cell("B", row), cell("B", row), valueStyle)
		row++
	}
	row++

	// Addresses
	writeExcelAddress := func(label string, addr *DCExportAddress) {
		f.SetCellValue(sheetName, cell("A", row), label)
		f.SetCellStyle(sheetName, cell("A", row), cell("A", row), labelStyle)
		row++
		if addr != nil {
			if addr.CompanyName != "" {
				f.SetCellValue(sheetName, cell("A", row), addr.CompanyName)
				f.SetCellStyle(sheetName, cell("A", row), cell("A", row), valueStyle)
				row++
			}
			if addr.AddressLines != "" {
				f.SetCellValue(sheetName, cell("A", row), addr.AddressLines)
				f.SetCellStyle(sheetName, cell("A", row), cell("A", row), valueStyle)
				row++
			}
			if addr.GSTIN != "" {
				f.SetCellValue(sheetName, cell("A", row), fmt.Sprintf("GSTIN: %s", addr.GSTIN))
				f.SetCellStyle(sheetName, cell("A", row), cell("A", row), valueStyle)
				row++
			}
			if addr.ContactPerson != "" || addr.Phone != "" {
				f.SetCellValue(sheetName, cell("A", row), fmt.Sprintf("%s %s", addr.ContactPerson, addr.Phone))
				f.SetCellStyle(sheetName, cell("A", row), cell("A", row), valueStyle)
				row++
			}
		} else {
			f.SetCellValue(sheetName, cell("A", row), "Not specified")
			row++
		}
		row++
	}

	writeExcelAddress("BILL FROM", data.BillFrom)
	writeExcelAddress("DISPATCH FROM", data.DispatchFrom)
	writeExcelAddress("BILL TO", data.BillTo)
	writeExcelAddress("SHIP TO", data.ShipTo)

	// Transport
	if data.Transport != nil {
		f.SetCellValue(sheetName, cell("A", row), "TRANSPORT DETAILS")
		f.SetCellStyle(sheetName, cell("A", row), cell("A", row), labelStyle)
		row++
		transportFields := []struct{ label, value string }{
			{"Transporter", data.Transport.TransporterName},
			{"Vehicle", data.Transport.VehicleNumber},
			{"E-Way Bill", data.Transport.EwayBillNumber},
			{"Docket No", data.Transport.DocketNumber},
		}
		for _, tf := range transportFields {
			if tf.value != "" {
				f.SetCellValue(sheetName, cell("A", row), tf.label)
				f.SetCellStyle(sheetName, cell("A", row), cell("A", row), labelStyle)
				f.SetCellValue(sheetName, cell("B", row), tf.value)
				f.SetCellStyle(sheetName, cell("B", row), cell("B", row), valueStyle)
				row++
			}
		}
		row++
	}

	// Line Items header
	f.SetCellValue(sheetName, cell("A", row), "LINE ITEMS")
	f.SetCellStyle(sheetName, cell("A", row), cell("A", row), labelStyle)
	row++

	if data.DCType == "official" {
		// Official DC: simplified columns
		headers := []string{"SI No", "Description", "HSN Code", "UoM", "Qty"}
		cols := []string{"A", "B", "C", "D", "E"}
		for i, h := range headers {
			f.SetCellValue(sheetName, cell(cols[i], row), h)
			f.SetCellStyle(sheetName, cell(cols[i], row), cell(cols[i], row), headerStyle)
		}
		row++

		for _, item := range data.LineItems {
			f.SetCellValue(sheetName, cell("A", row), item.SINo)
			f.SetCellValue(sheetName, cell("B", row), item.Description)
			f.SetCellValue(sheetName, cell("C", row), item.HSNCode)
			f.SetCellValue(sheetName, cell("D", row), item.UOM)
			f.SetCellValue(sheetName, cell("E", row), item.Qty)
			row++
		}
	} else {
		// Transit/Transfer: full columns
		headers := []string{"SI No", "Description", "HSN Code", "Qty", "UoM", "Rate", "Taxable", "Tax%", "Tax Amt", "Total"}
		cols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
		for i, h := range headers {
			f.SetCellValue(sheetName, cell(cols[i], row), h)
			f.SetCellStyle(sheetName, cell(cols[i], row), cell(cols[i], row), headerStyle)
		}
		row++

		for _, item := range data.LineItems {
			f.SetCellValue(sheetName, cell("A", row), item.SINo)
			f.SetCellValue(sheetName, cell("B", row), item.Description)
			f.SetCellValue(sheetName, cell("C", row), item.HSNCode)
			f.SetCellValue(sheetName, cell("D", row), item.Qty)
			f.SetCellValue(sheetName, cell("E", row), item.UOM)
			f.SetCellValue(sheetName, cell("F", row), item.Rate)
			f.SetCellStyle(sheetName, cell("F", row), cell("F", row), currencyStyle)
			f.SetCellValue(sheetName, cell("G", row), item.Taxable)
			f.SetCellStyle(sheetName, cell("G", row), cell("G", row), currencyStyle)
			f.SetCellValue(sheetName, cell("H", row), fmt.Sprintf("%.1f%%", item.TaxPercent))
			f.SetCellValue(sheetName, cell("I", row), item.TaxAmount)
			f.SetCellStyle(sheetName, cell("I", row), cell("I", row), currencyStyle)
			f.SetCellValue(sheetName, cell("J", row), item.Total)
			f.SetCellStyle(sheetName, cell("J", row), cell("J", row), currencyStyle)
			row++
		}

		row++
		// Totals
		totals := []struct{ label string; value float64 }{
			{"Taxable Amount", data.TotalTaxable},
			{"Tax Amount", data.TotalTax},
			{"Grand Total", data.GrandTotal},
		}
		for _, t := range totals {
			f.SetCellValue(sheetName, cell("I", row), t.label)
			f.SetCellStyle(sheetName, cell("I", row), cell("I", row), totalLabelStyle)
			f.SetCellValue(sheetName, cell("J", row), t.value)
			f.SetCellStyle(sheetName, cell("J", row), cell("J", row), totalValueStyle)
			row++
		}
	}

	// Serial Numbers sheet (if any items have serials)
	hasSerials := false
	for _, item := range data.LineItems {
		if len(item.Serials) > 0 {
			hasSerials = true
			break
		}
	}

	if hasSerials {
		serialSheet := "Serial Numbers"
		f.NewSheet(serialSheet)
		_ = f.SetColWidth(serialSheet, "A", "A", 35)
		_ = f.SetColWidth(serialSheet, "B", "B", 25)

		sRow := 1
		f.SetCellValue(serialSheet, cell("A", sRow), "Item")
		f.SetCellStyle(serialSheet, cell("A", sRow), cell("A", sRow), headerStyle)
		f.SetCellValue(serialSheet, cell("B", sRow), "Serial Number")
		f.SetCellStyle(serialSheet, cell("B", sRow), cell("B", sRow), headerStyle)
		sRow++

		for _, item := range data.LineItems {
			for _, serial := range item.Serials {
				f.SetCellValue(serialSheet, cell("A", sRow), item.Description)
				f.SetCellValue(serialSheet, cell("B", sRow), serial)
				sRow++
			}
		}
	}

	// Transfer DC destinations sheet
	if data.DCType == "transfer" && len(data.Destinations) > 0 {
		destSheet := "Destinations"
		f.NewSheet(destSheet)
		_ = f.SetColWidth(destSheet, "A", "A", 6)
		_ = f.SetColWidth(destSheet, "B", "B", 50)

		dRow := 1
		f.SetCellValue(destSheet, cell("A", dRow), "#")
		f.SetCellStyle(destSheet, cell("A", dRow), cell("A", dRow), headerStyle)
		f.SetCellValue(destSheet, cell("B", dRow), "Destination")
		f.SetCellStyle(destSheet, cell("B", dRow), cell("B", dRow), headerStyle)
		dRow++

		if data.HubAddress != "" {
			f.SetCellValue(destSheet, cell("A", dRow), "Hub")
			f.SetCellStyle(destSheet, cell("A", dRow), cell("A", dRow), labelStyle)
			f.SetCellValue(destSheet, cell("B", dRow), data.HubAddress)
			dRow++
		}

		for i, dest := range data.Destinations {
			f.SetCellValue(destSheet, cell("A", dRow), i+1)
			f.SetCellValue(destSheet, cell("B", dRow), dest)
			dRow++
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write excel: %w", err)
	}

	return buf.Bytes(), nil
}

// cell builds an Excel cell reference like "A1".
func cell(col string, row int) string {
	return fmt.Sprintf("%s%d", col, row)
}
