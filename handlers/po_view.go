package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// HandlePOView returns a handler that renders the Purchase Order view/detail page.
func HandlePOView(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")

		// 1. Find the PO record — 404 if not found
		po, err := app.FindRecordById("purchase_orders", id)
		if err != nil {
			log.Printf("po_view: could not find PO %s: %v", id, err)
			return e.String(http.StatusNotFound, "Purchase order not found")
		}

		// 2. Verify PO belongs to the requested project
		if po.GetString("project") != projectId {
			log.Printf("po_view: PO %s does not belong to project %s", id, projectId)
			return e.String(http.StatusNotFound, "Purchase order not found")
		}

		// 3. Fetch vendor and build POViewVendor
		vendor := templates.POViewVendor{}
		vendorID := po.GetString("vendor")
		if vendorID != "" {
			v, err := app.FindRecordById("vendors", vendorID)
			if err != nil {
				log.Printf("po_view: could not find vendor %s: %v", vendorID, err)
			} else {
				vendor = templates.POViewVendor{
					Name:                v.GetString("name"),
					AddressLine1:        v.GetString("address_line_1"),
					AddressLine2:        v.GetString("address_line_2"),
					City:                v.GetString("city"),
					State:               v.GetString("state"),
					PinCode:             v.GetString("pin_code"),
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

		// 4. Fetch Bill To address (optional)
		var billTo *templates.POViewAddress
		if billToID := po.GetString("bill_to_address"); billToID != "" {
			if addr, err := app.FindRecordById("addresses", billToID); err == nil {
				billTo = &templates.POViewAddress{
					CompanyName:  addr.GetString("company_name"),
					AddressLine1: addr.GetString("address_line_1"),
					AddressLine2: addr.GetString("address_line_2"),
					City:         addr.GetString("city"),
					State:        addr.GetString("state"),
					PinCode:      addr.GetString("pin_code"),
					GSTIN:        addr.GetString("gstin"),
					ContactName:  addr.GetString("contact_name"),
					Phone:        addr.GetString("phone"),
				}
			} else {
				log.Printf("po_view: could not find bill_to address %s: %v", billToID, err)
			}
		}

		// 5. Fetch Ship To address (optional)
		var shipTo *templates.POViewAddress
		if shipToID := po.GetString("ship_to_address"); shipToID != "" {
			if addr, err := app.FindRecordById("addresses", shipToID); err == nil {
				shipTo = &templates.POViewAddress{
					CompanyName:  addr.GetString("company_name"),
					AddressLine1: addr.GetString("address_line_1"),
					AddressLine2: addr.GetString("address_line_2"),
					City:         addr.GetString("city"),
					State:        addr.GetString("state"),
					PinCode:      addr.GetString("pin_code"),
					GSTIN:        addr.GetString("gstin"),
					ContactName:  addr.GetString("contact_name"),
					Phone:        addr.GetString("phone"),
				}
			} else {
				log.Printf("po_view: could not find ship_to address %s: %v", shipToID, err)
			}
		}

		// 6. Fetch line items sorted by sort_order
		lineItemRecords, err := app.FindRecordsByFilter(
			"po_line_items",
			"purchase_order = {:poId}",
			"sort_order",
			0,
			0,
			map[string]any{"poId": id},
		)
		if err != nil {
			log.Printf("po_view: could not fetch line items for PO %s: %v", id, err)
			lineItemRecords = nil
		}

		// 7. Build view line items and accumulate calc items for totals
		var lineItems []templates.POViewLineItem
		var calcItems []services.POLineItemCalc

		for _, item := range lineItemRecords {
			rate := item.GetFloat("rate")
			qty := item.GetFloat("qty")
			gstPercent := item.GetFloat("gst_percent")
			calc := services.CalcPOLineItem(rate, qty, gstPercent)
			calcItems = append(calcItems, calc)
			lineItems = append(lineItems, templates.POViewLineItem{
				SortOrder:   item.GetInt("sort_order"),
				Description: item.GetString("description"),
				HSNCode:     item.GetString("hsn_code"),
				Qty:         fmt.Sprintf("%.2f", qty),
				UoM:         item.GetString("uom"),
				Rate:        services.FormatINR(rate),
				BeforeGST:   services.FormatINR(calc.BeforeGST),
				GSTPercent:  fmt.Sprintf("%.2f", gstPercent),
				GSTAmount:   services.FormatINR(calc.GSTAmount),
				Total:       services.FormatINR(calc.Total),
			})
		}

		// 8. Calculate order-level totals
		totals := services.CalcPOTotals(calcItems)

		// 9. Format date fields for display
		orderDate := po.GetString("order_date")
		refDate := po.GetString("ref_date")

		// 10. Build the view data struct
		data := templates.POViewData{
			ProjectID:      projectId,
			POID:           po.Id,
			PONumber:       po.GetString("po_number"),
			Status:         po.GetString("status"),
			OrderDate:      orderDate,
			QuotationRef:   po.GetString("quotation_ref"),
			RefDate:        refDate,
			PaymentTerms:   po.GetString("payment_terms"),
			DeliveryTerms:  po.GetString("delivery_terms"),
			WarrantyTerms:  po.GetString("warranty_terms"),
			Comments:       po.GetString("comments"),
			Vendor:         vendor,
			BillTo:         billTo,
			ShipTo:         shipTo,
			LineItems:      lineItems,
			TotalBeforeTax: services.FormatINR(totals.TotalBeforeTax),
			IGSTPercent:    fmt.Sprintf("%.2f", totals.IGSTPercent),
			IGSTAmount:     services.FormatINR(totals.IGSTAmount),
			RoundOff:       services.FormatINR(totals.RoundOff),
			GrandTotal:     services.FormatINR(totals.GrandTotal),
			AmountInWords:  services.AmountToWords(totals.GrandTotal),
		}

		// 11. HTMX detection: render partial or full page
		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.POViewContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.POViewPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}
