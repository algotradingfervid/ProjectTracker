package services

import (
	"testing"
)

func TestGeneratePDF_BasicBOQ(t *testing.T) {
	data := ExportData{
		Title:           "Test BOQ PDF",
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

	result, err := GeneratePDF(data)
	if err != nil {
		t.Fatalf("GeneratePDF() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GeneratePDF() returned empty bytes")
	}
	// PDF files start with %PDF
	if len(result) > 4 && string(result[:5]) != "%PDF-" {
		t.Errorf("result does not start with PDF header, got %q", string(result[:5]))
	}
}

func TestGeneratePDF_EmptyItems(t *testing.T) {
	data := ExportData{
		Title:       "Empty BOQ PDF",
		CreatedDate: "2025-01-15",
		Rows:        []ExportRow{},
	}

	result, err := GeneratePDF(data)
	if err != nil {
		t.Fatalf("GeneratePDF() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GeneratePDF() returned empty bytes")
	}
}

func TestGeneratePDF_AllLevels(t *testing.T) {
	data := ExportData{
		Title:       "Multi Level PDF",
		CreatedDate: "2025-01-15",
		Rows: []ExportRow{
			{Level: 0, Index: "1", Description: "Main", Qty: 1, UOM: "Lot"},
			{Level: 1, Index: "1.1", Description: "Sub", Qty: 2, UOM: "Nos"},
			{Level: 2, Index: "1.1.1", Description: "Sub-Sub", Qty: 3, UOM: "Sqm"},
		},
		TotalQuoted:   500,
		TotalBudgeted: 400,
		Margin:        100,
		MarginPercent: 20,
	}

	result, err := GeneratePDF(data)
	if err != nil {
		t.Fatalf("GeneratePDF() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GeneratePDF() returned empty bytes")
	}
}

func TestFormatQty(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{"whole number", 10, "10"},
		{"zero", 0, "0"},
		{"decimal", 10.5, "10.50"},
		{"small decimal", 0.25, "0.25"},
		{"large whole", 1000, "1000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatQty(tt.input)
			if got != tt.want {
				t.Errorf("formatQty(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
