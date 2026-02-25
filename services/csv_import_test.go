package services

import (
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseCSV_Valid(t *testing.T) {
	input := "Name,City,State\nAcme,Mumbai,Maharashtra\nBeta,Delhi,Delhi\n"
	headers, rows, err := parseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseCSV() error = %v", err)
	}
	if len(headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(headers))
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 data rows, got %d", len(rows))
	}
}

func TestParseCSV_HeaderOnly(t *testing.T) {
	input := "Name,City,State\n"
	_, _, err := parseCSV(strings.NewReader(input))
	if err == nil {
		t.Error("expected error for header-only file")
	}
	if !strings.Contains(err.Error(), "at least one data row") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseCSV_Empty(t *testing.T) {
	input := ""
	_, _, err := parseCSV(strings.NewReader(input))
	if err == nil {
		t.Error("expected error for empty file")
	}
}

func TestParseCSV_SingleRow(t *testing.T) {
	input := "Name\n"
	_, _, err := parseCSV(strings.NewReader(input))
	if err == nil {
		t.Error("expected error for single-row file")
	}
}

func TestMapHeadersToFields(t *testing.T) {
	fields := []TemplateField{
		{Key: "company_name", Label: "Company Name"},
		{Key: "city", Label: "City"},
		{Key: "state", Label: "State"},
	}

	t.Run("exact match", func(t *testing.T) {
		headers := []string{"Company Name", "City", "State"}
		mapped, unrecognized := mapHeadersToFields(headers, fields)
		if len(unrecognized) != 0 {
			t.Errorf("expected no unrecognized, got %v", unrecognized)
		}
		if mapped[0] != "company_name" || mapped[1] != "city" || mapped[2] != "state" {
			t.Errorf("unexpected mapping: %v", mapped)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		headers := []string{"company name", "CITY", "State"}
		mapped, unrecognized := mapHeadersToFields(headers, fields)
		if len(unrecognized) != 0 {
			t.Errorf("expected no unrecognized, got %v", unrecognized)
		}
		if mapped[0] != "company_name" {
			t.Errorf("expected 'company_name', got %q", mapped[0])
		}
	})

	t.Run("with required asterisk", func(t *testing.T) {
		headers := []string{"Company Name *", "City *", "State"}
		mapped, unrecognized := mapHeadersToFields(headers, fields)
		if len(unrecognized) != 0 {
			t.Errorf("expected no unrecognized, got %v", unrecognized)
		}
		if mapped[0] != "company_name" {
			t.Errorf("expected 'company_name', got %q", mapped[0])
		}
	})

	t.Run("unrecognized columns", func(t *testing.T) {
		headers := []string{"Company Name", "Unknown Column", "City"}
		mapped, unrecognized := mapHeadersToFields(headers, fields)
		if len(unrecognized) != 1 || unrecognized[0] != "Unknown Column" {
			t.Errorf("expected ['Unknown Column'], got %v", unrecognized)
		}
		if mapped[1] != "" {
			t.Errorf("expected empty for unrecognized column, got %q", mapped[1])
		}
	})

	t.Run("with extra whitespace", func(t *testing.T) {
		headers := []string{"  Company Name  ", " City ", "State"}
		mapped, _ := mapHeadersToFields(headers, fields)
		if mapped[0] != "company_name" {
			t.Errorf("expected 'company_name', got %q", mapped[0])
		}
	})
}

func TestValidateImportFieldFormats(t *testing.T) {
	t.Run("all valid", func(t *testing.T) {
		data := map[string]string{
			"pin_code": "400001",
			"phone":    "9876543210",
			"email":    "test@example.com",
			"gstin":    "27AAPFU0939F1ZV",
			"pan":      "ABCDE1234F",
			"cin":      "U74999MH2000PTC123456",
		}
		errs := validateImportFieldFormats(2, data)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("all empty is valid", func(t *testing.T) {
		data := map[string]string{}
		errs := validateImportFieldFormats(2, data)
		if len(errs) != 0 {
			t.Errorf("expected no errors for empty data, got %v", errs)
		}
	})

	t.Run("invalid pin_code", func(t *testing.T) {
		data := map[string]string{"pin_code": "ABC"}
		errs := validateImportFieldFormats(2, data)
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Field != "PIN Code" {
			t.Errorf("expected field 'PIN Code', got %q", errs[0].Field)
		}
		if errs[0].Row != 2 {
			t.Errorf("expected row 2, got %d", errs[0].Row)
		}
	})

	t.Run("multiple invalid fields", func(t *testing.T) {
		data := map[string]string{
			"phone": "123",
			"email": "notanemail",
			"gstin": "INVALID",
		}
		errs := validateImportFieldFormats(5, data)
		if len(errs) != 3 {
			t.Errorf("expected 3 errors, got %d: %v", len(errs), errs)
		}
	})
}

func TestGenerateErrorReport_WithErrors(t *testing.T) {
	errors := []ValidationError{
		{Row: 2, Field: "Company Name", Message: "Company Name is required"},
		{Row: 3, Field: "PIN Code", Message: "PIN Code must be exactly 6 digits"},
		{Row: 5, Field: "GSTIN", Message: "GSTIN must be 15 characters"},
	}

	result, err := GenerateErrorReport(errors)
	if err != nil {
		t.Fatalf("GenerateErrorReport() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GenerateErrorReport() returned empty bytes")
	}

	// Verify it's valid Excel
	f, err := excelize.OpenReader(bytesReader(result))
	if err != nil {
		t.Fatalf("result is not valid Excel: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetList()[0]
	if sheet != "Errors" {
		t.Errorf("expected sheet name 'Errors', got %q", sheet)
	}

	// Check header row
	a1, _ := f.GetCellValue(sheet, "A1")
	b1, _ := f.GetCellValue(sheet, "B1")
	c1, _ := f.GetCellValue(sheet, "C1")
	if a1 != "Row #" || b1 != "Field" || c1 != "Error" {
		t.Errorf("unexpected headers: %q, %q, %q", a1, b1, c1)
	}

	// Check first data row
	a2, _ := f.GetCellValue(sheet, "A2")
	b2, _ := f.GetCellValue(sheet, "B2")
	if a2 != "2" {
		t.Errorf("expected row '2' in A2, got %q", a2)
	}
	if b2 != "Company Name" {
		t.Errorf("expected 'Company Name' in B2, got %q", b2)
	}
}

func TestGenerateErrorReport_NoErrors(t *testing.T) {
	result, err := GenerateErrorReport([]ValidationError{})
	if err != nil {
		t.Fatalf("GenerateErrorReport() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GenerateErrorReport() returned empty bytes")
	}
}
