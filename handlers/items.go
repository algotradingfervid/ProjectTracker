package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// HandleDeleteMainItem deletes a main item (PocketBase cascade handles children)
// and re-renders the full edit page to refresh pricing state.
func HandleDeleteMainItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		boqID := e.Request.PathValue("id")
		itemID := e.Request.PathValue("itemId")
		if boqID == "" || itemID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing required IDs")
		}

		record, err := app.FindRecordById("main_boq_items", itemID)
		if err != nil {
			log.Printf("delete_main_item: not found %s: %v", itemID, err)
			return ErrorToast(e, http.StatusNotFound, "Item not found")
		}

		if err := app.Delete(record); err != nil {
			log.Printf("delete_main_item: error deleting %s: %v", itemID, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		SetToast(e, "success", "Item deleted")

		// Re-render the full edit page so Alpine pricing state is rebuilt
		data, err := buildBOQEditData(app, boqID, nil, nil)
		if err != nil {
			log.Printf("delete_main_item: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}
		data.ProjectID = projectID
		return renderBOQEdit(e, data)
	}
}

// HandleDeleteSubItem deletes a sub-item (cascade deletes sub-sub-items)
// and re-renders the edit page.
func HandleDeleteSubItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		boqID := e.Request.PathValue("id")
		subItemID := e.Request.PathValue("subItemId")
		if boqID == "" || subItemID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing required IDs")
		}

		// Find the parent main item before deleting so we can keep its accordion open
		subItemRecord, err := app.FindRecordById("sub_items", subItemID)
		if err != nil {
			log.Printf("delete_sub_item: not found %s: %v", subItemID, err)
			return ErrorToast(e, http.StatusNotFound, "Sub-item not found")
		}
		mainItemID := subItemRecord.GetString("main_item")

		if err := app.Delete(subItemRecord); err != nil {
			log.Printf("delete_sub_item: error deleting %s: %v", subItemID, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		// Recalculate parent main item budgeted price
		recalcMainItemBudgeted(app, mainItemID)

		SetToast(e, "success", "Sub-item deleted")

		data, err := buildBOQEditData(app, boqID, map[string]bool{mainItemID: true}, nil)
		if err != nil {
			log.Printf("delete_sub_item: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}
		data.ProjectID = projectID
		return renderBOQEdit(e, data)
	}
}

// HandleDeleteSubSubItem deletes a sub-sub-item and re-renders the edit page.
func HandleDeleteSubSubItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		boqID := e.Request.PathValue("id")
		subSubItemID := e.Request.PathValue("subSubItemId")
		if boqID == "" || subSubItemID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing required IDs")
		}

		record, err := app.FindRecordById("sub_sub_items", subSubItemID)
		if err != nil {
			log.Printf("delete_sub_sub_item: not found %s: %v", subSubItemID, err)
			return ErrorToast(e, http.StatusNotFound, "Sub-sub-item not found")
		}
		subItemID := record.GetString("sub_item")

		if err := app.Delete(record); err != nil {
			log.Printf("delete_sub_sub_item: error deleting %s: %v", subSubItemID, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		// Recalculate parent chain
		recalcSubItemBudgeted(app, subItemID)

		// Find the main item ID for keeping accordions open
		subItemRecord, err := app.FindRecordById("sub_items", subItemID)
		if err != nil {
			log.Printf("delete_sub_sub_item: could not find parent sub item %s: %v", subItemID, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}
		mainItemID := subItemRecord.GetString("main_item")
		recalcMainItemBudgeted(app, mainItemID)

		SetToast(e, "success", "Sub-sub-item deleted")

		data, err := buildBOQEditData(app, boqID, map[string]bool{mainItemID: true}, map[string]bool{subItemID: true})
		if err != nil {
			log.Printf("delete_sub_sub_item: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}
		data.ProjectID = projectID
		return renderBOQEdit(e, data)
	}
}

// HandleExpandMainItem returns the sub-items (and their sub-sub-items) for a given main item.
// Used for lazy loading when expanding a main item accordion.
func HandleExpandMainItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		boqID := e.Request.PathValue("id")
		mainItemID := e.Request.PathValue("itemId")
		if boqID == "" || mainItemID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing required IDs")
		}

		subItemsCol, err := app.FindCollectionByNameOrId("sub_items")
		if err != nil {
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		subSubItemsCol, err := app.FindCollectionByNameOrId("sub_sub_items")
		if err != nil {
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		subItemRecords, err := app.FindRecordsByFilter(subItemsCol, "main_item = {:mainId}", "sort_order", 0, 0, map[string]any{"mainId": mainItemID})
		if err != nil {
			log.Printf("expand_main_item: could not query sub items: %v", err)
			subItemRecords = nil
		}

		var subItems []templates.SubItemEdit
		for _, si := range subItemRecords {
			subSubItemRecords, err := app.FindRecordsByFilter(subSubItemsCol, "sub_item = {:subId}", "sort_order", 0, 0, map[string]any{"subId": si.Id})
			if err != nil {
				log.Printf("expand_main_item: could not query sub-sub items: %v", err)
				subSubItemRecords = nil
			}

			var subSubItems []templates.SubSubItemEdit
			for _, ssi := range subSubItemRecords {
				subSubItems = append(subSubItems, templates.SubSubItemEdit{
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

			subItems = append(subItems, templates.SubItemEdit{
				ID:            si.Id,
				Type:          si.GetString("type"),
				Description:   si.GetString("description"),
				UOM:           si.GetString("uom"),
				HSNCode:       si.GetString("hsn_code"),
				QtyPerUnit:    si.GetFloat("qty_per_unit"),
				UnitPrice:     si.GetFloat("unit_price"),
				BudgetedPrice: si.GetFloat("budgeted_price"),
				GSTPercent:    si.GetFloat("gst_percent"),
				SubSubItems:   subSubItems,
			})
		}

		component := templates.EditSubItemsBlock(projectID, boqID, subItems, services.UOMOptions, services.GSTOptions, nil)
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandlePatchMainItem updates individual fields on a main item.
func HandlePatchMainItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		boqID := e.Request.PathValue("id")
		itemID := e.Request.PathValue("itemId")
		if boqID == "" || itemID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing required IDs")
		}

		record, err := app.FindRecordById("main_boq_items", itemID)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Item not found")
		}

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		updated := false
		for key, values := range e.Request.Form {
			if len(values) == 0 {
				continue
			}
			val := values[0]
			switch key {
			case "description":
				record.Set("description", val)
				updated = true
			case "qty":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					record.Set("qty", f)
					updated = true
				}
			case "uom":
				record.Set("uom", val)
				updated = true
			case "quoted_price":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					record.Set("quoted_price", f)
					updated = true
				}
			case "budgeted_price":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					record.Set("budgeted_price", f)
					updated = true
				}
			case "hsn_code":
				record.Set("hsn_code", val)
				updated = true
			case "gst_percent":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					record.Set("gst_percent", f)
					updated = true
				}
			}
		}

		if updated {
			if err := app.Save(record); err != nil {
				log.Printf("patch_main_item: error saving %s: %v", itemID, err)
				return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
			}
		}

		// Return updated budgeted price info as JSON
		SetToast(e, "info", "Item saved")
		e.Response.Header().Set("Content-Type", "application/json")
		return e.String(200, fmt.Sprintf(`{"budgeted_price": %.2f}`, record.GetFloat("budgeted_price")))
	}
}

// HandlePatchSubItem updates individual fields on a sub-item and recalculates prices.
func HandlePatchSubItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		boqID := e.Request.PathValue("id")
		subItemID := e.Request.PathValue("subItemId")
		if boqID == "" || subItemID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing required IDs")
		}

		record, err := app.FindRecordById("sub_items", subItemID)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Sub-item not found")
		}

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		updated := false
		for key, values := range e.Request.Form {
			if len(values) == 0 {
				continue
			}
			val := values[0]
			switch key {
			case "type":
				record.Set("type", val)
				updated = true
			case "description":
				record.Set("description", val)
				updated = true
			case "qty_per_unit":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					record.Set("qty_per_unit", f)
					updated = true
				}
			case "uom":
				record.Set("uom", val)
				updated = true
			case "unit_price":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					record.Set("unit_price", f)
					updated = true
				}
			case "hsn_code":
				record.Set("hsn_code", val)
				updated = true
			case "gst_percent":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					record.Set("gst_percent", f)
					updated = true
				}
			}
		}

		if updated {
			// Recalculate budgeted price
			recalcSubItemBudgeted(app, subItemID)
			// Reload record to get updated budgeted_price
			record, _ = app.FindRecordById("sub_items", subItemID)
			// Recalculate parent
			recalcMainItemBudgeted(app, record.GetString("main_item"))
		}

		SetToast(e, "info", "Item saved")
		e.Response.Header().Set("Content-Type", "application/json")
		return e.String(200, fmt.Sprintf(`{"budgeted_price": %.2f}`, record.GetFloat("budgeted_price")))
	}
}

// HandlePatchSubSubItem updates individual fields on a sub-sub-item and recalculates prices.
func HandlePatchSubSubItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		boqID := e.Request.PathValue("id")
		subSubItemID := e.Request.PathValue("subSubItemId")
		if boqID == "" || subSubItemID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing required IDs")
		}

		record, err := app.FindRecordById("sub_sub_items", subSubItemID)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Sub-sub-item not found")
		}

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		updated := false
		for key, values := range e.Request.Form {
			if len(values) == 0 {
				continue
			}
			val := values[0]
			switch key {
			case "type":
				record.Set("type", val)
				updated = true
			case "description":
				record.Set("description", val)
				updated = true
			case "qty_per_unit":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					record.Set("qty_per_unit", f)
					updated = true
				}
			case "uom":
				record.Set("uom", val)
				updated = true
			case "unit_price":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					record.Set("unit_price", f)
					updated = true
				}
			case "hsn_code":
				record.Set("hsn_code", val)
				updated = true
			case "gst_percent":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					record.Set("gst_percent", f)
					updated = true
				}
			}
		}

		if updated {
			// Recalculate this item's budgeted price
			budgeted := record.GetFloat("qty_per_unit") * record.GetFloat("unit_price")
			record.Set("budgeted_price", budgeted)
			if err := app.Save(record); err != nil {
				log.Printf("patch_sub_sub_item: error saving %s: %v", subSubItemID, err)
				return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
			}

			// Recalculate parent chain
			subItemID := record.GetString("sub_item")
			recalcSubItemBudgeted(app, subItemID)
			subItemRecord, err := app.FindRecordById("sub_items", subItemID)
			if err == nil {
				recalcMainItemBudgeted(app, subItemRecord.GetString("main_item"))
			}
		}

		SetToast(e, "info", "Item saved")
		e.Response.Header().Set("Content-Type", "application/json")
		return e.String(200, fmt.Sprintf(`{"budgeted_price": %.2f}`, record.GetFloat("budgeted_price")))
	}
}

// recalcSubItemBudgeted recalculates a sub-item's budgeted_price from its sub-sub-items.
func recalcSubItemBudgeted(app *pocketbase.PocketBase, subItemID string) {
	subSubItemsCol, err := app.FindCollectionByNameOrId("sub_sub_items")
	if err != nil {
		return
	}

	subSubItems, err := app.FindRecordsByFilter(subSubItemsCol, "sub_item = {:subId}", "sort_order", 0, 0, map[string]any{"subId": subItemID})
	if err != nil {
		return
	}

	siRecord, err := app.FindRecordById("sub_items", subItemID)
	if err != nil {
		return
	}

	if len(subSubItems) > 0 {
		var total float64
		for _, ssi := range subSubItems {
			total += ssi.GetFloat("budgeted_price")
		}
		siRecord.Set("budgeted_price", total)
	} else {
		budgeted := siRecord.GetFloat("qty_per_unit") * siRecord.GetFloat("unit_price")
		siRecord.Set("budgeted_price", budgeted)
	}

	if err := app.Save(siRecord); err != nil {
		log.Printf("recalcSubItemBudgeted: error saving %s: %v", subItemID, err)
	}
}

// recalcMainItemBudgeted recalculates a main item's budgeted_price from its sub-items.
func recalcMainItemBudgeted(app *pocketbase.PocketBase, mainItemID string) {
	subItemsCol, err := app.FindCollectionByNameOrId("sub_items")
	if err != nil {
		return
	}

	subItems, err := app.FindRecordsByFilter(subItemsCol, "main_item = {:mainId}", "sort_order", 0, 0, map[string]any{"mainId": mainItemID})
	if err != nil {
		return
	}

	miRecord, err := app.FindRecordById("main_boq_items", mainItemID)
	if err != nil {
		return
	}

	if len(subItems) > 0 {
		var total float64
		for _, si := range subItems {
			total += si.GetFloat("budgeted_price")
		}
		miRecord.Set("budgeted_price", total*miRecord.GetFloat("qty"))
	}

	if err := app.Save(miRecord); err != nil {
		log.Printf("recalcMainItemBudgeted: error saving %s: %v", mainItemID, err)
	}
}
