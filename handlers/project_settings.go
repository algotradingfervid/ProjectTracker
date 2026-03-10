package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

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
			ProjectRef:            project.GetString("reference_number"),
			Errors:                make(map[string]string),
			ShipToEqualsInstallAt: shipToEqualsInstallAt,
		}

		// Address type configs
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

		// PO numbering config
		data.PONumbering = templates.NumberingConfig{
			Prefix:    project.GetString("po_prefix"),
			Format:    project.GetString("po_number_format"),
			Separator: project.GetString("po_separator"),
			Padding:   project.GetInt("po_seq_padding"),
			SeqStart:  project.GetInt("po_seq_start"),
		}
		if data.PONumbering.Padding == 0 {
			data.PONumbering.Padding = 3
		}
		if data.PONumbering.SeqStart == 0 {
			data.PONumbering.SeqStart = 1
		}

		// DC numbering config
		data.DCNumbering = templates.NumberingConfig{
			Prefix:       project.GetString("dc_prefix"),
			Format:       project.GetString("dc_number_format"),
			Separator:    project.GetString("dc_separator"),
			Padding:      project.GetInt("dc_seq_padding"),
			SeqStartTDC:  project.GetInt("dc_seq_start_tdc"),
			SeqStartODC:  project.GetInt("dc_seq_start_odc"),
			SeqStartSTDC: project.GetInt("dc_seq_start_stdc"),
		}
		if data.DCNumbering.Padding == 0 {
			data.DCNumbering.Padding = 3
		}
		if data.DCNumbering.SeqStartTDC == 0 {
			data.DCNumbering.SeqStartTDC = 1
		}
		if data.DCNumbering.SeqStartODC == 0 {
			data.DCNumbering.SeqStartODC = 1
		}
		if data.DCNumbering.SeqStartSTDC == 0 {
			data.DCNumbering.SeqStartSTDC = 1
		}

		// Default addresses
		data.DefaultBillFromID = project.GetString("default_bill_from")
		data.DefaultDispatchFromID = project.GetString("default_dispatch_from")
		data.BillFromAddresses = fetchDefaultAddressOptions(app, projectID, "bill_from")
		data.DispatchFromAddresses = fetchDefaultAddressOptions(app, projectID, "dispatch_from")

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

		// Save address config columns
		for _, addrType := range addressTypeOrder {
			configRec, colDefs := getOrCreateAddressConfig(app, projectID, AddressType(addrType))
			if configRec == nil {
				continue
			}

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

			syncLegacyAddressSettings(app, projectID, addrType, colDefs)
		}

		// Save PO numbering config
		project.Set("po_prefix", e.Request.FormValue("po_prefix"))
		project.Set("po_number_format", e.Request.FormValue("po_number_format"))
		project.Set("po_separator", e.Request.FormValue("po_separator"))
		project.Set("po_seq_padding", formInt(e.Request.FormValue("po_seq_padding"), 3))
		project.Set("po_seq_start", formInt(e.Request.FormValue("po_seq_start"), 1))

		// Save DC numbering config
		project.Set("dc_prefix", e.Request.FormValue("dc_prefix"))
		project.Set("dc_number_format", e.Request.FormValue("dc_number_format"))
		project.Set("dc_separator", e.Request.FormValue("dc_separator"))
		project.Set("dc_seq_padding", formInt(e.Request.FormValue("dc_seq_padding"), 3))
		project.Set("dc_seq_start_tdc", formInt(e.Request.FormValue("dc_seq_start_tdc"), 1))
		project.Set("dc_seq_start_odc", formInt(e.Request.FormValue("dc_seq_start_odc"), 1))
		project.Set("dc_seq_start_stdc", formInt(e.Request.FormValue("dc_seq_start_stdc"), 1))

		// Save default addresses
		project.Set("default_bill_from", e.Request.FormValue("default_bill_from"))
		project.Set("default_dispatch_from", e.Request.FormValue("default_dispatch_from"))

		if err := app.Save(project); err != nil {
			log.Printf("project_settings_save: failed to save project fields: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
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

// formInt parses a form value as int, returning defaultVal if empty or invalid.
func formInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}

// fetchDefaultAddressOptions returns address select items for the default address pickers.
func fetchDefaultAddressOptions(app *pocketbase.PocketBase, projectID, addressType string) []templates.DefaultAddressOption {
	// For dispatch_from, the DB type might be "ship_from" (legacy) — handle both
	dbType := addressType
	records, err := app.FindRecordsByFilter(
		"addresses",
		"project = {:pid} && address_type = {:type}",
		"created", 0, 0,
		map[string]any{"pid": projectID, "type": dbType},
	)
	if err != nil || len(records) == 0 {
		// Try legacy ship_from if dispatch_from found nothing
		if addressType == "dispatch_from" {
			records, _ = app.FindRecordsByFilter(
				"addresses",
				"project = {:pid} && address_type = 'ship_from'",
				"created", 0, 0,
				map[string]any{"pid": projectID},
			)
		}
	}

	var options []templates.DefaultAddressOption
	for _, rec := range records {
		data := readAddressData(rec)
		options = append(options, templates.DefaultAddressOption{
			ID:          rec.Id,
			CompanyName: data["company_name"],
			City:        data["city"],
		})
	}
	return options
}
