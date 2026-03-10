package collections

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// DefaultColumnDefs returns the default column definitions for an address type,
// matching the current fixed fields.
func DefaultColumnDefs(addressType string) []map[string]any {
	base := []map[string]any{
		{"name": "company_name", "label": "Company Name", "required": true, "type": "text", "fixed": false, "show_in_table": true, "show_in_print": true, "sort_order": 1},
		{"name": "contact_person", "label": "Contact Person", "required": false, "type": "text", "fixed": false, "show_in_table": true, "show_in_print": true, "sort_order": 2},
		{"name": "phone", "label": "Phone", "required": false, "type": "text", "fixed": false, "show_in_table": true, "show_in_print": true, "sort_order": 3},
		{"name": "email", "label": "Email", "required": false, "type": "text", "fixed": false, "show_in_table": false, "show_in_print": false, "sort_order": 4},
		{"name": "gstin", "label": "GSTIN", "required": false, "type": "text", "fixed": false, "show_in_table": true, "show_in_print": true, "sort_order": 5},
		{"name": "pan", "label": "PAN", "required": false, "type": "text", "fixed": false, "show_in_table": false, "show_in_print": false, "sort_order": 6},
		{"name": "cin", "label": "CIN", "required": false, "type": "text", "fixed": false, "show_in_table": false, "show_in_print": false, "sort_order": 7},
		{"name": "address_line_1", "label": "Address Line 1", "required": true, "type": "text", "fixed": false, "show_in_table": true, "show_in_print": true, "sort_order": 8},
		{"name": "address_line_2", "label": "Address Line 2", "required": false, "type": "text", "fixed": false, "show_in_table": false, "show_in_print": true, "sort_order": 9},
		{"name": "landmark", "label": "Landmark", "required": false, "type": "text", "fixed": false, "show_in_table": false, "show_in_print": false, "sort_order": 10},
		{"name": "district", "label": "District", "required": false, "type": "text", "fixed": false, "show_in_table": false, "show_in_print": false, "sort_order": 11},
		{"name": "city", "label": "City", "required": true, "type": "text", "fixed": false, "show_in_table": true, "show_in_print": true, "sort_order": 12},
		{"name": "state", "label": "State", "required": true, "type": "text", "fixed": false, "show_in_table": true, "show_in_print": true, "sort_order": 13},
		{"name": "pin_code", "label": "PIN Code", "required": true, "type": "text", "fixed": false, "show_in_table": true, "show_in_print": true, "sort_order": 14},
		{"name": "country", "label": "Country", "required": true, "type": "text", "fixed": false, "show_in_table": false, "show_in_print": true, "sort_order": 15},
		{"name": "fax", "label": "Fax", "required": false, "type": "text", "fixed": false, "show_in_table": false, "show_in_print": false, "sort_order": 16},
		{"name": "website", "label": "Website", "required": false, "type": "text", "fixed": false, "show_in_table": false, "show_in_print": false, "sort_order": 17},
	}

	// Adjust required fields based on address type (matching old project_address_settings defaults)
	switch addressType {
	case "bill_from":
		setRequired(base, "company_name", "address_line_1", "city", "state", "pin_code", "country", "gstin")
	case "dispatch_from":
		setRequired(base, "company_name", "address_line_1", "city", "state", "pin_code", "country")
	case "bill_to":
		setRequired(base, "company_name", "contact_person", "address_line_1", "city", "state", "pin_code", "country", "gstin")
	case "ship_to", "install_at":
		setRequired(base, "contact_person", "address_line_1", "city", "state", "pin_code", "country", "phone")
	}

	return base
}

func setRequired(cols []map[string]any, names ...string) {
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	for i := range cols {
		cols[i]["required"] = nameSet[cols[i]["name"].(string)]
	}
}

// MigrateAddressesToFlexible migrates addresses from fixed fields to the new
// address_configs + JSON data model. Idempotent — skips if address_configs exist.
func MigrateAddressesToFlexible(app *pocketbase.PocketBase) error {
	// Check if migration already done
	configCol, err := app.FindCollectionByNameOrId("address_configs")
	if err != nil {
		return nil // Collection doesn't exist yet, skip
	}

	projectsCol, err := app.FindCollectionByNameOrId("projects")
	if err != nil {
		return nil
	}

	projects, err := app.FindAllRecords(projectsCol)
	if err != nil {
		return nil
	}

	addressTypes := []string{"bill_from", "dispatch_from", "bill_to", "ship_to", "install_at"}
	fixedFields := []string{
		"company_name", "contact_person", "phone", "email", "gstin", "pan", "cin",
		"address_line_1", "address_line_2", "landmark", "district", "city", "state",
		"pin_code", "country", "fax", "website",
	}

	for _, project := range projects {
		for _, addrType := range addressTypes {
			// Check if config already exists
			existing, _ := app.FindRecordsByFilter(
				configCol, "project = {:pid} && address_type = {:type}",
				"", 1, 0,
				map[string]any{"pid": project.Id, "type": addrType},
			)
			if len(existing) > 0 {
				continue // Already migrated
			}

			// Create address_config record
			configRec := core.NewRecord(configCol)
			configRec.Set("project", project.Id)
			configRec.Set("address_type", addrType)
			columnsJSON, _ := json.Marshal(DefaultColumnDefs(addrType))
			configRec.Set("columns", string(columnsJSON))
			if err := app.Save(configRec); err != nil {
				return fmt.Errorf("failed to create address_config for %s/%s: %w", project.Id, addrType, err)
			}

			// Migrate existing addresses for this project/type
			addressesCol, _ := app.FindCollectionByNameOrId("addresses")
			if addressesCol == nil {
				continue
			}

			// Find addresses that still have the old address_type field
			oldAddresses, _ := app.FindRecordsByFilter(
				addressesCol,
				"project = {:pid} && address_type = {:type}",
				"", 0, 0,
				map[string]any{"pid": project.Id, "type": addrType},
			)

			for _, addr := range oldAddresses {
				// Skip if already migrated (has config relation set)
				if addr.GetString("config") != "" {
					continue
				}

				// Build data JSON from fixed fields
				data := make(map[string]string)
				for _, field := range fixedFields {
					val := addr.GetString(field)
					if val != "" {
						data[field] = val
					}
				}
				dataJSON, _ := json.Marshal(data)

				// Generate address_code
				code := addr.GetString("company_name")
				if code == "" {
					code = addr.Id
				}
				code = strings.ReplaceAll(strings.ToUpper(code), " ", "-")

				addr.Set("config", configRec.Id)
				addr.Set("address_code", code)
				addr.Set("data", string(dataJSON))
				if err := app.Save(addr); err != nil {
					return fmt.Errorf("failed to migrate address %s: %w", addr.Id, err)
				}
			}
		}
	}

	return nil
}
