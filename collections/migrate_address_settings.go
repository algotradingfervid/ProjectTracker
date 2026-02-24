package collections

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// addressTypes is the canonical list of address types used throughout the app.
var addressTypes = []string{"bill_from", "ship_from", "bill_to", "ship_to", "install_at"}

// defaultRequiredFields defines which fields are required by default for each address type.
var defaultRequiredFields = map[string]map[string]bool{
	"bill_from": {
		"req_company_name":   true,
		"req_address_line_1": true,
		"req_city":           true,
		"req_state":          true,
		"req_pin_code":       true,
		"req_country":        true,
		"req_gstin":          true,
	},
	"ship_from": {
		"req_company_name":   true,
		"req_address_line_1": true,
		"req_city":           true,
		"req_state":          true,
		"req_pin_code":       true,
		"req_country":        true,
	},
	"bill_to": {
		"req_company_name":   true,
		"req_contact_person": true,
		"req_address_line_1": true,
		"req_city":           true,
		"req_state":          true,
		"req_pin_code":       true,
		"req_country":        true,
		"req_gstin":          true,
	},
	"ship_to": {
		"req_contact_person": true,
		"req_address_line_1": true,
		"req_city":           true,
		"req_state":          true,
		"req_pin_code":       true,
		"req_country":        true,
		"req_phone":          true,
	},
	"install_at": {
		"req_contact_person": true,
		"req_address_line_1": true,
		"req_city":           true,
		"req_state":          true,
		"req_pin_code":       true,
		"req_country":        true,
		"req_phone":          true,
	},
}

// MigrateDefaultAddressSettings creates default project_address_settings records
// for every project that is missing them. Safe to call on every startup.
func MigrateDefaultAddressSettings(app *pocketbase.PocketBase) error {
	projectsCol, err := app.FindCollectionByNameOrId("projects")
	if err != nil {
		return fmt.Errorf("migrate_settings: could not find projects collection: %w", err)
	}

	settingsCol, err := app.FindCollectionByNameOrId("project_address_settings")
	if err != nil {
		return fmt.Errorf("migrate_settings: could not find project_address_settings collection: %w", err)
	}

	projects, err := app.FindAllRecords(projectsCol)
	if err != nil {
		return fmt.Errorf("migrate_settings: could not query projects: %w", err)
	}

	for _, project := range projects {
		for _, addrType := range addressTypes {
			existing, _ := app.FindRecordsByFilter(
				settingsCol,
				"project = {:projectId} && address_type = {:addrType}",
				"",
				1, 0,
				map[string]any{
					"projectId": project.Id,
					"addrType":  addrType,
				},
			)
			if len(existing) > 0 {
				continue
			}

			record := core.NewRecord(settingsCol)
			record.Set("project", project.Id)
			record.Set("address_type", addrType)

			defaults := defaultRequiredFields[addrType]
			for field, required := range defaults {
				record.Set(field, required)
			}

			if err := app.Save(record); err != nil {
				log.Printf("migrate_settings: failed to create settings for project %s, type %s: %v\n",
					project.Id, addrType, err)
				continue
			}
		}
	}

	return nil
}
