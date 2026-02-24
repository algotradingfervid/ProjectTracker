package services

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// GenerateExcel creates an Excel file from the given ExportData and returns
// the file contents as a byte slice.
func GenerateExcel(data ExportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Determine sheet name (max 31 chars).
	sheetName := data.Title
	if len(sheetName) > 31 {
		sheetName = sheetName[:31]
	}
	if sheetName == "" {
		sheetName = "BOQ"
	}

	// Rename default sheet.
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheetName); err != nil {
		return nil, fmt.Errorf("set sheet name: %w", err)
	}

	// Column references (A through H).
	columns := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	lastCol := columns[len(columns)-1] // "H"

	// Set column widths.
	widths := []float64{6, 40, 10, 10, 18, 18, 14, 8}
	for i, col := range columns {
		if err := f.SetColWidth(sheetName, col, col, widths[i]); err != nil {
			return nil, fmt.Errorf("set col width %s: %w", col, err)
		}
	}

	// ── Styles ──────────────────────────────────────────────────────────

	// Title style: bold, 16pt.
	titleStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 16,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create title style: %w", err)
	}

	// Subtitle style (reference, date).
	subtitleStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size: 11,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create subtitle style: %w", err)
	}

	// Column header style: bold, white text, charcoal background, centered.
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "#FFFFFF",
			Size:  11,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#333333"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: thinBorders(),
	})
	if err != nil {
		return nil, fmt.Errorf("create header style: %w", err)
	}

	// Main item style (level 0): bold with borders.
	mainItemStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 10,
		},
		Border: thinBorders(),
	})
	if err != nil {
		return nil, fmt.Errorf("create main item style: %w", err)
	}

	// Sub/sub-sub item style (level 1, 2): normal with borders.
	subItemStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size: 10,
		},
		Border: thinBorders(),
	})
	if err != nil {
		return nil, fmt.Errorf("create sub item style: %w", err)
	}

	// Summary label style: bold, right-aligned.
	summaryLabelStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 11,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create summary label style: %w", err)
	}

	// Summary value style: bold.
	summaryValueStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 11,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create summary value style: %w", err)
	}

	// ── Header Rows (1-3) ───────────────────────────────────────────────

	// Row 1: Title merged across all columns.
	if err := f.MergeCell(sheetName, "A1", lastCol+"1"); err != nil {
		return nil, fmt.Errorf("merge title: %w", err)
	}
	f.SetCellValue(sheetName, "A1", sanitizeExcelCell(data.Title))
	f.SetCellStyle(sheetName, "A1", lastCol+"1", titleStyle)

	// Row 2: Reference number (if present).
	if data.ReferenceNumber != "" {
		if err := f.MergeCell(sheetName, "A2", lastCol+"2"); err != nil {
			return nil, fmt.Errorf("merge ref: %w", err)
		}
		f.SetCellValue(sheetName, "A2", "Ref: "+data.ReferenceNumber)
		f.SetCellStyle(sheetName, "A2", lastCol+"2", subtitleStyle)
	}

	// Row 3: Date.
	if err := f.MergeCell(sheetName, "A3", lastCol+"3"); err != nil {
		return nil, fmt.Errorf("merge date: %w", err)
	}
	f.SetCellValue(sheetName, "A3", "Date: "+data.CreatedDate)
	f.SetCellStyle(sheetName, "A3", lastCol+"3", subtitleStyle)

	// ── Row 5: Column Headers ───────────────────────────────────────────

	headers := []string{"#", "Description", "Qty", "UOM", "Quoted Price", "Budgeted Price", "HSN", "GST%"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s5", columns[i])
		f.SetCellValue(sheetName, cell, h)
	}
	f.SetCellStyle(sheetName, "A5", lastCol+"5", headerStyle)

	// ── Data Rows (starting row 6) ──────────────────────────────────────

	row := 6
	for _, r := range data.Rows {
		rowStr := fmt.Sprintf("%d", row)

		// Index column.
		f.SetCellValue(sheetName, "A"+rowStr, r.Index)

		// Description with indentation based on level.
		desc := r.Description
		switch r.Level {
		case 1:
			desc = "  " + desc
		case 2:
			desc = "    " + desc
		}
		f.SetCellValue(sheetName, "B"+rowStr, sanitizeExcelCell(desc))

		// Qty.
		f.SetCellValue(sheetName, "C"+rowStr, r.Qty)

		// UOM.
		f.SetCellValue(sheetName, "D"+rowStr, sanitizeExcelCell(r.UOM))

		// Quoted Price (formatted string).
		f.SetCellValue(sheetName, "E"+rowStr, FormatINR(r.QuotedPrice))

		// Budgeted Price (formatted string).
		f.SetCellValue(sheetName, "F"+rowStr, FormatINR(r.BudgetedPrice))

		// HSN Code.
		f.SetCellValue(sheetName, "G"+rowStr, sanitizeExcelCell(r.HSNCode))

		// GST%.
		f.SetCellValue(sheetName, "H"+rowStr, r.GSTPercent)

		// Apply row style based on level.
		style := subItemStyle
		if r.Level == 0 {
			style = mainItemStyle
		}
		f.SetCellStyle(sheetName, "A"+rowStr, lastCol+rowStr, style)

		row++
	}

	// ── Summary Rows ────────────────────────────────────────────────────

	// Skip a blank row.
	row++

	// Total Quoted.
	summaryRow := fmt.Sprintf("%d", row)
	f.SetCellValue(sheetName, "D"+summaryRow, "Total Quoted:")
	f.SetCellStyle(sheetName, "D"+summaryRow, "D"+summaryRow, summaryLabelStyle)
	f.SetCellValue(sheetName, "E"+summaryRow, FormatINR(data.TotalQuoted))
	f.SetCellStyle(sheetName, "E"+summaryRow, "E"+summaryRow, summaryValueStyle)
	row++

	// Total Budgeted.
	summaryRow = fmt.Sprintf("%d", row)
	f.SetCellValue(sheetName, "D"+summaryRow, "Total Budgeted:")
	f.SetCellStyle(sheetName, "D"+summaryRow, "D"+summaryRow, summaryLabelStyle)
	f.SetCellValue(sheetName, "F"+summaryRow, FormatINR(data.TotalBudgeted))
	f.SetCellStyle(sheetName, "F"+summaryRow, "F"+summaryRow, summaryValueStyle)
	row++

	// Margin.
	summaryRow = fmt.Sprintf("%d", row)
	marginLabel := fmt.Sprintf("Margin (%.1f%%):", data.MarginPercent)
	f.SetCellValue(sheetName, "D"+summaryRow, marginLabel)
	f.SetCellStyle(sheetName, "D"+summaryRow, "D"+summaryRow, summaryLabelStyle)
	f.SetCellValue(sheetName, "E"+summaryRow, FormatINR(data.Margin))
	f.SetCellStyle(sheetName, "E"+summaryRow, "E"+summaryRow, summaryValueStyle)

	// ── Write to buffer ─────────────────────────────────────────────────

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write excel: %w", err)
	}

	return buf.Bytes(), nil
}

// sanitizeExcelCell prevents formula injection by prefixing dangerous leading
// characters with a single quote. Excel interprets cells starting with =, +, -,
// @, \t or \r as formulas, which can be abused for code execution or data theft.
func sanitizeExcelCell(s string) string {
	if len(s) == 0 {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r', '|':
		return "'" + s
	}
	return s
}

// thinBorders returns a slice of excelize.Border for thin borders on all four sides.
func thinBorders() []excelize.Border {
	sides := []string{"left", "top", "bottom", "right"}
	borders := make([]excelize.Border, len(sides))
	for i, side := range sides {
		borders[i] = excelize.Border{
			Type:  side,
			Color: "#000000",
			Style: 1, // thin
		}
	}
	return borders
}
