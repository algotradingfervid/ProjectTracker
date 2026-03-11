package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/collections"
	"projectcreation/services"
	"projectcreation/templates"
)

// HandleDCPrint renders a standalone HTML page optimized for browser printing.
func HandleDCPrint(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		dcId := e.Request.PathValue("id")

		dc, err := app.FindRecordById("delivery_challans", dcId)
		if err != nil {
			return e.String(http.StatusNotFound, "Delivery challan not found")
		}
		if dc.GetString("project") != projectId {
			return e.String(http.StatusNotFound, "Delivery challan not found in this project")
		}

		companyName := collections.GetCompanyName(app)
		logoURL := collections.GetLogoURL(app)
		dcType := dc.GetString("dc_type")

		// Resolve addresses
		billFrom := resolvePrintAddress(app, dc.GetString("bill_from_address"))
		dispatchFrom := resolvePrintAddress(app, dc.GetString("dispatch_from_address"))
		billTo := resolvePrintAddress(app, dc.GetString("bill_to_address"))
		shipTo := resolvePrintAddress(app, dc.GetString("ship_to_address"))

		// Fetch line items
		lineItems, _ := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "line_order", 0, 0, map[string]any{"did": dcId})
		var items []templates.DCPrintLineItem
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

			var serials []string
			serialRecs, _ := app.FindRecordsByFilter("serial_numbers", "line_item = {:lid}", "serial_number", 0, 0, map[string]any{"lid": li.Id})
			for _, sr := range serialRecs {
				serials = append(serials, sr.GetString("serial_number"))
			}

			items = append(items, templates.DCPrintLineItem{
				LineOrder:   i + 1,
				Description: description,
				HSNCode:     hsnCode,
				UOM:         uom,
				Qty:         strconv.Itoa(qty),
				Rate:        services.FormatINR(rate),
				TaxPercent:  fmt.Sprintf("%.1f", taxPct),
				Taxable:     services.FormatINR(taxable),
				TaxAmount:   services.FormatINR(tax),
				Total:       services.FormatINR(total),
				HasPricing:  rate > 0,
				Serials:     serials,
			})

			totalTaxable += taxable
			totalTax += tax
			grandTotal += total
		}

		// Fetch transit details
		var transit *templates.DCPrintTransit
		transitRecs, _ := app.FindRecordsByFilter("dc_transit_details", "dc = {:did}", "", 1, 0, map[string]any{"did": dcId})
		if len(transitRecs) > 0 {
			tr := transitRecs[0]
			transporterName := ""
			if tID := tr.GetString("transporter"); tID != "" {
				if tRec, err := app.FindRecordById("transporters", tID); err == nil {
					transporterName = tRec.GetString("company_name")
				}
			}
			transit = &templates.DCPrintTransit{
				TransporterName: transporterName,
				VehicleNumber:   tr.GetString("vehicle_number"),
				EwayBillNumber:  tr.GetString("eway_bill_number"),
				DocketNumber:    tr.GetString("docket_number"),
			}
		}

		// Transfer DC destinations
		var destinations []string
		var hubAddress string
		if dcType == "transfer" {
			transferRecs, _ := app.FindRecordsByFilter("transfer_dcs", "dc = {:did}", "", 1, 0, map[string]any{"did": dcId})
			if len(transferRecs) > 0 {
				tdc := transferRecs[0]
				hubAddr := resolveExportAddress(app, tdc.GetString("hub_address"))
				if hubAddr != nil {
					hubAddress = hubAddr.CompanyName
					if hubAddr.AddressLines != "" {
						hubAddress += ", " + hubAddr.AddressLines
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
						destinations = append(destinations, name)
					}
				}
			}
		}

		data := templates.DCPrintData{
			CompanyName:  companyName,
			LogoURL:      logoURL,
			DCNumber:     dc.GetString("dc_number"),
			DCType:       dcType,
			Status:       dc.GetString("status"),
			ChallanDate:  dc.GetString("challan_date"),
			BillFrom:     billFrom,
			DispatchFrom: dispatchFrom,
			BillTo:       billTo,
			ShipTo:       shipTo,
			LineItems:    items,
			TotalTaxable: services.FormatINR(totalTaxable),
			TotalTax:     services.FormatINR(totalTax),
			GrandTotal:   services.FormatINR(grandTotal),
			Transit:      transit,
			HubAddress:   hubAddress,
			Destinations: destinations,
		}

		component := templates.DCPrintPage(data)
		if err := component.Render(e.Request.Context(), e.Response); err != nil {
			log.Printf("dc_print: render error: %v", err)
			return e.String(http.StatusInternalServerError, "Failed to render print view")
		}
		return nil
	}
}

func resolvePrintAddress(app *pocketbase.PocketBase, addressID string) *templates.DCPrintAddress {
	if addressID == "" {
		return nil
	}
	rec, err := app.FindRecordById("addresses", addressID)
	if err != nil {
		return nil
	}
	data := readAddressData(rec)
	return &templates.DCPrintAddress{
		CompanyName:  data["company_name"],
		AddressLine1: data["address_line_1"],
		AddressLine2: data["address_line_2"],
		City:         data["city"],
		State:        data["state"],
		PinCode:      data["pin_code"],
		GSTIN:        data["gstin"],
		ContactName:  data["contact_person"],
		Phone:        data["phone"],
	}
}
