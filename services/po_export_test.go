package services

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"projectcreation/testhelpers"
)

func TestBuildPOExportData_Complete(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Laksh Ribbons")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-001")

	// Set additional fields on PO
	po.Set("order_date", "2025-01-15")
	po.Set("quotation_ref", "QR-2025-001")
	po.Set("ref_date", "2025-01-10")
	po.Set("payment_terms", "Net 30")
	po.Set("delivery_terms", "FOB Bangalore")
	po.Set("warranty_terms", "12 months")
	po.Set("comments", "Urgent delivery required")
	if err := app.Save(po); err != nil {
		t.Fatalf("failed to update PO: %v", err)
	}

	testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "LED Panel 40W", 10, 1500.0, 18)
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 2, "LED Driver", 10, 500.0, 18)

	data, err := BuildPOExportData(app, po.Id)
	if err != nil {
		t.Fatalf("BuildPOExportData failed: %v", err)
	}

	// Header
	if data.PONumber != "FSS-PO-001" {
		t.Errorf("PONumber = %q, want %q", data.PONumber, "FSS-PO-001")
	}
	if data.CompanyName != "FSS ENGINEERING" {
		t.Errorf("CompanyName = %q, want %q", data.CompanyName, "FSS ENGINEERING")
	}
	if data.OrderDate != "2025-01-15" {
		t.Errorf("OrderDate = %q, want %q", data.OrderDate, "2025-01-15")
	}
	if data.QuotationRef != "QR-2025-001" {
		t.Errorf("QuotationRef = %q, want %q", data.QuotationRef, "QR-2025-001")
	}

	// Vendor
	if data.Vendor.Name != "Laksh Ribbons" {
		t.Errorf("Vendor.Name = %q, want %q", data.Vendor.Name, "Laksh Ribbons")
	}
	if data.Vendor.GSTIN != "27AADCB2230M1ZV" {
		t.Errorf("Vendor.GSTIN = %q, want %q", data.Vendor.GSTIN, "27AADCB2230M1ZV")
	}

	// Line items
	if len(data.LineItems) != 2 {
		t.Fatalf("LineItems count = %d, want 2", len(data.LineItems))
	}

	// Terms
	if data.PaymentTerms != "Net 30" {
		t.Errorf("PaymentTerms = %q, want %q", data.PaymentTerms, "Net 30")
	}
	if data.DeliveryTerms != "FOB Bangalore" {
		t.Errorf("DeliveryTerms = %q, want %q", data.DeliveryTerms, "FOB Bangalore")
	}
	if data.Comments != "Urgent delivery required" {
		t.Errorf("Comments = %q, want %q", data.Comments, "Urgent delivery required")
	}

	// Amount in words should be non-empty
	if data.AmountInWords == "" {
		t.Error("AmountInWords should not be empty")
	}
}

func TestBuildPOExportData_AddressMapping(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")

	billAddr := testhelpers.CreateTestAddress(t, app, project.Id, "bill_from", "FSS Engineering Pvt Ltd")
	shipAddr := testhelpers.CreateTestAddress(t, app, project.Id, "ship_from", "FSS Warehouse")

	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-002")
	po.Set("bill_to_address", billAddr.Id)
	po.Set("ship_to_address", shipAddr.Id)
	if err := app.Save(po); err != nil {
		t.Fatalf("failed to update PO: %v", err)
	}

	data, err := BuildPOExportData(app, po.Id)
	if err != nil {
		t.Fatalf("BuildPOExportData failed: %v", err)
	}

	if data.BillTo == nil {
		t.Fatal("BillTo should not be nil")
	}
	if data.BillTo.CompanyName != "FSS Engineering Pvt Ltd" {
		t.Errorf("BillTo.CompanyName = %q, want %q", data.BillTo.CompanyName, "FSS Engineering Pvt Ltd")
	}

	if data.ShipTo == nil {
		t.Fatal("ShipTo should not be nil")
	}
	if data.ShipTo.CompanyName != "FSS Warehouse" {
		t.Errorf("ShipTo.CompanyName = %q, want %q", data.ShipTo.CompanyName, "FSS Warehouse")
	}
}

func TestBuildPOExportData_AddressMapping_NilBillTo(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-003")

	data, err := BuildPOExportData(app, po.Id)
	if err != nil {
		t.Fatalf("BuildPOExportData failed: %v", err)
	}

	if data.BillTo != nil {
		t.Errorf("BillTo should be nil when no bill_to_address is set")
	}
}

func TestBuildPOExportData_AddressMapping_NilShipTo(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-004")

	data, err := BuildPOExportData(app, po.Id)
	if err != nil {
		t.Fatalf("BuildPOExportData failed: %v", err)
	}

	if data.ShipTo != nil {
		t.Errorf("ShipTo should be nil when no ship_to_address is set")
	}
}

func TestBuildPOExportData_LineItemCalc(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-005")

	// rate=1000, qty=10, gst=18%
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "Widget", 10, 1000.0, 18)

	data, err := BuildPOExportData(app, po.Id)
	if err != nil {
		t.Fatalf("BuildPOExportData failed: %v", err)
	}

	if len(data.LineItems) != 1 {
		t.Fatalf("LineItems count = %d, want 1", len(data.LineItems))
	}

	item := data.LineItems[0]
	// BeforeGST = rate * qty = 1000 * 10 = 10000
	if item.BeforeGST != 10000 {
		t.Errorf("BeforeGST = %.2f, want 10000.00", item.BeforeGST)
	}
	// GSTAmount = 10000 * 18/100 = 1800
	if item.GSTAmount != 1800 {
		t.Errorf("GSTAmount = %.2f, want 1800.00", item.GSTAmount)
	}
	// TotalAmount = 10000 + 1800 = 11800
	if item.TotalAmount != 11800 {
		t.Errorf("TotalAmount = %.2f, want 11800.00", item.TotalAmount)
	}
}

func TestBuildPOExportData_Totals(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-006")

	// Item 1: rate=1000, qty=10, gst=18% → before=10000, gst=1800, total=11800
	// Item 2: rate=500, qty=20, gst=18% → before=10000, gst=1800, total=11800
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "Widget A", 10, 1000.0, 18)
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 2, "Widget B", 20, 500.0, 18)

	data, err := BuildPOExportData(app, po.Id)
	if err != nil {
		t.Fatalf("BuildPOExportData failed: %v", err)
	}

	// TotalBeforeTax = 10000 + 10000 = 20000
	if data.TotalBeforeTax != 20000 {
		t.Errorf("TotalBeforeTax = %.2f, want 20000.00", data.TotalBeforeTax)
	}

	// IGSTAmount = 1800 + 1800 = 3600
	if data.IGSTAmount != 3600 {
		t.Errorf("IGSTAmount = %.2f, want 3600.00", data.IGSTAmount)
	}

	// GrandTotal = round(20000 + 3600) = 23600
	if data.GrandTotal != 23600 {
		t.Errorf("GrandTotal = %.2f, want 23600.00", data.GrandTotal)
	}
}

func TestBuildPOExportData_AmountInWords(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-007")

	// Single item: rate=1000, qty=10, gst=18% → total=11800
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "Widget", 10, 1000.0, 18)

	data, err := BuildPOExportData(app, po.Id)
	if err != nil {
		t.Fatalf("BuildPOExportData failed: %v", err)
	}

	// GrandTotal = 11800, amount in words should contain "Eleven Thousand Eight Hundred"
	expected := "Eleven Thousand Eight Hundred Rupees Only/-"
	if data.AmountInWords != expected {
		t.Errorf("AmountInWords = %q, want %q", data.AmountInWords, expected)
	}
}

func TestBuildPOExportData_VendorBankDetails(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")

	// Create vendor with bank details
	col, err := app.FindCollectionByNameOrId("vendors")
	if err != nil {
		t.Fatalf("failed to find vendors collection: %v", err)
	}
	vendorRec := core.NewRecord(col)
	vendorRec.Set("name", "Bank Vendor")
	vendorRec.Set("bank_beneficiary_name", "Bank Vendor Pvt Ltd")
	vendorRec.Set("bank_name", "HDFC Bank")
	vendorRec.Set("bank_account_no", "12345678901234")
	vendorRec.Set("bank_ifsc", "HDFC0001234")
	vendorRec.Set("bank_branch", "Koramangala")
	if err := app.Save(vendorRec); err != nil {
		t.Fatalf("failed to save vendor: %v", err)
	}

	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendorRec.Id, "FSS-PO-008")

	data, err := BuildPOExportData(app, po.Id)
	if err != nil {
		t.Fatalf("BuildPOExportData failed: %v", err)
	}

	if data.Vendor.BankBeneficiaryName != "Bank Vendor Pvt Ltd" {
		t.Errorf("BankBeneficiaryName = %q, want %q", data.Vendor.BankBeneficiaryName, "Bank Vendor Pvt Ltd")
	}
	if data.Vendor.BankName != "HDFC Bank" {
		t.Errorf("BankName = %q, want %q", data.Vendor.BankName, "HDFC Bank")
	}
	if data.Vendor.BankAccountNo != "12345678901234" {
		t.Errorf("BankAccountNo = %q, want %q", data.Vendor.BankAccountNo, "12345678901234")
	}
	if data.Vendor.BankIFSC != "HDFC0001234" {
		t.Errorf("BankIFSC = %q, want %q", data.Vendor.BankIFSC, "HDFC0001234")
	}
	if data.Vendor.BankBranch != "Koramangala" {
		t.Errorf("BankBranch = %q, want %q", data.Vendor.BankBranch, "Koramangala")
	}
}

func TestBuildPOExportData_EmptyLineItems(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-009")

	data, err := BuildPOExportData(app, po.Id)
	if err != nil {
		t.Fatalf("BuildPOExportData failed: %v", err)
	}

	if len(data.LineItems) != 0 {
		t.Errorf("LineItems count = %d, want 0", len(data.LineItems))
	}
	if data.GrandTotal != 0 {
		t.Errorf("GrandTotal = %.2f, want 0.00", data.GrandTotal)
	}
	if data.AmountInWords != "Zero Rupees Only/-" {
		t.Errorf("AmountInWords = %q, want %q", data.AmountInWords, "Zero Rupees Only/-")
	}
}

func TestPOExportPDF_GeneratesFile(t *testing.T) {
	data := &POExportData{
		CompanyName:    "FSS ENGINEERING",
		CompanyAddress: "Bangalore, Karnataka",
		CompanyEmail:   "info@fssengineering.com",
		PONumber:       "FSS-PO-TEST-001",
		OrderDate:      "2025-01-15",
		QuotationRef:   "QR-001",
		RefDate:        "2025-01-10",
		Status:         "draft",
		Vendor: POExportVendor{
			Name:                "Test Vendor",
			Address:             "123 Test Street\nMumbai, Maharashtra, 400001",
			GSTIN:               "27AADCB2230M1ZV",
			ContactName:         "John Doe",
			Phone:               "9876543210",
			Email:               "john@test.com",
			BankBeneficiaryName: "Test Vendor Pvt Ltd",
			BankName:            "HDFC Bank",
			BankAccountNo:       "1234567890",
			BankIFSC:            "HDFC0001234",
			BankBranch:          "Main Branch",
		},
		BillTo: &POExportAddress{
			CompanyName:   "FSS Engineering",
			AddressLines:  "456 Bill Street\nBangalore, Karnataka, 560001",
			ContactNo:     "9876543211",
			ContactPerson: "Jane Smith",
			GSTIN:         "29AADCB2230M1ZV",
		},
		ShipTo: &POExportAddress{
			CompanyName:   "FSS Warehouse",
			AddressLines:  "789 Ship Avenue\nBangalore, Karnataka, 560002",
			ContactNo:     "9876543212",
			ContactPerson: "Bob Wilson",
			GSTIN:         "29AADCB2230M1ZV",
		},
		LineItems: []POExportLineItem{
			{SINo: 1, Description: "LED Panel 40W", HSNCode: "8504", Qty: 10, UoM: "Nos", Rate: 1500, BeforeGST: 15000, GSTPercent: 18, GSTAmount: 2700, TotalAmount: 17700},
			{SINo: 2, Description: "LED Driver", HSNCode: "8504", Qty: 10, UoM: "Nos", Rate: 500, BeforeGST: 5000, GSTPercent: 18, GSTAmount: 900, TotalAmount: 5900},
		},
		TotalBeforeTax: 20000,
		IGSTPercent:    18,
		IGSTAmount:     3600,
		RoundOff:       0,
		GrandTotal:     23600,
		AmountInWords:  "Twenty Three Thousand Six Hundred Rupees Only/-",
		PaymentTerms:   "Net 30",
		DeliveryTerms:  "FOB Bangalore",
		WarrantyTerms:  "12 months",
		Comments:       "Rush order",
	}

	pdfBytes, err := GeneratePOPDF(data)
	if err != nil {
		t.Fatalf("GeneratePOPDF failed: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Error("PDF bytes length should be > 0")
	}

	// Verify it starts with PDF magic bytes
	if len(pdfBytes) >= 4 && string(pdfBytes[:4]) != "%PDF" {
		t.Error("Generated bytes do not start with PDF header")
	}
}

func TestPOExportPDF_MatchesFormat(t *testing.T) {
	data := &POExportData{
		CompanyName:    "FSS ENGINEERING",
		CompanyAddress: "Bangalore",
		PONumber:       "FSS-PO-G10X-4/25-26/001",
		Vendor: POExportVendor{
			Name: "Laksh Ribbons",
		},
		LineItems:     []POExportLineItem{},
		AmountInWords: "Zero Rupees Only/-",
	}

	pdfBytes, err := GeneratePOPDF(data)
	if err != nil {
		t.Fatalf("GeneratePOPDF failed: %v", err)
	}

	pdfStr := string(pdfBytes)

	// Check that key text appears in the PDF content
	checks := []string{
		"PURCHASE ORDER",
		"FSS ENGINEERING",
		"FSS-PO-G10X-4/25-26/001",
		"Laksh Ribbons",
	}

	for _, check := range checks {
		found := false
		for i := 0; i <= len(pdfStr)-len(check); i++ {
			if pdfStr[i:i+len(check)] == check {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PDF should contain %q", check)
		}
	}
}

func TestPOExportPDF_NilAddresses(t *testing.T) {
	data := &POExportData{
		CompanyName: "FSS ENGINEERING",
		PONumber:    "FSS-PO-NIL",
		Vendor:      POExportVendor{Name: "Test"},
		BillTo:      nil,
		ShipTo:      nil,
		LineItems:   []POExportLineItem{},
	}

	pdfBytes, err := GeneratePOPDF(data)
	if err != nil {
		t.Fatalf("GeneratePOPDF should handle nil addresses without error: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Error("PDF bytes should not be empty even with nil addresses")
	}
}
