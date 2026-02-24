package handlers

import (
	"fmt"
	"log"
	"math"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// formatQty formats a quantity value: whole numbers without decimals, others with 2 decimals.
func formatQty(val float64) string {
	if val == math.Trunc(val) {
		return fmt.Sprintf("%.0f", val)
	}
	return fmt.Sprintf("%.2f", val)
}

// HandleBOQView returns a handler that renders the BOQ detail/view page.
func HandleBOQView(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		boqID := e.Request.PathValue("id")
		if boqID == "" {
			return e.String(400, "Missing BOQ ID")
		}

		// 1. Fetch the BOQ record
		boqRecord, err := app.FindRecordById("boqs", boqID)
		if err != nil {
			log.Printf("boq_view: could not find BOQ %s: %v", boqID, err)
			return e.String(404, "BOQ not found")
		}

		// 2. Fetch main_boq_items for this BOQ
		mainItemsCol, err := app.FindCollectionByNameOrId("main_boq_items")
		if err != nil {
			log.Printf("boq_view: could not find main_boq_items collection: %v", err)
			return e.String(500, "Internal error")
		}

		mainItemRecords, err := app.FindRecordsByFilter(mainItemsCol, "boq = {:boqId}", "sort_order", 0, 0, map[string]any{"boqId": boqID})
		if err != nil {
			log.Printf("boq_view: could not query main items for BOQ %s: %v", boqID, err)
			mainItemRecords = nil
		}

		// 3. Fetch sub_items and sub_sub_items collections once
		subItemsCol, err := app.FindCollectionByNameOrId("sub_items")
		if err != nil {
			log.Printf("boq_view: could not find sub_items collection: %v", err)
			return e.String(500, "Internal error")
		}

		subSubItemsCol, err := app.FindCollectionByNameOrId("sub_sub_items")
		if err != nil {
			log.Printf("boq_view: could not find sub_sub_items collection: %v", err)
			return e.String(500, "Internal error")
		}

		// 4. Build main item views and compute totals
		var totalQuoted, totalBudgeted float64
		var mainItemViews []templates.MainItemView

		for i, mi := range mainItemRecords {
			qty := mi.GetFloat("qty")
			quotedPrice := mi.GetFloat("quoted_price")
			budgetedPrice := mi.GetFloat("budgeted_price")

			totalQuoted += qty * quotedPrice
			totalBudgeted += budgetedPrice

			// Compute budgeted per unit for display on the main item row
			budgetedPerUnit := budgetedPrice
			if qty != 0 {
				budgetedPerUnit = budgetedPrice / qty
			}

			// Fetch sub_items for this main item
			subItemRecords, err := app.FindRecordsByFilter(subItemsCol, "main_item = {:mainItemId}", "sort_order", 0, 0, map[string]any{"mainItemId": mi.Id})
			if err != nil {
				log.Printf("boq_view: could not query sub items for main item %s: %v", mi.Id, err)
				subItemRecords = nil
			}

			var subItemViews []templates.SubItemView
			for _, si := range subItemRecords {
				// Fetch sub_sub_items for this sub item
				subSubItemRecords, err := app.FindRecordsByFilter(subSubItemsCol, "sub_item = {:subItemId}", "sort_order", 0, 0, map[string]any{"subItemId": si.Id})
				if err != nil {
					log.Printf("boq_view: could not query sub-sub items for sub item %s: %v", si.Id, err)
					subSubItemRecords = nil
				}

				var subSubItemViews []templates.SubSubItemView
				for _, ssi := range subSubItemRecords {
					subSubItemViews = append(subSubItemViews, templates.SubSubItemView{
						ID:            ssi.Id,
						Description:   ssi.GetString("description"),
						Type:          ssi.GetString("type"),
						QtyPerUnit:    formatQty(ssi.GetFloat("qty_per_unit")),
						UOM:           ssi.GetString("uom"),
						UnitPrice:     services.FormatINR(ssi.GetFloat("unit_price")),
						BudgetedPrice: services.FormatINR(ssi.GetFloat("budgeted_price")),
						HSNCode:       ssi.GetString("hsn_code"),
						GSTPercent:    fmt.Sprintf("%.0f%%", ssi.GetFloat("gst_percent")),
					})
				}

				subItemViews = append(subItemViews, templates.SubItemView{
					ID:            si.Id,
					Description:   si.GetString("description"),
					Type:          si.GetString("type"),
					QtyPerUnit:    formatQty(si.GetFloat("qty_per_unit")),
					UOM:           si.GetString("uom"),
					UnitPrice:     services.FormatINR(si.GetFloat("unit_price")),
					BudgetedPrice: services.FormatINR(si.GetFloat("budgeted_price")),
					HSNCode:       si.GetString("hsn_code"),
					GSTPercent:    fmt.Sprintf("%.0f%%", si.GetFloat("gst_percent")),
					SubSubItems:   subSubItemViews,
				})
			}

			mainItemViews = append(mainItemViews, templates.MainItemView{
				ID:            mi.Id,
				Index:         i + 1,
				Description:   mi.GetString("description"),
				Qty:           formatQty(qty),
				UOM:           mi.GetString("uom"),
				QuotedPrice:   services.FormatINR(quotedPrice),
				BudgetedPrice: services.FormatINR(budgetedPerUnit),
				HSNCode:       mi.GetString("hsn_code"),
				GSTPercent:    fmt.Sprintf("%.0f%%", mi.GetFloat("gst_percent")),
				SubItems:      subItemViews,
			})
		}

		// 5. Compute margin
		margin := totalQuoted - totalBudgeted
		var marginPercent string
		if totalQuoted != 0 {
			marginPercent = fmt.Sprintf("%.1f%%", (margin/totalQuoted)*100)
		} else {
			marginPercent = "0.0%"
		}

		// 6. Format created date
		createdDate := "—"
		if dt := boqRecord.GetDateTime("created"); !dt.IsZero() {
			createdDate = dt.Time().Format("02 Jan 2006")
		}

		// 7. Build view data
		data := templates.BOQViewData{
			ProjectID:        projectID,
			ID:               boqRecord.Id,
			Title:            boqRecord.GetString("title"),
			ReferenceNumber:  boqRecord.GetString("reference_number"),
			CreatedDate:      createdDate,
			MainItems:        mainItemViews,
			TotalQuoted:      services.FormatINR(totalQuoted),
			TotalBudgeted:    services.FormatINR(totalBudgeted),
			Margin:           services.FormatINR(margin),
			MarginPercent:    marginPercent,
			IsPositiveMargin: margin >= 0,
		}

		// 8. Render
		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.BOQViewContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.BOQViewPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}
