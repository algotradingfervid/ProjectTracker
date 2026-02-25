package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// fetchLinkedVendors returns vendor select items for a project by querying
// project_vendors and then resolving each vendor record.
func fetchLinkedVendors(app *pocketbase.PocketBase, projectId string) []templates.VendorSelectItem {
	links, err := app.FindRecordsByFilter(
		"project_vendors",
		"project = {:projectId}",
		"",
		0,
		0,
		map[string]any{"projectId": projectId},
	)
	if err != nil {
		log.Printf("po_create: fetchLinkedVendors: could not query project_vendors: %v", err)
		return nil
	}

	var items []templates.VendorSelectItem
	for _, link := range links {
		vendorID := link.GetString("vendor")
		vendor, err := app.FindRecordById("vendors", vendorID)
		if err != nil {
			log.Printf("po_create: fetchLinkedVendors: could not find vendor %s: %v", vendorID, err)
			continue
		}
		items = append(items, templates.VendorSelectItem{
			ID:   vendor.Id,
			Name: vendor.GetString("name"),
		})
	}
	return items
}

// fetchAddressesByType returns address select items for a project filtered by address_type.
func fetchAddressesByType(app *pocketbase.PocketBase, projectId, addressType string) []templates.AddressSelectItem {
	records, err := app.FindRecordsByFilter(
		"addresses",
		"project = {:projectId} && address_type = {:addressType}",
		"",
		0,
		0,
		map[string]any{
			"projectId":   projectId,
			"addressType": addressType,
		},
	)
	if err != nil {
		log.Printf("po_create: fetchAddressesByType: could not query addresses (type=%s): %v", addressType, err)
		return nil
	}

	var items []templates.AddressSelectItem
	for _, rec := range records {
		items = append(items, templates.AddressSelectItem{
			ID:           rec.Id,
			CompanyName:  rec.GetString("company_name"),
			AddressLine1: rec.GetString("address_line_1"),
			City:         rec.GetString("city"),
			State:        rec.GetString("state"),
		})
	}
	return items
}

func HandlePOCreate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")

		_, err := app.FindRecordById("projects", projectId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Project not found")
		}

		vendors := fetchLinkedVendors(app, projectId)
		billToAddresses := fetchAddressesByType(app, projectId, "bill_from")
		shipToAddresses := fetchAddressesByType(app, projectId, "ship_from")

		data := templates.POCreateData{
			ProjectID:       projectId,
			Vendors:         vendors,
			BillToAddresses: billToAddresses,
			ShipToAddresses: shipToAddresses,
			Errors:          make(map[string]string),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.POCreateContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.POCreatePage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

func HandlePOSave(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")

		vendorID := strings.TrimSpace(e.Request.FormValue("vendor_id"))
		orderDate := strings.TrimSpace(e.Request.FormValue("order_date"))
		quotationRef := strings.TrimSpace(e.Request.FormValue("quotation_ref"))
		refDate := strings.TrimSpace(e.Request.FormValue("ref_date"))
		billToAddressID := strings.TrimSpace(e.Request.FormValue("bill_to_address_id"))
		shipToAddressID := strings.TrimSpace(e.Request.FormValue("ship_to_address_id"))
		paymentTerms := strings.TrimSpace(e.Request.FormValue("payment_terms"))
		deliveryTerms := strings.TrimSpace(e.Request.FormValue("delivery_terms"))
		warrantyTerms := strings.TrimSpace(e.Request.FormValue("warranty_terms"))
		comments := strings.TrimSpace(e.Request.FormValue("comments"))

		errors := make(map[string]string)
		if vendorID == "" {
			errors["vendor_id"] = "Vendor is required"
		}

		if len(errors) > 0 {
			SetToast(e, "warning", "Please fix the errors below")
			vendors := fetchLinkedVendors(app, projectId)
			billToAddresses := fetchAddressesByType(app, projectId, "bill_from")
			shipToAddresses := fetchAddressesByType(app, projectId, "ship_from")

			data := templates.POCreateData{
				ProjectID:       projectId,
				Vendors:         vendors,
				BillToAddresses: billToAddresses,
				ShipToAddresses: shipToAddresses,
				VendorID:        vendorID,
				OrderDate:       orderDate,
				QuotationRef:    quotationRef,
				RefDate:         refDate,
				BillToAddressID: billToAddressID,
				ShipToAddressID: shipToAddressID,
				PaymentTerms:    paymentTerms,
				DeliveryTerms:   deliveryTerms,
				WarrantyTerms:   warrantyTerms,
				Comments:        comments,
				Errors:          errors,
			}

			var component templ.Component
			if e.Request.Header.Get("HX-Request") == "true" {
				component = templates.POCreateContent(data)
			} else {
				headerData := GetHeaderData(e.Request)
				sidebarData := GetSidebarData(e.Request)
				component = templates.POCreatePage(data, headerData, sidebarData)
			}
			return component.Render(e.Request.Context(), e.Response)
		}

		poNumber, err := services.GeneratePONumber(app, projectId, time.Now())
		if err != nil {
			log.Printf("po_create: could not generate PO number: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		col, err := app.FindCollectionByNameOrId("purchase_orders")
		if err != nil {
			log.Printf("po_create: could not find purchase_orders collection: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		record := core.NewRecord(col)
		record.Set("project", projectId)
		record.Set("vendor", vendorID)
		record.Set("po_number", poNumber)
		record.Set("status", "draft")
		record.Set("order_date", orderDate)
		record.Set("quotation_ref", quotationRef)
		record.Set("ref_date", refDate)
		record.Set("bill_to_address", billToAddressID)
		record.Set("ship_to_address", shipToAddressID)
		record.Set("payment_terms", paymentTerms)
		record.Set("delivery_terms", deliveryTerms)
		record.Set("warranty_terms", warrantyTerms)
		record.Set("comments", comments)

		if err := app.Save(record); err != nil {
			log.Printf("po_create: could not save purchase order: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		redirectURL := fmt.Sprintf("/projects/%s/po/%s/edit", projectId, record.Id)

		SetToast(e, "success", "Purchase order created")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}
