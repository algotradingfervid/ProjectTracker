package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// fetchDCTemplatesForProject returns DC template select items for a project.
func fetchDCTemplatesForProject(app *pocketbase.PocketBase, projectId string) []templates.DCTemplateSelectItem {
	records, err := app.FindRecordsByFilter("dc_templates", "project = {:pid}", "name", 0, 0, map[string]any{"pid": projectId})
	if err != nil {
		return nil
	}

	var items []templates.DCTemplateSelectItem
	for _, rec := range records {
		// Count items in this template
		templateItems, _ := app.FindRecordsByFilter("dc_template_items", "template = {:tid}", "", 0, 0, map[string]any{"tid": rec.Id})
		items = append(items, templates.DCTemplateSelectItem{
			ID:        rec.Id,
			Name:      rec.GetString("name"),
			Purpose:   rec.GetString("purpose"),
			ItemCount: len(templateItems),
		})
	}
	return items
}

// fetchActiveTransporters returns active transporter select items for a project.
func fetchActiveTransporters(app *pocketbase.PocketBase, projectId string) []templates.TransporterSelectItem {
	records, err := app.FindRecordsByFilter("transporters", "project = {:pid} && is_active = true", "company_name", 0, 0, map[string]any{"pid": projectId})
	if err != nil {
		return nil
	}

	var items []templates.TransporterSelectItem
	for _, rec := range records {
		vehicles := fetchVehiclesForSelect(app, rec.Id)
		items = append(items, templates.TransporterSelectItem{
			ID:            rec.Id,
			CompanyName:   rec.GetString("company_name"),
			ContactPerson: rec.GetString("contact_person"),
			Phone:         rec.GetString("phone"),
			Vehicles:      vehicles,
		})
	}
	return items
}

// fetchVehiclesForSelect returns vehicle select items for a transporter.
func fetchVehiclesForSelect(app *pocketbase.PocketBase, transporterID string) []templates.VehicleSelectItem {
	records, err := app.FindRecordsByFilter("transporter_vehicles", "transporter = {:tid}", "vehicle_number", 0, 0, map[string]any{"tid": transporterID})
	if err != nil {
		return nil
	}

	var items []templates.VehicleSelectItem
	for _, rec := range records {
		items = append(items, templates.VehicleSelectItem{
			ID:            rec.Id,
			VehicleNumber: rec.GetString("vehicle_number"),
			VehicleType:   rec.GetString("vehicle_type"),
			DriverName:    rec.GetString("driver_name"),
			DriverPhone:   rec.GetString("driver_phone"),
		})
	}
	return items
}

// fetchTemplateItemsWithBOQ returns template items enriched with BOQ source data.
func fetchTemplateItemsWithBOQ(app *pocketbase.PocketBase, templateID string) []templates.DCWizardItem {
	templateItems, err := app.FindRecordsByFilter("dc_template_items", "template = {:tid}", "created", 0, 0, map[string]any{"tid": templateID})
	if err != nil {
		return nil
	}

	var items []templates.DCWizardItem
	for i, ti := range templateItems {
		itemType := ti.GetString("source_item_type")
		itemID := ti.GetString("source_item_id")
		serialTracking := ti.GetString("serial_tracking")
		defaultQty := ti.GetFloat("default_quantity")

		item := templates.DCWizardItem{
			SourceItemType:  itemType,
			SourceItemID:    itemID,
			SerialTracking:  serialTracking,
			DefaultQuantity: defaultQty,
			LineOrder:       i + 1,
		}

		// Resolve BOQ source data
		var collection string
		if itemType == "sub_item" {
			collection = "sub_items"
		} else {
			collection = "sub_sub_items"
		}
		sourceRec, err := app.FindRecordById(collection, itemID)
		if err != nil {
			log.Printf("dc_wizard: could not find %s %s: %v", collection, itemID, err)
			continue
		}

		item.Description = sourceRec.GetString("description")
		item.UOM = sourceRec.GetString("uom")
		item.HSNCode = sourceRec.GetString("hsn_code")
		item.Rate = sourceRec.GetFloat("quoted_price")
		item.TaxPercentage = sourceRec.GetFloat("gst_percentage")

		items = append(items, item)
	}
	return items
}

// HandleDCWizardStep1 renders the DC wizard step 1 (Setup).
func HandleDCWizardStep1(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")

		dcTemplates := fetchDCTemplatesForProject(app, projectId)
		transporters := fetchActiveTransporters(app, projectId)

		data := templates.DCWizardStep1Data{
			ProjectID:    projectId,
			Templates:    dcTemplates,
			Transporters: transporters,
			ChallanDate:  time.Now().Format("2006-01-02"),
			DCType:       "direct",
			Errors:       make(map[string]string),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.DCWizardStep1Content(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.DCWizardStep1Page(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleDCWizardStep2 processes step 1 form data and renders step 2 (Destinations).
func HandleDCWizardStep2(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")

		// Extract step 1 values
		dcType := strings.TrimSpace(e.Request.FormValue("dc_type"))
		templateID := strings.TrimSpace(e.Request.FormValue("template_id"))
		challanDate := strings.TrimSpace(e.Request.FormValue("challan_date"))
		transporterID := strings.TrimSpace(e.Request.FormValue("transporter_id"))
		vehicleID := strings.TrimSpace(e.Request.FormValue("vehicle_id"))
		ewayBillNumber := strings.TrimSpace(e.Request.FormValue("eway_bill_number"))
		docketNumber := strings.TrimSpace(e.Request.FormValue("docket_number"))
		reverseCharge := e.Request.FormValue("reverse_charge") == "on"

		// Validate step 1
		errors := make(map[string]string)
		if dcType == "" {
			dcType = "direct"
		}
		if templateID == "" {
			errors["template_id"] = "DC Template is required"
		}
		if challanDate == "" {
			errors["challan_date"] = "Challan date is required"
		}

		if len(errors) > 0 {
			SetToast(e, "warning", "Please fix the errors below")
			dcTemplates := fetchDCTemplatesForProject(app, projectId)
			transporters := fetchActiveTransporters(app, projectId)
			data := templates.DCWizardStep1Data{
				ProjectID:      projectId,
				Templates:      dcTemplates,
				Transporters:   transporters,
				DCType:         dcType,
				TemplateID:     templateID,
				ChallanDate:    challanDate,
				TransporterID:  transporterID,
				VehicleID:      vehicleID,
				EwayBillNumber: ewayBillNumber,
				DocketNumber:   docketNumber,
				ReverseCharge:  reverseCharge,
				Errors:         errors,
			}
			var component templ.Component
			if e.Request.Header.Get("HX-Request") == "true" {
				component = templates.DCWizardStep1Content(data)
			} else {
				headerData := GetHeaderData(e.Request)
				sidebarData := GetSidebarData(e.Request)
				component = templates.DCWizardStep1Page(data, headerData, sidebarData)
			}
			return component.Render(e.Request.Context(), e.Response)
		}

		// Fetch data for step 2
		project, _ := app.FindRecordById("projects", projectId)

		// Default addresses from project settings
		defaultBillFromID := ""
		defaultDispatchFromID := ""
		if project != nil {
			defaultBillFromID = project.GetString("default_bill_from")
			defaultDispatchFromID = project.GetString("default_dispatch_from")
		}

		billFromAddresses := fetchAddressesByType(app, projectId, "bill_from")
		dispatchFromAddresses := fetchAddressesByType(app, projectId, "ship_from")
		billToAddresses := fetchAddressesByType(app, projectId, "bill_to")
		shipToAddresses := fetchAddressesByType(app, projectId, "ship_to")

		// Resolve transporter name for display
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

		// Resolve template name
		templateName := ""
		if templateID != "" {
			if tRec, err := app.FindRecordById("dc_templates", templateID); err == nil {
				templateName = tRec.GetString("name")
			}
		}

		data := templates.DCWizardStep2Data{
			ProjectID:             projectId,
			DCType:                dcType,
			TemplateID:            templateID,
			TemplateName:          templateName,
			ChallanDate:           challanDate,
			TransporterID:         transporterID,
			TransporterName:       transporterName,
			VehicleID:             vehicleID,
			VehicleNumber:         vehicleNumber,
			EwayBillNumber:        ewayBillNumber,
			DocketNumber:          docketNumber,
			ReverseCharge:         reverseCharge,
			BillFromAddresses:     billFromAddresses,
			DispatchFromAddresses: dispatchFromAddresses,
			BillToAddresses:       billToAddresses,
			ShipToAddresses:       shipToAddresses,
			DefaultBillFromID:     defaultBillFromID,
			DefaultDispatchFromID: defaultDispatchFromID,
			BillFromID:            defaultBillFromID,
			DispatchFromID:        defaultDispatchFromID,
			NumDestinations:       1,
			Errors:                make(map[string]string),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.DCWizardStep2Content(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.DCWizardStep2Page(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleDCWizardBackToStep1 renders step 1 with previously submitted data.
func HandleDCWizardBackToStep1(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")
		dcTemplates := fetchDCTemplatesForProject(app, projectId)
		transporters := fetchActiveTransporters(app, projectId)

		data := templates.DCWizardStep1Data{
			ProjectID:      projectId,
			Templates:      dcTemplates,
			Transporters:   transporters,
			DCType:         e.Request.FormValue("dc_type"),
			TemplateID:     e.Request.FormValue("template_id"),
			ChallanDate:    e.Request.FormValue("challan_date"),
			TransporterID:  e.Request.FormValue("transporter_id"),
			VehicleID:      e.Request.FormValue("vehicle_id"),
			EwayBillNumber: e.Request.FormValue("eway_bill_number"),
			DocketNumber:   e.Request.FormValue("docket_number"),
			ReverseCharge:  e.Request.FormValue("reverse_charge") == "on",
			Errors:         make(map[string]string),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.DCWizardStep1Content(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.DCWizardStep1Page(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleDCWizardStep3 processes step 2 form data and renders step 3 (Items & Serials).
func HandleDCWizardStep3(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")

		// Extract step 1 values (carried forward)
		dcType := e.Request.FormValue("dc_type")
		templateID := e.Request.FormValue("template_id")
		challanDate := e.Request.FormValue("challan_date")
		transporterID := e.Request.FormValue("transporter_id")
		vehicleID := e.Request.FormValue("vehicle_id")
		ewayBillNumber := e.Request.FormValue("eway_bill_number")
		docketNumber := e.Request.FormValue("docket_number")
		reverseCharge := e.Request.FormValue("reverse_charge") == "on"

		// Extract step 2 values
		billFromID := strings.TrimSpace(e.Request.FormValue("bill_from_id"))
		dispatchFromID := strings.TrimSpace(e.Request.FormValue("dispatch_from_id"))
		billToID := strings.TrimSpace(e.Request.FormValue("bill_to_id"))
		numDestStr := strings.TrimSpace(e.Request.FormValue("num_destinations"))
		numDest, _ := strconv.Atoi(numDestStr)
		if numDest < 1 {
			numDest = 1
		}
		hubAddressID := strings.TrimSpace(e.Request.FormValue("hub_address_id"))
		taxTypeOverride := strings.TrimSpace(e.Request.FormValue("tax_type_override"))

		// Collect ship-to addresses
		shipToIDs := make([]string, numDest)
		for i := 0; i < numDest; i++ {
			shipToIDs[i] = strings.TrimSpace(e.Request.FormValue(fmt.Sprintf("ship_to_id_%d", i)))
		}

		// Validate step 2
		errors := make(map[string]string)
		if billToID == "" {
			errors["bill_to_id"] = "Bill To address is required"
		}
		for i, sid := range shipToIDs {
			if sid == "" {
				errors[fmt.Sprintf("ship_to_id_%d", i)] = fmt.Sprintf("Ship To address %d is required", i+1)
			}
		}
		if dcType == "transfer" && hubAddressID == "" {
			errors["hub_address_id"] = "Hub address is required for Transfer DCs"
		}

		if len(errors) > 0 {
			SetToast(e, "warning", "Please fix the errors below")
			// Re-render step 2 with errors
			project, _ := app.FindRecordById("projects", projectId)
			defaultBillFromID := ""
			defaultDispatchFromID := ""
			if project != nil {
				defaultBillFromID = project.GetString("default_bill_from")
				defaultDispatchFromID = project.GetString("default_dispatch_from")
			}

			templateName := ""
			if tRec, err := app.FindRecordById("dc_templates", templateID); err == nil {
				templateName = tRec.GetString("name")
			}
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

			data := templates.DCWizardStep2Data{
				ProjectID:             projectId,
				DCType:                dcType,
				TemplateID:            templateID,
				TemplateName:          templateName,
				ChallanDate:           challanDate,
				TransporterID:         transporterID,
				TransporterName:       transporterName,
				VehicleID:             vehicleID,
				VehicleNumber:         vehicleNumber,
				EwayBillNumber:        ewayBillNumber,
				DocketNumber:          docketNumber,
				ReverseCharge:         reverseCharge,
				BillFromAddresses:     fetchAddressesByType(app, projectId, "bill_from"),
				DispatchFromAddresses: fetchAddressesByType(app, projectId, "ship_from"),
				BillToAddresses:       fetchAddressesByType(app, projectId, "bill_to"),
				ShipToAddresses:       fetchAddressesByType(app, projectId, "ship_to"),
				DefaultBillFromID:     defaultBillFromID,
				DefaultDispatchFromID: defaultDispatchFromID,
				BillFromID:            billFromID,
				DispatchFromID:        dispatchFromID,
				BillToID:              billToID,
				ShipToIDs:             shipToIDs,
				NumDestinations:       numDest,
				HubAddressID:          hubAddressID,
				TaxTypeOverride:       taxTypeOverride,
				Errors:                errors,
			}
			var component templ.Component
			if e.Request.Header.Get("HX-Request") == "true" {
				component = templates.DCWizardStep2Content(data)
			} else {
				headerData := GetHeaderData(e.Request)
				sidebarData := GetSidebarData(e.Request)
				component = templates.DCWizardStep2Page(data, headerData, sidebarData)
			}
			return component.Render(e.Request.Context(), e.Response)
		}

		// Determine tax type
		taxType := taxTypeOverride
		if taxType == "" {
			taxType = determineTaxType(app, billFromID, shipToIDs)
		}

		// Fetch template items with BOQ data
		wizardItems := fetchTemplateItemsWithBOQ(app, templateID)

		// Resolve destination display names
		destinations := make([]templates.DCWizardDestination, numDest)
		for i, sid := range shipToIDs {
			dest := templates.DCWizardDestination{
				Index:     i,
				AddressID: sid,
			}
			if rec, err := app.FindRecordById("addresses", sid); err == nil {
				data := readAddressData(rec)
				dest.DisplayName = data["company_name"]
				if dest.DisplayName == "" {
					dest.DisplayName = data["contact_person"]
				}
				if city := data["city"]; city != "" {
					dest.DisplayName += ", " + city
				}
			}
			destinations[i] = dest
		}

		// Resolve template name
		templateName := ""
		if tRec, err := app.FindRecordById("dc_templates", templateID); err == nil {
			templateName = tRec.GetString("name")
		}

		data := templates.DCWizardStep3Data{
			ProjectID:      projectId,
			DCType:         dcType,
			TemplateID:     templateID,
			TemplateName:   templateName,
			ChallanDate:    challanDate,
			TransporterID:  transporterID,
			VehicleID:      vehicleID,
			EwayBillNumber: ewayBillNumber,
			DocketNumber:   docketNumber,
			ReverseCharge:  reverseCharge,
			BillFromID:     billFromID,
			DispatchFromID: dispatchFromID,
			BillToID:       billToID,
			ShipToIDs:      shipToIDs,
			HubAddressID:   hubAddressID,
			TaxType:        taxType,
			Items:          wizardItems,
			Destinations:   destinations,
			Errors:         make(map[string]string),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.DCWizardStep3Content(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.DCWizardStep3Page(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleDCWizardBackToStep2 renders step 2 with previously submitted data.
func HandleDCWizardBackToStep2(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")

		dcType := e.Request.FormValue("dc_type")
		templateID := e.Request.FormValue("template_id")
		challanDate := e.Request.FormValue("challan_date")
		transporterID := e.Request.FormValue("transporter_id")
		vehicleID := e.Request.FormValue("vehicle_id")
		ewayBillNumber := e.Request.FormValue("eway_bill_number")
		docketNumber := e.Request.FormValue("docket_number")
		reverseCharge := e.Request.FormValue("reverse_charge") == "on"

		billFromID := e.Request.FormValue("bill_from_id")
		dispatchFromID := e.Request.FormValue("dispatch_from_id")
		billToID := e.Request.FormValue("bill_to_id")
		numDestStr := e.Request.FormValue("num_destinations")
		numDest, _ := strconv.Atoi(numDestStr)
		if numDest < 1 {
			numDest = 1
		}
		hubAddressID := e.Request.FormValue("hub_address_id")
		taxTypeOverride := e.Request.FormValue("tax_type_override")

		shipToIDs := make([]string, numDest)
		for i := 0; i < numDest; i++ {
			shipToIDs[i] = e.Request.FormValue(fmt.Sprintf("ship_to_id_%d", i))
		}

		project, _ := app.FindRecordById("projects", projectId)
		defaultBillFromID := ""
		defaultDispatchFromID := ""
		if project != nil {
			defaultBillFromID = project.GetString("default_bill_from")
			defaultDispatchFromID = project.GetString("default_dispatch_from")
		}

		templateName := ""
		if tRec, err := app.FindRecordById("dc_templates", templateID); err == nil {
			templateName = tRec.GetString("name")
		}
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

		data := templates.DCWizardStep2Data{
			ProjectID:             projectId,
			DCType:                dcType,
			TemplateID:            templateID,
			TemplateName:          templateName,
			ChallanDate:           challanDate,
			TransporterID:         transporterID,
			TransporterName:       transporterName,
			VehicleID:             vehicleID,
			VehicleNumber:         vehicleNumber,
			EwayBillNumber:        ewayBillNumber,
			DocketNumber:          docketNumber,
			ReverseCharge:         reverseCharge,
			BillFromAddresses:     fetchAddressesByType(app, projectId, "bill_from"),
			DispatchFromAddresses: fetchAddressesByType(app, projectId, "ship_from"),
			BillToAddresses:       fetchAddressesByType(app, projectId, "bill_to"),
			ShipToAddresses:       fetchAddressesByType(app, projectId, "ship_to"),
			DefaultBillFromID:     defaultBillFromID,
			DefaultDispatchFromID: defaultDispatchFromID,
			BillFromID:            billFromID,
			DispatchFromID:        dispatchFromID,
			BillToID:              billToID,
			ShipToIDs:             shipToIDs,
			NumDestinations:       numDest,
			HubAddressID:          hubAddressID,
			TaxTypeOverride:       taxTypeOverride,
			Errors:                make(map[string]string),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.DCWizardStep2Content(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.DCWizardStep2Page(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleSerialValidate provides real-time serial number validation via HTMX.
func HandleSerialValidate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")
		serialsRaw := strings.TrimSpace(e.Request.FormValue("serials"))
		expectedQtyStr := strings.TrimSpace(e.Request.FormValue("expected_qty"))
		expectedQty, _ := strconv.Atoi(expectedQtyStr)

		serials := splitSerials(serialsRaw)

		// Get existing serials for project
		existingSerials := getExistingSerials(app, projectId)

		result := services.ValidateSerials(serials, expectedQty, existingSerials)

		// Return validation HTML snippet
		resultJSON, _ := json.Marshal(result)
		return e.String(http.StatusOK, string(resultJSON))
	}
}

// determineTaxType determines IGST or CGST+SGST based on state comparison.
func determineTaxType(app *pocketbase.PocketBase, billFromID string, shipToIDs []string) string {
	if billFromID == "" || len(shipToIDs) == 0 {
		return "cgst_sgst"
	}

	billFromRec, err := app.FindRecordById("addresses", billFromID)
	if err != nil {
		return "cgst_sgst"
	}
	billFromData := readAddressData(billFromRec)
	billFromState := strings.ToLower(strings.TrimSpace(billFromData["state"]))

	for _, sid := range shipToIDs {
		if sid == "" {
			continue
		}
		shipToRec, err := app.FindRecordById("addresses", sid)
		if err != nil {
			continue
		}
		shipToData := readAddressData(shipToRec)
		shipToState := strings.ToLower(strings.TrimSpace(shipToData["state"]))

		if billFromState != shipToState {
			return "igst"
		}
	}

	return "cgst_sgst"
}

// splitSerials splits a newline/comma-separated serial string into individual serials.
func splitSerials(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, ",", "\n")
	parts := strings.Split(raw, "\n")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// HandleDCWizardStep4 processes step 3 form data and renders step 4 (Review & Confirm).
func HandleDCWizardStep4(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")

		// Extract all carried-forward values
		dcType := e.Request.FormValue("dc_type")
		templateID := e.Request.FormValue("template_id")
		challanDate := e.Request.FormValue("challan_date")
		transporterID := e.Request.FormValue("transporter_id")
		vehicleID := e.Request.FormValue("vehicle_id")
		ewayBillNumber := e.Request.FormValue("eway_bill_number")
		docketNumber := e.Request.FormValue("docket_number")
		reverseCharge := e.Request.FormValue("reverse_charge") == "on"
		billFromID := e.Request.FormValue("bill_from_id")
		dispatchFromID := e.Request.FormValue("dispatch_from_id")
		billToID := e.Request.FormValue("bill_to_id")
		numDestStr := e.Request.FormValue("num_destinations")
		numDest, _ := strconv.Atoi(numDestStr)
		if numDest < 1 {
			numDest = 1
		}
		hubAddressID := e.Request.FormValue("hub_address_id")
		taxType := e.Request.FormValue("tax_type")

		shipToIDs := make([]string, numDest)
		for i := 0; i < numDest; i++ {
			shipToIDs[i] = e.Request.FormValue(fmt.Sprintf("ship_to_id_%d", i))
		}

		// Extract item keys (multiple values with same name)
		itemKeys := e.Request.Form["item_keys"]
		// Deduplicate item keys while preserving order
		itemKeys = deduplicateStrings(itemKeys)

		// Extract item metadata and quantities
		itemFormData := make(map[string]map[string]string)
		qtyFormData := make(map[string]string)
		serialFormData := make(map[string]string)

		for _, key := range itemKeys {
			itemFormData[key] = map[string]string{
				"type":             e.Request.FormValue(fmt.Sprintf("item_type_%s", key)),
				"id":               e.Request.FormValue(fmt.Sprintf("item_id_%s", key)),
				"rate":             e.Request.FormValue(fmt.Sprintf("item_rate_%s", key)),
				"tax":              e.Request.FormValue(fmt.Sprintf("item_tax_%s", key)),
				"serial_tracking":  e.Request.FormValue(fmt.Sprintf("item_serial_tracking_%s", key)),
			}

			// Quantities per destination
			for i := 0; i < numDest; i++ {
				qtyName := fmt.Sprintf("qty_%s_dest_%d", key, i)
				qtyFormData[qtyName] = e.Request.FormValue(qtyName)
			}

			// Serials
			serialName := fmt.Sprintf("serials_%s", key)
			serialFormData[serialName] = e.Request.FormValue(serialName)
		}

		// Build review items with pricing calculations
		var reviewItems []templates.DCWizardStep4ReviewItem
		var grandTaxable, grandTax, grandTotal float64
		errors := make(map[string]string)

		for _, key := range itemKeys {
			fields := itemFormData[key]

			rate := parseFloat(fields["rate"])
			taxPct := parseFloat(fields["tax"])

			// Collect quantities per destination
			quantities := make([]int, numDest)
			totalQty := 0
			for i := 0; i < numDest; i++ {
				qtyStr := e.Request.FormValue(fmt.Sprintf("qty_%s_dest_%d", key, i))
				qty, _ := strconv.Atoi(qtyStr)
				quantities[i] = qty
				totalQty += qty
			}

			// Parse serials
			serialsRaw := e.Request.FormValue(fmt.Sprintf("serials_%s", key))
			serials := splitSerials(serialsRaw)

			// Validate serials for required tracking
			serialTracking := fields["serial_tracking"]
			if serialTracking == "required" && totalQty > 0 {
				if len(serials) != totalQty {
					errors[fmt.Sprintf("serials_%s", key)] = fmt.Sprintf("Item requires %d serial(s), got %d", totalQty, len(serials))
				}
			}

			// Pricing
			taxableAmount := rate * float64(totalQty)
			taxAmount := taxableAmount * taxPct / 100.0
			totalAmount := taxableAmount + taxAmount

			grandTaxable += taxableAmount
			grandTax += taxAmount
			grandTotal += totalAmount

			// Resolve description from BOQ source
			description := ""
			hsnCode := ""
			uom := ""
			collection := "sub_items"
			if fields["type"] == "sub_sub_item" {
				collection = "sub_sub_items"
			}
			if sourceRec, err := app.FindRecordById(collection, fields["id"]); err == nil {
				description = sourceRec.GetString("description")
				hsnCode = sourceRec.GetString("hsn_code")
				uom = sourceRec.GetString("uom")
			}

			reviewItems = append(reviewItems, templates.DCWizardStep4ReviewItem{
				SourceItemType: fields["type"],
				SourceItemID:   fields["id"],
				Description:    description,
				HSNCode:        hsnCode,
				UOM:            uom,
				Rate:           rate,
				TaxPercentage:  taxPct,
				SerialTracking: serialTracking,
				Quantities:     quantities,
				TotalQty:       totalQty,
				Serials:        serials,
				TaxableAmount:  taxableAmount,
				TaxAmount:      taxAmount,
				TotalAmount:    totalAmount,
			})
		}

		// Validate serials against existing in project
		if len(errors) == 0 {
			existingSerials := getExistingSerials(app, projectId)
			for _, item := range reviewItems {
				if len(item.Serials) > 0 {
					result := services.ValidateSerials(item.Serials, item.TotalQty, existingSerials)
					if !result.Valid {
						if len(result.DuplicatesInInput) > 0 {
							errors["serials"] = fmt.Sprintf("Duplicate serial numbers found: %s", strings.Join(result.DuplicatesInInput, ", "))
						}
						for _, conflict := range result.DuplicatesInDB {
							errors["serials_db"] = fmt.Sprintf("Serial %s already used in %s", conflict.Serial, conflict.ExistingDC)
						}
					}
				}
			}
		}

		// Resolve addresses for display
		billFrom := resolveAddressDisplay(app, billFromID, "Bill From")
		dispatchFrom := resolveAddressDisplay(app, dispatchFromID, "Dispatch From")
		billTo := resolveAddressDisplay(app, billToID, "Bill To")

		shipTos := make([]templates.DCWizardStep4AddressDisplay, numDest)
		for i, sid := range shipToIDs {
			shipTos[i] = resolveAddressDisplay(app, sid, fmt.Sprintf("Ship To %d", i+1))
		}

		hubAddress := templates.DCWizardStep4AddressDisplay{}
		if dcType == "transfer" {
			hubAddress = resolveAddressDisplay(app, hubAddressID, "Hub Address")
		}

		// Resolve display names
		templateName := ""
		if tRec, err := app.FindRecordById("dc_templates", templateID); err == nil {
			templateName = tRec.GetString("name")
		}
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

		// Build destination display names for table headers
		destinations := make([]templates.DCWizardDestination, numDest)
		for i, sid := range shipToIDs {
			dest := templates.DCWizardDestination{Index: i, AddressID: sid}
			if rec, err := app.FindRecordById("addresses", sid); err == nil {
				data := readAddressData(rec)
				dest.DisplayName = data["company_name"]
				if dest.DisplayName == "" {
					dest.DisplayName = data["contact_person"]
				}
				if city := data["city"]; city != "" {
					dest.DisplayName += ", " + city
				}
			}
			destinations[i] = dest
		}

		data := templates.DCWizardStep4Data{
			ProjectID:       projectId,
			DCType:          dcType,
			TemplateID:      templateID,
			TemplateName:    templateName,
			ChallanDate:     challanDate,
			TransporterID:   transporterID,
			TransporterName: transporterName,
			VehicleID:       vehicleID,
			VehicleNumber:   vehicleNumber,
			EwayBillNumber:  ewayBillNumber,
			DocketNumber:    docketNumber,
			ReverseCharge:   reverseCharge,
			BillFromID:      billFromID,
			DispatchFromID:  dispatchFromID,
			BillToID:        billToID,
			ShipToIDs:       shipToIDs,
			HubAddressID:    hubAddressID,
			TaxType:         taxType,
			BillFrom:        billFrom,
			DispatchFrom:    dispatchFrom,
			BillTo:          billTo,
			ShipTos:         shipTos,
			HubAddress:      hubAddress,
			Items:           reviewItems,
			Destinations:    destinations,
			GrandTaxable:    grandTaxable,
			GrandTax:        grandTax,
			GrandTotal:      grandTotal,
			ItemKeys:        itemKeys,
			ItemFormData:    itemFormData,
			QtyFormData:     qtyFormData,
			SerialFormData:  serialFormData,
			Errors:          errors,
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.DCWizardStep4Content(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.DCWizardStep4Page(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleDCWizardBackToStep3 renders step 3 with previously submitted data.
func HandleDCWizardBackToStep3(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")

		dcType := e.Request.FormValue("dc_type")
		templateID := e.Request.FormValue("template_id")
		challanDate := e.Request.FormValue("challan_date")
		transporterID := e.Request.FormValue("transporter_id")
		vehicleID := e.Request.FormValue("vehicle_id")
		ewayBillNumber := e.Request.FormValue("eway_bill_number")
		docketNumber := e.Request.FormValue("docket_number")
		reverseCharge := e.Request.FormValue("reverse_charge") == "on"
		billFromID := e.Request.FormValue("bill_from_id")
		dispatchFromID := e.Request.FormValue("dispatch_from_id")
		billToID := e.Request.FormValue("bill_to_id")
		numDestStr := e.Request.FormValue("num_destinations")
		numDest, _ := strconv.Atoi(numDestStr)
		if numDest < 1 {
			numDest = 1
		}
		hubAddressID := e.Request.FormValue("hub_address_id")
		taxType := e.Request.FormValue("tax_type")

		shipToIDs := make([]string, numDest)
		for i := 0; i < numDest; i++ {
			shipToIDs[i] = e.Request.FormValue(fmt.Sprintf("ship_to_id_%d", i))
		}

		// Fetch template items with BOQ data
		wizardItems := fetchTemplateItemsWithBOQ(app, templateID)

		// Resolve destination display names
		destinations := make([]templates.DCWizardDestination, numDest)
		for i, sid := range shipToIDs {
			dest := templates.DCWizardDestination{Index: i, AddressID: sid}
			if rec, err := app.FindRecordById("addresses", sid); err == nil {
				addrData := readAddressData(rec)
				dest.DisplayName = addrData["company_name"]
				if dest.DisplayName == "" {
					dest.DisplayName = addrData["contact_person"]
				}
				if city := addrData["city"]; city != "" {
					dest.DisplayName += ", " + city
				}
			}
			destinations[i] = dest
		}

		templateName := ""
		if tRec, err := app.FindRecordById("dc_templates", templateID); err == nil {
			templateName = tRec.GetString("name")
		}

		data := templates.DCWizardStep3Data{
			ProjectID:      projectId,
			DCType:         dcType,
			TemplateID:     templateID,
			TemplateName:   templateName,
			ChallanDate:    challanDate,
			TransporterID:  transporterID,
			VehicleID:      vehicleID,
			EwayBillNumber: ewayBillNumber,
			DocketNumber:   docketNumber,
			ReverseCharge:  reverseCharge,
			BillFromID:     billFromID,
			DispatchFromID: dispatchFromID,
			BillToID:       billToID,
			ShipToIDs:      shipToIDs,
			HubAddressID:   hubAddressID,
			TaxType:        taxType,
			Items:          wizardItems,
			Destinations:   destinations,
			Errors:         make(map[string]string),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.DCWizardStep3Content(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.DCWizardStep3Page(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleDCCreate processes the final wizard submission and creates the DC records.
func HandleDCCreate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")
		dcType := e.Request.FormValue("dc_type")
		numDestStr := e.Request.FormValue("num_destinations")
		numDest, _ := strconv.Atoi(numDestStr)
		if numDest < 1 {
			numDest = 1
		}

		shipToIDs := make([]string, numDest)
		for i := 0; i < numDest; i++ {
			shipToIDs[i] = e.Request.FormValue(fmt.Sprintf("ship_to_id_%d", i))
		}

		items := services.ParseShipmentItemsFromForm(e.Request.Form, numDest)

		if dcType == "transfer" {
			params := services.TransferDCParams{
				ProjectID:      projectId,
				TemplateID:     e.Request.FormValue("template_id"),
				ChallanDate:    e.Request.FormValue("challan_date"),
				TransporterID:  e.Request.FormValue("transporter_id"),
				VehicleID:      e.Request.FormValue("vehicle_id"),
				EwayBillNumber: e.Request.FormValue("eway_bill_number"),
				DocketNumber:   e.Request.FormValue("docket_number"),
				ReverseCharge:  e.Request.FormValue("reverse_charge") == "on",
				BillFromID:     e.Request.FormValue("bill_from_id"),
				DispatchFromID: e.Request.FormValue("dispatch_from_id"),
				BillToID:       e.Request.FormValue("bill_to_id"),
				ShipToIDs:      shipToIDs,
				HubAddressID:   e.Request.FormValue("hub_address_id"),
				TaxType:        e.Request.FormValue("tax_type"),
				Items:          items,
			}

			result, err := services.CreateTransferDC(app, params)
			if err != nil {
				return ErrorToast(e, http.StatusInternalServerError, "Failed to create transfer DC: "+err.Error())
			}

			// Redirect to DC detail
			redirectURL := fmt.Sprintf("/projects/%s/dcs/%s", projectId, result.DCID)
			if e.Request.Header.Get("HX-Request") == "true" {
				e.Response.Header().Set("HX-Redirect", redirectURL)
				return e.String(http.StatusOK, "")
			}
			return e.Redirect(http.StatusFound, redirectURL)
		}

		// Direct shipment
		params := services.ShipmentParams{
			ProjectID:      projectId,
			TemplateID:     e.Request.FormValue("template_id"),
			ChallanDate:    e.Request.FormValue("challan_date"),
			TransporterID:  e.Request.FormValue("transporter_id"),
			VehicleID:      e.Request.FormValue("vehicle_id"),
			EwayBillNumber: e.Request.FormValue("eway_bill_number"),
			DocketNumber:   e.Request.FormValue("docket_number"),
			ReverseCharge:  e.Request.FormValue("reverse_charge") == "on",
			BillFromID:     e.Request.FormValue("bill_from_id"),
			DispatchFromID: e.Request.FormValue("dispatch_from_id"),
			BillToID:       e.Request.FormValue("bill_to_id"),
			ShipToIDs:      shipToIDs,
			TaxType:        e.Request.FormValue("tax_type"),
			Items:          items,
		}

		result, err := services.CreateDirectShipment(app, params)
		if err != nil {
			return ErrorToast(e, http.StatusInternalServerError, "Failed to create shipment: "+err.Error())
		}

		// Redirect to transit DC detail (primary view for shipment group)
		redirectURL := fmt.Sprintf("/projects/%s/dcs/%s", projectId, result.TransitDCID)
		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}

// resolveAddressDisplay resolves an address ID to a display struct.
func resolveAddressDisplay(app *pocketbase.PocketBase, addressID, label string) templates.DCWizardStep4AddressDisplay {
	display := templates.DCWizardStep4AddressDisplay{Label: label}
	if addressID == "" {
		return display
	}

	rec, err := app.FindRecordById("addresses", addressID)
	if err != nil {
		return display
	}

	data := readAddressData(rec)
	display.CompanyName = data["company_name"]
	if display.CompanyName == "" {
		display.CompanyName = data["contact_person"]
	}
	display.Address = data["address_line_1"]
	if line2 := data["address_line_2"]; line2 != "" {
		display.Address += ", " + line2
	}
	display.City = data["city"]
	display.State = data["state"]
	display.PinCode = data["pin_code"]

	return display
}

// deduplicateStrings removes duplicates from a slice while preserving order.
func deduplicateStrings(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// parseFloat safely parses a string to float64.
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

// getExistingSerials returns a map of serial_number -> dc_number for all serials in a project.
func getExistingSerials(app *pocketbase.PocketBase, projectId string) map[string]string {
	serials, err := app.FindRecordsByFilter("serial_numbers", "project = {:pid}", "", 0, 0, map[string]any{"pid": projectId})
	if err != nil {
		return nil
	}

	result := make(map[string]string)
	for _, s := range serials {
		serial := s.GetString("serial_number")
		lineItemID := s.GetString("line_item")
		// Resolve DC number via line_item → dc → dc_number
		lineItem, err := app.FindRecordById("dc_line_items", lineItemID)
		if err != nil {
			result[serial] = "unknown"
			continue
		}
		dcID := lineItem.GetString("dc")
		dc, err := app.FindRecordById("delivery_challans", dcID)
		if err != nil {
			result[serial] = "unknown"
			continue
		}
		result[serial] = dc.GetString("dc_number")
	}
	return result
}
