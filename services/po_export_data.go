package services

import (
	"fmt"
	"log"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// POExportData holds all data needed to generate a PO PDF.
type POExportData struct {
	// Company (hardcoded for now)
	CompanyName    string
	CompanyAddress string
	CompanyEmail   string

	// PO Header
	PONumber     string
	OrderDate    string
	QuotationRef string
	RefDate      string
	Status       string

	// Vendor
	Vendor POExportVendor

	// Addresses (mapped: Bill To = bill_from, Ship To = ship_from)
	BillTo *POExportAddress
	ShipTo *POExportAddress

	// Line Items
	LineItems []POExportLineItem

	// Totals
	TotalBeforeTax float64
	IGSTPercent    float64
	IGSTAmount     float64
	RoundOff       float64
	GrandTotal     float64
	AmountInWords  string

	// Terms
	PaymentTerms  string
	DeliveryTerms string
	WarrantyTerms string
	Comments      string
}

// POExportVendor holds vendor details for PDF export.
type POExportVendor struct {
	Name        string
	Address     string // formatted multi-line
	GSTIN       string
	ContactName string
	Phone       string
	Email       string
	// Bank details
	BankBeneficiaryName string
	BankName            string
	BankAccountNo       string
	BankIFSC            string
	BankBranch          string
}

// POExportAddress holds address details for PDF export.
type POExportAddress struct {
	CompanyName   string
	AddressLines  string // formatted multi-line
	ContactNo     string
	ContactPerson string
	GSTIN         string
}

// POExportLineItem holds a single line item for PDF export.
type POExportLineItem struct {
	SINo        int
	Description string
	HSNCode     string
	Qty         float64
	UoM         string
	Rate        float64
	BeforeGST   float64
	GSTPercent  float64
	GSTAmount   float64
	TotalAmount float64
}

// BuildPOExportData assembles all data needed for PDF generation from PocketBase records.
func BuildPOExportData(app *pocketbase.PocketBase, poId string) (*POExportData, error) {
	// 1. Find PO record
	po, err := app.FindRecordById("purchase_orders", poId)
	if err != nil {
		return nil, fmt.Errorf("purchase order not found: %w", err)
	}

	// 2. Build vendor
	vendor := POExportVendor{}
	vendorID := po.GetString("vendor")
	if vendorID != "" {
		v, err := app.FindRecordById("vendors", vendorID)
		if err != nil {
			log.Printf("po_export: could not find vendor %s: %v", vendorID, err)
		} else {
			// Build formatted address
			addrParts := []string{}
			if line1 := v.GetString("address_line_1"); line1 != "" {
				addrParts = append(addrParts, line1)
			}
			if line2 := v.GetString("address_line_2"); line2 != "" {
				addrParts = append(addrParts, line2)
			}
			cityStateParts := []string{}
			if city := v.GetString("city"); city != "" {
				cityStateParts = append(cityStateParts, city)
			}
			if state := v.GetString("state"); state != "" {
				cityStateParts = append(cityStateParts, state)
			}
			if pin := v.GetString("pin_code"); pin != "" {
				cityStateParts = append(cityStateParts, pin)
			}
			if len(cityStateParts) > 0 {
				addrParts = append(addrParts, strings.Join(cityStateParts, ", "))
			}

			vendor = POExportVendor{
				Name:                v.GetString("name"),
				Address:             strings.Join(addrParts, "\n"),
				GSTIN:               v.GetString("gstin"),
				ContactName:         v.GetString("contact_name"),
				Phone:               v.GetString("phone"),
				Email:               v.GetString("email"),
				BankBeneficiaryName: v.GetString("bank_beneficiary_name"),
				BankName:            v.GetString("bank_name"),
				BankAccountNo:       v.GetString("bank_account_no"),
				BankIFSC:            v.GetString("bank_ifsc"),
				BankBranch:          v.GetString("bank_branch"),
			}
		}
	}

	// 3. Fetch Bill To address
	var billTo *POExportAddress
	if billToID := po.GetString("bill_to_address"); billToID != "" {
		if addr, err := app.FindRecordById("addresses", billToID); err == nil {
			billTo = buildExportAddress(addr)
		} else {
			log.Printf("po_export: could not find bill_to address %s: %v", billToID, err)
		}
	}

	// 4. Fetch Ship To address
	var shipTo *POExportAddress
	if shipToID := po.GetString("ship_to_address"); shipToID != "" {
		if addr, err := app.FindRecordById("addresses", shipToID); err == nil {
			shipTo = buildExportAddress(addr)
		} else {
			log.Printf("po_export: could not find ship_to address %s: %v", shipToID, err)
		}
	}

	// 5. Fetch line items
	lineItemRecords, err := app.FindRecordsByFilter(
		"po_line_items",
		"purchase_order = {:poId}",
		"sort_order",
		0,
		0,
		map[string]any{"poId": poId},
	)
	if err != nil {
		log.Printf("po_export: could not fetch line items for PO %s: %v", poId, err)
		lineItemRecords = nil
	}

	// 6. Build line items and calculate totals
	var lineItems []POExportLineItem
	var calcItems []POLineItemCalc

	for i, item := range lineItemRecords {
		rate := item.GetFloat("rate")
		qty := item.GetFloat("qty")
		gstPercent := item.GetFloat("gst_percent")
		calc := CalcPOLineItem(rate, qty, gstPercent)
		calcItems = append(calcItems, calc)

		lineItems = append(lineItems, POExportLineItem{
			SINo:        i + 1,
			Description: item.GetString("description"),
			HSNCode:     item.GetString("hsn_code"),
			Qty:         qty,
			UoM:         item.GetString("uom"),
			Rate:        rate,
			BeforeGST:   calc.BeforeGST,
			GSTPercent:  gstPercent,
			GSTAmount:   calc.GSTAmount,
			TotalAmount: calc.Total,
		})
	}

	// 7. Calculate order-level totals
	totals := CalcPOTotals(calcItems)

	return &POExportData{
		CompanyName:    "FSS ENGINEERING",
		CompanyAddress: "Bangalore, Karnataka",
		CompanyEmail:   "info@fssengineering.com",

		PONumber:     po.GetString("po_number"),
		OrderDate:    po.GetString("order_date"),
		QuotationRef: po.GetString("quotation_ref"),
		RefDate:      po.GetString("ref_date"),
		Status:       po.GetString("status"),

		Vendor: vendor,
		BillTo: billTo,
		ShipTo: shipTo,

		LineItems: lineItems,

		TotalBeforeTax: totals.TotalBeforeTax,
		IGSTPercent:    totals.IGSTPercent,
		IGSTAmount:     totals.IGSTAmount,
		RoundOff:       totals.RoundOff,
		GrandTotal:     totals.GrandTotal,
		AmountInWords:  AmountToWords(totals.GrandTotal),

		PaymentTerms:  po.GetString("payment_terms"),
		DeliveryTerms: po.GetString("delivery_terms"),
		WarrantyTerms: po.GetString("warranty_terms"),
		Comments:      po.GetString("comments"),
	}, nil
}

// buildExportAddress creates a POExportAddress from a PocketBase address record.
func buildExportAddress(addr *core.Record) *POExportAddress {
	addrParts := []string{}
	if line1 := addr.GetString("address_line_1"); line1 != "" {
		addrParts = append(addrParts, line1)
	}
	if line2 := addr.GetString("address_line_2"); line2 != "" {
		addrParts = append(addrParts, line2)
	}
	cityStateParts := []string{}
	if city := addr.GetString("city"); city != "" {
		cityStateParts = append(cityStateParts, city)
	}
	if state := addr.GetString("state"); state != "" {
		cityStateParts = append(cityStateParts, state)
	}
	if pin := addr.GetString("pin_code"); pin != "" {
		cityStateParts = append(cityStateParts, pin)
	}
	if len(cityStateParts) > 0 {
		addrParts = append(addrParts, strings.Join(cityStateParts, ", "))
	}

	return &POExportAddress{
		CompanyName:   addr.GetString("company_name"),
		AddressLines:  strings.Join(addrParts, "\n"),
		ContactNo:     addr.GetString("phone"),
		ContactPerson: addr.GetString("contact_name"),
		GSTIN:         addr.GetString("gstin"),
	}
}
