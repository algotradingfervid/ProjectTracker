package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// HandleSplitStep1 renders step 1 of the split wizard: select destinations to split.
func HandleSplitStep1(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		dcId := e.Request.PathValue("id")

		// Fetch and validate the transfer DC
		dc, err := app.FindRecordById("delivery_challans", dcId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Delivery challan not found")
		}
		if dc.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "DC not found in this project")
		}
		if dc.GetString("dc_type") != "transfer" {
			return ErrorToast(e, http.StatusBadRequest, "Only transfer DCs can be split")
		}
		status := dc.GetString("status")
		if status != "issued" && status != "splitting" {
			return ErrorToast(e, http.StatusBadRequest, "DC must be issued before splitting")
		}

		// Fetch transfer DC metadata
		transferRecs, _ := app.FindRecordsByFilter("transfer_dcs", "dc = {:did}", "", 1, 0, map[string]any{"did": dcId})
		if len(transferRecs) == 0 {
			return ErrorToast(e, http.StatusNotFound, "Transfer DC metadata not found")
		}
		transferDC := transferRecs[0]

		// Fetch destinations
		destRecs, _ := app.FindRecordsByFilter("transfer_dc_destinations", "transfer_dc = {:tid}", "", 0, 0, map[string]any{"tid": transferDC.Id})
		var destinations []templates.SplitDestination
		hasUnsplit := false
		for _, dest := range destRecs {
			addr := resolveDCAddressDisplay(app, dest.GetString("ship_to_address"))
			isSplit := dest.GetBool("is_split")

			// Fetch per-item quantities for this destination
			destQtyRecs, _ := app.FindRecordsByFilter("transfer_dc_dest_quantities", "destination = {:did}", "", 0, 0, map[string]any{"did": dest.Id})
			totalQty := 0
			for _, dq := range destQtyRecs {
				totalQty += dq.GetInt("quantity")
			}

			sd := templates.SplitDestination{
				ID:       dest.Id,
				IsSplit:  isSplit,
				TotalQty: totalQty,
			}
			if addr != nil {
				sd.ShipToName = addr.CompanyName
				sd.ShipToCity = addr.City
				sd.ShipToState = addr.State
			}

			destinations = append(destinations, sd)
			if !isSplit {
				hasUnsplit = true
			}
		}

		if !hasUnsplit {
			return ErrorToast(e, http.StatusBadRequest, "All destinations have already been split")
		}

		data := templates.SplitWizardStep1Data{
			ProjectID:    projectId,
			DCID:         dcId,
			DCNumber:     dc.GetString("dc_number"),
			Destinations: destinations,
			Errors:       make(map[string]string),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.SplitWizardStep1Content(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.SplitWizardStep1Page(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleSplitStep2 renders step 2: transporter selection and serial assignment.
func HandleSplitStep2(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		dcId := e.Request.PathValue("id")

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		// Get selected destination IDs
		selectedDestIDs := e.Request.Form["destination_ids"]
		if len(selectedDestIDs) == 0 {
			// Re-render step 1 with error
			return handleSplitStep1WithError(app, e, projectId, dcId, "Please select at least one destination")
		}

		// Fetch DC and transfer metadata
		dc, err := app.FindRecordById("delivery_challans", dcId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "DC not found")
		}

		transferRecs, _ := app.FindRecordsByFilter("transfer_dcs", "dc = {:did}", "", 1, 0, map[string]any{"did": dcId})
		if len(transferRecs) == 0 {
			return ErrorToast(e, http.StatusNotFound, "Transfer DC metadata not found")
		}

		// Fetch parent line items
		parentLineItems, _ := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "line_order", 0, 0, map[string]any{"did": dcId})

		// Build items with quantities for selected destinations
		var items []templates.SplitWizardItem
		for _, li := range parentLineItems {
			sourceType := li.GetString("source_item_type")
			sourceID := li.GetString("source_item_id")

			// Calculate total qty across selected destinations
			totalQty := 0
			for _, destID := range selectedDestIDs {
				destQtyRecs, _ := app.FindRecordsByFilter("transfer_dc_dest_quantities",
					"destination = {:did} && source_item_type = {:sit} && source_item_id = {:sid}",
					"", 1, 0,
					map[string]any{"did": destID, "sit": sourceType, "sid": sourceID})
				if len(destQtyRecs) > 0 {
					totalQty += destQtyRecs[0].GetInt("quantity")
				}
			}

			if totalQty == 0 {
				continue
			}

			// Resolve description from BOQ
			description := ""
			collection := "sub_items"
			if sourceType == "sub_sub_item" {
				collection = "sub_sub_items"
			}
			if sourceRec, err := app.FindRecordById(collection, sourceID); err == nil {
				description = sourceRec.GetString("description")
			}

			// Check serial tracking from template items
			serialTracking := "none"
			templateItems, _ := app.FindRecordsByFilter("dc_template_items",
				"source_item_type = {:sit} && source_item_id = {:sid}",
				"", 1, 0,
				map[string]any{"sit": sourceType, "sid": sourceID})
			if len(templateItems) > 0 {
				serialTracking = templateItems[0].GetString("serial_tracking")
			}

			// Fetch available serials from parent DC for this item
			var availableSerials []string
			serialRecs, _ := app.FindRecordsByFilter("serial_numbers", "line_item = {:lid}", "serial_number", 0, 0, map[string]any{"lid": li.Id})
			for _, sr := range serialRecs {
				availableSerials = append(availableSerials, sr.GetString("serial_number"))
			}

			items = append(items, templates.SplitWizardItem{
				ParentLineItemID: li.Id,
				SourceItemType:   sourceType,
				SourceItemID:     sourceID,
				Description:      description,
				TotalQty:         totalQty,
				SerialTracking:   serialTracking,
				AvailableSerials: availableSerials,
			})
		}

		// Fetch transporters
		transporters := fetchActiveTransporters(app, projectId)

		data := templates.SplitWizardStep2Data{
			ProjectID:       projectId,
			DCID:            dcId,
			DCNumber:        dc.GetString("dc_number"),
			SelectedDestIDs: selectedDestIDs,
			Items:           items,
			Transporters:    transporters,
			Errors:          make(map[string]string),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.SplitWizardStep2Content(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.SplitWizardStep2Page(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleSplitStep3 renders step 3: review and confirm.
func HandleSplitStep3(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		dcId := e.Request.PathValue("id")

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		dc, err := app.FindRecordById("delivery_challans", dcId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "DC not found")
		}

		selectedDestIDs := e.Request.Form["destination_ids"]
		transporterID := strings.TrimSpace(e.Request.FormValue("transporter_id"))
		vehicleID := strings.TrimSpace(e.Request.FormValue("vehicle_id"))
		ewayBillNumber := strings.TrimSpace(e.Request.FormValue("eway_bill_number"))
		docketNumber := strings.TrimSpace(e.Request.FormValue("docket_number"))

		// Resolve transporter name
		transporterName := ""
		vehicleNumber := ""
		if transporterID != "" {
			if tRec, err := app.FindRecordById("transporters", transporterID); err == nil {
				transporterName = tRec.GetString("company_name")
			}
		}
		if vehicleID != "" {
			if vRec, err := app.FindRecordById("transporter_vehicles", vehicleID); err == nil {
				vehicleNumber = vRec.GetString("vehicle_number")
			}
		}

		// Resolve destination names
		var destNames []string
		for _, destID := range selectedDestIDs {
			if destRec, err := app.FindRecordById("transfer_dc_destinations", destID); err == nil {
				addr := resolveDCAddressDisplay(app, destRec.GetString("ship_to_address"))
				if addr != nil {
					name := addr.CompanyName
					if addr.City != "" {
						name += ", " + addr.City
					}
					destNames = append(destNames, name)
				}
			}
		}

		// Build item review data with serial assignments
		parentLineItems, _ := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "line_order", 0, 0, map[string]any{"did": dcId})
		var reviewItems []templates.SplitReviewItem
		serialAssignments := make(map[string][]string)

		for _, li := range parentLineItems {
			sourceType := li.GetString("source_item_type")
			sourceID := li.GetString("source_item_id")

			totalQty := 0
			for _, destID := range selectedDestIDs {
				destQtyRecs, _ := app.FindRecordsByFilter("transfer_dc_dest_quantities",
					"destination = {:did} && source_item_type = {:sit} && source_item_id = {:sid}",
					"", 1, 0,
					map[string]any{"did": destID, "sit": sourceType, "sid": sourceID})
				if len(destQtyRecs) > 0 {
					totalQty += destQtyRecs[0].GetInt("quantity")
				}
			}

			if totalQty == 0 {
				continue
			}

			description := ""
			collection := "sub_items"
			if sourceType == "sub_sub_item" {
				collection = "sub_sub_items"
			}
			if sourceRec, err := app.FindRecordById(collection, sourceID); err == nil {
				description = sourceRec.GetString("description")
			}

			// Parse serial assignments from form
			serialsRaw := strings.TrimSpace(e.Request.FormValue(fmt.Sprintf("serials_%s", li.Id)))
			var serials []string
			if serialsRaw != "" {
				serialsRaw = strings.ReplaceAll(serialsRaw, "\r\n", "\n")
				serialsRaw = strings.ReplaceAll(serialsRaw, ",", "\n")
				for _, s := range strings.Split(serialsRaw, "\n") {
					s = strings.TrimSpace(s)
					if s != "" {
						serials = append(serials, s)
					}
				}
			}
			serialAssignments[li.Id] = serials

			reviewItems = append(reviewItems, templates.SplitReviewItem{
				ParentLineItemID: li.Id,
				Description:      description,
				TotalQty:         totalQty,
				Serials:          serials,
			})
		}

		data := templates.SplitWizardStep3Data{
			ProjectID:        projectId,
			DCID:             dcId,
			DCNumber:         dc.GetString("dc_number"),
			SelectedDestIDs:  selectedDestIDs,
			DestinationNames: destNames,
			TransporterID:    transporterID,
			TransporterName:  transporterName,
			VehicleID:        vehicleID,
			VehicleNumber:    vehicleNumber,
			EwayBillNumber:   ewayBillNumber,
			DocketNumber:     docketNumber,
			Items:            reviewItems,
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.SplitWizardStep3Content(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.SplitWizardStep3Page(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleSplitCreate processes the final split creation.
func HandleSplitCreate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		dcId := e.Request.PathValue("id")

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		selectedDestIDs := e.Request.Form["destination_ids"]
		transporterID := strings.TrimSpace(e.Request.FormValue("transporter_id"))
		vehicleID := strings.TrimSpace(e.Request.FormValue("vehicle_id"))
		ewayBillNumber := strings.TrimSpace(e.Request.FormValue("eway_bill_number"))
		docketNumber := strings.TrimSpace(e.Request.FormValue("docket_number"))

		// Parse serial assignments
		serialAssignments := make(map[string][]string)
		parentLineItems, _ := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "line_order", 0, 0, map[string]any{"did": dcId})
		for _, li := range parentLineItems {
			serialsRaw := strings.TrimSpace(e.Request.FormValue(fmt.Sprintf("serials_%s", li.Id)))
			if serialsRaw != "" {
				serialsRaw = strings.ReplaceAll(serialsRaw, "\r\n", "\n")
				serialsRaw = strings.ReplaceAll(serialsRaw, ",", "\n")
				var serials []string
				for _, s := range strings.Split(serialsRaw, "\n") {
					s = strings.TrimSpace(s)
					if s != "" {
						serials = append(serials, s)
					}
				}
				if len(serials) > 0 {
					serialAssignments[li.Id] = serials
				}
			}
		}

		params := services.SplitParams{
			ProjectID:         projectId,
			TransferDCID:      dcId,
			DestinationIDs:    selectedDestIDs,
			TransporterID:     transporterID,
			VehicleID:         vehicleID,
			EwayBillNumber:    ewayBillNumber,
			DocketNumber:      docketNumber,
			SerialAssignments: serialAssignments,
		}

		result, err := services.CreateSplit(app, params)
		if err != nil {
			return ErrorToast(e, http.StatusInternalServerError, fmt.Sprintf("Failed to create split: %s", err.Error()))
		}

		// Redirect to the new transit DC detail
		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", fmt.Sprintf("/projects/%s/dcs/%s", projectId, result.TransitDCID))
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, fmt.Sprintf("/projects/%s/dcs/%s", projectId, result.TransitDCID))
	}
}

// HandleSplitUndo reverses a split operation.
func HandleSplitUndo(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		dcId := e.Request.PathValue("id")
		splitId := e.Request.PathValue("sid")

		if err := services.UndoSplit(app, splitId); err != nil {
			return ErrorToast(e, http.StatusInternalServerError, fmt.Sprintf("Failed to undo split: %s", err.Error()))
		}

		// Redirect back to the transfer DC detail
		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", fmt.Sprintf("/projects/%s/dcs/%s", projectId, dcId))
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, fmt.Sprintf("/projects/%s/dcs/%s", projectId, dcId))
	}
}

// handleSplitStep1WithError re-renders step 1 with an error message.
func handleSplitStep1WithError(app *pocketbase.PocketBase, e *core.RequestEvent, projectId, dcId, errMsg string) error {
	dc, err := app.FindRecordById("delivery_challans", dcId)
	if err != nil {
		return ErrorToast(e, http.StatusNotFound, "DC not found")
	}

	transferRecs, _ := app.FindRecordsByFilter("transfer_dcs", "dc = {:did}", "", 1, 0, map[string]any{"did": dcId})
	if len(transferRecs) == 0 {
		return ErrorToast(e, http.StatusNotFound, "Transfer DC metadata not found")
	}
	transferDC := transferRecs[0]

	destRecs, _ := app.FindRecordsByFilter("transfer_dc_destinations", "transfer_dc = {:tid}", "", 0, 0, map[string]any{"tid": transferDC.Id})
	var destinations []templates.SplitDestination
	for _, dest := range destRecs {
		addr := resolveDCAddressDisplay(app, dest.GetString("ship_to_address"))
		destQtyRecs, _ := app.FindRecordsByFilter("transfer_dc_dest_quantities", "destination = {:did}", "", 0, 0, map[string]any{"did": dest.Id})
		totalQty := 0
		for _, dq := range destQtyRecs {
			totalQty += dq.GetInt("quantity")
		}

		sd := templates.SplitDestination{
			ID:       dest.Id,
			IsSplit:  dest.GetBool("is_split"),
			TotalQty: totalQty,
		}
		if addr != nil {
			sd.ShipToName = addr.CompanyName
			sd.ShipToCity = addr.City
			sd.ShipToState = addr.State
		}
		destinations = append(destinations, sd)
	}

	data := templates.SplitWizardStep1Data{
		ProjectID:    projectId,
		DCID:         dcId,
		DCNumber:     dc.GetString("dc_number"),
		Destinations: destinations,
		Errors:       map[string]string{"destinations": errMsg},
	}

	var component templ.Component
	if e.Request.Header.Get("HX-Request") == "true" {
		component = templates.SplitWizardStep1Content(data)
	} else {
		headerData := GetHeaderData(e.Request)
		sidebarData := GetSidebarData(e.Request)
		component = templates.SplitWizardStep1Page(data, headerData, sidebarData)
	}
	return component.Render(e.Request.Context(), e.Response)
}
