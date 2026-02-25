package services

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestGetAddressColumns_Common(t *testing.T) {
	types := []string{"bill_from", "ship_from", "bill_to", "ship_to"}
	for _, addrType := range types {
		t.Run(addrType, func(t *testing.T) {
			cols := GetAddressColumns(addrType)
			if len(cols) == 0 {
				t.Fatal("expected non-empty columns")
			}
			// Common types should have 11 columns
			if len(cols) != 11 {
				t.Errorf("expected 11 columns for %s, got %d", addrType, len(cols))
			}
			// First column should be Company Name
			if cols[0].Field != "company_name" {
				t.Errorf("expected first column 'company_name', got %q", cols[0].Field)
			}
		})
	}
}

func TestGetAddressColumns_InstallAt(t *testing.T) {
	cols := GetAddressColumns("install_at")
	if len(cols) == 0 {
		t.Fatal("expected non-empty columns")
	}
	// Install At has an extra "Ship To Parent" column
	if len(cols) != 12 {
		t.Errorf("expected 12 columns for install_at, got %d", len(cols))
	}
	if cols[0].Field != "_ship_to_parent_name" {
		t.Errorf("expected first column '_ship_to_parent_name', got %q", cols[0].Field)
	}
}

func TestGenerateAddressExcel_WithData(t *testing.T) {
	data := AddressExportData{
		ProjectName: "Test Project",
		AddressType: "ship_to",
		TypeLabel:   "Ship To",
		Columns:     GetAddressColumns("ship_to"),
		Rows: []map[string]string{
			{"company_name": "Acme Corp", "city": "Mumbai", "state": "Maharashtra"},
			{"company_name": "Beta Inc", "city": "Delhi", "state": "Delhi"},
		},
	}

	result, err := GenerateAddressExcel(data)
	if err != nil {
		t.Fatalf("GenerateAddressExcel() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GenerateAddressExcel() returned empty bytes")
	}

	f, err := excelize.OpenReader(bytesReader(result))
	if err != nil {
		t.Fatalf("result is not valid Excel: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetList()[0]
	if sheet != "Ship To Addresses" {
		t.Errorf("expected sheet name 'Ship To Addresses', got %q", sheet)
	}
}

func TestGenerateAddressExcel_EmptyData(t *testing.T) {
	data := AddressExportData{
		ProjectName: "Test Project",
		AddressType: "bill_from",
		TypeLabel:   "Bill From",
		Columns:     GetAddressColumns("bill_from"),
		Rows:        []map[string]string{},
	}

	result, err := GenerateAddressExcel(data)
	if err != nil {
		t.Fatalf("GenerateAddressExcel() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GenerateAddressExcel() returned empty bytes for empty data")
	}
}

func TestAddrColName(t *testing.T) {
	tests := []struct {
		name  string
		index int
		want  string
	}{
		{"first column", 0, "A"},
		{"second column", 1, "B"},
		{"last single letter", 25, "Z"},
		{"first double letter", 26, "AA"},
		{"27th column", 27, "AB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addrColName(tt.index)
			if got != tt.want {
				t.Errorf("addrColName(%d) = %q, want %q", tt.index, got, tt.want)
			}
		})
	}
}
