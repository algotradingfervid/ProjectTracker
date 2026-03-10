package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// addressFieldDefs defines the order and labels for address fields shown in settings.
var addressFieldDefs = []struct {
	Field string
	Label string
}{
	{"req_company_name", "Company Name"},
	{"req_contact_person", "Contact Person"},
	{"req_address_line_1", "Address Line 1"},
	{"req_address_line_2", "Address Line 2"},
	{"req_city", "City"},
	{"req_state", "State"},
	{"req_pin_code", "PIN Code"},
	{"req_country", "Country"},
	{"req_landmark", "Landmark"},
	{"req_district", "District"},
	{"req_phone", "Phone"},
	{"req_email", "Email"},
	{"req_fax", "Fax"},
	{"req_website", "Website"},
	{"req_gstin", "GSTIN"},
	{"req_pan", "PAN"},
	{"req_cin", "CIN"},
}

// addressTypeLabels maps address type values to display labels.
var addressTypeLabels = map[string]string{
	"bill_from":     "BILL FROM",
	"ship_from":     "SHIP FROM",
	"dispatch_from": "DISPATCH FROM",
	"bill_to":       "BILL TO",
	"ship_to":       "SHIP TO",
	"install_at":    "INSTALL AT",
}

var addressTypeOrder = []string{"bill_from", "dispatch_from", "bill_to", "ship_to", "install_at"}

func HandleProjectSettings(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("id")
		if projectID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing project ID")
		}

		project, err := app.FindRecordById("projects", projectID)
		if err != nil {
			log.Printf("project_settings: could not find project %s: %v", projectID, err)
			return ErrorToast(e, http.StatusNotFound, "Project not found")
		}

		shipToEqualsInstallAt := project.GetBool("ship_to_equals_install_at")

		data := templates.ProjectSettingsData{
			ProjectID:             projectID,
			ProjectName:           project.GetString("name"),
			Errors:                make(map[string]string),
			ShipToEqualsInstallAt: shipToEqualsInstallAt,
		}

		for _, addrType := range addressTypeOrder {
			_, colDefs := getOrCreateAddressConfig(app, projectID, AddressType(addrType))

			var columns []templates.AddressColumnConfig
			for _, col := range colDefs {
				columns = append(columns, templates.AddressColumnConfig{
					Name:        col.Name,
					Label:       col.Label,
					Required:    col.Required,
					ShowInTable: col.ShowInTable,
					ShowInPrint: col.ShowInPrint,
					SortOrder:   col.SortOrder,
				})
			}

			data.AddressTypes = append(data.AddressTypes, templates.AddressTypeConfig{
				Type:    addrType,
				Label:   addressTypeLabels[addrType],
				Columns: columns,
			})
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.ProjectSettingsContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.ProjectSettingsPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

func HandleProjectSettingsSave(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("id")
		if projectID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing project ID")
		}

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		project, err := app.FindRecordById("projects", projectID)
		if err != nil {
			log.Printf("project_settings_save: could not find project %s: %v", projectID, err)
			return ErrorToast(e, http.StatusNotFound, "Project not found")
		}
		shipToEqualsInstallAt := e.Request.FormValue("ship_to_equals_install_at") == "on" ||
			e.Request.FormValue("ship_to_equals_install_at") == "true"
		project.Set("ship_to_equals_install_at", shipToEqualsInstallAt)
		if err := app.Save(project); err != nil {
			log.Printf("project_settings_save: failed to save ship_to toggle: %v", err)
		}

		for _, addrType := range addressTypeOrder {
			configRec, colDefs := getOrCreateAddressConfig(app, projectID, AddressType(addrType))
			if configRec == nil {
				continue
			}

			// Update column definitions from form data
			for i, col := range colDefs {
				colDefs[i].Required = e.Request.FormValue(addrType+"."+col.Name+".required") == "true"
				colDefs[i].ShowInTable = e.Request.FormValue(addrType+"."+col.Name+".show_in_table") == "true"
				colDefs[i].ShowInPrint = e.Request.FormValue(addrType+"."+col.Name+".show_in_print") == "true"
			}

			columnsJSON, _ := json.Marshal(colDefs)
			configRec.Set("columns", string(columnsJSON))
			if err := app.Save(configRec); err != nil {
				log.Printf("project_settings_save: failed to save address_config for %s/%s: %v", projectID, addrType, err)
				return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
			}

			// Also update legacy project_address_settings for backward compat
			syncLegacyAddressSettings(app, projectID, addrType, colDefs)
		}

		SetToast(e, "success", "Settings saved")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Trigger", "settings-saved")
			// Re-render the settings page content to reflect saved state
			// Redirect back to settings page
			e.Response.Header().Set("HX-Redirect", fmt.Sprintf("/projects/%s/settings", projectID))
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, fmt.Sprintf("/projects/%s/settings", projectID))
	}
}

// syncLegacyAddressSettings updates the old project_address_settings table
// to keep backward compatibility with existing validation code.
func syncLegacyAddressSettings(app *pocketbase.PocketBase, projectID, addrType string, colDefs []services.ColumnDef) {
	settingsCol, err := app.FindCollectionByNameOrId("project_address_settings")
	if err != nil {
		return
	}

	// Map dispatch_from back to ship_from for legacy settings
	legacyType := addrType
	if addrType == "dispatch_from" {
		legacyType = "ship_from"
	}

	records, _ := app.FindRecordsByFilter(
		settingsCol,
		"project = {:pid} && address_type = {:atype}",
		"", 1, 0,
		map[string]any{"pid": projectID, "atype": legacyType},
	)

	var record *core.Record
	if len(records) > 0 {
		record = records[0]
	} else {
		record = core.NewRecord(settingsCol)
		record.Set("project", projectID)
		record.Set("address_type", legacyType)
	}

	for _, col := range colDefs {
		reqField := "req_" + col.Name
		record.Set(reqField, col.Required)
	}

	if err := app.Save(record); err != nil {
		log.Printf("syncLegacyAddressSettings: failed for %s/%s: %v", projectID, addrType, err)
	}
}
