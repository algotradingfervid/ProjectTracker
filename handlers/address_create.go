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

// HandleAddressCreate returns a handler that renders the address creation form.
func HandleAddressCreate(app *pocketbase.PocketBase, addressType AddressType) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		if projectID == "" {
			return e.String(400, "Missing project ID")
		}

		// Set active project cookie
		http.SetCookie(e.Response, &http.Cookie{
			Name:   "active_project",
			Value:  projectID,
			Path:   "/",
			MaxAge: 86400 * 30,
		})

		// Verify project exists
		projectRecord, err := app.FindRecordById("projects", projectID)
		if err != nil {
			log.Printf("address_create: could not find project %s: %v", projectID, err)
			return e.String(404, "Project not found")
		}

		// Fetch required fields from project_address_settings
		requiredFields := services.GetRequiredFields(app, projectID, string(addressType))

		// Prepare Ship To parent options for Install At type
		var shipToAddresses []templates.ShipToOption
		var showShipToParent bool
		if addressType == AddressTypeInstallAt {
			showShipToParent, shipToAddresses = fetchShipToOptions(app, projectID)
		}

		// Build form data
		data := templates.AddressFormData{
			IsEdit:           false,
			ProjectID:        projectID,
			ProjectName:      projectRecord.GetString("name"),
			AddressType:      string(addressType),
			AddressLabel:     addressTypeLabel(addressType),
			Country:          "India",
			RequiredFields:   requiredFields,
			Errors:           make(map[string]string),
			StateOptions:     services.IndianStates,
			CountryOptions:   services.Countries,
			ShowShipToParent: showShipToParent,
			ShipToAddresses:  shipToAddresses,
		}

		// Render
		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.AddressFormContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.AddressFormPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleAddressSave returns a handler that processes the address creation form submission.
func HandleAddressSave(app *pocketbase.PocketBase, addressType AddressType) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		if projectID == "" {
			return e.String(400, "Missing project ID")
		}

		if err := e.Request.ParseForm(); err != nil {
			return e.String(400, "Invalid form data")
		}

		// Verify project exists
		projectRecord, err := app.FindRecordById("projects", projectID)
		if err != nil {
			log.Printf("address_save: could not find project %s: %v", projectID, err)
			return e.String(404, "Project not found")
		}

		// Extract form fields
		fields := extractAddressFields(e)

		// Validate: required fields + format validation
		errors := services.ValidateAddress(app, projectID, string(addressType), fields)
		formatErrors := services.ValidateAddressFormat(fields)
		for k, v := range formatErrors {
			errors[k] = v
		}

		if len(errors) > 0 {
			// Fetch required fields for re-rendering
			requiredFields := services.GetRequiredFields(app, projectID, string(addressType))

			var shipToAddresses []templates.ShipToOption
			var showShipToParent bool
			if addressType == AddressTypeInstallAt {
				showShipToParent, shipToAddresses = fetchShipToOptions(app, projectID)
			}

			data := templates.AddressFormData{
				IsEdit:           false,
				ProjectID:        projectID,
				ProjectName:      projectRecord.GetString("name"),
				AddressType:      string(addressType),
				AddressLabel:     addressTypeLabel(addressType),
				CompanyName:      fields["company_name"],
				ContactPerson:    fields["contact_person"],
				AddressLine1:     fields["address_line_1"],
				AddressLine2:     fields["address_line_2"],
				City:             fields["city"],
				State:            fields["state"],
				PinCode:          fields["pin_code"],
				Country:          fields["country"],
				Phone:            fields["phone"],
				Email:            fields["email"],
				GSTIN:            fields["gstin"],
				PAN:              fields["pan"],
				CIN:              fields["cin"],
				Website:          fields["website"],
				Fax:              fields["fax"],
				Landmark:         fields["landmark"],
				District:         fields["district"],
				RequiredFields:   requiredFields,
				Errors:           errors,
				StateOptions:     services.IndianStates,
				CountryOptions:   services.Countries,
				ShowShipToParent: showShipToParent,
				ShipToAddresses:  shipToAddresses,
				SelectedShipToID: e.Request.FormValue("ship_to_parent"),
			}

			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component := templates.AddressFormPage(data, headerData, sidebarData)
			return component.Render(e.Request.Context(), e.Response)
		}

		// Create the address record
		addressesCol, err := app.FindCollectionByNameOrId("addresses")
		if err != nil {
			log.Printf("address_save: could not find addresses collection: %v", err)
			return e.String(500, "Internal error")
		}

		record := core.NewRecord(addressesCol)
		record.Set("project", projectID)
		record.Set("address_type", string(addressType))
		setAddressRecordFields(record, fields)

		// Set ship_to_parent for install_at type
		if addressType == AddressTypeInstallAt {
			if parentID := e.Request.FormValue("ship_to_parent"); parentID != "" {
				record.Set("ship_to_parent", parentID)
			}
		}

		if err := app.Save(record); err != nil {
			log.Printf("address_save: could not save address: %v", err)
			return e.String(500, "Failed to save address")
		}

		// Redirect to address list
		slugType := addressTypeToSlug(addressType)
		redirectURL := fmt.Sprintf("/projects/%s/addresses/%s", projectID, slugType)
		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}

// --- Helper functions ---

// extractAddressFields extracts all address fields from the request form.
func extractAddressFields(e *core.RequestEvent) map[string]string {
	return map[string]string{
		"company_name":   strings.TrimSpace(e.Request.FormValue("company_name")),
		"contact_person": strings.TrimSpace(e.Request.FormValue("contact_person")),
		"address_line_1": strings.TrimSpace(e.Request.FormValue("address_line_1")),
		"address_line_2": strings.TrimSpace(e.Request.FormValue("address_line_2")),
		"city":           strings.TrimSpace(e.Request.FormValue("city")),
		"state":          strings.TrimSpace(e.Request.FormValue("state")),
		"pin_code":       strings.TrimSpace(e.Request.FormValue("pin_code")),
		"country":        strings.TrimSpace(e.Request.FormValue("country")),
		"phone":          strings.TrimSpace(e.Request.FormValue("phone")),
		"email":          strings.TrimSpace(e.Request.FormValue("email")),
		"gstin":          strings.TrimSpace(strings.ToUpper(e.Request.FormValue("gstin"))),
		"pan":            strings.TrimSpace(strings.ToUpper(e.Request.FormValue("pan"))),
		"cin":            strings.TrimSpace(strings.ToUpper(e.Request.FormValue("cin"))),
		"website":        strings.TrimSpace(e.Request.FormValue("website")),
		"fax":            strings.TrimSpace(e.Request.FormValue("fax")),
		"landmark":       strings.TrimSpace(e.Request.FormValue("landmark")),
		"district":       strings.TrimSpace(e.Request.FormValue("district")),
	}
}

// setAddressRecordFields sets all address fields on a PocketBase record.
func setAddressRecordFields(record *core.Record, fields map[string]string) {
	for key, val := range fields {
		record.Set(key, val)
	}
}

// fetchShipToOptions loads Ship To addresses for the Install At parent dropdown.
func fetchShipToOptions(app *pocketbase.PocketBase, projectID string) (bool, []templates.ShipToOption) {
	// Check if project has ship_to_equals_install_at toggle ON
	projectRecord, err := app.FindRecordById("projects", projectID)
	if err != nil {
		return false, nil
	}

	// If the toggle is ON, no dropdown needed (addresses auto-linked)
	if projectRecord.GetBool("ship_to_equals_install_at") {
		return false, nil
	}

	// Fetch all ship_to addresses for this project
	addressesCol, err := app.FindCollectionByNameOrId("addresses")
	if err != nil {
		return true, nil
	}

	records, err := app.FindRecordsByFilter(
		addressesCol,
		"project = {:projectId} && address_type = 'ship_to'",
		"company_name", 0, 0,
		map[string]any{"projectId": projectID},
	)
	if err != nil {
		return true, nil
	}

	var options []templates.ShipToOption
	for _, rec := range records {
		options = append(options, templates.ShipToOption{
			ID:          rec.Id,
			CompanyName: rec.GetString("company_name"),
			City:        rec.GetString("city"),
		})
	}

	return true, options
}

// addressTypeToSlug converts an AddressType to URL slug format.
func addressTypeToSlug(at AddressType) string {
	return strings.ReplaceAll(string(at), "_", "-")
}
