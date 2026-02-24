package services

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// AddressExportColumn defines a column in the address export spreadsheet.
type AddressExportColumn struct {
	Header string
	Field  string  // field name on the PocketBase record
	Width  float64 // column width in Excel units
}

// AddressExportData holds all data needed for address export.
type AddressExportData struct {
	ProjectName string
	AddressType string            // "bill_from", "ship_from", "bill_to", "ship_to", "install_at"
	TypeLabel   string            // "Bill From", "Ship From", etc.
	Columns     []AddressExportColumn
	Rows        []map[string]string // each row is field -> value
}

// GetAddressColumns returns the export columns for a given address type.
func GetAddressColumns(addressType string) []AddressExportColumn {
	common := []AddressExportColumn{
		{Header: "Company Name", Field: "company_name", Width: 30},
		{Header: "Contact Person", Field: "contact_person", Width: 25},
		{Header: "Email", Field: "email", Width: 30},
		{Header: "Phone", Field: "phone", Width: 18},
		{Header: "Address Line 1", Field: "address_line_1", Width: 35},
		{Header: "Address Line 2", Field: "address_line_2", Width: 35},
		{Header: "City", Field: "city", Width: 20},
		{Header: "State", Field: "state", Width: 20},
		{Header: "PIN Code", Field: "pin_code", Width: 12},
		{Header: "Country", Field: "country", Width: 18},
		{Header: "GSTIN", Field: "gstin", Width: 20},
	}

	if addressType == "install_at" {
		parentCol := AddressExportColumn{
			Header: "Ship To Parent (Company Name)",
			Field:  "_ship_to_parent_name",
			Width:  35,
		}
		return append([]AddressExportColumn{parentCol}, common...)
	}

	return common
}

// GenerateAddressExcel creates an Excel file from address export data.
func GenerateAddressExcel(data AddressExportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := data.TypeLabel + " Addresses"
	if len(sheetName) > 31 {
		sheetName = sheetName[:31]
	}

	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheetName); err != nil {
		return nil, fmt.Errorf("set sheet name: %w", err)
	}

	// --- Styles (matching existing BOQ export patterns) ---

	titleStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16},
	})
	if err != nil {
		return nil, fmt.Errorf("create title style: %w", err)
	}

	subtitleStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 11},
	})
	if err != nil {
		return nil, fmt.Errorf("create subtitle style: %w", err)
	}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#333333"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: thinBorders(),
	})
	if err != nil {
		return nil, fmt.Errorf("create header style: %w", err)
	}

	dataStyle, err := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Size: 10},
		Border: thinBorders(),
		Alignment: &excelize.Alignment{
			Vertical: "center",
			WrapText: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create data style: %w", err)
	}

	// --- Column widths ---
	for i, col := range data.Columns {
		colLetter := addrColName(i)
		f.SetColWidth(sheetName, colLetter, colLetter, col.Width)
	}

	// --- Row 1: Title ---
	lastCol := addrColName(len(data.Columns) - 1)
	f.MergeCell(sheetName, "A1", lastCol+"1")
	f.SetCellValue(sheetName, "A1", sanitizeExcelCell(data.ProjectName)+" - "+data.TypeLabel+" Addresses")
	f.SetCellStyle(sheetName, "A1", lastCol+"1", titleStyle)

	// --- Row 2: Subtitle with count ---
	f.MergeCell(sheetName, "A2", lastCol+"2")
	f.SetCellValue(sheetName, "A2", fmt.Sprintf("Total: %d addresses", len(data.Rows)))
	f.SetCellStyle(sheetName, "A2", lastCol+"2", subtitleStyle)

	// --- Row 4: Column headers ---
	for i, col := range data.Columns {
		cell := fmt.Sprintf("%s4", addrColName(i))
		f.SetCellValue(sheetName, cell, col.Header)
	}
	f.SetCellStyle(sheetName, "A4", lastCol+"4", headerStyle)

	// --- Freeze header row (freeze pane below row 4) ---
	f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      4,
		TopLeftCell: "A5",
		ActivePane:  "bottomLeft",
	})

	// --- Data rows starting at row 5 ---
	for rowIdx, rowData := range data.Rows {
		rowNum := rowIdx + 5
		rowStr := fmt.Sprintf("%d", rowNum)
		for colIdx, col := range data.Columns {
			cell := fmt.Sprintf("%s%s", addrColName(colIdx), rowStr)
			f.SetCellValue(sheetName, cell, sanitizeExcelCell(rowData[col.Field]))
		}
		f.SetCellStyle(sheetName, "A"+rowStr, lastCol+rowStr, dataStyle)
	}

	// --- Write to buffer ---
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write excel: %w", err)
	}

	return buf.Bytes(), nil
}

// addrColName converts a 0-based column index to an Excel column letter (A, B, ..., Z, AA, ...).
func addrColName(index int) string {
	name := ""
	for index >= 0 {
		name = string(rune('A'+index%26)) + name
		index = index/26 - 1
	}
	return name
}
