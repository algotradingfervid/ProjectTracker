package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// buildPOEditData fetches all data needed to render the PO edit page.
func buildPOEditData(app *pocketbase.PocketBase, projectID, id string) (templates.POEditData, error) {
	record, err := app.FindRecordById("purchase_orders", id)
	if err != nil {
		return templates.POEditData{}, fmt.Errorf("purchase order not found: %w", err)
	}

	if record.GetString("project") != projectID {
		return templates.POEditData{}, fmt.Errorf("purchase order does not belong to project")
	}

	// Fetch vendor name
	vendorName := ""
	vendorID := record.GetString("vendor")
	if vendorID != "" {
		if vendor, err := app.FindRecordById("vendors", vendorID); err == nil {
			vendorName = vendor.GetString("name")
		} else {
			log.Printf("po_edit: could not find vendor %s: %v", vendorID, err)
		}
	}

	// Fetch line items
	lineItemRecords, err := app.FindRecordsByFilter(
		"po_line_items",
		"purchase_order = {:poId}",
		"sort_order",
		0, 0,
		map[string]any{"poId": id},
	)
	if err != nil {
		log.Printf("po_edit: could not fetch line items for PO %s: %v", id, err)
		lineItemRecords = nil
	}
	var lineItems []templates.POLineItemDisplay
	var calcItems []services.POLineItemCalc

	for _, item := range lineItemRecords {
		rate := item.GetFloat("rate")
		qty := item.GetFloat("qty")
		gstPercent := item.GetFloat("gst_percent")
		calc := services.CalcPOLineItem(rate, qty, gstPercent)
		calcItems = append(calcItems, calc)
		lineItems = append(lineItems, templates.POLineItemDisplay{
			ID:          item.Id,
			SortOrder:   item.GetInt("sort_order"),
			Description: item.GetString("description"),
			HSNCode:     item.GetString("hsn_code"),
			Qty:         qty,
			UoM:         item.GetString("uom"),
			Rate:        rate,
			BeforeGST:   calc.BeforeGST,
			GSTPercent:  gstPercent,
			GSTAmount:   calc.GSTAmount,
			Total:       calc.Total,
			SourceType:  item.GetString("source_item_type"),
		})
	}

	totals := services.CalcPOTotals(calcItems)

	// Fetch address options
	billToAddresses := fetchAddressesByType(app, projectID, "bill_from")
	shipToAddresses := fetchAddressesByType(app, projectID, "ship_from")

	data := templates.POEditData{
		ProjectID:       projectID,
		POID:            id,
		PONumber:        record.GetString("po_number"),
		Status:          record.GetString("status"),
		VendorID:        vendorID,
		VendorName:      vendorName,
		OrderDate:       record.GetString("order_date"),
		QuotationRef:    record.GetString("quotation_ref"),
		RefDate:         record.GetString("ref_date"),
		BillToAddressID: record.GetString("bill_to_address"),
		ShipToAddressID: record.GetString("ship_to_address"),
		BillToAddresses: billToAddresses,
		ShipToAddresses: shipToAddresses,
		PaymentTerms:    record.GetString("payment_terms"),
		DeliveryTerms:   record.GetString("delivery_terms"),
		WarrantyTerms:   record.GetString("warranty_terms"),
		Comments:        record.GetString("comments"),
		LineItems:       lineItems,
		TotalBeforeTax:  services.FormatINR(totals.TotalBeforeTax),
		IGSTPercent:     fmt.Sprintf("%.2f", totals.IGSTPercent),
		IGSTAmount:      services.FormatINR(totals.IGSTAmount),
		RoundOff:        services.FormatINR(totals.RoundOff),
		GrandTotal:      services.FormatINR(totals.GrandTotal),
		AmountInWords:   services.AmountToWords(totals.GrandTotal),
		Errors:          make(map[string]string),
	}

	return data, nil
}

// renderPOEdit renders either the partial content or full page depending on HX-Request header.
func renderPOEdit(e *core.RequestEvent, data templates.POEditData) error {
	var component templ.Component
	if e.Request.Header.Get("HX-Request") == "true" {
		component = templates.POEditContent(data)
	} else {
		headerData := GetHeaderData(e.Request)
		sidebarData := GetSidebarData(e.Request)
		component = templates.POEditPage(data, headerData, sidebarData)
	}
	return component.Render(e.Request.Context(), e.Response)
}

// HandlePOEdit returns a handler that renders the PO edit form with all data populated.
func HandlePOEdit(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")

		data, err := buildPOEditData(app, projectID, id)
		if err != nil {
			log.Printf("po_edit: %v", err)
			return ErrorToast(e, http.StatusNotFound, "Purchase order not found")
		}

		return renderPOEdit(e, data)
	}
}

// isValidStatusTransition returns true when transitioning from currentStatus to newStatus is allowed.
// Valid transitions:
//   - draft → sent
//   - sent → acknowledged
//   - acknowledged → completed
//   - any → cancelled
//
// Reversing back to "draft" from any other status is not permitted.
func isValidStatusTransition(currentStatus, newStatus string) bool {
	if newStatus == "cancelled" {
		return true
	}
	switch currentStatus {
	case "draft":
		return newStatus == "sent"
	case "sent":
		return newStatus == "acknowledged"
	case "acknowledged":
		return newStatus == "completed"
	default:
		return false
	}
}

// HandlePOUpdate returns a handler that saves edits to an existing purchase order.
func HandlePOUpdate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		record, err := app.FindRecordById("purchase_orders", id)
		if err != nil {
			log.Printf("po_update: could not find PO %s: %v", id, err)
			return ErrorToast(e, http.StatusNotFound, "Purchase order not found")
		}

		// Verify ownership
		if record.GetString("project") != projectID {
			return ErrorToast(e, http.StatusNotFound, "Purchase order not found")
		}

		// Extract and trim form values
		orderDate := strings.TrimSpace(e.Request.FormValue("order_date"))
		quotationRef := strings.TrimSpace(e.Request.FormValue("quotation_ref"))
		refDate := strings.TrimSpace(e.Request.FormValue("ref_date"))
		billToAddressID := strings.TrimSpace(e.Request.FormValue("bill_to_address_id"))
		shipToAddressID := strings.TrimSpace(e.Request.FormValue("ship_to_address_id"))
		paymentTerms := strings.TrimSpace(e.Request.FormValue("payment_terms"))
		deliveryTerms := strings.TrimSpace(e.Request.FormValue("delivery_terms"))
		warrantyTerms := strings.TrimSpace(e.Request.FormValue("warranty_terms"))
		comments := strings.TrimSpace(e.Request.FormValue("comments"))
		newStatus := strings.TrimSpace(e.Request.FormValue("new_status"))

		errors := make(map[string]string)

		// Status transition validation
		currentStatus := record.GetString("status")
		if newStatus != "" {
			if newStatus == "draft" && currentStatus != "draft" {
				errors["status"] = "Cannot change status back to draft"
			} else if newStatus != "draft" && !isValidStatusTransition(currentStatus, newStatus) {
				errors["status"] = fmt.Sprintf("Invalid status transition from %s to %s", currentStatus, newStatus)
			}
		}

		if len(errors) > 0 {
			SetToast(e, "warning", "Please fix the errors below")
			data, buildErr := buildPOEditData(app, projectID, id)
			if buildErr != nil {
				log.Printf("po_update: could not rebuild data for PO %s: %v", id, buildErr)
				return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
			}
			data.Errors = errors
			// Restore submitted form values so the user doesn't lose their edits
			data.OrderDate = orderDate
			data.QuotationRef = quotationRef
			data.RefDate = refDate
			data.BillToAddressID = billToAddressID
			data.ShipToAddressID = shipToAddressID
			data.PaymentTerms = paymentTerms
			data.DeliveryTerms = deliveryTerms
			data.WarrantyTerms = warrantyTerms
			data.Comments = comments
			return renderPOEdit(e, data)
		}

		// Update record fields (vendor and po_number are immutable)
		record.Set("order_date", orderDate)
		record.Set("quotation_ref", quotationRef)
		record.Set("ref_date", refDate)
		record.Set("bill_to_address", billToAddressID)
		record.Set("ship_to_address", shipToAddressID)
		record.Set("payment_terms", paymentTerms)
		record.Set("delivery_terms", deliveryTerms)
		record.Set("warranty_terms", warrantyTerms)
		record.Set("comments", comments)

		if newStatus != "" {
			record.Set("status", newStatus)
		}

		if err := app.Save(record); err != nil {
			log.Printf("po_update: could not save PO %s: %v", id, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		redirectURL := fmt.Sprintf("/projects/%s/po/%s/edit", projectID, id)

		SetToast(e, "success", "Purchase order updated")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}
