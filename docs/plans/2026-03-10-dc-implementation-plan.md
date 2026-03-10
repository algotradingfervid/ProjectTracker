# Delivery Challan Integration — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate Delivery Challan management into the ProjectCreation app, including address restructure, configurable numbering, DC templates, transporters, unified DC wizard, split workflow, and PDF/Excel exports.

**Architecture:** PocketBase collections created idempotently in `collections/setup.go`. Handlers follow closure pattern over `*pocketbase.PocketBase`. Templates use three-component pattern (Data struct + Content partial + Page full). All navigation via HTMX with `hx-target="#main-content"`.

**Tech Stack:** Go + PocketBase + Templ + HTMX + Alpine.js + Tailwind CSS v4 + DaisyUI v5 + maroto (PDF) + excelize (Excel)

**Design Doc:** `docs/plans/2026-03-10-dc-integration-design.md`

---

## Progress Tracker

| Phase | Status | Tasks Done | Next Task |
|-------|--------|------------|-----------|
| 1: Address Restructure | **In Progress** | 1.1, 1.2, 1.3, 1.4 | 1.5: Update address handlers |
| 2: Numbering System | Not started | — | 2.1: Create number_sequences collection |
| 3: DC Master Data | Not started | — | 3.1: Create DC collections |
| 4: DC Wizard | Not started | — | 4.1: Wizard step 1 |
| 5: DC Lifecycle | Not started | — | 5.1: DC list view |
| 6: Exports | Not started | — | 6.1: DC PDF export |
| 7: Shipment Groups | Not started | — | 7.1: Group detail |
| 8: Testing & Polish | Not started | — | 8.1: Integration tests |

**Resume from:** Task 1.5 — Update Address Handlers for Flexible Schema

---

## Phase 1: Address System Restructure

### Task 1.1: Create `address_configs` Collection ✅ DONE

**Files:**
- Modify: `collections/setup.go`

**Step 1: Add `address_configs` collection definition**

In `collections/setup.go`, inside `Setup()`, add after the existing `project_address_settings` block:

```go
// Address configs — flexible column definitions per address type per project
addressConfigsCol := ensureCollection(app, "address_configs", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{
        Name:         "project",
        Required:     true,
        CollectionId: projectsCol.Id,
        CascadeDelete: true,
        MaxSelect:    1,
    })
    c.Fields.Add(&core.SelectField{
        Name:      "address_type",
        Required:  true,
        Values:    []string{"bill_from", "dispatch_from", "bill_to", "ship_to", "install_at"},
        MaxSelect: 1,
    })
    c.Fields.Add(&core.JSONField{
        Name:     "columns",
        Required: true,
        MaxSize:  10000,
    })
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})
```

**Step 2: Run `make run` to verify collection is created**

Run: `go run main.go serve`
Expected: Server starts, `address_configs` collection visible in PocketBase admin at `http://localhost:8090/_/`

**Step 3: Commit**

```bash
git add collections/setup.go
git commit -m "feat: add address_configs collection for flexible address columns"
```

---

### Task 1.2: Restructure `addresses` Collection ✅ DONE

**Files:**
- Modify: `collections/setup.go`

**Step 1: Replace the existing `addresses` collection definition**

Replace the current `addresses` `ensureCollection` block with:

```go
addressesCol := ensureCollection(app, "addresses", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{
        Name:          "config",
        Required:      true,
        CollectionId:  addressConfigsCol.Id,
        CascadeDelete: true,
        MaxSelect:     1,
    })
    c.Fields.Add(&core.TextField{
        Name:     "address_code",
        Required: true,
    })
    c.Fields.Add(&core.JSONField{
        Name:     "data",
        Required: true,
        MaxSize:  50000,
    })
    // Fixed fields for ship_to / install_at
    c.Fields.Add(&core.TextField{Name: "district_name"})
    c.Fields.Add(&core.TextField{Name: "mandal_name"})
    c.Fields.Add(&core.TextField{Name: "mandal_code"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})
```

Note: Keep the old `addresses` definition commented out temporarily during migration. The `ensureCollection` pattern will add new fields to the existing collection, but we need a migration step (Task 1.3) to move data before removing old fields.

**Step 2: Commit**

```bash
git add collections/setup.go
git commit -m "feat: restructure addresses collection for flexible JSON data"
```

---

### Task 1.3: Write Address Data Migration ✅ DONE

**Files:**
- Create: `collections/migrate_addresses.go`

**Step 1: Write migration function**

```go
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
```

**Step 2: Register migration in `main.go`**

Add after `collections.MigrateDefaultAddressSettings(app)`:

```go
collections.MigrateAddressesToFlexible(app)
```

**Step 3: Run and verify migration works**

Run: `go run main.go serve`
Expected: Server starts, `address_configs` populated with default column defs per project/type. Existing addresses have `config`, `address_code`, and `data` fields populated.

**Step 4: Commit**

```bash
git add collections/migrate_addresses.go main.go
git commit -m "feat: add address data migration from fixed fields to flexible JSON"
```

---

### Task 1.4: Address Config Service Layer ✅ DONE

**Files:**
- Create: `services/address_config.go`
- Create: `services/address_config_test.go`

**Step 1: Write tests**

```go
package services

import (
    "testing"
)

func TestDefaultColumnDefs_HasExpectedFields(t *testing.T) {
    types := []string{"bill_from", "dispatch_from", "bill_to", "ship_to", "install_at"}
    for _, addrType := range types {
        t.Run(addrType, func(t *testing.T) {
            cols := ParseColumnDefs(DefaultColumnDefsJSON(addrType))
            if len(cols) == 0 {
                t.Error("expected columns, got none")
            }
            // Every type should have company_name
            found := false
            for _, c := range cols {
                if c.Name == "company_name" {
                    found = true
                    break
                }
            }
            if !found {
                t.Error("expected company_name column")
            }
        })
    }
}

func TestParseColumnDefs_RoundTrips(t *testing.T) {
    json := DefaultColumnDefsJSON("bill_to")
    cols := ParseColumnDefs(json)
    if len(cols) < 10 {
        t.Errorf("expected at least 10 columns, got %d", len(cols))
    }
    // Check bill_to required fields
    for _, c := range cols {
        if c.Name == "gstin" && !c.Required {
            t.Error("bill_to should require gstin")
        }
    }
}
```

**Step 2: Write service**

```go
package services

import "encoding/json"

// ColumnDef describes a single column in an address configuration.
type ColumnDef struct {
    Name        string `json:"name"`
    Label       string `json:"label"`
    Required    bool   `json:"required"`
    Type        string `json:"type"`
    Fixed       bool   `json:"fixed"`
    ShowInTable bool   `json:"show_in_table"`
    ShowInPrint bool   `json:"show_in_print"`
    SortOrder   int    `json:"sort_order"`
}

// ParseColumnDefs parses the JSON columns field from an address_configs record.
func ParseColumnDefs(jsonStr string) []ColumnDef {
    var cols []ColumnDef
    if err := json.Unmarshal([]byte(jsonStr), &cols); err != nil {
        return nil
    }
    return cols
}

// ColumnDefsToJSON serializes column definitions to JSON.
func ColumnDefsToJSON(cols []ColumnDef) string {
    b, _ := json.Marshal(cols)
    return string(b)
}

// DefaultColumnDefsJSON returns the default columns JSON string for an address type.
func DefaultColumnDefsJSON(addressType string) string {
    cols := defaultColumnDefs(addressType)
    b, _ := json.Marshal(cols)
    return string(b)
}

func defaultColumnDefs(addressType string) []ColumnDef {
    base := []ColumnDef{
        {Name: "company_name", Label: "Company Name", Required: true, Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 1},
        {Name: "contact_person", Label: "Contact Person", Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 2},
        {Name: "phone", Label: "Phone", Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 3},
        {Name: "email", Label: "Email", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 4},
        {Name: "gstin", Label: "GSTIN", Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 5},
        {Name: "pan", Label: "PAN", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 6},
        {Name: "cin", Label: "CIN", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 7},
        {Name: "address_line_1", Label: "Address Line 1", Required: true, Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 8},
        {Name: "address_line_2", Label: "Address Line 2", Type: "text", ShowInTable: false, ShowInPrint: true, SortOrder: 9},
        {Name: "landmark", Label: "Landmark", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 10},
        {Name: "district", Label: "District", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 11},
        {Name: "city", Label: "City", Required: true, Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 12},
        {Name: "state", Label: "State", Required: true, Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 13},
        {Name: "pin_code", Label: "PIN Code", Required: true, Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 14},
        {Name: "country", Label: "Country", Required: true, Type: "text", ShowInTable: false, ShowInPrint: true, SortOrder: 15},
        {Name: "fax", Label: "Fax", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 16},
        {Name: "website", Label: "Website", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 17},
    }

    switch addressType {
    case "bill_from":
        setReq(base, "company_name", "address_line_1", "city", "state", "pin_code", "country", "gstin")
    case "dispatch_from":
        setReq(base, "company_name", "address_line_1", "city", "state", "pin_code", "country")
    case "bill_to":
        setReq(base, "company_name", "contact_person", "address_line_1", "city", "state", "pin_code", "country", "gstin")
    case "ship_to", "install_at":
        setReq(base, "contact_person", "address_line_1", "city", "state", "pin_code", "country", "phone")
    }

    return base
}

func setReq(cols []ColumnDef, names ...string) {
    nameSet := make(map[string]bool)
    for _, n := range names {
        nameSet[n] = true
    }
    for i := range cols {
        cols[i].Required = nameSet[cols[i].Name]
    }
}
```

**Step 3: Run tests**

Run: `go test ./services/ -run TestDefaultColumnDefs -v && go test ./services/ -run TestParseColumnDefs -v`
Expected: PASS

**Step 4: Commit**

```bash
git add services/address_config.go services/address_config_test.go
git commit -m "feat: add address config service with column definition parsing"
```

---

### Task 1.5: Update Address Handlers for Flexible Schema

**Files:**
- Modify: `handlers/address_list.go`
- Modify: `handlers/address_create.go`
- Modify: `handlers/address_edit.go`
- Modify: `templates/address_list.templ`
- Modify: `templates/address_form.templ`

This is a large refactor. The key changes:

**Step 1: Update address list handler to read from JSON `data` field**

In `address_list.go`, change the address record mapping from fixed fields to reading from `data` JSON:

```go
// Before: item.CompanyName = record.GetString("company_name")
// After:
dataJSON := record.GetString("data")
var data map[string]string
json.Unmarshal([]byte(dataJSON), &data)
// Access: data["company_name"], data["city"], etc.
```

Also fetch the `address_configs` record for this project/type to get column definitions for dynamic table rendering.

**Step 2: Update address create handler to write JSON `data` field**

In `address_create.go`, change `HandleAddressSave`:
- Read the address_config for this project/type
- Validate against column definitions (required fields)
- Build `data` map from form values
- Set `record.Set("data", dataJSON)` and `record.Set("config", configId)`

**Step 3: Update address edit handler similarly**

Same pattern as create but loading existing data from JSON.

**Step 4: Update address list template for dynamic columns**

The template receives column definitions and renders table headers/cells dynamically:

```go
type AddressListData struct {
    ProjectID    string
    AddressType  string
    AddressLabel string
    Columns      []services.ColumnDef  // visible table columns
    Items        []AddressListItem
    // ... pagination fields
}

type AddressListItem struct {
    ID          string
    AddressCode string
    Data        map[string]string
    // Fixed fields for ship_to/install_at
    DistrictName string
    MandalName   string
    MandalCode   string
}
```

**Step 5: Update address form template for dynamic fields**

The form renders fields dynamically from column definitions:

```templ
for _, col := range data.Columns {
    @addressField(col.Name, col.Label, "text", data.Values[col.Name], "", col.Required, data.Errors[col.Name])
}
```

**Step 6: Run `make run` and test all address pages**

Expected: Address list, create, edit pages work with the new flexible schema. Existing data is visible.

**Step 7: Commit**

```bash
git add handlers/address_list.go handlers/address_create.go handlers/address_edit.go templates/address_list.templ templates/address_form.templ
git commit -m "refactor: update address handlers and templates for flexible JSON schema"
```

---

### Task 1.6: Update Address Config UI in Project Settings

**Files:**
- Modify: `handlers/project_settings.go`
- Modify: `templates/project_settings.templ`

**Step 1: Update settings handler to load/save address_configs**

Replace the `project_address_settings` logic with `address_configs` logic. The settings page should show configurable columns per address type with:
- Column name and label
- Required toggle
- Show in table toggle
- Show in print toggle
- Add/remove custom columns
- Reorder via sort_order

**Step 2: Update settings template**

Use DaisyUI tabs (existing pattern) with a column configuration grid per address type.

**Step 3: Test settings page**

Run: `make run`, navigate to project settings
Expected: Column configuration UI per address type, changes persist

**Step 4: Commit**

```bash
git add handlers/project_settings.go templates/project_settings.templ
git commit -m "refactor: update project settings for configurable address columns"
```

---

### Task 1.7: Update PO Handlers for New Address Format

**Files:**
- Modify: `handlers/po_create.go`
- Modify: `handlers/po_view.go`
- Modify: `handlers/po_edit.go` (if exists)
- Modify: `templates/po_view.templ`
- Modify: `templates/po_create.templ`

**Step 1: Update address fetching in PO handlers**

`fetchAddressesByType()` now reads from `data` JSON field instead of fixed fields:

```go
func fetchAddressesByType(app *pocketbase.PocketBase, projectId, addressType string) []templates.AddressSelectItem {
    // Find config for this project/type
    // Find addresses with that config
    // Parse data JSON to get display fields
}
```

**Step 2: Update PO view template address display**

Read address fields from the JSON `data` instead of fixed record fields.

**Step 3: Test PO creation and viewing**

Run: `make run`, create and view a PO
Expected: Addresses display correctly from JSON data

**Step 4: Commit**

```bash
git add handlers/po_create.go handlers/po_view.go templates/po_view.templ templates/po_create.templ
git commit -m "refactor: update PO handlers for new address JSON format"
```

---

## Phase 2: Configurable Numbering System

PO and DC have **separate** numbering configurations with the same UI pattern. Each has its own prefix, format, separator, padding, and start numbers.

### Task 2.1: Create `number_sequences` Collection and Project Fields

**Files:**
- Modify: `collections/setup.go`

**Step 1: Add collection and project fields**

```go
// Number sequences — atomic counter per document type
ensureCollection(app, "number_sequences", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{
        Name:          "project",
        Required:      true,
        CollectionId:  projectsCol.Id,
        CascadeDelete: true,
        MaxSelect:     1,
    })
    c.Fields.Add(&core.SelectField{
        Name:      "sequence_type",
        Required:  true,
        Values:    []string{"po", "tdc", "odc", "stdc"},
        MaxSelect: 1,
    })
    c.Fields.Add(&core.TextField{Name: "financial_year", Required: true})
    c.Fields.Add(&core.NumberField{Name: "last_number", Required: true})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})

// PO numbering fields on projects
ensureField(app, "projects", &core.TextField{Name: "po_prefix"})
ensureField(app, "projects", &core.TextField{Name: "po_number_format"})
ensureField(app, "projects", &core.TextField{Name: "po_separator"})
ensureField(app, "projects", &core.NumberField{Name: "po_seq_padding"})
ensureField(app, "projects", &core.NumberField{Name: "po_seq_start"})

// DC numbering fields on projects
ensureField(app, "projects", &core.TextField{Name: "dc_prefix"})
ensureField(app, "projects", &core.TextField{Name: "dc_number_format"})
ensureField(app, "projects", &core.TextField{Name: "dc_separator"})
ensureField(app, "projects", &core.NumberField{Name: "dc_seq_padding"})
ensureField(app, "projects", &core.NumberField{Name: "dc_seq_start_tdc"})
ensureField(app, "projects", &core.NumberField{Name: "dc_seq_start_odc"})
ensureField(app, "projects", &core.NumberField{Name: "dc_seq_start_stdc"})

// Default addresses for DC wizard
ensureField(app, "projects", &core.RelationField{
    Name:         "default_bill_from",
    CollectionId: addressesCol.Id,
    MaxSelect:    1,
})
ensureField(app, "projects", &core.RelationField{
    Name:         "default_dispatch_from",
    CollectionId: addressesCol.Id,
    MaxSelect:    1,
})
```

**Step 2: Verify and commit**

```bash
git add collections/setup.go
git commit -m "feat: add number_sequences collection with separate PO and DC numbering fields"
```

---

### Task 2.2: Numbering Service

**Files:**
- Create: `services/numbering.go`
- Create: `services/numbering_test.go`

**Step 1: Write tests**

```go
package services

import (
    "testing"
    "time"
)

func TestGetFinancialYear(t *testing.T) {
    tests := []struct {
        name   string
        month  int
        year   int
        expect string
    }{
        {"april_start", 4, 2025, "2526"},
        {"march_end", 3, 2026, "2526"},
        {"january", 1, 2026, "2526"},
        {"december", 12, 2025, "2526"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            d := time.Date(tt.year, time.Month(tt.month), 15, 0, 0, 0, 0, time.UTC)
            got := GetFinancialYear(d)
            if got != tt.expect {
                t.Errorf("GetFinancialYear(%v) = %q, want %q", d, got, tt.expect)
            }
        })
    }
}

func TestFormatDocNumber(t *testing.T) {
    tests := []struct {
        name     string
        format   string
        sep      string
        prefix   string
        seqType  string
        fy       string
        seq      int
        padding  int
        projRef  string
        expect   string
    }{
        {"po_standard", "{PREFIX}{SEP}{TYPE}{SEP}{PROJECT_REF}{SEP}{FY}{SEP}{SEQ}", "-", "FSS", "po", "2526", 1, 3, "OAVS", "FSS-PO-OAVS-2526-001"},
        {"dc_standard", "{PREFIX}{SEP}{TYPE}{SEP}{FY}{SEP}{SEQ}", "-", "ABC", "tdc", "2526", 1, 3, "", "ABC-TDC-2526-001"},
        {"dc_official", "{PREFIX}{SEP}{TYPE}{SEP}{FY}{SEP}{SEQ}", "-", "ABC", "odc", "2526", 5, 3, "", "ABC-ODC-2526-005"},
        {"dc_transfer", "{PREFIX}{SEP}{TYPE}{SEP}{FY}{SEP}{SEQ}", "-", "ABC", "stdc", "2526", 1, 4, "", "ABC-STDC-2526-0001"},
        {"custom_sep", "{PREFIX}{SEP}{TYPE}{SEP}{FY}{SEP}{SEQ}", "/", "XYZ", "odc", "2526", 42, 4, "", "XYZ/ODC/2526/0042"},
        {"po_with_ref", "{PREFIX}{SEP}{TYPE}{SEP}{PROJECT_REF}{SEP}{FY}{SEP}{SEQ}", "-", "FSS", "po", "2526", 42, 4, "OAVS", "FSS-PO-OAVS-2526-0042"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := FormatDocNumber(tt.format, tt.sep, tt.prefix, tt.seqType, tt.fy, tt.seq, tt.padding, tt.projRef)
            if got != tt.expect {
                t.Errorf("got %q, want %q", got, tt.expect)
            }
        })
    }
}

func TestNumberingConfigForType(t *testing.T) {
    tests := []struct {
        seqType    string
        expectGroup string
    }{
        {"po", "po"},
        {"tdc", "dc"},
        {"odc", "dc"},
        {"stdc", "dc"},
    }
    for _, tt := range tests {
        t.Run(tt.seqType, func(t *testing.T) {
            got := ConfigGroupForType(tt.seqType)
            if got != tt.expectGroup {
                t.Errorf("ConfigGroupForType(%q) = %q, want %q", tt.seqType, got, tt.expectGroup)
            }
        })
    }
}
```

**Step 2: Implement service**

```go
package services

import (
    "fmt"
    "strings"
    "time"

    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"
)

// GetFinancialYear returns Indian fiscal year as "YYZZ" (e.g., "2526" for Apr 2025-Mar 2026).
func GetFinancialYear(d time.Time) string {
    startYear := d.Year()
    if d.Month() < time.April {
        startYear--
    }
    endYear := startYear + 1
    return fmt.Sprintf("%02d%02d", startYear%100, endYear%100)
}

// TypeCode returns the display code for a sequence type.
func TypeCode(seqType string) string {
    switch seqType {
    case "po":
        return "PO"
    case "tdc":
        return "TDC"
    case "odc":
        return "ODC"
    case "stdc":
        return "STDC"
    default:
        return strings.ToUpper(seqType)
    }
}

// ConfigGroupForType returns "po" or "dc" based on the sequence type.
// PO types use po_* project fields; DC types use dc_* project fields.
func ConfigGroupForType(seqType string) string {
    if seqType == "po" {
        return "po"
    }
    return "dc"
}

// FormatDocNumber formats a document number from a template and parameters.
func FormatDocNumber(format, sep, prefix, seqType, fy string, seq, padding int, projRef string) string {
    padFmt := fmt.Sprintf("%%0%dd", padding)
    result := format
    result = strings.ReplaceAll(result, "{PREFIX}", prefix)
    result = strings.ReplaceAll(result, "{SEP}", sep)
    result = strings.ReplaceAll(result, "{TYPE}", TypeCode(seqType))
    result = strings.ReplaceAll(result, "{FY}", fy)
    result = strings.ReplaceAll(result, "{SEQ}", fmt.Sprintf(padFmt, seq))
    result = strings.ReplaceAll(result, "{PROJECT_REF}", projRef)
    return result
}

// NextDocNumber atomically increments and returns the next document number.
// It reads the appropriate config (PO or DC) based on the sequence type.
func NextDocNumber(app *pocketbase.PocketBase, projectID, seqType string, docDate time.Time) (string, error) {
    project, err := app.FindRecordById("projects", projectID)
    if err != nil {
        return "", fmt.Errorf("project not found: %w", err)
    }

    group := ConfigGroupForType(seqType) // "po" or "dc"
    fy := GetFinancialYear(docDate)

    // Read config fields based on group
    format := project.GetString(group + "_number_format")
    if format == "" {
        format = "{PREFIX}{SEP}{TYPE}{SEP}{FY}{SEP}{SEQ}"
    }
    sep := project.GetString(group + "_separator")
    if sep == "" {
        sep = "-"
    }
    prefix := project.GetString(group + "_prefix")
    padding := project.GetInt(group + "_seq_padding")
    if padding == 0 {
        padding = 3
    }
    projRef := project.GetString("reference_number")

    // Determine start number: PO has one start, DC has per-type starts
    var seqStart int
    if group == "po" {
        seqStart = project.GetInt("po_seq_start")
    } else {
        seqStart = project.GetInt("dc_seq_start_" + seqType)
    }
    if seqStart == 0 {
        seqStart = 1
    }

    // Find or create sequence record
    seqCol, err := app.FindCollectionByNameOrId("number_sequences")
    if err != nil {
        return "", fmt.Errorf("number_sequences collection not found: %w", err)
    }

    records, err := app.FindRecordsByFilter(
        seqCol,
        "project = {:pid} && sequence_type = {:type} && financial_year = {:fy}",
        "", 1, 0,
        map[string]any{"pid": projectID, "type": seqType, "fy": fy},
    )
    if err != nil {
        return "", fmt.Errorf("failed to query sequences: %w", err)
    }

    var nextNum int
    if len(records) > 0 {
        rec := records[0]
        nextNum = rec.GetInt("last_number") + 1
        rec.Set("last_number", nextNum)
        if err := app.Save(rec); err != nil {
            return "", fmt.Errorf("failed to update sequence: %w", err)
        }
    } else {
        nextNum = seqStart
        rec := core.NewRecord(seqCol)
        rec.Set("project", projectID)
        rec.Set("sequence_type", seqType)
        rec.Set("financial_year", fy)
        rec.Set("last_number", nextNum)
        if err := app.Save(rec); err != nil {
            return "", fmt.Errorf("failed to create sequence: %w", err)
        }
    }

    return FormatDocNumber(format, sep, prefix, seqType, fy, nextNum, padding, projRef), nil
}
```

**Step 3: Run tests**

Run: `go test ./services/ -run TestGetFinancialYear -v && go test ./services/ -run TestFormatDocNumber -v && go test ./services/ -run TestNumberingConfigForType -v`
Expected: PASS

**Step 4: Commit**

```bash
git add services/numbering.go services/numbering_test.go
git commit -m "feat: add document numbering service with separate PO and DC configurations"
```

---

### Task 2.3: Migrate PO Number Generation

**Files:**
- Modify: `services/po_number.go`
- Modify: `handlers/po_create.go`

**Step 1: Update `GeneratePONumber` to use the new numbering service**

Replace the existing PO number logic with a call to `NextDocNumber(app, projectID, "po", time.Now())`. The service reads `po_prefix`, `po_number_format`, `po_separator`, `po_seq_padding`, `po_seq_start` from the project record.

**Step 2: Test PO creation still works**

Run: `make run`, create a PO
Expected: PO number generated using new configurable format

**Step 3: Commit**

```bash
git add services/po_number.go handlers/po_create.go
git commit -m "refactor: migrate PO number generation to configurable numbering service"
```

---

### Task 2.4: Add Numbering Config to Project Settings UI

**Files:**
- Modify: `handlers/project_settings.go`
- Modify: `templates/project_settings.templ`

**Step 1: Add numbering fields to settings handler**

Load and save two sets of fields:

**PO Numbering:** `po_prefix`, `po_number_format`, `po_separator`, `po_seq_padding`, `po_seq_start`

**DC Numbering:** `dc_prefix`, `dc_number_format`, `dc_separator`, `dc_seq_padding`, `dc_seq_start_tdc`, `dc_seq_start_odc`, `dc_seq_start_stdc`

**Default Addresses:** `default_bill_from`, `default_dispatch_from`

**Step 2: Add numbering config sections to settings template**

Two separate sections (or tabs) with the **same UI layout**:

**PO Numbering section:**
- PO Prefix text input
- PO Number Format text input with token reference ({PREFIX}, {TYPE}, {FY}, {SEQ}, {SEP}, {PROJECT_REF})
- PO Separator text input
- PO Sequence Padding number input
- PO Starting Number input
- Live preview: shows example PO number as you type (Alpine.js computed)

**DC Numbering section:**
- DC Prefix text input
- DC Number Format text input (same tokens)
- DC Separator text input
- DC Sequence Padding number input
- Starting numbers per DC type:
  - Transit DC (TDC) start number
  - Official DC (ODC) start number
  - Transfer DC (STDC) start number
- Live preview: shows example numbers for TDC, ODC, STDC (Alpine.js computed)

**Default Addresses section:**
- Default Bill From address picker
- Default Dispatch From address picker

**Step 3: Test settings**

Run: `make run`, configure PO and DC numbering separately, verify previews update independently
Expected: PO settings and DC settings persist independently, previews show correctly formatted numbers

**Step 4: Commit**

```bash
git add handlers/project_settings.go templates/project_settings.templ
git commit -m "feat: add separate PO and DC numbering configuration to project settings"
```

---

## Phase 3: DC Master Data (Templates + Transporters)

### Task 3.1: Create DC Collections

**Files:**
- Modify: `collections/setup.go`

**Step 1: Add all DC-related collections**

Add these collections in `Setup()`:

```go
// DC Templates
dcTemplatesCol := ensureCollection(app, "dc_templates", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "project", Required: true, CollectionId: projectsCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.TextField{Name: "name", Required: true})
    c.Fields.Add(&core.TextField{Name: "purpose"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})

// DC Template Items — references BOQ sub_items or sub_sub_items
ensureCollection(app, "dc_template_items", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "template", Required: true, CollectionId: dcTemplatesCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.SelectField{Name: "source_item_type", Required: true, Values: []string{"sub_item", "sub_sub_item"}, MaxSelect: 1})
    c.Fields.Add(&core.TextField{Name: "source_item_id", Required: true})
    c.Fields.Add(&core.NumberField{Name: "default_quantity"})
    c.Fields.Add(&core.SelectField{Name: "serial_tracking", Required: true, Values: []string{"none", "optional", "required"}, MaxSelect: 1})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
})

// Transporters
transportersCol := ensureCollection(app, "transporters", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "project", Required: true, CollectionId: projectsCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.TextField{Name: "company_name", Required: true})
    c.Fields.Add(&core.TextField{Name: "contact_person"})
    c.Fields.Add(&core.TextField{Name: "phone"})
    c.Fields.Add(&core.TextField{Name: "gst_number"})
    c.Fields.Add(&core.BoolField{Name: "is_active"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})

// Transporter Vehicles
ensureCollection(app, "transporter_vehicles", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "transporter", Required: true, CollectionId: transportersCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.TextField{Name: "vehicle_number", Required: true})
    c.Fields.Add(&core.TextField{Name: "vehicle_type"})
    c.Fields.Add(&core.TextField{Name: "driver_name"})
    c.Fields.Add(&core.TextField{Name: "driver_phone"})
    c.Fields.Add(&core.FileField{Name: "rc_image", MaxSelect: 1, MaxSize: 5242880, MimeTypes: []string{"image/jpeg", "image/png", "application/pdf"}})
    c.Fields.Add(&core.FileField{Name: "driver_license", MaxSelect: 1, MaxSize: 5242880, MimeTypes: []string{"image/jpeg", "image/png", "application/pdf"}})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
})

// Shipment Groups
shipmentGroupsCol := ensureCollection(app, "shipment_groups", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "project", Required: true, CollectionId: projectsCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.RelationField{Name: "template", CollectionId: dcTemplatesCol.Id, MaxSelect: 1})
    c.Fields.Add(&core.NumberField{Name: "num_locations", Required: true})
    c.Fields.Add(&core.SelectField{Name: "tax_type", Required: true, Values: []string{"cgst_sgst", "igst"}, MaxSelect: 1})
    c.Fields.Add(&core.BoolField{Name: "reverse_charge"})
    c.Fields.Add(&core.SelectField{Name: "status", Required: true, Values: []string{"draft", "issued"}, MaxSelect: 1})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})

// Delivery Challans
dcCol := ensureCollection(app, "delivery_challans", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "project", Required: true, CollectionId: projectsCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.TextField{Name: "dc_number", Required: true})
    c.Fields.Add(&core.SelectField{Name: "dc_type", Required: true, Values: []string{"transit", "official", "transfer"}, MaxSelect: 1})
    c.Fields.Add(&core.SelectField{Name: "status", Required: true, Values: []string{"draft", "issued", "splitting", "split"}, MaxSelect: 1})
    c.Fields.Add(&core.RelationField{Name: "template", CollectionId: dcTemplatesCol.Id, MaxSelect: 1})
    c.Fields.Add(&core.RelationField{Name: "bill_from_address", CollectionId: addressesCol.Id, MaxSelect: 1})
    c.Fields.Add(&core.RelationField{Name: "dispatch_from_address", CollectionId: addressesCol.Id, MaxSelect: 1})
    c.Fields.Add(&core.RelationField{Name: "bill_to_address", CollectionId: addressesCol.Id, MaxSelect: 1})
    c.Fields.Add(&core.RelationField{Name: "ship_to_address", CollectionId: addressesCol.Id, MaxSelect: 1})
    c.Fields.Add(&core.TextField{Name: "challan_date", Required: true})
    c.Fields.Add(&core.TextField{Name: "issued_at"})
    c.Fields.Add(&core.RelationField{Name: "shipment_group", CollectionId: shipmentGroupsCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})

// DC Line Items
dcLineItemsCol := ensureCollection(app, "dc_line_items", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "dc", Required: true, CollectionId: dcCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.SelectField{Name: "source_item_type", Required: true, Values: []string{"sub_item", "sub_sub_item"}, MaxSelect: 1})
    c.Fields.Add(&core.TextField{Name: "source_item_id", Required: true})
    c.Fields.Add(&core.NumberField{Name: "quantity", Required: true})
    c.Fields.Add(&core.NumberField{Name: "rate"})
    c.Fields.Add(&core.NumberField{Name: "tax_percentage"})
    c.Fields.Add(&core.NumberField{Name: "taxable_amount"})
    c.Fields.Add(&core.NumberField{Name: "tax_amount"})
    c.Fields.Add(&core.NumberField{Name: "total_amount"})
    c.Fields.Add(&core.NumberField{Name: "line_order"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})

// Serial Numbers
ensureCollection(app, "serial_numbers", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "project", Required: true, CollectionId: projectsCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.RelationField{Name: "line_item", Required: true, CollectionId: dcLineItemsCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.TextField{Name: "serial_number", Required: true})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
})

// DC Transit Details
ensureCollection(app, "dc_transit_details", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "dc", Required: true, CollectionId: dcCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.RelationField{Name: "transporter", CollectionId: transportersCol.Id, MaxSelect: 1})
    c.Fields.Add(&core.TextField{Name: "vehicle_number"})
    c.Fields.Add(&core.TextField{Name: "eway_bill_number"})
    c.Fields.Add(&core.TextField{Name: "docket_number"})
    c.Fields.Add(&core.TextField{Name: "notes"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
})

// Transfer DCs
transferDCsCol := ensureCollection(app, "transfer_dcs", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "dc", Required: true, CollectionId: dcCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.RelationField{Name: "hub_address", CollectionId: addressesCol.Id, MaxSelect: 1})
    c.Fields.Add(&core.RelationField{Name: "template", CollectionId: dcTemplatesCol.Id, MaxSelect: 1})
    c.Fields.Add(&core.SelectField{Name: "tax_type", Required: true, Values: []string{"cgst_sgst", "igst"}, MaxSelect: 1})
    c.Fields.Add(&core.BoolField{Name: "reverse_charge"})
    c.Fields.Add(&core.RelationField{Name: "transporter", CollectionId: transportersCol.Id, MaxSelect: 1})
    c.Fields.Add(&core.TextField{Name: "vehicle_number"})
    c.Fields.Add(&core.TextField{Name: "eway_bill_number"})
    c.Fields.Add(&core.TextField{Name: "docket_number"})
    c.Fields.Add(&core.TextField{Name: "notes"})
    c.Fields.Add(&core.NumberField{Name: "num_destinations", Required: true})
    c.Fields.Add(&core.NumberField{Name: "num_split"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})

// Transfer DC Destinations
transferDCDestsCol := ensureCollection(app, "transfer_dc_destinations", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "transfer_dc", Required: true, CollectionId: transferDCsCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.RelationField{Name: "ship_to_address", Required: true, CollectionId: addressesCol.Id, MaxSelect: 1})
    c.Fields.Add(&core.TextField{Name: "split_group"})
    c.Fields.Add(&core.BoolField{Name: "is_split"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
})

// Transfer DC Destination Quantities
ensureCollection(app, "transfer_dc_dest_quantities", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "destination", Required: true, CollectionId: transferDCDestsCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.SelectField{Name: "source_item_type", Required: true, Values: []string{"sub_item", "sub_sub_item"}, MaxSelect: 1})
    c.Fields.Add(&core.TextField{Name: "source_item_id", Required: true})
    c.Fields.Add(&core.NumberField{Name: "quantity", Required: true})
})

// Transfer DC Splits
ensureCollection(app, "transfer_dc_splits", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{Name: "transfer_dc", Required: true, CollectionId: transferDCsCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.RelationField{Name: "shipment_group", Required: true, CollectionId: shipmentGroupsCol.Id, CascadeDelete: true, MaxSelect: 1})
    c.Fields.Add(&core.NumberField{Name: "split_number", Required: true})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
})
```

Also add `transfer_dc` and `split` relation fields to `shipment_groups`:

```go
ensureField(app, "shipment_groups", &core.RelationField{Name: "transfer_dc", CollectionId: transferDCsCol.Id, MaxSelect: 1})
ensureField(app, "shipment_groups", &core.RelationField{Name: "split", CollectionId: transferDCSplitsCol.Id, MaxSelect: 1})
```

**Step 2: Run and verify**

Run: `go run main.go serve`
Expected: All collections created in PocketBase admin

**Step 3: Commit**

```bash
git add collections/setup.go
git commit -m "feat: add all DC-related collections (templates, transporters, DCs, serials, splits)"
```

---

### Task 3.2: DC Templates — Handlers & Templates

**Files:**
- Create: `handlers/dc_template_list.go`
- Create: `handlers/dc_template_create.go`
- Create: `handlers/dc_template_edit.go`
- Create: `handlers/dc_template_delete.go`
- Create: `templates/dc_template_list.templ`
- Create: `templates/dc_template_form.templ`

**Step 1: Write DC Template list handler**

Follow the `HandleBOQList` pattern. Closure over `app`, path value `projectId`, HTMX detection, dual rendering.

Data struct:
```go
type DCTemplateListData struct {
    ProjectID string
    Templates []DCTemplateListItem
    Total     int
}

type DCTemplateListItem struct {
    ID           string
    Name         string
    Purpose      string
    ItemCount    int
    Created      string
}
```

**Step 2: Write DC Template create handler**

Two handlers: GET (form) and POST (save). The create form needs:
- Name, Purpose fields
- BOQ item selector: fetch all sub_items and sub_sub_items for the project
- For each selected item: default_quantity input, serial_tracking select (none/optional/required)

**Step 3: Write templates**

Follow `po_list.templ` for list layout, `po_create.templ` for form layout.

**Step 4: Register routes in `main.go`**

```go
// DC Templates
se.Router.GET("/projects/{projectId}/dc-templates/", handlers.HandleDCTemplateList(app))
se.Router.GET("/projects/{projectId}/dc-templates/create", handlers.HandleDCTemplateCreate(app))
se.Router.POST("/projects/{projectId}/dc-templates/create", handlers.HandleDCTemplateSave(app))
se.Router.GET("/projects/{projectId}/dc-templates/{id}", handlers.HandleDCTemplateDetail(app))
se.Router.GET("/projects/{projectId}/dc-templates/{id}/edit", handlers.HandleDCTemplateEdit(app))
se.Router.POST("/projects/{projectId}/dc-templates/{id}/edit", handlers.HandleDCTemplateUpdate(app))
se.Router.POST("/projects/{projectId}/dc-templates/{id}/delete", handlers.HandleDCTemplateDelete(app))
se.Router.POST("/projects/{projectId}/dc-templates/{id}/duplicate", handlers.HandleDCTemplateDuplicate(app))
```

**Step 5: Test end-to-end**

Run: `make run`, navigate to DC Templates, create/edit/delete templates
Expected: Full CRUD working with HTMX navigation

**Step 6: Commit**

```bash
git add handlers/dc_template_*.go templates/dc_template_*.templ main.go
git commit -m "feat: add DC template CRUD with BOQ item selection"
```

---

### Task 3.3: Transporters — Handlers & Templates

**Files:**
- Create: `handlers/transporter_list.go`
- Create: `handlers/transporter_create.go`
- Create: `handlers/transporter_detail.go`
- Create: `handlers/transporter_vehicles.go`
- Create: `templates/transporter_list.templ`
- Create: `templates/transporter_form.templ`
- Create: `templates/transporter_detail.templ`

**Step 1: Write transporter list handler**

Pattern: closure over `app`, fetch transporters for project with vehicle counts.

Data struct:
```go
type TransporterListData struct {
    ProjectID    string
    Transporters []TransporterListItem
    Total        int
}

type TransporterListItem struct {
    ID           string
    CompanyName  string
    ContactPerson string
    Phone        string
    GSTNumber    string
    IsActive     bool
    VehicleCount int
}
```

**Step 2: Write transporter create/edit handlers**

Form fields: company_name, contact_person, phone, gst_number. Toggle active status.

**Step 3: Write transporter detail handler with vehicles**

Detail page shows transporter info + vehicle list. Vehicle CRUD:
- Add vehicle form: vehicle_number, vehicle_type, driver_name, driver_phone, rc_image upload, driver_license upload
- Delete vehicle action
- File uploads use `ParseMultipartForm(5 << 20)` and PocketBase file handling

**Step 4: Write templates**

Follow existing list/form patterns. Vehicle list uses inline expandable section (Alpine.js).

**Step 5: Register routes in `main.go`**

```go
// Transporters
se.Router.GET("/projects/{projectId}/transporters/", handlers.HandleTransporterList(app))
se.Router.GET("/projects/{projectId}/transporters/create", handlers.HandleTransporterCreate(app))
se.Router.POST("/projects/{projectId}/transporters/create", handlers.HandleTransporterSave(app))
se.Router.GET("/projects/{projectId}/transporters/{id}", handlers.HandleTransporterDetail(app))
se.Router.POST("/projects/{projectId}/transporters/{id}/edit", handlers.HandleTransporterUpdate(app))
se.Router.POST("/projects/{projectId}/transporters/{id}/toggle", handlers.HandleTransporterToggle(app))
se.Router.POST("/projects/{projectId}/transporters/{id}/vehicles", handlers.HandleVehicleAdd(app))
se.Router.DELETE("/projects/{projectId}/transporters/{id}/vehicles/{vid}", handlers.HandleVehicleDelete(app))
```

**Step 6: Test and commit**

```bash
git add handlers/transporter_*.go templates/transporter_*.templ main.go
git commit -m "feat: add transporter management with vehicle CRUD and file uploads"
```

---

### Task 3.4: Update Sidebar Navigation

**Files:**
- Modify: `templates/sidebar.templ`
- Modify: `handlers/sidebar_helpers.go`

**Step 1: Add DC counts to SidebarData**

In `templates/sidebar.templ`, update `SidebarData`:
```go
type SidebarData struct {
    // ... existing fields
    DCTemplateCount  int
    TransporterCount int
    DCCount          int
}
```

**Step 2: Update BuildSidebarData to query DC counts**

In `handlers/sidebar_helpers.go`:
```go
// DC Template count
dcTemplates, _ := app.FindRecordsByFilter("dc_templates", "project = {:pid}", "", 0, 0, map[string]any{"pid": projectId})
data.DCTemplateCount = len(dcTemplates)

// Transporter count
transporters, _ := app.FindRecordsByFilter("transporters", "project = {:pid}", "", 0, 0, map[string]any{"pid": projectId})
data.TransporterCount = len(transporters)

// DC count
dcs, _ := app.FindRecordsByFilter("delivery_challans", "project = {:pid}", "", 0, 0, map[string]any{"pid": projectId})
data.DCCount = len(dcs)
```

**Step 3: Add DC Management section to sidebar template**

Between "Purchase Orders" and "Vendors" in the sidebar, add:

```templ
// DC MANAGEMENT section
<div style="padding: 0 16px; margin-top: 8px; margin-bottom: 8px;">
    <div style="font-family: 'Space Grotesk', sans-serif; font-size: 10px; font-weight: 600; color: #666; text-transform: uppercase; letter-spacing: 1.5px; padding: 8px 0;">
        DC MANAGEMENT
    </div>
</div>
@SidebarSubLink(fmt.Sprintf("/projects/%s/dc-templates/", data.ActiveProject.ID), "DC Templates", isActive, data.DCTemplateCount)
@SidebarSubLink(fmt.Sprintf("/projects/%s/transporters/", data.ActiveProject.ID), "Transporters", isActive, data.TransporterCount)
@SidebarSubLink(fmt.Sprintf("/projects/%s/dcs/", data.ActiveProject.ID), "Delivery Challans", isActive, data.DCCount)
```

**Step 4: Generate templ and test**

Run: `templ generate && make run`
Expected: Sidebar shows DC Management section with links and counts

**Step 5: Commit**

```bash
git add templates/sidebar.templ handlers/sidebar_helpers.go
git commit -m "feat: add DC management section to sidebar navigation"
```

---

## Phase 4: Unified DC Wizard

### Task 4.1: DC Wizard — Step 1 (Setup)

**Files:**
- Create: `handlers/dc_wizard.go`
- Create: `templates/dc_wizard_step1.templ`

**Step 1: Write wizard step 1 handler**

```go
func HandleDCWizardStep1(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectId := e.Request.PathValue("projectId")

        // Fetch DC templates for this project
        templates, _ := app.FindRecordsByFilter("dc_templates", "project = {:pid}", "name", 0, 0, map[string]any{"pid": projectId})

        // Fetch active transporters
        transporters, _ := app.FindRecordsByFilter("transporters", "project = {:pid} && is_active = true", "company_name", 0, 0, map[string]any{"pid": projectId})

        data := DCWizardStep1Data{
            ProjectID:    projectId,
            Templates:    mapTemplates(templates),
            Transporters: mapTransporters(transporters),
            ChallanDate:  time.Now().Format("2006-01-02"),
            Errors:       make(map[string]string),
        }

        // HTMX detection, dual rendering
        // ...
    }
}
```

**Step 2: Write step 1 template**

Form fields:
- DC Type: radio toggle (Direct Shipment / Transfer) — Alpine.js `x-data="{ dcType: 'direct' }"`
- DC Template: select dropdown
- Challan Date: date input (default today)
- Transporter: select dropdown → on change, fetch vehicles via HTMX (`hx-get="/projects/{id}/transporters/{tid}/vehicles-json"`)
- Vehicle: select dropdown (populated from transporter selection)
- Eway Bill Number: text input
- Docket Number: text input
- Reverse Charge: checkbox toggle (default off)

**Step 3: Register route**

```go
se.Router.GET("/projects/{projectId}/dcs/create", handlers.HandleDCWizardStep1(app))
```

**Step 4: Test and commit**

```bash
git add handlers/dc_wizard.go templates/dc_wizard_step1.templ main.go
git commit -m "feat: add DC wizard step 1 - setup (type, template, transporter)"
```

---

### Task 4.2: DC Wizard — Step 2 (Destinations)

**Files:**
- Modify: `handlers/dc_wizard.go`
- Create: `templates/dc_wizard_step2.templ`

**Step 1: Write step 2 handler**

POST handler that receives step 1 data and renders step 2. Carries forward step 1 values in hidden fields or session.

Fetches:
- Project's default bill_from and dispatch_from addresses
- All bill_to addresses for the project
- All ship_to addresses for the project

**Step 2: Write step 2 template**

Form fields:
- Bill From: pre-filled from project defaults, shown as read-only with "Override" toggle (Alpine.js)
- Dispatch From: same pattern
- Bill To: searchable address picker (HTMX search endpoint)
- Number of Destinations: number input
- Ship To addresses: dynamic list of address pickers (one per destination) — Alpine.js `x-data="{ numDest: 1 }"`, adds pickers on change
- **Transfer only** (shown via `x-show="dcType === 'transfer'"`): Hub Address picker
- Tax Type: auto-calculated badge (IGST/CGST+SGST) based on state comparison, with override option

**Step 3: Register route**

```go
se.Router.POST("/projects/{projectId}/dcs/create/step2", handlers.HandleDCWizardStep2(app))
se.Router.POST("/projects/{projectId}/dcs/create/back-to-step1", handlers.HandleDCWizardBackToStep1(app))
```

**Step 4: Test and commit**

```bash
git add handlers/dc_wizard.go templates/dc_wizard_step2.templ main.go
git commit -m "feat: add DC wizard step 2 - destinations with auto tax type"
```

---

### Task 4.3: DC Wizard — Step 3 (Items & Serials)

**Files:**
- Modify: `handlers/dc_wizard.go`
- Create: `templates/dc_wizard_step3.templ`
- Create: `services/serial_validation.go`
- Create: `services/serial_validation_test.go`

**Step 1: Write serial validation service**

```go
package services

// ValidateSerials checks serial numbers for duplicates and count mismatches.
func ValidateSerials(input []string, expectedQty int, existingSerials map[string]string) SerialValidationResult {
    result := SerialValidationResult{Valid: true}

    // Check duplicates within input
    seen := make(map[string]bool)
    for _, s := range input {
        s = strings.TrimSpace(s)
        if s == "" {
            continue
        }
        if seen[s] {
            result.DuplicatesInInput = append(result.DuplicatesInInput, s)
            result.Valid = false
        }
        seen[s] = true
    }

    // Check duplicates in database
    for _, s := range input {
        s = strings.TrimSpace(s)
        if dcNum, exists := existingSerials[s]; exists {
            result.DuplicatesInDB = append(result.DuplicatesInDB, SerialConflict{Serial: s, ExistingDC: dcNum})
            result.Valid = false
        }
    }

    // Check count
    uniqueCount := len(seen)
    if uniqueCount != expectedQty {
        result.CountMismatch = true
        result.Expected = expectedQty
        result.Got = uniqueCount
        result.Valid = false
    }

    return result
}

type SerialValidationResult struct {
    Valid            bool
    DuplicatesInInput []string
    DuplicatesInDB   []SerialConflict
    CountMismatch    bool
    Expected, Got    int
}

type SerialConflict struct {
    Serial     string
    ExistingDC string
}
```

**Step 2: Write tests for serial validation**

Table-driven tests covering: valid input, duplicates within, duplicates in DB, count mismatch, empty strings filtered.

**Step 3: Write step 3 handler**

POST handler receives step 1+2 data, loads template items with their BOQ source details (description, HSN, UoM, price, GST). Renders quantity grid + serial entry areas.

**Step 4: Write step 3 template**

Complex interactive template:
- Quantity grid table: rows = items, columns = destinations + total
- Each cell is a number input
- Below each item row: expandable serial entry (based on `serial_tracking`)
  - `required`: red border if count != total, textarea + live count badge
  - `optional`: textarea + count badge (no enforcement)
  - `none`: hidden
- Alpine.js handles: quantity totals, serial count display, expand/collapse
- HTMX endpoint for real-time serial validation: `POST /api/serials/validate`

**Step 5: Register routes**

```go
se.Router.POST("/projects/{projectId}/dcs/create/step3", handlers.HandleDCWizardStep3(app))
se.Router.POST("/projects/{projectId}/dcs/create/back-to-step2", handlers.HandleDCWizardBackToStep2(app))
se.Router.POST("/projects/{projectId}/api/serials/validate", handlers.HandleSerialValidate(app))
```

**Step 6: Test and commit**

```bash
git add handlers/dc_wizard.go templates/dc_wizard_step3.templ services/serial_validation.go services/serial_validation_test.go main.go
git commit -m "feat: add DC wizard step 3 - items grid with serial number validation"
```

---

### Task 4.4: DC Wizard — Step 4 (Review & Confirm)

**Files:**
- Modify: `handlers/dc_wizard.go`
- Create: `templates/dc_wizard_step4.templ`

**Step 1: Write step 4 handler**

POST handler receives all previous step data. Builds a comprehensive review data struct with:
- All addresses resolved to display names
- All items with quantities, pricing, tax calculations
- Serial number summaries
- Grand totals (taxable, tax, total)

**Step 2: Write step 4 template**

Read-only summary with collapsible sections:
- **Addresses**: Bill From, Dispatch From, Bill To, Ship To (per destination)
- **Transport**: Transporter, Vehicle, Eway Bill, Docket Number
- **Items**: Table with per-destination quantities, rates, tax, totals
- **Serials**: Collapsed by default, expandable per item
- **Totals**: Taxable amount, Tax amount, Grand total
- Each section has "Edit" link (posts back to that step with data)
- Confirm button posts to creation endpoint

**Step 3: Register routes**

```go
se.Router.POST("/projects/{projectId}/dcs/create/step4", handlers.HandleDCWizardStep4(app))
se.Router.POST("/projects/{projectId}/dcs/create/back-to-step3", handlers.HandleDCWizardBackToStep3(app))
```

**Step 4: Test and commit**

```bash
git add handlers/dc_wizard.go templates/dc_wizard_step4.templ main.go
git commit -m "feat: add DC wizard step 4 - review and confirm"
```

---

### Task 4.5: DC Creation Service (Atomic)

**Files:**
- Create: `services/dc_creation.go`
- Create: `services/dc_creation_test.go`

**Step 1: Write DC creation service**

```go
// CreateDirectShipment creates a shipment group with 1 transit DC + N official DCs.
func CreateDirectShipment(app *pocketbase.PocketBase, params ShipmentParams) (*ShipmentResult, error)

// CreateTransferDC creates a transfer DC with destination plan.
func CreateTransferDC(app *pocketbase.PocketBase, params TransferDCParams) (*TransferDCResult, error)
```

For Direct Shipment:
1. Create Shipment Group (status=draft)
2. Generate Transit DC number via `NextDocNumber(app, projectID, "tdc", date)`
3. Create Transit DC (all items, serials, pricing, transport details)
4. For each destination with qty > 0: generate Official DC number, create Official DC (per-location qty, no pricing/serials)
5. Return result with all created record IDs

For Transfer DC:
1. Generate Transfer DC number via `NextDocNumber(app, projectID, "stdc", date)`
2. Create Delivery Challan record (type=transfer)
3. Create Transfer DC metadata (hub, destinations, transporter)
4. Create line items with serials and pricing
5. For each destination: create destination record with per-product quantities
6. Return result

**Step 2: Write tests using test helpers**

Test both creation paths with `testhelpers.NewTestApp(t)`.

**Step 3: Wire creation handler**

```go
se.Router.POST("/projects/{projectId}/dcs/create", handlers.HandleDCCreate(app))
```

The handler calls the appropriate service based on `dcType`, then redirects to the DC detail page.

**Step 4: Commit**

```bash
git add services/dc_creation.go services/dc_creation_test.go handlers/dc_wizard.go main.go
git commit -m "feat: add atomic DC creation service for direct shipments and transfer DCs"
```

---

## Phase 5: DC Lifecycle

### Task 5.1: DC List View

**Files:**
- Create: `handlers/dc_list.go`
- Create: `templates/dc_list.templ`

**Step 1: Write DC list handler**

Unified list showing all DC types with filters:
- Filter by type: all / transit / official / transfer
- Filter by status: all / draft / issued / splitting / split
- Search by DC number
- Sort by date, number, type, status

Data struct:
```go
type DCListData struct {
    ProjectID  string
    DCs        []DCListItem
    TypeFilter string
    StatusFilter string
    Search     string
    Total      int
    // Pagination
}

type DCListItem struct {
    ID          string
    DCNumber    string
    DCType      string
    Status      string
    TemplateName string
    ChallanDate string
    ShipTo      string  // display name
    ItemCount   int
    Created     string
}
```

**Step 2: Write list template**

Follow `po_list.templ` pattern with filter bar (type + status dropdowns + search), table, pagination. Status badges styled per status.

**Step 3: Register route and test**

```bash
git add handlers/dc_list.go templates/dc_list.templ main.go
git commit -m "feat: add delivery challan list view with type and status filters"
```

---

### Task 5.2: DC Detail View

**Files:**
- Create: `handlers/dc_detail.go`
- Create: `templates/dc_detail.templ`

**Step 1: Write DC detail handler**

Fetches DC record + related data (addresses, line items, serials, transit details, shipment group info). Builds comprehensive view data.

**Step 2: Write detail template**

Follow `po_view.templ` pattern — document-style layout (max-width 900px centered):
- Header: DC Number, Type badge, Status badge, Date
- Actions bar: Edit (draft), Delete (draft), Issue (draft), Print (issued), Export PDF (issued), Export Excel (issued), Split (transfer + issued)
- Addresses section: Bill From, Dispatch From, Bill To, Ship To
- Transport section: Transporter, Vehicle, Eway Bill, Docket
- Line items table: Item, HSN, UoM, Qty, Rate, Tax%, Taxable, Tax, Total
- Serial numbers (expandable per item)
- Totals: Taxable, Tax, Grand Total
- For shipment groups: link to Transit DC + list of Official DCs
- For transfer DCs: destination plan table + split status

**Step 3: Register route and test**

```bash
git add handlers/dc_detail.go templates/dc_detail.templ main.go
git commit -m "feat: add delivery challan detail view with full document display"
```

---

### Task 5.3: DC Issue Flow

**Files:**
- Create: `services/dc_issue.go`
- Create: `services/dc_issue_test.go`
- Create: `handlers/dc_issue.go`

**Step 1: Write issue service**

```go
// IssueShipmentGroup validates and issues all DCs in a shipment group atomically.
func IssueShipmentGroup(app *pocketbase.PocketBase, groupID string) error

// IssueTransferDC validates and issues a transfer DC.
func IssueTransferDC(app *pocketbase.PocketBase, dcID string) error
```

Validation:
- All DCs must be in draft status
- Transit/Transfer DCs: all items with `serial_tracking=required` must have serial count == quantity
- Serial uniqueness check across project
- On success: update status to "issued", set `issued_at`

**Step 2: Write tests**

Test valid issuance, missing serials rejection, duplicate serial rejection.

**Step 3: Write handler**

POST handler that calls appropriate issue service, returns toast notification.

**Step 4: Register route and commit**

```bash
git add services/dc_issue.go services/dc_issue_test.go handlers/dc_issue.go main.go
git commit -m "feat: add DC issuance with serial number validation"
```

---

### Task 5.4: DC Edit and Delete

**Files:**
- Create: `handlers/dc_edit.go`
- Create: `handlers/dc_delete.go`

**Step 1: Write edit handler (draft only)**

Re-enters the wizard with existing data pre-populated. Redirects to step 1 with all current values loaded.

**Step 2: Write delete handler (draft only)**

Validates DC is in draft status, then cascading delete (PocketBase handles via CascadeDelete on relations).

**Step 3: Register routes and commit**

```bash
git add handlers/dc_edit.go handlers/dc_delete.go main.go
git commit -m "feat: add DC edit (re-enter wizard) and delete (draft only)"
```

---

### Task 5.5: Split Wizard (Transfer DCs)

**Files:**
- Create: `handlers/split_wizard.go`
- Create: `templates/split_wizard_step1.templ`
- Create: `templates/split_wizard_step2.templ`
- Create: `templates/split_wizard_step3.templ`
- Create: `services/dc_split.go`
- Create: `services/dc_split_test.go`

**Step 1: Write split service**

```go
// CreateSplit creates a child shipment group from selected transfer DC destinations.
func CreateSplit(app *pocketbase.PocketBase, params SplitParams) (*SplitResult, error)

// UndoSplit reverses a split operation.
func UndoSplit(app *pocketbase.PocketBase, splitID string) error
```

CreateSplit:
1. Validate transfer DC is issued/splitting
2. Create child Shipment Group (status=draft)
3. Create Transit DC (with selected destinations' total quantities + assigned serials)
4. Create Official DCs (one per selected destination)
5. Mark destinations as `is_split=1`
6. Increment `num_split` on transfer_dcs record
7. Update status based on split count vs destination count

UndoSplit:
1. Delete child shipment group (cascades to its DCs)
2. Reset `is_split=0` on affected destinations
3. Decrement `num_split`
4. Recompute status

**Step 2: Write split wizard handlers (3 steps)**

Step 1: Show unsplit destinations with checkboxes
Step 2: Transporter selection + serial assignment from parent pool
Step 3: Review and confirm

**Step 3: Write templates for each step**

**Step 4: Register routes**

```go
se.Router.GET("/projects/{projectId}/transfer-dcs/{id}/split", handlers.HandleSplitStep1(app))
se.Router.POST("/projects/{projectId}/transfer-dcs/{id}/split/step2", handlers.HandleSplitStep2(app))
se.Router.POST("/projects/{projectId}/transfer-dcs/{id}/split/step3", handlers.HandleSplitStep3(app))
se.Router.POST("/projects/{projectId}/transfer-dcs/{id}/split", handlers.HandleSplitCreate(app))
se.Router.POST("/projects/{projectId}/transfer-dcs/{id}/splits/{sid}/undo", handlers.HandleSplitUndo(app))
```

**Step 5: Test and commit**

```bash
git add handlers/split_wizard.go templates/split_wizard_*.templ services/dc_split.go services/dc_split_test.go main.go
git commit -m "feat: add transfer DC split wizard with undo support"
```

---

## Phase 6: Exports

### Task 6.1: DC PDF Export

**Files:**
- Create: `services/dc_export_pdf.go`
- Create: `services/dc_export_pdf_test.go`
- Create: `handlers/dc_export.go`

**Step 1: Write PDF export service using maroto**

Follow `services/export_pdf.go` pattern. Three PDF layouts:

```go
func GenerateTransitDCPDF(data TransitDCExportData) ([]byte, error)
func GenerateOfficialDCPDF(data OfficialDCExportData) ([]byte, error)
func GenerateTransferDCPDF(data TransferDCExportData) ([]byte, error)
func GenerateShipmentGroupPDF(data ShipmentGroupExportData) ([]byte, error) // merged
```

Transit DC PDF sections: Header (company, DC#, date), Addresses (bill_from, dispatch_from, bill_to, ship_to), Transport details, Line items table (with pricing + tax), Serial numbers, Totals, Signature area.

Official DC PDF: Same but no pricing, no serials.

Transfer DC PDF: Same as transit + destination summary table.

Shipment Group PDF: Transit DC page + all Official DC pages merged.

**Step 2: Write tests**

Test each PDF generator produces non-empty bytes with valid data.

**Step 3: Write export handler**

```go
func HandleDCExportPDF(app *pocketbase.PocketBase) func(*core.RequestEvent) error
func HandleShipmentGroupExportPDF(app *pocketbase.PocketBase) func(*core.RequestEvent) error
```

Pattern: fetch data → generate PDF → set Content-Type + Content-Disposition → write bytes.

**Step 4: Register routes and commit**

```bash
git add services/dc_export_pdf.go services/dc_export_pdf_test.go handlers/dc_export.go main.go
git commit -m "feat: add DC PDF export with transit, official, transfer, and merged layouts"
```

---

### Task 6.2: DC Excel Export

**Files:**
- Create: `services/dc_export_excel.go`
- Create: `services/dc_export_excel_test.go`

**Step 1: Write Excel export service using excelize**

Follow `services/export_data.go` pattern.

```go
func GenerateDCExcel(data DCExcelData) ([]byte, error)
func GenerateShipmentGroupExcel(data ShipmentGroupExcelData) ([]byte, error)
```

Single DC: Sheet 1 (DC details + addresses), Sheet 2 (line items + pricing), Sheet 3 (serial numbers).

Shipment Group: One sheet per DC.

**Step 2: Write tests and handler**

**Step 3: Register routes and commit**

```bash
git add services/dc_export_excel.go services/dc_export_excel_test.go handlers/dc_export.go main.go
git commit -m "feat: add DC Excel export with multi-sheet workbooks"
```

---

### Task 6.3: DC Print View (HTML)

**Files:**
- Create: `handlers/dc_print.go`
- Create: `templates/dc_print.templ`

**Step 1: Write print handler**

Renders a standalone HTML page (no sidebar/header) optimized for browser printing.

**Step 2: Write print template**

Full-page document layout with print CSS:
- `@media print` styles
- Company header, DC details, addresses, items table, totals, signature area
- Different layouts for transit vs official vs transfer

**Step 3: Register route and commit**

```bash
git add handlers/dc_print.go templates/dc_print.templ main.go
git commit -m "feat: add DC HTML print view with print-optimized layout"
```

---

## Phase 7: Shipment Group Views

### Task 7.1: Shipment Group Detail

**Files:**
- Create: `handlers/shipment_group.go`
- Create: `templates/shipment_group_detail.templ`

**Step 1: Write handler**

Fetches shipment group + all DCs in the group (transit + officials). Builds a view showing:
- Group status and metadata
- Transit DC summary with link to detail
- List of Official DCs with links
- Actions: Issue All (draft), Export Merged PDF (issued), Delete (draft)

**Step 2: Write template and register route**

```bash
git add handlers/shipment_group.go templates/shipment_group_detail.templ main.go
git commit -m "feat: add shipment group detail view with DC listing"
```

---

## Phase 8: Testing & Polish

### Task 8.1: Integration Tests for DC Handlers

**Files:**
- Create: `handlers/dc_wizard_test.go`
- Create: `handlers/dc_list_test.go`

**Step 1: Write handler integration tests**

Test key flows:
- Wizard step 1 renders with templates and transporters
- DC creation produces correct records
- DC list shows filtered results
- Issue validates serials correctly
- Delete only works on draft DCs

Follow the pattern in existing handler tests: `testhelpers.NewTestApp(t)`, create test data, construct request events, call handler, assert response.

**Step 2: Run all tests**

Run: `make test`
Expected: All tests pass

**Step 3: Commit**

```bash
git add handlers/dc_wizard_test.go handlers/dc_list_test.go
git commit -m "test: add integration tests for DC wizard and list handlers"
```

---

### Task 8.2: Test Helpers for DC Data

**Files:**
- Modify: `testhelpers/testhelpers.go`

**Step 1: Add DC test helper functions**

```go
func CreateTestDCTemplate(t *testing.T, app *pocketbase.PocketBase, projectID, name string) *core.Record
func CreateTestTransporter(t *testing.T, app *pocketbase.PocketBase, projectID, name string) *core.Record
func CreateTestVehicle(t *testing.T, app *pocketbase.PocketBase, transporterID, vehicleNumber string) *core.Record
func CreateTestDeliveryChallan(t *testing.T, app *pocketbase.PocketBase, projectID, dcNumber, dcType, status string) *core.Record
func CreateTestShipmentGroup(t *testing.T, app *pocketbase.PocketBase, projectID string) *core.Record
```

**Step 2: Commit**

```bash
git add testhelpers/testhelpers.go
git commit -m "test: add DC-related test helper functions"
```

---

### Task 8.3: Generate Templ Files and Build Verification

**Step 1: Generate all templ files**

Run: `templ generate`
Expected: All `_templ.go` files generated without errors

**Step 2: Build CSS**

Run: `npx @tailwindcss/cli -i input.css -o static/css/output.css`
Expected: CSS compiles with new Tailwind classes

**Step 3: Build and vet**

Run: `go build ./... && go vet ./...`
Expected: No build errors, no vet warnings

**Step 4: Run full test suite**

Run: `make test`
Expected: All tests pass

**Step 5: Start server and smoke test**

Run: `make run`
Manual verification:
- [ ] Sidebar shows DC Management section
- [ ] DC Templates CRUD works
- [ ] Transporters CRUD with vehicles works
- [ ] DC Wizard completes all 4 steps
- [ ] DC creation produces correct records
- [ ] DC list shows all DCs with filters
- [ ] DC detail view displays correctly
- [ ] Issue flow works with serial validation
- [ ] PDF export generates downloadable PDF
- [ ] Excel export generates downloadable XLSX
- [ ] Print view renders correctly
- [ ] Split wizard works for transfer DCs
- [ ] Address pages work with new flexible schema
- [ ] PO pages still work with migrated addresses
- [ ] Project settings has numbering config

**Step 6: Final commit**

```bash
git commit -m "chore: verify full build, tests, and smoke test pass"
```

---

## Summary

| Phase | Tasks | Key Deliverables |
|-------|-------|-----------------|
| 1: Address Restructure | 1.1–1.7 | Flexible JSON addresses, migration, updated handlers |
| 2: Numbering | 2.1–2.4 | Unified number_sequences, configurable format, PO migration |
| 3: Master Data | 3.1–3.4 | DC Templates, Transporters, Sidebar integration |
| 4: DC Wizard | 4.1–4.5 | 4-step unified wizard, serial validation, atomic creation |
| 5: DC Lifecycle | 5.1–5.5 | List, Detail, Issue, Edit, Delete, Split |
| 6: Exports | 6.1–6.3 | PDF (maroto), Excel (excelize), HTML print |
| 7: Shipment Groups | 7.1 | Group detail view |
| 8: Testing & Polish | 8.1–8.3 | Integration tests, test helpers, build verification |

**Total: 8 phases, 26 tasks**

Dependencies: Phase 1 must complete before Phase 4 (wizard uses new addresses). Phase 2 must complete before Phase 4 (wizard uses numbering). Phase 3 must complete before Phase 4 (wizard uses templates + transporters). Phases 5-7 depend on Phase 4. Phase 8 runs last.
