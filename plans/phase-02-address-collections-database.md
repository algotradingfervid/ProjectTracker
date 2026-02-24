# Phase 2: Address & Address Settings Collections

## Overview & Objectives

Create two new PocketBase collections to support the address management system:

1. **`addresses`** -- stores all address records for a project, differentiated by `address_type` (bill_from, ship_from, bill_to, ship_to, install_at). Each project can have multiple addresses of each type.
2. **`project_address_settings`** -- stores per-project, per-address-type configuration for which fields are required. This enables each project to have different mandatory field rules for different address types.

Additionally, install_at addresses can optionally reference a parent ship_to address, supporting the "Ship To equals Install At" relationship at the individual address level.

---

## Files to Create / Modify

| Action | Path | Purpose |
|--------|------|---------|
| Modify | `collections/setup.go` | Add `addresses` and `project_address_settings` collections |
| Create | `collections/migrate_address_settings.go` | Seed default address settings for existing projects |

---

## Detailed Implementation Steps

### Step 1 -- Add `addresses` collection in `collections/setup.go`

Add after the `projects` collection creation (requires `projects.Id` for the relation field).

```go
// ── Addresses ────────────────────────────────────────────────────
addresses := ensureCollection(app, "addresses", func(c *core.Collection) {
    c.Fields.Add(&core.SelectField{
        Name:      "address_type",
        Required:  true,
        Values:    []string{"bill_from", "ship_from", "bill_to", "ship_to", "install_at"},
        MaxSelect: 1,
    })
    c.Fields.Add(&core.RelationField{
        Name:          "project",
        Required:      true,
        CollectionId:  projects.Id,
        CascadeDelete: true,
        MaxSelect:     1,
    })

    // Optional: link install_at address to its parent ship_to address
    // This is a self-relation (addresses -> addresses)
    // Only used when address_type = "install_at"
    // Note: We set CollectionId after saving using ensureField, since
    // the collection ID isn't available during creation.

    // ── Company & contact fields ─────────────────────────────────
    c.Fields.Add(&core.TextField{Name: "company_name", Required: false})
    c.Fields.Add(&core.TextField{Name: "contact_person", Required: false})

    // ── Address fields ───────────────────────────────────────────
    c.Fields.Add(&core.TextField{Name: "address_line_1", Required: false})
    c.Fields.Add(&core.TextField{Name: "address_line_2", Required: false})
    c.Fields.Add(&core.TextField{Name: "city", Required: false})
    c.Fields.Add(&core.TextField{Name: "state", Required: false})
    c.Fields.Add(&core.TextField{Name: "pin_code", Required: false})
    c.Fields.Add(&core.TextField{Name: "country", Required: false})
    c.Fields.Add(&core.TextField{Name: "landmark", Required: false})
    c.Fields.Add(&core.TextField{Name: "district", Required: false})

    // ── Contact fields ───────────────────────────────────────────
    c.Fields.Add(&core.TextField{Name: "phone", Required: false})
    c.Fields.Add(&core.EmailField{Name: "email", Required: false})
    c.Fields.Add(&core.TextField{Name: "fax", Required: false})
    c.Fields.Add(&core.URLField{Name: "website", Required: false})

    // ── Tax / registration fields ────────────────────────────────
    c.Fields.Add(&core.TextField{Name: "gstin", Required: false})
    c.Fields.Add(&core.TextField{Name: "pan", Required: false})
    c.Fields.Add(&core.TextField{Name: "cin", Required: false})

    // ── Timestamps ───────────────────────────────────────────────
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})
```

### Step 1b -- Add self-relation for `ship_to_parent`

The `ship_to_parent` field is a self-relation (addresses -> addresses). Since we need the collection's own ID, add it after the collection is created using the `ensureField` helper from Phase 1:

```go
// Add ship_to_parent self-relation to addresses collection
// This links an install_at address to its parent ship_to address
ensureField(app, "addresses", &core.RelationField{
    Name:          "ship_to_parent",
    Required:      false,
    CollectionId:  addresses.Id,  // self-relation
    CascadeDelete: false,         // don't delete install_at if ship_to is deleted
    MaxSelect:     1,
})
```

### Step 2 -- Add `project_address_settings` collection

This collection stores which address fields are required per project per address type. Each record represents one project + address_type combination.

```go
// ── Project Address Settings ─────────────────────────────────────
ensureCollection(app, "project_address_settings", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{
        Name:          "project",
        Required:      true,
        CollectionId:  projects.Id,
        CascadeDelete: true,
        MaxSelect:     1,
    })
    c.Fields.Add(&core.SelectField{
        Name:      "address_type",
        Required:  true,
        Values:    []string{"bill_from", "ship_from", "bill_to", "ship_to", "install_at"},
        MaxSelect: 1,
    })

    // Each field below is a boolean: true = required for this address type in this project
    c.Fields.Add(&core.BoolField{Name: "req_company_name"})
    c.Fields.Add(&core.BoolField{Name: "req_contact_person"})
    c.Fields.Add(&core.BoolField{Name: "req_address_line_1"})
    c.Fields.Add(&core.BoolField{Name: "req_address_line_2"})
    c.Fields.Add(&core.BoolField{Name: "req_city"})
    c.Fields.Add(&core.BoolField{Name: "req_state"})
    c.Fields.Add(&core.BoolField{Name: "req_pin_code"})
    c.Fields.Add(&core.BoolField{Name: "req_country"})
    c.Fields.Add(&core.BoolField{Name: "req_landmark"})
    c.Fields.Add(&core.BoolField{Name: "req_district"})
    c.Fields.Add(&core.BoolField{Name: "req_phone"})
    c.Fields.Add(&core.BoolField{Name: "req_email"})
    c.Fields.Add(&core.BoolField{Name: "req_fax"})
    c.Fields.Add(&core.BoolField{Name: "req_website"})
    c.Fields.Add(&core.BoolField{Name: "req_gstin"})
    c.Fields.Add(&core.BoolField{Name: "req_pan"})
    c.Fields.Add(&core.BoolField{Name: "req_cin"})

    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})
```

### Step 3 -- Default address settings migration

Create `collections/migrate_address_settings.go` to seed default settings for existing projects:

```go
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
// These can be customized per-project later via the UI.
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
			// Check if settings already exist for this project + address_type
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
				continue // already exists
			}

			// Create default settings record
			record := core.NewRecord(settingsCol)
			record.Set("project", project.Id)
			record.Set("address_type", addrType)

			// Apply defaults
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
```

### Step 4 -- Update `main.go` to call address settings migration

```go
// main.go  (add after MigrateOrphanBOQsToProjects call)
if err := collections.MigrateDefaultAddressSettings(app); err != nil {
    log.Printf("Warning: address settings migration failed: %v", err)
}
```

### Step 5 -- Updated `collections/setup.go` complete function

The full ordering of collection creation in `Setup()` should be:

```go
func Setup(app *pocketbase.PocketBase) {
    // 1. projects (new)
    projects := ensureCollection(app, "projects", func(c *core.Collection) { ... })

    // 2. boqs (existing, now with project relation)
    boqs := ensureCollection(app, "boqs", func(c *core.Collection) { ... })

    // 3. main_boq_items (existing, unchanged)
    mainBOQItems := ensureCollection(app, "main_boq_items", func(c *core.Collection) { ... })

    // 4. sub_items (existing, unchanged)
    subItems := ensureCollection(app, "sub_items", func(c *core.Collection) { ... })

    // 5. sub_sub_items (existing, unchanged)
    ensureCollection(app, "sub_sub_items", func(c *core.Collection) { ... })

    // 6. addresses (new)
    addresses := ensureCollection(app, "addresses", func(c *core.Collection) { ... })

    // 7. Add ship_to_parent self-relation (must be after addresses is created)
    ensureField(app, "addresses", &core.RelationField{
        Name:          "ship_to_parent",
        Required:      false,
        CollectionId:  addresses.Id,
        CascadeDelete: false,
        MaxSelect:     1,
    })

    // 8. Add project field to existing boqs (idempotent)
    ensureField(app, "boqs", &core.RelationField{
        Name:          "project",
        Required:      false,
        CollectionId:  projects.Id,
        CascadeDelete: false,
        MaxSelect:     1,
    })

    // 9. project_address_settings (new)
    ensureCollection(app, "project_address_settings", func(c *core.Collection) { ... })
}
```

---

## Collection Schema Summary

### `addresses` Collection

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `address_type` | SelectField | Yes | Values: `bill_from`, `ship_from`, `bill_to`, `ship_to`, `install_at` |
| `project` | RelationField | Yes | -> `projects`, CascadeDelete=true |
| `ship_to_parent` | RelationField | No | Self-relation -> `addresses`, for install_at only |
| `company_name` | TextField | No | |
| `contact_person` | TextField | No | |
| `address_line_1` | TextField | No | |
| `address_line_2` | TextField | No | |
| `city` | TextField | No | |
| `state` | TextField | No | |
| `pin_code` | TextField | No | Stored as text for leading zeros |
| `country` | TextField | No | |
| `landmark` | TextField | No | |
| `district` | TextField | No | |
| `phone` | TextField | No | |
| `email` | EmailField | No | PocketBase validates email format |
| `fax` | TextField | No | |
| `website` | URLField | No | PocketBase validates URL format |
| `gstin` | TextField | No | 15-character GSTIN |
| `pan` | TextField | No | 10-character PAN |
| `cin` | TextField | No | Corporate Identity Number |
| `created` | AutodateField | -- | |
| `updated` | AutodateField | -- | |

**Design notes on `ship_to_parent`:**
- Only meaningful when `address_type = "install_at"`.
- When the project-level `ship_to_equals_install_at` toggle is `true`, the UI will auto-fill install_at addresses from ship_to data and set this field.
- When the toggle is `false`, install_at addresses are independently managed; `ship_to_parent` can still be set manually for reference.
- `CascadeDelete: false` -- if a ship_to address is deleted, the install_at address survives but loses its parent reference.

### `project_address_settings` Collection

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `project` | RelationField | Yes | -> `projects`, CascadeDelete=true |
| `address_type` | SelectField | Yes | Values: same as addresses |
| `req_company_name` | BoolField | No | true = field is required |
| `req_contact_person` | BoolField | No | |
| `req_address_line_1` | BoolField | No | |
| `req_address_line_2` | BoolField | No | |
| `req_city` | BoolField | No | |
| `req_state` | BoolField | No | |
| `req_pin_code` | BoolField | No | |
| `req_country` | BoolField | No | |
| `req_landmark` | BoolField | No | |
| `req_district` | BoolField | No | |
| `req_phone` | BoolField | No | |
| `req_email` | BoolField | No | |
| `req_fax` | BoolField | No | |
| `req_website` | BoolField | No | |
| `req_gstin` | BoolField | No | |
| `req_pan` | BoolField | No | |
| `req_cin` | BoolField | No | |
| `created` | AutodateField | -- | |
| `updated` | AutodateField | -- | |

**Unique constraint:** Each (project, address_type) pair should be unique. Enforce in the handler layer since PocketBase base collections don't natively support multi-field unique constraints. Alternatively, use a PocketBase `@request` rule or check before insert.

---

## Helper Service: Address Validation

Create `services/address_validation.go` for use by future handlers:

```go
package services

import (
	"fmt"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// AddressField represents a single address field with its DB name and display label.
type AddressField struct {
	Name  string // DB column name, e.g. "company_name"
	Label string // Display label, e.g. "Company Name"
}

// AllAddressFields is the ordered list of all address fields that can be configured.
var AllAddressFields = []AddressField{
	{Name: "company_name", Label: "Company Name"},
	{Name: "contact_person", Label: "Contact Person"},
	{Name: "address_line_1", Label: "Address Line 1"},
	{Name: "address_line_2", Label: "Address Line 2"},
	{Name: "city", Label: "City"},
	{Name: "state", Label: "State"},
	{Name: "pin_code", Label: "PIN Code"},
	{Name: "country", Label: "Country"},
	{Name: "landmark", Label: "Landmark"},
	{Name: "district", Label: "District"},
	{Name: "phone", Label: "Phone"},
	{Name: "email", Label: "Email"},
	{Name: "fax", Label: "Fax"},
	{Name: "website", Label: "Website"},
	{Name: "gstin", Label: "GSTIN"},
	{Name: "pan", Label: "PAN"},
	{Name: "cin", Label: "CIN"},
}

// ValidateAddress checks an address record against the project's required-field settings
// for the given address type. Returns a map of field -> error message for any violations.
func ValidateAddress(app *pocketbase.PocketBase, projectID, addressType string, formData map[string]string) map[string]string {
	errors := make(map[string]string)

	settingsCol, err := app.FindCollectionByNameOrId("project_address_settings")
	if err != nil {
		return errors // can't validate without settings
	}

	settings, err := app.FindRecordsByFilter(
		settingsCol,
		"project = {:projectId} && address_type = {:addrType}",
		"", 1, 0,
		map[string]any{
			"projectId": projectID,
			"addrType":  addressType,
		},
	)
	if err != nil || len(settings) == 0 {
		return errors // no settings found, nothing required
	}

	setting := settings[0]

	for _, field := range AllAddressFields {
		reqFieldName := "req_" + field.Name
		if setting.GetBool(reqFieldName) {
			val := formData[field.Name]
			if val == "" {
				errors[field.Name] = fmt.Sprintf("%s is required", field.Label)
			}
		}
	}

	return errors
}

// GetRequiredFields returns a map of field names that are required for a given
// project + address type. Used by templates to mark required fields in the form.
func GetRequiredFields(app *pocketbase.PocketBase, projectID, addressType string) map[string]bool {
	required := make(map[string]bool)

	settingsCol, err := app.FindCollectionByNameOrId("project_address_settings")
	if err != nil {
		return required
	}

	settings, err := app.FindRecordsByFilter(
		settingsCol,
		"project = {:projectId} && address_type = {:addrType}",
		"", 1, 0,
		map[string]any{
			"projectId": projectID,
			"addrType":  addressType,
		},
	)
	if err != nil || len(settings) == 0 {
		return required
	}

	setting := settings[0]
	for _, field := range AllAddressFields {
		reqFieldName := "req_" + field.Name
		if setting.GetBool(reqFieldName) {
			required[field.Name] = true
		}
	}

	return required
}
```

---

## Data Model Relationships Diagram

```
projects (1) ──────┬──── (*) boqs
                    │
                    ├──── (*) addresses
                    │         ├── address_type: bill_from | ship_from | bill_to | ship_to | install_at
                    │         └── ship_to_parent ──> addresses (self, optional)
                    │
                    └──── (*) project_address_settings
                              └── address_type (one record per type per project)
```

---

## Dependencies on Other Phases

- **Depends on Phase 1:** Requires the `projects` collection to exist (for relation fields).
- **Depends on Phase 1:** Requires the `ensureField` helper function.
- Phase 3 (project CRUD) will use these collections for address management UI.

---

## Testing / Verification Steps

1. **Fresh database:**
   - Delete `pb_data/` and start the app.
   - Verify `addresses` collection exists with all 21+ fields.
   - Verify `project_address_settings` collection exists with 17 `req_*` bool fields.
   - Verify seed project has 5 address settings records (one per type).

2. **Schema verification via PocketBase Admin:**
   - Open `/_/` admin panel.
   - Inspect `addresses` collection: confirm `address_type` select has 5 values.
   - Confirm `project` relation points to `projects`.
   - Confirm `ship_to_parent` relation points to `addresses` (self-relation).
   - Confirm `email` field is EmailField type (validates format).
   - Confirm `website` field is URLField type (validates format).

3. **Address settings defaults:**
   - Create a new project via PocketBase admin.
   - Restart the app (to trigger migration).
   - Verify 5 `project_address_settings` records are created for the new project.
   - Verify `bill_from` settings have `req_company_name = true`, `req_gstin = true`, etc.

4. **Self-relation test:**
   - Create a `ship_to` address for a project.
   - Create an `install_at` address and set `ship_to_parent` to the ship_to address.
   - Verify the relation is saved correctly.
   - Delete the ship_to address and verify the install_at address still exists.

5. **Validation service test (manual):**
   - Call `ValidateAddress` with an empty `company_name` for `bill_from` type.
   - Verify it returns `{"company_name": "Company Name is required"}`.
   - Call with all required fields populated -- should return empty error map.

---

## Acceptance Criteria

- [ ] `addresses` collection exists with all extended fields (company_name through cin, plus address_type, project, ship_to_parent).
- [ ] `project_address_settings` collection exists with 17 `req_*` boolean fields plus project and address_type relations.
- [ ] `ship_to_parent` is a self-relation on addresses, pointing to the same collection.
- [ ] Default address settings are auto-created for every project (5 records per project).
- [ ] Migration is idempotent.
- [ ] `ValidateAddress` service function correctly validates against per-project settings.
- [ ] `GetRequiredFields` service function returns the correct required field map.
- [ ] `AllAddressFields` list is available as a shared constant for templates and handlers.
- [ ] `address_type` select field has exactly 5 values: bill_from, ship_from, bill_to, ship_to, install_at.
- [ ] Cascade delete: deleting a project deletes all its addresses and address settings.
