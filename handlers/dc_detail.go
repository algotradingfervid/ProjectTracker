package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// HandleDCDetail renders the DC detail view for a delivery challan.
func HandleDCDetail(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		dcId := e.Request.PathValue("id")

		// Fetch DC record
		dc, err := app.FindRecordById("delivery_challans", dcId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Delivery challan not found")
		}

		// Verify DC belongs to this project
		if dc.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Delivery challan not found in this project")
		}

		dcType := dc.GetString("dc_type")
		status := dc.GetString("status")

		// Resolve template name
		templateName := ""
		if templateID := dc.GetString("template"); templateID != "" {
			if tRec, err := app.FindRecordById("dc_templates", templateID); err == nil {
				templateName = tRec.GetString("name")
			}
		}

		// Resolve addresses
		billFrom := resolveDCAddressDisplay(app, dc.GetString("bill_from_address"))
		dispatchFrom := resolveDCAddressDisplay(app, dc.GetString("dispatch_from_address"))
		billTo := resolveDCAddressDisplay(app, dc.GetString("bill_to_address"))
		shipTo := resolveDCAddressDisplay(app, dc.GetString("ship_to_address"))

		// Fetch line items
		lineItems, _ := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "line_order", 0, 0, map[string]any{"did": dcId})
		var items []templates.DCDetailLineItem
		var totalTaxable, totalTax, grandTotal float64

		for i, li := range lineItems {
			qty := li.GetInt("quantity")
			rate := li.GetFloat("rate")
			taxPct := li.GetFloat("tax_percentage")
			taxable := li.GetFloat("taxable_amount")
			tax := li.GetFloat("tax_amount")
			total := li.GetFloat("total_amount")

			// Resolve source item description
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

			// Fetch serial numbers for this line item
			var serials []string
			serialRecs, _ := app.FindRecordsByFilter("serial_numbers", "line_item = {:lid}", "serial_number", 0, 0, map[string]any{"lid": li.Id})
			for _, sr := range serialRecs {
				serials = append(serials, sr.GetString("serial_number"))
			}

			items = append(items, templates.DCDetailLineItem{
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
		var transitDetail *templates.DCDetailTransit
		transitRecs, _ := app.FindRecordsByFilter("dc_transit_details", "dc = {:did}", "", 1, 0, map[string]any{"did": dcId})
		if len(transitRecs) > 0 {
			tr := transitRecs[0]
			transporterName := ""
			if tID := tr.GetString("transporter"); tID != "" {
				if tRec, err := app.FindRecordById("transporters", tID); err == nil {
					transporterName = tRec.GetString("company_name")
				}
			}
			transitDetail = &templates.DCDetailTransit{
				TransporterName: transporterName,
				VehicleNumber:   tr.GetString("vehicle_number"),
				EwayBillNumber:  tr.GetString("eway_bill_number"),
				DocketNumber:    tr.GetString("docket_number"),
				Notes:           tr.GetString("notes"),
			}
		}

		// For shipment group DCs: fetch related DCs
		var shipmentGroupInfo *templates.DCDetailShipmentGroup
		if sgID := dc.GetString("shipment_group"); sgID != "" {
			sgRec, err := app.FindRecordById("shipment_groups", sgID)
			if err == nil {
				// Find all DCs in this shipment group
				relatedDCs, _ := app.FindRecordsByFilter("delivery_challans", "shipment_group = {:sgid}", "dc_type,dc_number", 0, 0, map[string]any{"sgid": sgID})
				var transitDC *templates.DCDetailRelatedDC
				var officialDCs []templates.DCDetailRelatedDC
				for _, rdc := range relatedDCs {
					if rdc.Id == dcId {
						continue // skip current DC
					}
					related := templates.DCDetailRelatedDC{
						ID:       rdc.Id,
						DCNumber: rdc.GetString("dc_number"),
						DCType:   rdc.GetString("dc_type"),
						Status:   rdc.GetString("status"),
					}
					if rdc.GetString("dc_type") == "transit" {
						transitDC = &related
					} else {
						officialDCs = append(officialDCs, related)
					}
				}
				shipmentGroupInfo = &templates.DCDetailShipmentGroup{
					ID:          sgID,
					Status:      sgRec.GetString("status"),
					NumLocations: sgRec.GetInt("num_locations"),
					TransitDC:   transitDC,
					OfficialDCs: officialDCs,
				}
			}
		}

		// For transfer DCs: fetch destination plan
		var transferInfo *templates.DCDetailTransferInfo
		if dcType == "transfer" {
			transferRecs, _ := app.FindRecordsByFilter("transfer_dcs", "dc = {:did}", "", 1, 0, map[string]any{"did": dcId})
			if len(transferRecs) > 0 {
				tdc := transferRecs[0]
				numDest := tdc.GetInt("num_destinations")
				numSplit := tdc.GetInt("num_split")

				// Fetch destinations
				destRecs, _ := app.FindRecordsByFilter("transfer_dc_destinations", "transfer_dc = {:tid}", "", 0, 0, map[string]any{"tid": tdc.Id})
				var destinations []templates.DCDetailDestination
				for _, dest := range destRecs {
					shipToAddr := resolveDCAddressDisplay(app, dest.GetString("ship_to_address"))
					isSplit := dest.GetBool("is_split")

					destinations = append(destinations, templates.DCDetailDestination{
						ID:          dest.Id,
						ShipToName:  shipToAddr.CompanyName,
						ShipToCity:  shipToAddr.City,
						ShipToState: shipToAddr.State,
						IsSplit:     isSplit,
					})
				}

				// Resolve hub address
				hubAddr := resolveDCAddressDisplay(app, tdc.GetString("hub_address"))

				transferInfo = &templates.DCDetailTransferInfo{
					HubAddress:      hubAddr,
					NumDestinations: numDest,
					NumSplit:        numSplit,
					Destinations:    destinations,
				}
			}
		}

		// Format challan date
		challanDate := dc.GetString("challan_date")
		issuedAt := dc.GetString("issued_at")

		data := templates.DCDetailData{
			ProjectID:     projectId,
			DCID:          dcId,
			DCNumber:      dc.GetString("dc_number"),
			DCType:        dcType,
			Status:        status,
			TemplateName:  templateName,
			ChallanDate:   challanDate,
			IssuedAt:      issuedAt,
			BillFrom:      billFrom,
			DispatchFrom:  dispatchFrom,
			BillTo:        billTo,
			ShipTo:        shipTo,
			LineItems:     items,
			TotalTaxable:  services.FormatINR(totalTaxable),
			TotalTax:      services.FormatINR(totalTax),
			GrandTotal:    services.FormatINR(grandTotal),
			Transit:       transitDetail,
			ShipmentGroup: shipmentGroupInfo,
			TransferInfo:  transferInfo,
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.DCDetailContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.DCDetailPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// resolveDCAddressDisplay resolves an address record into a display struct.
func resolveDCAddressDisplay(app *pocketbase.PocketBase, addressID string) *templates.DCDetailAddress {
	if addressID == "" {
		return nil
	}
	rec, err := app.FindRecordById("addresses", addressID)
	if err != nil {
		return nil
	}
	data := readAddressData(rec)
	return &templates.DCDetailAddress{
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
