package services

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestGenerateExcel_BasicBOQ(t *testing.T) {
	data := ExportData{
		Title:           "Test BOQ",
		ReferenceNumber: "REF-001",
		CreatedDate:     "2025-01-15",
		Rows: []ExportRow{
			{Level: 0, Index: "1", Description: "Main Item", Qty: 10, UOM: "Nos", QuotedPrice: 1000, BudgetedPrice: 900, HSNCode: "8541", GSTPercent: 18},
			{Level: 1, Index: "1.1", Description: "Sub Item", Qty: 5, UOM: "Sqm", QuotedPrice: 200, BudgetedPrice: 180, HSNCode: "8542", GSTPercent: 12},
		},
		TotalQuoted:   1200,
		TotalBudgeted: 1080,
		Margin:        120,
		MarginPercent: 10,
	}

	result, err := GenerateExcel(data)
	if err != nil {
		t.Fatalf("GenerateExcel() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GenerateExcel() returned empty bytes")
	}

	// Verify it's a valid Excel file
	f, err := excelize.OpenReader(bytesReader(result))
	if err != nil {
		t.Fatalf("result is not valid Excel: %v", err)
	}
	defer f.Close()

	// Check sheet name
	sheets := f.GetSheetList()
	if len(sheets) == 0 || sheets[0] != "Test BOQ" {
		t.Errorf("expected sheet name 'Test BOQ', got %v", sheets)
	}

	// Check title cell
	title, _ := f.GetCellValue(sheets[0], "A1")
	if title != "Test BOQ" {
		t.Errorf("expected title 'Test BOQ', got %q", title)
	}
}

func TestGenerateExcel_EmptyItems(t *testing.T) {
	data := ExportData{
		Title:       "Empty BOQ",
		CreatedDate: "2025-01-15",
		Rows:        []ExportRow{},
	}

	result, err := GenerateExcel(data)
	if err != nil {
		t.Fatalf("GenerateExcel() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GenerateExcel() returned empty bytes")
	}
}

func TestGenerateExcel_LongTitle(t *testing.T) {
	data := ExportData{
		Title:       "This is a very long title that exceeds thirty one characters",
		CreatedDate: "2025-01-15",
		Rows:        []ExportRow{},
	}

	result, err := GenerateExcel(data)
	if err != nil {
		t.Fatalf("GenerateExcel() error = %v", err)
	}

	f, err := excelize.OpenReader(bytesReader(result))
	if err != nil {
		t.Fatalf("result is not valid Excel: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets[0]) > 31 {
		t.Errorf("sheet name exceeds 31 chars: %d", len(sheets[0]))
	}
}

func TestGenerateExcel_EmptyTitle(t *testing.T) {
	data := ExportData{
		Title:       "",
		CreatedDate: "2025-01-15",
		Rows:        []ExportRow{},
	}

	result, err := GenerateExcel(data)
	if err != nil {
		t.Fatalf("GenerateExcel() error = %v", err)
	}

	f, err := excelize.OpenReader(bytesReader(result))
	if err != nil {
		t.Fatalf("result is not valid Excel: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if sheets[0] != "BOQ" {
		t.Errorf("expected default sheet name 'BOQ', got %q", sheets[0])
	}
}

func TestGenerateExcel_SubSubItemHierarchy(t *testing.T) {
	data := ExportData{
		Title:       "Hierarchy Test",
		CreatedDate: "2025-01-15",
		Rows: []ExportRow{
			{Level: 0, Index: "1", Description: "Main", Qty: 1, UOM: "Lot"},
			{Level: 1, Index: "1.1", Description: "Sub", Qty: 2, UOM: "Nos"},
			{Level: 2, Index: "1.1.1", Description: "Sub-Sub", Qty: 3, UOM: "Sqm"},
		},
	}

	result, err := GenerateExcel(data)
	if err != nil {
		t.Fatalf("GenerateExcel() error = %v", err)
	}

	f, err := excelize.OpenReader(bytesReader(result))
	if err != nil {
		t.Fatalf("result is not valid Excel: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetList()[0]

	// Row 6 = first data row, B6 = description
	mainDesc, _ := f.GetCellValue(sheet, "B6")
	subDesc, _ := f.GetCellValue(sheet, "B7")
	subSubDesc, _ := f.GetCellValue(sheet, "B8")

	if mainDesc != "Main" {
		t.Errorf("main item desc = %q, want 'Main'", mainDesc)
	}
	if subDesc != "  Sub" {
		t.Errorf("sub item desc = %q, want '  Sub'", subDesc)
	}
	if subSubDesc != "    Sub-Sub" {
		t.Errorf("sub-sub item desc = %q, want '    Sub-Sub'", subSubDesc)
	}
}

func TestSanitizeExcelCell(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"normal text", "Hello", "Hello"},
		{"starts with equals", "=SUM(A1:A10)", "'=SUM(A1:A10)"},
		{"starts with plus", "+1234", "'+1234"},
		{"starts with minus", "-100", "'-100"},
		{"starts with at", "@import", "'@import"},
		{"starts with tab", "\tdata", "'\tdata"},
		{"starts with pipe", "|command", "'|command"},
		{"starts with carriage return", "\rdata", "'\rdata"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeExcelCell(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeExcelCell(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestThinBorders(t *testing.T) {
	borders := thinBorders()
	if len(borders) != 4 {
		t.Errorf("thinBorders() returned %d borders, want 4", len(borders))
	}

	sides := map[string]bool{"left": false, "top": false, "bottom": false, "right": false}
	for _, b := range borders {
		sides[b.Type] = true
		if b.Style != 1 {
			t.Errorf("border %s style = %d, want 1 (thin)", b.Type, b.Style)
		}
	}
	for side, found := range sides {
		if !found {
			t.Errorf("missing border side: %s", side)
		}
	}
}
