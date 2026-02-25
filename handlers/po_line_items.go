package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

// getNextSortOrder queries the existing line items for a PO and returns the next sort_order value.
func getNextSortOrder(app *pocketbase.PocketBase, poId string) int {
	existing, err := app.FindRecordsByFilter(
		"po_line_items",
		"purchase_order = {:poId}",
		"-sort_order",
		1,
		0,
		map[string]any{"poId": poId},
	)
	if err != nil || len(existing) == 0 {
		return 1
	}
	return existing[0].GetInt("sort_order") + 1
}

// HandlePOAddLineItem handles POST /projects/:projectId/po/:id/line-items
// Creates a manual line item for the given PO.
func HandlePOAddLineItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		poId := e.Request.PathValue("id")

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		description := strings.TrimSpace(e.Request.FormValue("description"))
		hsnCode := strings.TrimSpace(e.Request.FormValue("hsn_code"))
		uom := strings.TrimSpace(e.Request.FormValue("uom"))

		qtyStr := strings.TrimSpace(e.Request.FormValue("qty"))
		rateStr := strings.TrimSpace(e.Request.FormValue("rate"))
		gstPercentStr := strings.TrimSpace(e.Request.FormValue("gst_percent"))

		qty, err := strconv.ParseFloat(qtyStr, 64)
		if err != nil {
			qty = 0
		}
		rate, err := strconv.ParseFloat(rateStr, 64)
		if err != nil {
			rate = 0
		}
		gstPercent, err := strconv.ParseFloat(gstPercentStr, 64)
		if err != nil {
			gstPercent = 0
		}

		// Validation
		errors := make(map[string]string)
		if description == "" {
			errors["description"] = "Description is required"
		}
		if uom == "" {
			errors["uom"] = "Unit of measure is required"
		}
		if qty <= 0 {
			errors["qty"] = "Quantity must be greater than zero"
		}
		if rate < 0 {
			errors["rate"] = "Rate must be zero or greater"
		}

		if len(errors) > 0 {
			SetToast(e, "warning", "Please fix the errors below")
			data, buildErr := buildPOEditData(app, projectId, poId)
			if buildErr != nil {
				log.Printf("po_line_items: HandlePOAddLineItem: buildPOEditData failed: %v", buildErr)
				return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
			}
			data.Errors = errors
			return templates.POLineItemsSection(data).Render(e.Request.Context(), e.Response)
		}

		nextSortOrder := getNextSortOrder(app, poId)

		col, err := app.FindCollectionByNameOrId("po_line_items")
		if err != nil {
			log.Printf("po_line_items: HandlePOAddLineItem: could not find po_line_items collection: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		record := core.NewRecord(col)
		record.Set("purchase_order", poId)
		record.Set("sort_order", nextSortOrder)
		record.Set("description", description)
		record.Set("hsn_code", hsnCode)
		record.Set("qty", qty)
		record.Set("uom", uom)
		record.Set("rate", rate)
		record.Set("gst_percent", gstPercent)
		record.Set("source_item_type", "manual")
		record.Set("source_item_id", "")

		if err := app.Save(record); err != nil {
			log.Printf("po_line_items: HandlePOAddLineItem: could not save line item: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		SetToast(e, "success", "Line item added")

		data, buildErr := buildPOEditData(app, projectId, poId)
		if buildErr != nil {
			log.Printf("po_line_items: HandlePOAddLineItem: buildPOEditData failed after save: %v", buildErr)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		return templates.POLineItemsSection(data).Render(e.Request.Context(), e.Response)
	}
}

// HandlePOAddLineItemFromBOQ handles POST /projects/:projectId/po/:id/line-items/from-boq
// Creates a line item sourced from a BOQ item.
func HandlePOAddLineItemFromBOQ(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		poId := e.Request.PathValue("id")

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		sourceItemType := strings.TrimSpace(e.Request.FormValue("source_item_type"))
		sourceItemId := strings.TrimSpace(e.Request.FormValue("source_item_id"))

		if sourceItemType == "" || sourceItemId == "" {
			return ErrorToast(e, http.StatusBadRequest, "Source item type and ID are required")
		}

		// Resolve collection name from source_item_type
		var collectionName string
		switch sourceItemType {
		case "main_item":
			collectionName = "main_boq_items"
		case "sub_item":
			collectionName = "sub_items"
		case "sub_sub_item":
			collectionName = "sub_sub_items"
		default:
			return ErrorToast(e, http.StatusBadRequest, "Unknown source item type")
		}

		sourceRecord, err := app.FindRecordById(collectionName, sourceItemId)
		if err != nil {
			log.Printf("po_line_items: HandlePOAddLineItemFromBOQ: could not find %s/%s: %v", collectionName, sourceItemId, err)
			return ErrorToast(e, http.StatusNotFound, "Source item not found")
		}

		description := sourceRecord.GetString("description")
		hsnCode := sourceRecord.GetString("hsn_code")
		uom := sourceRecord.GetString("uom")
		gstPercent := sourceRecord.GetFloat("gst_percent")

		var qty, rate float64
		if sourceItemType == "main_item" {
			qty = sourceRecord.GetFloat("qty")
			rate = sourceRecord.GetFloat("unit_price")
		} else {
			// sub_item and sub_sub_item use qty_per_unit
			qty = sourceRecord.GetFloat("qty_per_unit")
			rate = sourceRecord.GetFloat("unit_price")
		}

		nextSortOrder := getNextSortOrder(app, poId)

		col, err := app.FindCollectionByNameOrId("po_line_items")
		if err != nil {
			log.Printf("po_line_items: HandlePOAddLineItemFromBOQ: could not find po_line_items collection: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		record := core.NewRecord(col)
		record.Set("purchase_order", poId)
		record.Set("sort_order", nextSortOrder)
		record.Set("description", description)
		record.Set("hsn_code", hsnCode)
		record.Set("qty", qty)
		record.Set("uom", uom)
		record.Set("rate", rate)
		record.Set("gst_percent", gstPercent)
		record.Set("source_item_type", sourceItemType)
		record.Set("source_item_id", sourceItemId)

		if err := app.Save(record); err != nil {
			log.Printf("po_line_items: HandlePOAddLineItemFromBOQ: could not save line item: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		SetToast(e, "success", "BOQ items added to purchase order")

		data, buildErr := buildPOEditData(app, projectId, poId)
		if buildErr != nil {
			log.Printf("po_line_items: HandlePOAddLineItemFromBOQ: buildPOEditData failed after save: %v", buildErr)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		return templates.POLineItemsSection(data).Render(e.Request.Context(), e.Response)
	}
}

// HandlePOUpdateLineItem handles PATCH /projects/:projectId/po/:id/line-items/:itemId
// Updates fields on an existing PO line item.
func HandlePOUpdateLineItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		poId := e.Request.PathValue("id")
		itemId := e.Request.PathValue("itemId")

		item, err := app.FindRecordById("po_line_items", itemId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Line item not found")
		}

		if item.GetString("purchase_order") != poId {
			return ErrorToast(e, http.StatusForbidden, "Line item does not belong to this purchase order")
		}

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		// Update only fields that are provided (non-empty)
		if v := strings.TrimSpace(e.Request.FormValue("description")); v != "" {
			item.Set("description", v)
		}
		if v := strings.TrimSpace(e.Request.FormValue("hsn_code")); v != "" {
			item.Set("hsn_code", v)
		}
		if v := strings.TrimSpace(e.Request.FormValue("uom")); v != "" {
			item.Set("uom", v)
		}
		if v := strings.TrimSpace(e.Request.FormValue("qty")); v != "" {
			if qty, err := strconv.ParseFloat(v, 64); err == nil {
				item.Set("qty", qty)
			}
		}
		if v := strings.TrimSpace(e.Request.FormValue("rate")); v != "" {
			if rate, err := strconv.ParseFloat(v, 64); err == nil {
				item.Set("rate", rate)
			}
		}
		if v := strings.TrimSpace(e.Request.FormValue("gst_percent")); v != "" {
			if gst, err := strconv.ParseFloat(v, 64); err == nil {
				item.Set("gst_percent", gst)
			}
		}

		if err := app.Save(item); err != nil {
			log.Printf("po_line_items: HandlePOUpdateLineItem: could not save line item %s: %v", itemId, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		SetToast(e, "info", "Line item updated")

		data, buildErr := buildPOEditData(app, projectId, poId)
		if buildErr != nil {
			log.Printf("po_line_items: HandlePOUpdateLineItem: buildPOEditData failed after save: %v", buildErr)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		return templates.POLineItemsSection(data).Render(e.Request.Context(), e.Response)
	}
}

// HandlePODeleteLineItem handles DELETE /projects/:projectId/po/:id/line-items/:itemId
// Deletes a PO line item.
func HandlePODeleteLineItem(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		poId := e.Request.PathValue("id")
		itemId := e.Request.PathValue("itemId")

		item, err := app.FindRecordById("po_line_items", itemId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Line item not found")
		}

		if item.GetString("purchase_order") != poId {
			return ErrorToast(e, http.StatusForbidden, "Line item does not belong to this purchase order")
		}

		if err := app.Delete(item); err != nil {
			log.Printf("po_line_items: HandlePODeleteLineItem: could not delete line item %s: %v", itemId, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		SetToast(e, "success", "Line item removed")

		data, buildErr := buildPOEditData(app, projectId, poId)
		if buildErr != nil {
			log.Printf("po_line_items: HandlePODeleteLineItem: buildPOEditData failed after delete: %v", buildErr)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		return templates.POLineItemsSection(data).Render(e.Request.Context(), e.Response)
	}
}

// HandlePOBOQPicker handles GET /projects/:projectId/po/:id/boq-picker
// Renders the BOQ picker modal content for selecting items to add to a PO.
func HandlePOBOQPicker(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		poId := e.Request.PathValue("id")

		// Fetch all BOQs for the project
		boqRecords, err := app.FindRecordsByFilter(
			"boqs",
			"project = {:projectId}",
			"title",
			0,
			0,
			map[string]any{"projectId": projectId},
		)
		if err != nil {
			log.Printf("po_line_items: HandlePOBOQPicker: could not query boqs: %v", err)
			boqRecords = nil
		}

		var boqs []templates.BOQPickerBOQ
		for _, boqRec := range boqRecords {
			boqId := boqRec.Id

			// Fetch main items for this BOQ
			mainItemRecords, err := app.FindRecordsByFilter(
				"main_boq_items",
				"boq = {:boqId}",
				"sort_order",
				0,
				0,
				map[string]any{"boqId": boqId},
			)
			if err != nil {
				log.Printf("po_line_items: HandlePOBOQPicker: could not query main_boq_items for boq %s: %v", boqId, err)
				mainItemRecords = nil
			}

			var mainItems []templates.BOQPickerMainItem
			for _, miRec := range mainItemRecords {
				mainItemId := miRec.Id

				// Fetch sub items for this main item
				subItemRecords, err := app.FindRecordsByFilter(
					"sub_items",
					"main_item = {:mainId}",
					"sort_order",
					0,
					0,
					map[string]any{"mainId": mainItemId},
				)
				if err != nil {
					log.Printf("po_line_items: HandlePOBOQPicker: could not query sub_items for main_item %s: %v", mainItemId, err)
					subItemRecords = nil
				}

				var subItems []templates.BOQPickerSubItem
				for _, siRec := range subItemRecords {
					subItemId := siRec.Id

					// Fetch sub-sub items for this sub item
					subSubItemRecords, err := app.FindRecordsByFilter(
						"sub_sub_items",
						"sub_item = {:subId}",
						"sort_order",
						0,
						0,
						map[string]any{"subId": subItemId},
					)
					if err != nil {
						log.Printf("po_line_items: HandlePOBOQPicker: could not query sub_sub_items for sub_item %s: %v", subItemId, err)
						subSubItemRecords = nil
					}

					var subSubItems []templates.BOQPickerSubSubItem
					for _, ssiRec := range subSubItemRecords {
						subSubItems = append(subSubItems, templates.BOQPickerSubSubItem{
							ID:          ssiRec.Id,
							Description: ssiRec.GetString("description"),
							HSNCode:     ssiRec.GetString("hsn_code"),
							Qty:         ssiRec.GetFloat("qty_per_unit"),
							UoM:         ssiRec.GetString("uom"),
							Rate:        ssiRec.GetFloat("unit_price"),
							GSTPercent:  ssiRec.GetFloat("gst_percent"),
						})
					}

					subItems = append(subItems, templates.BOQPickerSubItem{
						ID:          siRec.Id,
						Description: siRec.GetString("description"),
						HSNCode:     siRec.GetString("hsn_code"),
						Qty:         siRec.GetFloat("qty_per_unit"),
						UoM:         siRec.GetString("uom"),
						Rate:        siRec.GetFloat("unit_price"),
						GSTPercent:  siRec.GetFloat("gst_percent"),
						SubSubItems: subSubItems,
					})
				}

				mainItems = append(mainItems, templates.BOQPickerMainItem{
					ID:          miRec.Id,
					Description: miRec.GetString("description"),
					HSNCode:     miRec.GetString("hsn_code"),
					Qty:         miRec.GetFloat("qty"),
					UoM:         miRec.GetString("uom"),
					Rate:        miRec.GetFloat("unit_price"),
					GSTPercent:  miRec.GetFloat("gst_percent"),
					SubItems:    subItems,
				})
			}

			boqs = append(boqs, templates.BOQPickerBOQ{
				ID:        boqRec.Id,
				Title:     boqRec.GetString("title"),
				MainItems: mainItems,
			})
		}

		data := templates.BOQPickerData{
			ProjectID: projectId,
			POID:      poId,
			BOQs:      boqs,
		}

		return templates.BOQPickerContent(data).Render(e.Request.Context(), e.Response)
	}
}
