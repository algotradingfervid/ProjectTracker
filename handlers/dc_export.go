package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/collections"
	"projectcreation/services"
)

// HandleDCExportPDF generates and downloads a PDF for a single DC.
func HandleDCExportPDF(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		dcId := e.Request.PathValue("id")

		data, err := buildDCExportData(app, projectId, dcId)
		if err != nil {
			log.Printf("dc_export_pdf: %v", err)
			return e.String(http.StatusNotFound, "DC not found")
		}

		pdfBytes, err := services.GenerateDCPDF(data)
		if err != nil {
			log.Printf("dc_export_pdf: failed to generate: %v", err)
			return e.String(http.StatusInternalServerError, "Failed to generate PDF")
		}

		filename := fmt.Sprintf("DC_%s_%s.pdf", sanitizeDCFilename(data.DCNumber), time.Now().Format("2006-01-02"))
		e.Response.Header().Set("Content-Type", "application/pdf")
		e.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		e.Response.Write(pdfBytes)
		return nil
	}
}

// HandleDCExportExcel generates and downloads an Excel file for a single DC.
func HandleDCExportExcel(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		dcId := e.Request.PathValue("id")

		data, err := buildDCExportData(app, projectId, dcId)
		if err != nil {
			log.Printf("dc_export_excel: %v", err)
			return e.String(http.StatusNotFound, "DC not found")
		}

		xlsxBytes, err := services.GenerateDCExcel(data)
		if err != nil {
			log.Printf("dc_export_excel: failed to generate: %v", err)
			return e.String(http.StatusInternalServerError, "Failed to generate Excel file")
		}

		filename := fmt.Sprintf("DC_%s_%s.xlsx", sanitizeDCFilename(data.DCNumber), time.Now().Format("2006-01-02"))
		e.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		e.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		e.Response.Write(xlsxBytes)
		return nil
	}
}

func sanitizeDCFilename(s string) string {
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = strings.ReplaceAll(s, ":", "-")
	return s
}

// buildDCExportData assembles all data needed for DC export from PocketBase records.
func buildDCExportData(app *pocketbase.PocketBase, projectId, dcId string) (*services.DCExportData, error) {
	dc, err := app.FindRecordById("delivery_challans", dcId)
	if err != nil {
		return nil, fmt.Errorf("delivery challan not found: %w", err)
	}
	if dc.GetString("project") != projectId {
		return nil, fmt.Errorf("DC does not belong to this project")
	}

	companyName := collections.GetCompanyName(app)

	data := &services.DCExportData{
		CompanyName: companyName,
		DCNumber:    dc.GetString("dc_number"),
		DCType:      dc.GetString("dc_type"),
		Status:      dc.GetString("status"),
		ChallanDate: dc.GetString("challan_date"),
	}

	// Resolve addresses
	data.BillFrom = resolveExportAddress(app, dc.GetString("bill_from_address"))
	data.DispatchFrom = resolveExportAddress(app, dc.GetString("dispatch_from_address"))
	data.BillTo = resolveExportAddress(app, dc.GetString("bill_to_address"))
	data.ShipTo = resolveExportAddress(app, dc.GetString("ship_to_address"))

	// Fetch transit details
	transitRecs, _ := app.FindRecordsByFilter("dc_transit_details", "dc = {:did}", "", 1, 0, map[string]any{"did": dcId})
	if len(transitRecs) > 0 {
		tr := transitRecs[0]
		transporterName := ""
		if tID := tr.GetString("transporter"); tID != "" {
			if tRec, err := app.FindRecordById("transporters", tID); err == nil {
				transporterName = tRec.GetString("company_name")
			}
		}
		data.Transport = &services.DCExportTransport{
			TransporterName: transporterName,
			VehicleNumber:   tr.GetString("vehicle_number"),
			EwayBillNumber:  tr.GetString("eway_bill_number"),
			DocketNumber:    tr.GetString("docket_number"),
		}
	}

	// Fetch line items
	lineItems, _ := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "line_order", 0, 0, map[string]any{"did": dcId})
	var totalTaxable, totalTax, grandTotal float64

	for i, li := range lineItems {
		qty := li.GetInt("quantity")
		rate := li.GetFloat("rate")
		taxPct := li.GetFloat("tax_percentage")
		taxable := li.GetFloat("taxable_amount")
		tax := li.GetFloat("tax_amount")
		total := li.GetFloat("total_amount")

		description := ""
		hsnCode := ""
		uom := ""
		collection := "sub_items"
		if li.GetString("source_item_type") == "sub_sub_item" {
			collection = "sub_sub_items"
		}
		if sourceRec, err := app.FindRecordById(collection, li.GetString("source_item_id")); err == nil {
			description = sourceRec.GetString("description")
			hsnCode = sourceRec.GetString("hsn_code")
			uom = sourceRec.GetString("uom")
		}

		// Fetch serial numbers
		var serials []string
		serialRecs, _ := app.FindRecordsByFilter("serial_numbers", "line_item = {:lid}", "serial_number", 0, 0, map[string]any{"lid": li.Id})
		for _, sr := range serialRecs {
			serials = append(serials, sr.GetString("serial_number"))
		}

		data.LineItems = append(data.LineItems, services.DCExportLineItem{
			SINo:        i + 1,
			Description: description,
			HSNCode:     hsnCode,
			Qty:         qty,
			UOM:         uom,
			Rate:        rate,
			TaxPercent:  taxPct,
			Taxable:     taxable,
			TaxAmount:   tax,
			Total:       total,
			Serials:     serials,
		})

		totalTaxable += taxable
		totalTax += tax
		grandTotal += total
	}

	data.TotalTaxable = totalTaxable
	data.TotalTax = totalTax
	data.GrandTotal = grandTotal

	// Transfer DC specific data
	if dc.GetString("dc_type") == "transfer" {
		transferRecs, _ := app.FindRecordsByFilter("transfer_dcs", "dc = {:did}", "", 1, 0, map[string]any{"did": dcId})
		if len(transferRecs) > 0 {
			tdc := transferRecs[0]
			hubAddr := resolveExportAddress(app, tdc.GetString("hub_address"))
			if hubAddr != nil {
				data.HubAddress = hubAddr.CompanyName
				if hubAddr.AddressLines != "" {
					data.HubAddress += ", " + hubAddr.AddressLines
				}
			}

			destRecs, _ := app.FindRecordsByFilter("transfer_dc_destinations", "transfer_dc = {:tid}", "", 0, 0, map[string]any{"tid": tdc.Id})
			for _, dest := range destRecs {
				addr := resolveExportAddress(app, dest.GetString("ship_to_address"))
				if addr != nil {
					name := addr.CompanyName
					if addr.AddressLines != "" {
						name += ", " + addr.AddressLines
					}
					data.Destinations = append(data.Destinations, name)
				}
			}
		}
	}

	return data, nil
}

// resolveExportAddress resolves an address record into export format.
func resolveExportAddress(app *pocketbase.PocketBase, addressID string) *services.DCExportAddress {
	if addressID == "" {
		return nil
	}
	rec, err := app.FindRecordById("addresses", addressID)
	if err != nil {
		return nil
	}
	addrData := readAddressData(rec)

	addrLines := []string{}
	if v := addrData["address_line_1"]; v != "" {
		addrLines = append(addrLines, v)
	}
	if v := addrData["address_line_2"]; v != "" {
		addrLines = append(addrLines, v)
	}
	cityStatePin := ""
	if v := addrData["city"]; v != "" {
		cityStatePin += v
	}
	if v := addrData["state"]; v != "" {
		if cityStatePin != "" {
			cityStatePin += ", "
		}
		cityStatePin += v
	}
	if v := addrData["pin_code"]; v != "" {
		cityStatePin += " - " + v
	}
	if cityStatePin != "" {
		addrLines = append(addrLines, cityStatePin)
	}

	return &services.DCExportAddress{
		CompanyName:   addrData["company_name"],
		AddressLines:  strings.Join(addrLines, ", "),
		ContactPerson: addrData["contact_person"],
		Phone:         addrData["phone"],
		GSTIN:         addrData["gstin"],
	}
}

