package services

import (
	"testing"
)

func TestGeneratePOPDF_Complete(t *testing.T) {
	data := &POExportData{
		CompanyName:    "FSS Engineering",
		CompanyAddress: "Bangalore, Karnataka",
		CompanyEmail:   "info@fss.com",
		PONumber:       "FY-24-25/001",
		OrderDate:      "2025-01-15",
		QuotationRef:   "QR-001",
		RefDate:        "2025-01-10",
		Vendor: POExportVendor{
			Name:                "Test Vendor",
			Address:             "123 Main St\nMumbai, Maharashtra, 400001",
			GSTIN:               "27AAPFU0939F1ZV",
			ContactName:         "John Doe",
			Phone:               "9876543210",
			Email:               "vendor@test.com",
			BankBeneficiaryName: "Test Vendor Pvt Ltd",
			BankName:            "HDFC Bank",
			BankAccountNo:       "123456789012",
			BankIFSC:            "HDFC0001234",
			BankBranch:          "Mumbai Main",
		},
		BillTo: &POExportAddress{
			CompanyName:   "Bill Corp",
			AddressLines:  "456 Bill St\nDelhi, Delhi, 110001",
			ContactNo:     "9876543211",
			ContactPerson: "Jane Doe",
			GSTIN:         "07AAPFU0939F1ZV",
		},
		ShipTo: &POExportAddress{
			CompanyName:  "Ship Corp",
			AddressLines: "789 Ship Rd\nBangalore, Karnataka, 560001",
		},
		LineItems: []POExportLineItem{
			{SINo: 1, Description: "Item A", HSNCode: "8541", Qty: 10, UoM: "Nos", Rate: 100, BeforeGST: 1000, GSTPercent: 18, GSTAmount: 180, TotalAmount: 1180},
			{SINo: 2, Description: "Item B", HSNCode: "8542", Qty: 5, UoM: "Sqm", Rate: 200, BeforeGST: 1000, GSTPercent: 12, GSTAmount: 120, TotalAmount: 1120},
		},
		TotalBeforeTax: 2000,
		IGSTPercent:    18,
		IGSTAmount:     300,
		RoundOff:       0,
		GrandTotal:     2300,
		AmountInWords:  "Two Thousand Three Hundred Rupees Only",
		PaymentTerms:   "Net 30",
		DeliveryTerms:  "FOB",
		WarrantyTerms:  "1 Year",
		Comments:       "Handle with care",
	}

	result, err := GeneratePOPDF(data)
	if err != nil {
		t.Fatalf("GeneratePOPDF() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GeneratePOPDF() returned empty bytes")
	}
	if len(result) > 5 && string(result[:5]) != "%PDF-" {
		t.Errorf("result does not start with PDF header")
	}
}

func TestGeneratePOPDF_EmptyLineItems(t *testing.T) {
	data := &POExportData{
		CompanyName: "FSS Engineering",
		PONumber:    "FY-24-25/002",
		Vendor:      POExportVendor{Name: "Test Vendor"},
		LineItems:   []POExportLineItem{},
	}

	result, err := GeneratePOPDF(data)
	if err != nil {
		t.Fatalf("GeneratePOPDF() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GeneratePOPDF() returned empty bytes")
	}
}

func TestGeneratePOPDF_NilAddresses(t *testing.T) {
	data := &POExportData{
		CompanyName: "FSS Engineering",
		PONumber:    "FY-24-25/003",
		Vendor:      POExportVendor{Name: "Test Vendor"},
		BillTo:      nil,
		ShipTo:      nil,
		LineItems: []POExportLineItem{
			{SINo: 1, Description: "Item", Qty: 1, Rate: 100, BeforeGST: 100, TotalAmount: 100},
		},
	}

	result, err := GeneratePOPDF(data)
	if err != nil {
		t.Fatalf("GeneratePOPDF() with nil addresses error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GeneratePOPDF() returned empty bytes")
	}
}

func TestGeneratePOPDF_NoTerms(t *testing.T) {
	data := &POExportData{
		CompanyName:   "FSS Engineering",
		PONumber:      "FY-24-25/004",
		Vendor:        POExportVendor{Name: "Test Vendor"},
		PaymentTerms:  "",
		DeliveryTerms: "",
		WarrantyTerms: "",
		Comments:      "",
	}

	result, err := GeneratePOPDF(data)
	if err != nil {
		t.Fatalf("GeneratePOPDF() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GeneratePOPDF() returned empty bytes")
	}
}

func TestGeneratePOPDF_NoBankDetails(t *testing.T) {
	data := &POExportData{
		CompanyName: "FSS Engineering",
		PONumber:    "FY-24-25/005",
		Vendor:      POExportVendor{Name: "Vendor Without Bank"},
	}

	result, err := GeneratePOPDF(data)
	if err != nil {
		t.Fatalf("GeneratePOPDF() error = %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GeneratePOPDF() returned empty bytes")
	}
}

func TestJoinNonEmpty(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		sep   string
		want  string
	}{
		{"all non-empty", []string{"a", "b", "c"}, " | ", "a | b | c"},
		{"some empty", []string{"a", "", "c"}, " | ", "a | c"},
		{"all empty", []string{"", "", ""}, " | ", ""},
		{"single", []string{"a"}, " | ", "a"},
		{"nil", nil, " | ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinNonEmpty(tt.parts, tt.sep)
			if got != tt.want {
				t.Errorf("joinNonEmpty(%v, %q) = %q, want %q", tt.parts, tt.sep, got, tt.want)
			}
		})
	}
}

func TestFmtField(t *testing.T) {
	tests := []struct {
		name  string
		label string
		value string
		want  string
	}{
		{"non-empty value", "Phone", "9876543210", "Phone: 9876543210"},
		{"empty value", "Phone", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmtField(tt.label, tt.value)
			if got != tt.want {
				t.Errorf("fmtField(%q, %q) = %q, want %q", tt.label, tt.value, got, tt.want)
			}
		})
	}
}
