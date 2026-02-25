package handlers

import (
	"log"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

func HandlePOList(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		statusFilter := e.Request.URL.Query().Get("status")

		var (
			filter string
			params map[string]any
		)

		if statusFilter != "" {
			filter = "project = {:projectId} && status = {:status}"
			params = map[string]any{
				"projectId": projectId,
				"status":    statusFilter,
			}
		} else {
			filter = "project = {:projectId}"
			params = map[string]any{
				"projectId": projectId,
			}
		}

		poRecords, err := app.FindRecordsByFilter(
			"purchase_orders",
			filter,
			"-created",
			0,
			0,
			params,
		)
		if err != nil {
			log.Printf("po_list: could not query purchase_orders: %v", err)
			poRecords = nil
		}

		var poItems []templates.POListItem
		for _, po := range poRecords {
			// Resolve vendor name
			vendorName := "Unknown Vendor"
			vendorID := po.GetString("vendor")
			if vendorID != "" {
				vendor, err := app.FindRecordById("vendors", vendorID)
				if err != nil {
					log.Printf("po_list: could not find vendor %s: %v", vendorID, err)
				} else {
					vendorName = vendor.GetString("name")
				}
			}

			// Fetch line items for count and grand total calculation
			lineItems, err := app.FindRecordsByFilter(
				"po_line_items",
				"purchase_order = {:poId}",
				"",
				0,
				0,
				map[string]any{"poId": po.Id},
			)
			if err != nil {
				log.Printf("po_list: could not query po_line_items for PO %s: %v", po.Id, err)
				lineItems = nil
			}

			var calcItems []services.POLineItemCalc
			for _, item := range lineItems {
				calc := services.CalcPOLineItem(
					item.GetFloat("rate"),
					item.GetFloat("qty"),
					item.GetFloat("gst_percent"),
				)
				calcItems = append(calcItems, calc)
			}

			totals := services.CalcPOTotals(calcItems)
			grandTotal := services.FormatINR(totals.GrandTotal)

			poItems = append(poItems, templates.POListItem{
				ID:            po.Id,
				PONumber:      po.GetString("po_number"),
				VendorName:    vendorName,
				OrderDate:     po.GetString("order_date"),
				Status:        po.GetString("status"),
				LineItemCount: len(lineItems),
				GrandTotal:    grandTotal,
			})
		}

		data := templates.POListData{
			POs:          poItems,
			ProjectID:    projectId,
			StatusFilter: statusFilter,
			TotalCount:   len(poItems),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.POListContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.POListPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}
