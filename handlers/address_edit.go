package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// HandleAddressEdit returns a handler that renders the address edit form pre-populated with existing data.
func HandleAddressEdit(app *pocketbase.PocketBase, addressType AddressType) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		addressID := e.Request.PathValue("addressId")
		if projectID == "" || addressID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing project ID or address ID")
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
			log.Printf("address_edit: could not find project %s: %v", projectID, err)
			return ErrorToast(e, http.StatusNotFound, "Project not found")
		}

		// Fetch the address record
		addressRecord, err := app.FindRecordById("addresses", addressID)
		if err != nil {
			log.Printf("address_edit: could not find address %s: %v", addressID, err)
			return ErrorToast(e, http.StatusNotFound, "Address not found")
		}

		// Verify address belongs to this project and address type
		if addressRecord.GetString("project") != projectID ||
			addressRecord.GetString("address_type") != string(addressType) {
			return ErrorToast(e, http.StatusForbidden, "Address does not belong to this project")
		}

		// Fetch required fields
		requiredFields := services.GetRequiredFields(app, projectID, string(addressType))

		// Prepare Ship To options for Install At
		var shipToAddresses []templates.ShipToOption
		var showShipToParent bool
		if addressType == AddressTypeInstallAt {
			showShipToParent, shipToAddresses = fetchShipToOptions(app, projectID)
		}

		// Build form data from existing record
		data := templates.AddressFormData{
			IsEdit:           true,
			AddressID:        addressID,
			ProjectID:        projectID,
			ProjectName:      projectRecord.GetString("name"),
			AddressType:      string(addressType),
			AddressLabel:     addressTypeLabel(addressType),
			CompanyName:      addressRecord.GetString("company_name"),
			ContactPerson:    addressRecord.GetString("contact_person"),
			AddressLine1:     addressRecord.GetString("address_line_1"),
			AddressLine2:     addressRecord.GetString("address_line_2"),
			City:             addressRecord.GetString("city"),
			State:            addressRecord.GetString("state"),
			PinCode:          addressRecord.GetString("pin_code"),
			Country:          addressRecord.GetString("country"),
			Phone:            addressRecord.GetString("phone"),
			Email:            addressRecord.GetString("email"),
			GSTIN:            addressRecord.GetString("gstin"),
			PAN:              addressRecord.GetString("pan"),
			CIN:              addressRecord.GetString("cin"),
			Website:          addressRecord.GetString("website"),
			Fax:              addressRecord.GetString("fax"),
			Landmark:         addressRecord.GetString("landmark"),
			District:         addressRecord.GetString("district"),
			RequiredFields:   requiredFields,
			Errors:           make(map[string]string),
			StateOptions:     services.IndianStates,
			CountryOptions:   services.Countries,
			ShowShipToParent: showShipToParent,
			ShipToAddresses:  shipToAddresses,
			SelectedShipToID: addressRecord.GetString("ship_to_parent"),
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

// HandleAddressUpdate returns a handler that processes the address edit form submission.
func HandleAddressUpdate(app *pocketbase.PocketBase, addressType AddressType) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		addressID := e.Request.PathValue("addressId")
		if projectID == "" || addressID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing project ID or address ID")
		}

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		// Verify project exists
		projectRecord, err := app.FindRecordById("projects", projectID)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Project not found")
		}

		// Fetch existing address
		addressRecord, err := app.FindRecordById("addresses", addressID)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Address not found")
		}

		if addressRecord.GetString("project") != projectID ||
			addressRecord.GetString("address_type") != string(addressType) {
			return ErrorToast(e, http.StatusForbidden, "Address does not belong to this project")
		}

		// Extract and validate fields
		fields := extractAddressFields(e)
		errors := services.ValidateAddress(app, projectID, string(addressType), fields)
		formatErrors := services.ValidateAddressFormat(fields)
		for k, v := range formatErrors {
			errors[k] = v
		}

		if len(errors) > 0 {
			SetToast(e, "warning", "Please fix the errors below")
			requiredFields := services.GetRequiredFields(app, projectID, string(addressType))

			var shipToAddresses []templates.ShipToOption
			var showShipToParent bool
			if addressType == AddressTypeInstallAt {
				showShipToParent, shipToAddresses = fetchShipToOptions(app, projectID)
			}

			data := templates.AddressFormData{
				IsEdit:           true,
				AddressID:        addressID,
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

		// Update the record
		setAddressRecordFields(addressRecord, fields)

		if addressType == AddressTypeInstallAt {
			if parentID := e.Request.FormValue("ship_to_parent"); parentID != "" {
				addressRecord.Set("ship_to_parent", parentID)
			} else {
				addressRecord.Set("ship_to_parent", "")
			}
		}

		if err := app.Save(addressRecord); err != nil {
			log.Printf("address_update: could not save address %s: %v", addressID, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		// Redirect to address list
		slugType := addressTypeToSlug(addressType)
		redirectURL := fmt.Sprintf("/projects/%s/addresses/%s", projectID, slugType)
		SetToast(e, "success", "Address updated successfully")
		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}
