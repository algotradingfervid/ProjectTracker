# Phase 16: Project Settings & Required Field Configuration

## Overview & Objectives

Create a project settings page that allows per-project, per-address-type configuration of which address fields are required. Each of the 5 address types (Bill From, Ship From, Bill To, Ship To, Install At) has a set of address fields. The settings page presents these as a checklist where checked means required and unchecked means optional. A "Ship To = Install At" toggle also lives on this page.

Settings are stored in a `project_address_settings` PocketBase collection and are loaded by address forms and the validation engine at runtime.

---

## Files to Create/Modify

| Action | Path |
|--------|------|
| **Modify** | `collections/setup.go` (add `project_address_settings` collection) |
| **Create** | `handlers/project_settings.go` |
| **Create** | `templates/project_settings.templ` |
| **Modify** | `main.go` (register settings routes) |
| **Modify** | `templates/sidebar.templ` (add Settings link under project nav) |
| **Modify** | Address form handler/template (load required-field config for validation) |

---

## Detailed Implementation Steps

### Step 1: Create the `project_address_settings` Collection

Update `collections/setup.go` to add the settings collection. Each record stores the required-field configuration for one address type within one project.

```go
// In Setup(), after the projects collection:
ensureCollection(app, "project_address_settings", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{
        Name:         "project",
        Required:     true,
        CollectionId: projects.Id,
        MaxSelect:    1,
    })
    c.Fields.Add(&core.SelectField{
        Name:      "address_type",
        Required:  true,
        Values:    []string{"bill_from", "ship_from", "bill_to", "ship_to", "install_at"},
        MaxSelect: 1,
    })
    // JSON field storing the list of required field names
    // e.g., ["company_name", "address_line_1", "city", "state", "pin_code"]
    c.Fields.Add(&core.JSONField{Name: "required_fields", Required: true, MaxSize: 5000})
    // Ship To = Install At toggle (only meaningful for the project, stored once)
    c.Fields.Add(&core.BoolField{Name: "ship_to_equals_install_at", Required: false})
})

// Add unique index on (project, address_type) to prevent duplicates
// This is done via a rule or application logic since PocketBase
// doesn't support compound unique indexes directly on creation.
```

### Step 2: Define Default Required Fields and All Available Fields

In `handlers/project_settings.go`:

```go
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

// AddressField represents a single field that can be toggled as required/optional.
type AddressField struct {
	Name  string // PocketBase field name
	Label string // Human-readable label
}

// AllAddressFields returns the complete list of address fields available for configuration.
func AllAddressFields() []AddressField {
	return []AddressField{
		{Name: "company_name", Label: "Company Name"},
		{Name: "contact_person", Label: "Contact Person"},
		{Name: "email", Label: "Email"},
		{Name: "phone", Label: "Phone"},
		{Name: "address_line_1", Label: "Address Line 1"},
		{Name: "address_line_2", Label: "Address Line 2"},
		{Name: "city", Label: "City"},
		{Name: "state", Label: "State"},
		{Name: "pin_code", Label: "PIN Code"},
		{Name: "country", Label: "Country"},
		{Name: "gstin", Label: "GSTIN"},
	}
}

// DefaultRequiredFields returns the field names that are required by default
// when no settings have been configured yet.
var DefaultRequiredFields = []string{
	"company_name",
	"address_line_1",
	"city",
	"state",
	"pin_code",
}

// AddressTypeConfig holds the settings display data for one address type section.
type AddressTypeConfig struct {
	Type           string         // "bill_from", etc.
	Label          string         // "Bill From", etc.
	Fields         []FieldConfig  // All fields with their required state
}

// FieldConfig represents a single field's configuration state.
type FieldConfig struct {
	Name       string
	Label      string
	IsRequired bool
}

// All 5 address types in display order
var addressTypeOrder = []struct {
	Type  string
	Label string
}{
	{"bill_from", "Bill From"},
	{"ship_from", "Ship From"},
	{"bill_to", "Bill To"},
	{"ship_to", "Ship To"},
	{"install_at", "Install At"},
}
```

### Step 3: Implement the GET Settings Handler

```go
// HandleProjectSettings renders the project settings page.
// GET /projects/{projectId}/settings
func HandleProjectSettings(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		if projectID == "" {
			return e.String(http.StatusBadRequest, "Missing project ID")
		}

		project, err := app.FindRecordById("projects", projectID)
		if err != nil {
			return e.String(http.StatusNotFound, "Project not found")
		}

		projectName := project.GetString("name")

		// Load existing settings for this project
		settingsCol, err := app.FindCollectionByNameOrId("project_address_settings")
		if err != nil {
			log.Printf("settings: collection not found: %v", err)
			return e.String(http.StatusInternalServerError, "Settings collection not found")
		}

		existingSettings, err := app.FindRecordsByFilter(
			settingsCol,
			"project = {:projectId}",
			"",
			0, 0,
			map[string]any{"projectId": projectID},
		)
		if err != nil {
			existingSettings = nil
		}

		// Build a map of address_type -> set of required field names
		settingsMap := make(map[string]map[string]bool)
		shipToEqualsInstallAt := false

		for _, rec := range existingSettings {
			addrType := rec.GetString("address_type")
			var requiredFields []string
			raw := rec.Get("required_fields")
			if rawBytes, err := json.Marshal(raw); err == nil {
				json.Unmarshal(rawBytes, &requiredFields)
			}
			fieldSet := make(map[string]bool)
			for _, f := range requiredFields {
				fieldSet[f] = true
			}
			settingsMap[addrType] = fieldSet

			// Ship To = Install At flag (read from any record, typically stored on project level)
			if rec.GetBool("ship_to_equals_install_at") {
				shipToEqualsInstallAt = true
			}
		}

		// Build template data: 5 sections, each with all fields and their required state
		allFields := AllAddressFields()
		var sections []templates.AddressTypeSection

		for _, at := range addressTypeOrder {
			var fields []templates.FieldSetting
			requiredSet, hasSettings := settingsMap[at.Type]

			for _, f := range allFields {
				isRequired := false
				if hasSettings {
					isRequired = requiredSet[f.Name]
				} else {
					// Use defaults if no settings saved yet
					for _, df := range DefaultRequiredFields {
						if df == f.Name {
							isRequired = true
							break
						}
					}
				}
				fields = append(fields, templates.FieldSetting{
					Name:       f.Name,
					Label:      f.Label,
					IsRequired: isRequired,
				})
			}

			sections = append(sections, templates.AddressTypeSection{
				Type:   at.Type,
				Label:  at.Label,
				Fields: fields,
			})
		}

		data := templates.ProjectSettingsData{
			ProjectID:             projectID,
			ProjectName:           projectName,
			Sections:              sections,
			ShipToEqualsInstallAt: shipToEqualsInstallAt,
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.ProjectSettingsContent(data)
		} else {
			component = templates.ProjectSettingsPage(data)
		}

		return component.Render(e.Request.Context(), e.Response)
	}
}
```

### Step 4: Implement the POST Settings Handler

```go
// HandleProjectSettingsSave saves the required field configuration.
// POST /projects/{projectId}/settings
// Form data contains fields like:
//   bill_from_fields=company_name&bill_from_fields=city&...
//   ship_from_fields=company_name&...
//   ship_to_equals_install_at=on
func HandleProjectSettingsSave(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		if projectID == "" {
			return e.String(http.StatusBadRequest, "Missing project ID")
		}

		// Validate project exists
		_, err := app.FindRecordById("projects", projectID)
		if err != nil {
			return e.String(http.StatusNotFound, "Project not found")
		}

		settingsCol, err := app.FindCollectionByNameOrId("project_address_settings")
		if err != nil {
			return e.String(http.StatusInternalServerError, "Settings collection not found")
		}

		// Parse form data
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "Invalid form data")
		}

		shipToEqualsInstallAt := e.Request.FormValue("ship_to_equals_install_at") == "on"

		// For each address type, get the checked fields from the form
		for _, at := range addressTypeOrder {
			formKey := at.Type + "_fields"
			requiredFields := e.Request.Form[formKey] // []string of checked field names

			if requiredFields == nil {
				requiredFields = []string{} // no fields checked
			}

			// Find existing setting record or create new one
			existing, err := app.FindRecordsByFilter(
				settingsCol,
				"project = {:projectId} && address_type = {:addrType}",
				"",
				1, 0,
				map[string]any{"projectId": projectID, "addrType": at.Type},
			)

			var record *core.Record
			if err == nil && len(existing) > 0 {
				record = existing[0]
			} else {
				record = core.NewRecord(settingsCol)
				record.Set("project", projectID)
				record.Set("address_type", at.Type)
			}

			record.Set("required_fields", requiredFields)
			record.Set("ship_to_equals_install_at", shipToEqualsInstallAt)

			if err := app.Save(record); err != nil {
				log.Printf("settings: failed to save %s settings: %v", at.Type, err)
				return e.String(http.StatusInternalServerError,
					fmt.Sprintf("Failed to save %s settings", at.Label))
			}
		}

		// HTMX response: show success feedback
		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Trigger", "settings-saved")
			// Re-render the settings content with a success message
			return e.String(http.StatusOK, "")
		}

		return e.Redirect(http.StatusFound,
			fmt.Sprintf("/projects/%s/settings", projectID))
	}
}
```

### Step 5: Create a Helper to Load Required Fields at Runtime

This helper is used by address form handlers and validation:

```go
// GetRequiredFieldsForType returns the set of required field names for a given
// address type within a project. Falls back to DefaultRequiredFields if no
// settings are configured.
func GetRequiredFieldsForType(app *pocketbase.PocketBase, projectID, addressType string) ([]string, error) {
	settingsCol, err := app.FindCollectionByNameOrId("project_address_settings")
	if err != nil {
		return DefaultRequiredFields, nil
	}

	records, err := app.FindRecordsByFilter(
		settingsCol,
		"project = {:projectId} && address_type = {:addrType}",
		"",
		1, 0,
		map[string]any{"projectId": projectID, "addrType": addressType},
	)
	if err != nil || len(records) == 0 {
		return DefaultRequiredFields, nil
	}

	var fields []string
	raw := records[0].Get("required_fields")
	if rawBytes, err := json.Marshal(raw); err == nil {
		json.Unmarshal(rawBytes, &fields)
	}

	if len(fields) == 0 {
		return DefaultRequiredFields, nil
	}

	return fields, nil
}

// IsShipToEqualsInstallAt checks if the project has the Ship To = Install At toggle enabled.
func IsShipToEqualsInstallAt(app *pocketbase.PocketBase, projectID string) bool {
	settingsCol, err := app.FindCollectionByNameOrId("project_address_settings")
	if err != nil {
		return false
	}

	records, err := app.FindRecordsByFilter(
		settingsCol,
		"project = {:projectId} && ship_to_equals_install_at = true",
		"",
		1, 0,
		map[string]any{"projectId": projectID},
	)

	return err == nil && len(records) > 0
}
```

### Step 6: Create the Settings Template

Create `templates/project_settings.templ`:

```go
package templates

import "fmt"

// FieldSetting represents a single field's required/optional state.
type FieldSetting struct {
    Name       string
    Label      string
    IsRequired bool
}

// AddressTypeSection represents one address type's settings block.
type AddressTypeSection struct {
    Type   string
    Label  string
    Fields []FieldSetting
}

// ProjectSettingsData holds all data for the settings page.
type ProjectSettingsData struct {
    ProjectID             string
    ProjectName           string
    Sections              []AddressTypeSection
    ShipToEqualsInstallAt bool
}

templ ProjectSettingsContent(data ProjectSettingsData) {
    <!-- Breadcrumb -->
    <nav class="flex items-center gap-2 mb-6"
         style="font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
        <a href={ templ.SafeURL(fmt.Sprintf("/projects/%s", data.ProjectID)) }
           style="color: var(--text-secondary); text-decoration: none;">
            { data.ProjectName }
        </a>
        <span>/</span>
        <span style="color: var(--text-primary);">Settings</span>
    </nav>

    <!-- Page Header -->
    <div class="flex justify-between items-center mb-8">
        <div>
            <h1 style="font-family: 'Space Grotesk', sans-serif; font-size: 36px; font-weight: 700; color: var(--text-primary); margin: 0;">
                Project Settings
            </h1>
            <p style="font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-secondary); margin-top: 8px;">
                Configure required fields for each address type
            </p>
        </div>
    </div>

    <!-- Settings Form -->
    <form
        hx-post={ fmt.Sprintf("/projects/%s/settings", data.ProjectID) }
        hx-target="#main-content"
        hx-swap="innerHTML"
        x-data="{ saved: false }"
        @settings-saved.window="saved = true; setTimeout(() => saved = false, 3000)"
    >
        <!-- Success notification -->
        <div x-show="saved" x-transition
             class="fixed top-4 right-4 z-50 flex items-center gap-2 px-4 py-3"
             style="background-color: var(--success); color: white; font-family: 'Inter', sans-serif; font-size: 14px;">
            Settings saved successfully
        </div>

        <!-- Ship To = Install At Toggle -->
        <div style="background-color: var(--bg-card); padding: 24px; margin-bottom: 24px;">
            <label class="flex items-center gap-3 cursor-pointer">
                <input
                    type="checkbox"
                    name="ship_to_equals_install_at"
                    if data.ShipToEqualsInstallAt {
                        checked
                    }
                    class="checkbox checkbox-sm"
                />
                <div>
                    <span style="font-family: 'Space Grotesk', sans-serif; font-size: 14px; font-weight: 600; color: var(--text-primary);">
                        Ship To = Install At
                    </span>
                    <p style="font-family: 'Inter', sans-serif; font-size: 12px; color: var(--text-secondary); margin-top: 2px;">
                        When enabled, the Ship To address is automatically used as the Install At address
                    </p>
                </div>
            </label>
        </div>

        <!-- Address Type Sections -->
        for _, section := range data.Sections {
            <div style="background-color: var(--bg-card); padding: 24px; margin-bottom: 16px;">
                <h2 style="font-family: 'Space Grotesk', sans-serif; font-size: 18px; font-weight: 600; color: var(--text-primary); margin-bottom: 16px; padding-bottom: 12px; border-bottom: 1px solid var(--border-light);">
                    { section.Label }
                </h2>
                <div class="grid grid-cols-2 gap-3" style="max-width: 600px;">
                    for _, field := range section.Fields {
                        <label class="flex items-center gap-3 cursor-pointer" style="padding: 8px 0;">
                            <input
                                type="checkbox"
                                name={ section.Type + "_fields" }
                                value={ field.Name }
                                if field.IsRequired {
                                    checked
                                }
                                class="checkbox checkbox-sm"
                            />
                            <span style="font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-primary);">
                                { field.Label }
                            </span>
                        </label>
                    }
                </div>
            </div>
        }

        <!-- Save Button -->
        <div class="flex justify-end" style="margin-top: 24px;">
            <button
                type="submit"
                class="flex items-center hover:opacity-90"
                style="background-color: var(--bg-sidebar); padding: 12px 24px; gap: 8px; border: none; cursor: pointer;"
            >
                <span style="font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 600; letter-spacing: 1px; color: var(--text-light);">
                    SAVE SETTINGS
                </span>
            </button>
        </div>
    </form>
}

templ ProjectSettingsPage(data ProjectSettingsData) {
    @PageShell("Settings — " + data.ProjectName, data.ProjectID) {
        @ProjectSettingsContent(data)
    }
}
```

### Step 7: Register Routes in `main.go`

```go
// Project settings
se.Router.GET("/projects/{projectId}/settings", handlers.HandleProjectSettings(app))
se.Router.POST("/projects/{projectId}/settings", handlers.HandleProjectSettingsSave(app))
```

### Step 8: Add Settings Link to Sidebar

Update `templates/sidebar.templ` to add a Settings link under the project navigation. Replace the existing static Settings link:

```go
<!-- Settings (project-scoped) -->
<a href={ templ.SafeURL(fmt.Sprintf("/projects/%s/settings", projectID)) }
   hx-get={ fmt.Sprintf("/projects/%s/settings", projectID) }
   hx-target="#main-content"
   hx-push-url="true"
   class="flex items-center"
   style="gap: 12px; padding: 14px 0; border-top: 1px solid var(--border-dark);">
    <svg style="width: 20px; height: 20px; color: #666666;" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"></path>
        <circle cx="12" cy="12" r="3"></circle>
    </svg>
    <span style="color: #666666; font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 500; letter-spacing: 1px;">SETTINGS</span>
</a>
```

### Step 9: Integrate Required Fields into Address Form Validation

When rendering an address form, load the required fields for the address type and mark form inputs accordingly:

```go
// In the address form handler:
requiredFields, _ := handlers.GetRequiredFieldsForType(app, projectID, addrType)

// Build a set for quick lookup
requiredSet := make(map[string]bool)
for _, f := range requiredFields {
    requiredSet[f] = true
}

// Pass to template
data := templates.AddressFormData{
    // ...
    RequiredFields: requiredSet,
}
```

In the address form template, use the required set to add the HTML `required` attribute and a visual indicator:

```go
// For each form field:
<div>
    <label style="...">
        { field.Label }
        if data.RequiredFields[field.Name] {
            <span style="color: var(--error);">*</span>
        }
    </label>
    <input
        type="text"
        name={ field.Name }
        value={ field.Value }
        if data.RequiredFields[field.Name] {
            required
        }
        style="..."
    />
</div>
```

Server-side validation in the address save handler should also check against the configured required fields:

```go
// In address save handler:
requiredFields, _ := GetRequiredFieldsForType(app, projectID, addrType)
for _, fieldName := range requiredFields {
    value := e.Request.FormValue(fieldName)
    if strings.TrimSpace(value) == "" {
        return e.String(http.StatusBadRequest,
            fmt.Sprintf("%s is required", fieldName))
    }
}
```

---

## Dependencies on Other Phases

- **Phase 10 (assumed)**: The `projects` collection must exist.
- **Phase 12 (assumed)**: Address form templates and handlers must exist to integrate the required-field configuration.
- **Phase 15**: The `PageShell` templ function signature is updated in Phase 15 to accept `projectID`. This phase depends on that change for `ProjectSettingsPage`.
- The sidebar update in this phase and Phase 15 must be coordinated so the `Sidebar` component receives `projectID`.

---

## Testing / Verification Steps

1. **Settings page rendering**:
   - Navigate to `/projects/{projectId}/settings`.
   - Verify all 5 address type sections are displayed.
   - Verify each section lists all 11 address fields as checkboxes.
   - Verify default required fields (Company Name, Address Line 1, City, State, PIN Code) are pre-checked when no settings have been saved.

2. **Saving settings**:
   - Uncheck "City" for Bill From and check "GSTIN". Click Save.
   - Verify success toast notification appears.
   - Reload the page and verify the changes persisted (City unchecked, GSTIN checked for Bill From).
   - Verify other address types were not affected.

3. **Ship To = Install At toggle**:
   - Enable the toggle and save. Reload and verify it is still enabled.
   - Disable it, save, and verify it is disabled.

4. **Form integration**:
   - Configure Bill To to require "Email" (check it in settings).
   - Go to create a new Bill To address.
   - Verify the Email field shows a required indicator (`*`).
   - Try to submit the form without Email. Verify server-side validation rejects it.
   - Fill in Email and verify submission succeeds.

5. **Default behavior**:
   - For a new project with no saved settings, verify the address form uses default required fields.
   - After saving settings once, verify the saved settings override defaults.

6. **Sidebar link**:
   - Verify the Settings link in the sidebar navigates to `/projects/{projectId}/settings`.
   - Verify HTMX partial navigation works (content swaps without full page reload).

7. **HTMX behavior**:
   - Save settings via HTMX form submit. Verify no full page reload.
   - Verify the `HX-Trigger: settings-saved` event fires and the Alpine.js toast shows.

8. **Idempotency**:
   - Save the same settings twice. Verify no duplicate records are created in `project_address_settings`.
   - Check PocketBase admin: each project should have at most 5 settings records (one per address type).

---

## Acceptance Criteria

- [ ] `GET /projects/{projectId}/settings` renders a settings page with 5 address type sections.
- [ ] Each section displays all address fields as checkboxes (checked = required, unchecked = optional).
- [ ] Default required fields are: Company Name, Address Line 1, City, State, PIN Code.
- [ ] `POST /projects/{projectId}/settings` saves the checked fields per address type to `project_address_settings` collection.
- [ ] Settings are upserted: saving again updates existing records rather than creating duplicates.
- [ ] Ship To = Install At toggle is present on the settings page and its state is persisted.
- [ ] Success feedback is shown via HTMX trigger and Alpine.js toast after saving.
- [ ] Address forms load the configured required fields and display required indicators.
- [ ] Server-side validation enforces the configured required fields when saving addresses.
- [ ] When no settings are configured for a project, default required fields are used.
- [ ] Settings link is visible in the sidebar under project navigation.
- [ ] Sidebar Settings link uses HTMX for partial navigation.
- [ ] The `project_address_settings` collection is created by `collections/setup.go` on startup.
- [ ] The settings page follows the same visual patterns as other pages (Space Grotesk headings, Inter body text, card backgrounds).
