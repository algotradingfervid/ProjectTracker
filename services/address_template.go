package services

import (
	"bytes"
	"fmt"

	"github.com/pocketbase/pocketbase"
	"github.com/xuri/excelize/v2"
)

// GenerateAddressTemplate creates a downloadable .xlsx template for the given
// project and address type ("ship_to" or "install_at").
func GenerateAddressTemplate(app *pocketbase.PocketBase, projectID, addressType string) ([]byte, error) {
	// 1. Determine fields for the address type
	var fields []TemplateField
	if addressType == "install_at" {
		fields = InstallAtTemplateFields()
	} else {
		fields = ShipToTemplateFields()
	}

	// 2. Fetch project-specific required field overrides
	requiredSet := GetRequiredFields(app, projectID, addressType)

	// 3. Build the workbook
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Addresses"
	defaultSheet := f.GetSheetName(0)
	f.SetSheetName(defaultSheet, sheetName)

	// --- Styles ---
	requiredHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1D4ED8"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    thinBorders(),
	})

	optionalHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#6B7280"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    thinBorders(),
	})

	// 4. Write header row and set column widths
	columns := columnLetters(len(fields))
	for i, field := range fields {
		cell := fmt.Sprintf("%s1", columns[i])

		headerText := field.Label
		isRequired := field.AlwaysRequired || requiredSet[field.Key]
		if isRequired {
			headerText += " *"
		}
		f.SetCellValue(sheetName, cell, headerText)

		if isRequired {
			f.SetCellStyle(sheetName, cell, cell, requiredHeaderStyle)
		} else {
			f.SetCellStyle(sheetName, cell, cell, optionalHeaderStyle)
		}

		width := float64(len(field.Label)) * 1.3
		if width < 15 {
			width = 15
		}
		f.SetColWidth(sheetName, columns[i], columns[i], width)
	}

	// 5. Add data validation dropdowns for State and Country
	for i, field := range fields {
		col := columns[i]
		rangeRef := fmt.Sprintf("%s2:%s1048576", col, col)

		switch field.Key {
		case "state":
			dv := excelize.NewDataValidation(true)
			dv.Sqref = rangeRef
			dv.SetDropList(IndianStates)
			f.AddDataValidation(sheetName, dv)
		case "country":
			dv := excelize.NewDataValidation(true)
			dv.Sqref = rangeRef
			dv.SetDropList(Countries)
			f.AddDataValidation(sheetName, dv)
		}
	}

	// 6. Freeze header row
	f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// 7. Create hidden Instructions sheet
	addInstructionsSheet(f, fields, requiredSet, addressType)

	// 8. Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write excel template: %w", err)
	}
	return buf.Bytes(), nil
}

// addInstructionsSheet creates a hidden sheet with field descriptions.
func addInstructionsSheet(f *excelize.File, fields []TemplateField, requiredSet map[string]bool, addressType string) {
	instSheet := "Instructions"
	f.NewSheet(instSheet)

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14},
	})
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E5E7EB"}, Pattern: 1},
	})

	typeName := "Ship To"
	if addressType == "install_at" {
		typeName = "Install At"
	}
	f.SetCellValue(instSheet, "A1", fmt.Sprintf("%s Address Import - Instructions", typeName))
	f.SetCellStyle(instSheet, "A1", "A1", titleStyle)

	instructionHeaders := []string{"Field Name", "Required?", "Format Rule", "Description", "Example"}
	cols := columnLetters(5)
	for i, h := range instructionHeaders {
		cell := fmt.Sprintf("%s3", cols[i])
		f.SetCellValue(instSheet, cell, h)
		f.SetCellStyle(instSheet, cell, cell, headerStyle)
	}

	for i, field := range fields {
		row := fmt.Sprintf("%d", i+4)
		reqLabel := "Optional"
		if field.AlwaysRequired || requiredSet[field.Key] {
			reqLabel = "Required"
		}
		f.SetCellValue(instSheet, cols[0]+row, field.Label)
		f.SetCellValue(instSheet, cols[1]+row, reqLabel)
		f.SetCellValue(instSheet, cols[2]+row, field.FormatRule)
		f.SetCellValue(instSheet, cols[3]+row, field.Description)
		f.SetCellValue(instSheet, cols[4]+row, field.ExampleValue)
	}

	widths := []float64{20, 12, 30, 45, 25}
	for i, w := range widths {
		f.SetColWidth(instSheet, cols[i], cols[i], w)
	}

	f.SetSheetVisible(instSheet, false)
}

// columnLetters returns Excel column letters for n columns: A, B, ... Z, AA, AB ...
func columnLetters(n int) []string {
	cols := make([]string, n)
	for i := 0; i < n; i++ {
		name, _ := excelize.ColumnNumberToName(i + 1)
		cols[i] = name
	}
	return cols
}
