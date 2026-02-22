package handlers

import (
	"fmt"
	"log"
	"strconv"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// buildBOQEditData fetches all data for the BOQ edit page.
// openMainItemIDs and openSubItemIDs control which accordions render open.
func buildBOQEditData(app *pocketbase.PocketBase, boqID string, openMainItemIDs, openSubItemIDs map[string]bool) (templates.BOQEditData, error) {
	boqRecord, err := app.FindRecordById("boqs", boqID)
	if err != nil {
		return templates.BOQEditData{}, fmt.Errorf("BOQ not found: %w", err)
	}

	mainItemsCol, err := app.FindCollectionByNameOrId("main_boq_items")
	if err != nil {
		return templates.BOQEditData{}, fmt.Errorf("collection error: %w", err)
	}

	mainItemRecords, err := app.FindRecordsByFilter(mainItemsCol, "boq = {:boqId}", "sort_order", 0, 0, map[string]any{"boqId": boqID})
	if err != nil {
		log.Printf("boq_edit: could not query main items for BOQ %s: %v", boqID, err)
		mainItemRecords = nil
	}

	subItemsCol, err := app.FindCollectionByNameOrId("sub_items")
	if err != nil {
		return templates.BOQEditData{}, fmt.Errorf("collection error: %w", err)
	}

	subSubItemsCol, err := app.FindCollectionByNameOrId("sub_sub_items")
	if err != nil {
		return templates.BOQEditData{}, fmt.Errorf("collection error: %w", err)
	}

	var totalQuoted, totalBudgeted float64
	var mainItemEdits []templates.MainItemEdit

	for i, mi := range mainItemRecords {
		qty := mi.GetFloat("qty")
		quotedPrice := mi.GetFloat("quoted_price")
		budgetedPrice := mi.GetFloat("budgeted_price")

		totalQuoted += qty * quotedPrice
		totalBudgeted += budgetedPrice

		// Compute budgeted per unit for display
		budgetedPerUnit := budgetedPrice
		if qty != 0 {
			budgetedPerUnit = budgetedPrice / qty
		}

		subItemRecords, err := app.FindRecordsByFilter(subItemsCol, "main_item = {:mainItemId}", "sort_order", 0, 0, map[string]any{"mainItemId": mi.Id})
		if err != nil {
			log.Printf("boq_edit: could not query sub items for main item %s: %v", mi.Id, err)
			subItemRecords = nil
		}

		var subItemEdits []templates.SubItemEdit
		for _, si := range subItemRecords {
			subSubItemRecords, err := app.FindRecordsByFilter(subSubItemsCol, "sub_item = {:subItemId}", "sort_order", 0, 0, map[string]any{"subItemId": si.Id})
			if err != nil {
				log.Printf("boq_edit: could not query sub-sub items for sub item %s: %v", si.Id, err)
				subSubItemRecords = nil
			}

			var subSubItemEdits []templates.SubSubItemEdit
			for _, ssi := range subSubItemRecords {
				subSubItemEdits = append(subSubItemEdits, templates.SubSubItemEdit{
					ID:            ssi.Id,
					Type:          ssi.GetString("type"),
					Description:   ssi.GetString("description"),
					UOM:           ssi.GetString("uom"),
					HSNCode:       ssi.GetString("hsn_code"),
					QtyPerUnit:    ssi.GetFloat("qty_per_unit"),
					UnitPrice:     ssi.GetFloat("unit_price"),
					BudgetedPrice: ssi.GetFloat("budgeted_price"),
					GSTPercent:    ssi.GetFloat("gst_percent"),
				})
			}

			subItemEdits = append(subItemEdits, templates.SubItemEdit{
				ID:            si.Id,
				Type:          si.GetString("type"),
				Description:   si.GetString("description"),
				UOM:           si.GetString("uom"),
				HSNCode:       si.GetString("hsn_code"),
				QtyPerUnit:    si.GetFloat("qty_per_unit"),
				UnitPrice:     si.GetFloat("unit_price"),
				BudgetedPrice: si.GetFloat("budgeted_price"),
				GSTPercent:    si.GetFloat("gst_percent"),
				SubSubItems:   subSubItemEdits,
			})
		}

		mainItemEdits = append(mainItemEdits, templates.MainItemEdit{
			ID:            mi.Id,
			Index:         i + 1,
			Description:   mi.GetString("description"),
			Qty:           qty,
			UOM:           mi.GetString("uom"),
			QuotedPrice:   quotedPrice,
			BudgetedPrice: budgetedPerUnit,
			HSNCode:       mi.GetString("hsn_code"),
			GSTPercent:    mi.GetFloat("gst_percent"),
			SubItems:      subItemEdits,
		})
	}

	margin := totalQuoted - totalBudgeted
	var marginPercent string
	if totalQuoted != 0 {
		marginPercent = fmt.Sprintf("%.1f%%", (margin/totalQuoted)*100)
	} else {
		marginPercent = "0.0%"
	}

	createdDate := "—"
	if dt := boqRecord.GetDateTime("created"); !dt.IsZero() {
		createdDate = dt.Time().Format("02 Jan 2006")
	}

	return templates.BOQEditData{
		ID:               boqRecord.Id,
		Title:            boqRecord.GetString("title"),
		ReferenceNumber:  boqRecord.GetString("reference_number"),
		CreatedDate:      createdDate,
		MainItems:        mainItemEdits,
		TotalQuoted:      services.FormatINR(totalQuoted),
		TotalBudgeted:    services.FormatINR(totalBudgeted),
		Margin:           services.FormatINR(margin),
		MarginPercent:    marginPercent,
		IsPositiveMargin: margin >= 0,
		UOMOptions:       services.UOMOptions,
		GSTOptions:       services.GSTOptions,
		OpenMainItemIDs:  openMainItemIDs,
		OpenSubItemIDs:   openSubItemIDs,
	}, nil
}

// renderBOQEdit renders the edit page, choosing partial or full page based on HX-Request header.
func renderBOQEdit(e *core.RequestEvent, data templates.BOQEditData) error {
	var component templ.Component
	if e.Request.Header.Get("HX-Request") == "true" {
		component = templates.BOQEditContent(data)
	} else {
		component = templates.BOQEditPage(data)
	}
	return component.Render(e.Request.Context(), e.Response)
}

// HandleBOQEdit returns a handler that renders the BOQ edit form with all data populated.
func HandleBOQEdit(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		boqID := e.Request.PathValue("id")
		if boqID == "" {
			return e.String(400, "Missing BOQ ID")
		}

		data, err := buildBOQEditData(app, boqID, nil, nil)
		if err != nil {
			log.Printf("boq_edit: %v", err)
			return e.String(500, "Internal error")
		}

		return renderBOQEdit(e, data)
	}
}

// HandleBOQViewMode returns a handler that renders the BOQ in view mode.
// This is used when canceling out of edit mode via HTMX.
// It delegates to HandleBOQView which already handles both full-page and HTMX partial rendering.
func HandleBOQViewMode(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return HandleBOQView(app)
}

// HandleBOQUpdate returns a handler that batch-saves all BOQ edit changes from form data,
// recalculates budgeted prices, and returns the BOQ view content.
func HandleBOQUpdate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		boqID := e.Request.PathValue("id")
		if boqID == "" {
			return e.String(400, "Missing BOQ ID")
		}

		if err := e.Request.ParseForm(); err != nil {
			return e.String(400, "Invalid form data")
		}

		// Fetch collections
		mainItemsCol, err := app.FindCollectionByNameOrId("main_boq_items")
		if err != nil {
			log.Printf("boq_save: could not find main_boq_items collection: %v", err)
			return e.String(500, "Internal error")
		}

		subItemsCol, err := app.FindCollectionByNameOrId("sub_items")
		if err != nil {
			log.Printf("boq_save: could not find sub_items collection: %v", err)
			return e.String(500, "Internal error")
		}

		subSubItemsCol, err := app.FindCollectionByNameOrId("sub_sub_items")
		if err != nil {
			log.Printf("boq_save: could not find sub_sub_items collection: %v", err)
			return e.String(500, "Internal error")
		}

		// Fetch existing main item records
		mainItems, err := app.FindRecordsByFilter(mainItemsCol, "boq = {:boqId}", "sort_order", 0, 0, map[string]any{"boqId": boqID})
		if err != nil {
			log.Printf("boq_save: could not query main items for BOQ %s: %v", boqID, err)
			return e.String(500, "Internal error")
		}

		for _, mi := range mainItems {
			prefix := "main_item_" + mi.Id + "_"

			if desc := e.Request.FormValue(prefix + "description"); desc != "" {
				mi.Set("description", desc)
			}
			if qtyStr := e.Request.FormValue(prefix + "qty"); qtyStr != "" {
				if qty, err := strconv.ParseFloat(qtyStr, 64); err == nil {
					mi.Set("qty", qty)
				}
			}
			if uom := e.Request.FormValue(prefix + "uom"); uom != "" {
				mi.Set("uom", uom)
			}
			if qpStr := e.Request.FormValue(prefix + "quoted_price"); qpStr != "" {
				if qp, err := strconv.ParseFloat(qpStr, 64); err == nil {
					mi.Set("quoted_price", qp)
				}
			}
			mi.Set("hsn_code", e.Request.FormValue(prefix+"hsn_code"))
			if gstStr := e.Request.FormValue(prefix + "gst_percent"); gstStr != "" {
				if gst, err := strconv.ParseFloat(gstStr, 64); err == nil {
					mi.Set("gst_percent", gst)
				}
			}

			// Update sub items for this main item
			subItems, err := app.FindRecordsByFilter(subItemsCol, "main_item = {:mainId}", "sort_order", 0, 0, map[string]any{"mainId": mi.Id})
			if err != nil {
				log.Printf("boq_save: could not query sub items for main item %s: %v", mi.Id, err)
				subItems = nil
			}

			var subBudgetTotal float64
			for _, si := range subItems {
				subPrefix := "sub_item_" + si.Id + "_"

				if t := e.Request.FormValue(subPrefix + "type"); t != "" {
					si.Set("type", t)
				}
				if desc := e.Request.FormValue(subPrefix + "description"); desc != "" {
					si.Set("description", desc)
				}
				if qStr := e.Request.FormValue(subPrefix + "qty_per_unit"); qStr != "" {
					if q, err := strconv.ParseFloat(qStr, 64); err == nil {
						si.Set("qty_per_unit", q)
					}
				}
				if uom := e.Request.FormValue(subPrefix + "uom"); uom != "" {
					si.Set("uom", uom)
				}
				if upStr := e.Request.FormValue(subPrefix + "unit_price"); upStr != "" {
					if up, err := strconv.ParseFloat(upStr, 64); err == nil {
						si.Set("unit_price", up)
					}
				}
				si.Set("hsn_code", e.Request.FormValue(subPrefix+"hsn_code"))
				if gstStr := e.Request.FormValue(subPrefix + "gst_percent"); gstStr != "" {
					if gst, err := strconv.ParseFloat(gstStr, 64); err == nil {
						si.Set("gst_percent", gst)
					}
				}

				// Update sub-sub items for this sub item
				subSubItems, err := app.FindRecordsByFilter(subSubItemsCol, "sub_item = {:subId}", "sort_order", 0, 0, map[string]any{"subId": si.Id})
				if err != nil {
					log.Printf("boq_save: could not query sub-sub items for sub item %s: %v", si.Id, err)
					subSubItems = nil
				}

				var ssTotal float64
				for _, ssi := range subSubItems {
					ssPrefix := "sub_sub_item_" + ssi.Id + "_"

					if t := e.Request.FormValue(ssPrefix + "type"); t != "" {
						ssi.Set("type", t)
					}
					if desc := e.Request.FormValue(ssPrefix + "description"); desc != "" {
						ssi.Set("description", desc)
					}
					if qStr := e.Request.FormValue(ssPrefix + "qty_per_unit"); qStr != "" {
						if q, err := strconv.ParseFloat(qStr, 64); err == nil {
							ssi.Set("qty_per_unit", q)
						}
					}
					if uom := e.Request.FormValue(ssPrefix + "uom"); uom != "" {
						ssi.Set("uom", uom)
					}
					if upStr := e.Request.FormValue(ssPrefix + "unit_price"); upStr != "" {
						if up, err := strconv.ParseFloat(upStr, 64); err == nil {
							ssi.Set("unit_price", up)
						}
					}
					ssi.Set("hsn_code", e.Request.FormValue(ssPrefix+"hsn_code"))
					if gstStr := e.Request.FormValue(ssPrefix + "gst_percent"); gstStr != "" {
						if gst, err := strconv.ParseFloat(gstStr, 64); err == nil {
							ssi.Set("gst_percent", gst)
						}
					}

					// Recalculate sub-sub budgeted price
					ssBudgeted := ssi.GetFloat("qty_per_unit") * ssi.GetFloat("unit_price")
					ssi.Set("budgeted_price", ssBudgeted)
					ssTotal += ssBudgeted

					if err := app.Save(ssi); err != nil {
						log.Printf("boq_save: error saving sub-sub item %s: %v", ssi.Id, err)
					}
				}

				// Recalculate sub budgeted price
				if len(subSubItems) > 0 {
					si.Set("budgeted_price", ssTotal)
				} else {
					subBudgeted := si.GetFloat("qty_per_unit") * si.GetFloat("unit_price")
					si.Set("budgeted_price", subBudgeted)
				}
				subBudgetTotal += si.GetFloat("budgeted_price")

				if err := app.Save(si); err != nil {
					log.Printf("boq_save: error saving sub item %s: %v", si.Id, err)
				}
			}

			// Recalculate main item budgeted price
			if len(subItems) > 0 {
				mi.Set("budgeted_price", subBudgetTotal*mi.GetFloat("qty"))
			}

			if err := app.Save(mi); err != nil {
				log.Printf("boq_save: error saving main item %s: %v", mi.Id, err)
			}
		}

		// After saving, rebuild view data and return the view content
		boqRecord, err := app.FindRecordById("boqs", boqID)
		if err != nil {
			log.Printf("boq_save: could not find BOQ %s after save: %v", boqID, err)
			return e.String(500, "Internal error")
		}

		// Re-fetch main items with updated data
		mainItemRecords, err := app.FindRecordsByFilter(mainItemsCol, "boq = {:boqId}", "sort_order", 0, 0, map[string]any{"boqId": boqID})
		if err != nil {
			log.Printf("boq_save: could not query main items for BOQ %s: %v", boqID, err)
			mainItemRecords = nil
		}

		var totalQuoted, totalBudgeted float64
		var mainItemViews []templates.MainItemView

		for i, mi := range mainItemRecords {
			qty := mi.GetFloat("qty")
			quotedPrice := mi.GetFloat("quoted_price")
			budgetedPrice := mi.GetFloat("budgeted_price")

			totalQuoted += qty * quotedPrice
			totalBudgeted += budgetedPrice

		// Compute budgeted per unit for display
		budgetedPerUnit := budgetedPrice
		if qty != 0 {
			budgetedPerUnit = budgetedPrice / qty
		}

			// Fetch sub_items for this main item
			subItemRecords, err := app.FindRecordsByFilter(subItemsCol, "main_item = {:mainItemId}", "sort_order", 0, 0, map[string]any{"mainItemId": mi.Id})
			if err != nil {
				log.Printf("boq_save: could not query sub items for main item %s: %v", mi.Id, err)
				subItemRecords = nil
			}

			var subItemViews []templates.SubItemView
			for _, si := range subItemRecords {
				// Fetch sub_sub_items for this sub item
				subSubItemRecords, err := app.FindRecordsByFilter(subSubItemsCol, "sub_item = {:subItemId}", "sort_order", 0, 0, map[string]any{"subItemId": si.Id})
				if err != nil {
					log.Printf("boq_save: could not query sub-sub items for sub item %s: %v", si.Id, err)
					subSubItemRecords = nil
				}

				var subSubItemViews []templates.SubSubItemView
				for _, ssi := range subSubItemRecords {
					subSubItemViews = append(subSubItemViews, templates.SubSubItemView{
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

		// Compute margin
		margin := totalQuoted - totalBudgeted
		var marginPercent string
		if totalQuoted != 0 {
			marginPercent = fmt.Sprintf("%.1f%%", (margin/totalQuoted)*100)
		} else {
			marginPercent = "0.0%"
		}

		// Format created date
		createdDate := "—"
		if dt := boqRecord.GetDateTime("created"); !dt.IsZero() {
			createdDate = dt.Time().Format("02 Jan 2006")
		}

		// Build view data
		viewData := templates.BOQViewData{
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

		// Set HX-Push-Url so the browser URL updates back to the view URL
		e.Response.Header().Set("HX-Push-Url", "/boq/"+boqID)

		// Render the view content partial
		component := templates.BOQViewContent(viewData)
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleAddMainItem creates a new empty main item for a BOQ and re-renders the edit page.
func HandleAddMainItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		boqID := e.Request.PathValue("id")
		if boqID == "" {
			return e.String(400, "Missing BOQ ID")
		}

		mainItemsCol, err := app.FindCollectionByNameOrId("main_boq_items")
		if err != nil {
			return e.String(500, "Internal error")
		}
		existing, _ := app.FindRecordsByFilter(mainItemsCol, "boq = {:boqId}", "sort_order", 0, 0, map[string]any{"boqId": boqID})

		record := core.NewRecord(mainItemsCol)
		record.Set("boq", boqID)
		record.Set("sort_order", len(existing)+1)
		record.Set("description", "New Item")
		record.Set("qty", 1)
		record.Set("uom", "Nos")
		record.Set("quoted_price", 1)
		record.Set("budgeted_price", 0)
		record.Set("hsn_code", "")
		record.Set("gst_percent", 18)

		if err := app.Save(record); err != nil {
			log.Printf("add_main_item: error creating record: %v", err)
			return e.String(500, fmt.Sprintf("Failed to create item: %v", err))
		}

		// Re-render with the new main item's accordion open
		data, err := buildBOQEditData(app, boqID, map[string]bool{record.Id: true}, nil)
		if err != nil {
			log.Printf("add_main_item: %v", err)
			return e.String(500, "Internal error")
		}
		return renderBOQEdit(e, data)
	}
}

// HandleAddSubItem creates a new empty sub item under a main item and re-renders the edit page.
func HandleAddSubItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		boqID := e.Request.PathValue("id")
		mainItemID := e.Request.PathValue("mainItemId")
		if boqID == "" || mainItemID == "" {
			return e.String(400, "Missing IDs")
		}

		subItemsCol, err := app.FindCollectionByNameOrId("sub_items")
		if err != nil {
			return e.String(500, "Internal error")
		}
		existing, _ := app.FindRecordsByFilter(subItemsCol, "main_item = {:mainId}", "sort_order", 0, 0, map[string]any{"mainId": mainItemID})

		record := core.NewRecord(subItemsCol)
		record.Set("main_item", mainItemID)
		record.Set("sort_order", len(existing)+1)
		record.Set("type", "product")
		record.Set("description", "New Sub Item")
		record.Set("qty_per_unit", 1)
		record.Set("uom", "Nos")
		record.Set("unit_price", 1)
		record.Set("budgeted_price", 1)
		record.Set("hsn_code", "")
		record.Set("gst_percent", 18)

		if err := app.Save(record); err != nil {
			log.Printf("add_sub_item: error creating record: %v", err)
			return e.String(500, fmt.Sprintf("Failed to create sub item: %v", err))
		}

		// Re-render with the parent main item's accordion open
		data, err := buildBOQEditData(app, boqID, map[string]bool{mainItemID: true}, nil)
		if err != nil {
			log.Printf("add_sub_item: %v", err)
			return e.String(500, "Internal error")
		}
		return renderBOQEdit(e, data)
	}
}

// HandleAddSubSubItem creates a new empty sub-sub item under a sub item and re-renders the edit page.
func HandleAddSubSubItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		boqID := e.Request.PathValue("id")
		subItemID := e.Request.PathValue("subItemId")
		if boqID == "" || subItemID == "" {
			return e.String(400, "Missing IDs")
		}

		subSubItemsCol, err := app.FindCollectionByNameOrId("sub_sub_items")
		if err != nil {
			return e.String(500, "Internal error")
		}
		existing, _ := app.FindRecordsByFilter(subSubItemsCol, "sub_item = {:subId}", "sort_order", 0, 0, map[string]any{"subId": subItemID})

		record := core.NewRecord(subSubItemsCol)
		record.Set("sub_item", subItemID)
		record.Set("sort_order", len(existing)+1)
		record.Set("type", "product")
		record.Set("description", "New Sub-Sub Item")
		record.Set("qty_per_unit", 1)
		record.Set("uom", "Nos")
		record.Set("unit_price", 1)
		record.Set("budgeted_price", 1)
		record.Set("hsn_code", "")
		record.Set("gst_percent", 18)

		if err := app.Save(record); err != nil {
			log.Printf("add_sub_sub_item: error creating record: %v", err)
			return e.String(500, fmt.Sprintf("Failed to create sub-sub item: %v", err))
		}

		// Find the parent main item ID for this sub item
		subItemRecord, err := app.FindRecordById("sub_items", subItemID)
		if err != nil {
			log.Printf("add_sub_sub_item: could not find sub item %s: %v", subItemID, err)
			return e.String(500, "Internal error")
		}
		mainItemID := subItemRecord.GetString("main_item")

		// Re-render with both the parent main item and sub item accordions open
		data, err := buildBOQEditData(app, boqID, map[string]bool{mainItemID: true}, map[string]bool{subItemID: true})
		if err != nil {
			log.Printf("add_sub_sub_item: %v", err)
			return e.String(500, "Internal error")
		}
		return renderBOQEdit(e, data)
	}
}
