# Phase 10: Excel Template Download for Address Import

## Overview & Objectives

Provide downloadable Excel templates that users fill out offline and later upload to bulk-import Ship To or Install At addresses. Each template is dynamically generated per-project so that **required fields** (driven by `project_address_settings`) are visually distinguished from optional fields, and data-validation dropdowns are embedded for State and Country columns.

### Key goals

1. Single service function `GenerateAddressTemplate(projectID, addressType)` returns `[]byte` (xlsx).
2. Separate templates for **Ship To** and **Install At** -- the Install At template includes an extra leading column **Ship To Reference**.
3. Color-coded headers: required fields get a blue background; optional fields get a grey background.
4. Excel data-validation dropdowns on State and Country columns.
5. Hidden **Instructions** sheet with field descriptions, format rules, and example rows.
6. Handler wired at `GET /projects/{projectId}/addresses/{type}/template`.

---

## Files to Create / Modify

| Action | Path | Purpose |
|--------|------|---------|
| **Create** | `services/address_template.go` | Template generation logic (Excelize) |
| **Create** | `services/address_fields.go` | Canonical field definitions, Indian state list, country list |
| **Modify** | `collections/setup.go` | Add `projects`, `ship_to_addresses`, `install_at_addresses`, and `project_address_settings` collections |
| **Modify** | `main.go` | Register the new GET route |
| **Create** | `handlers/address_template.go` | HTTP handler for template download |

---

## Detailed Implementation Steps

### Step 1: Define Address Field Metadata (`services/address_fields.go`)

This file provides a single source of truth for every column that can appear in an address template. The validation engine (Phase 11) will also import these definitions.

```go
package services

// AddressField describes one column in an address import template.
type AddressField struct {
    Key          string // internal name, matches PocketBase field name
    Label        string // human-readable header shown in Excel
    Description  string // shown on the Instructions sheet
    FormatRule   string // e.g. "6 digits", "15-char GSTIN", ""
    ExampleValue string // shown on the Instructions sheet
    AlwaysRequired bool // true = required regardless of project settings
}

// BaseShipToFields returns the ordered list of fields for Ship To addresses.
func BaseShipToFields() []AddressField {
    return []AddressField{
        {Key: "site_name", Label: "Site Name", Description: "Unique name/identifier for this site", FormatRule: "", ExampleValue: "Mumbai HQ", AlwaysRequired: true},
        {Key: "contact_person", Label: "Contact Person", Description: "Primary contact at the address", ExampleValue: "Rajesh Kumar"},
        {Key: "phone", Label: "Phone", Description: "10-digit mobile number", FormatRule: "10 digits, no spaces or dashes", ExampleValue: "9876543210"},
        {Key: "email", Label: "Email", Description: "Email address", FormatRule: "Valid email format", ExampleValue: "rajesh@example.com"},
        {Key: "address_line1", Label: "Address Line 1", Description: "Street address", ExampleValue: "123 MG Road", AlwaysRequired: true},
        {Key: "address_line2", Label: "Address Line 2", Description: "Locality / landmark", ExampleValue: "Near City Mall"},
        {Key: "city", Label: "City", Description: "City name", ExampleValue: "Mumbai", AlwaysRequired: true},
        {Key: "state", Label: "State", Description: "Indian state (select from dropdown)", ExampleValue: "Maharashtra", AlwaysRequired: true},
        {Key: "pin_code", Label: "PIN Code", Description: "6-digit Indian postal code", FormatRule: "Exactly 6 digits", ExampleValue: "400001"},
        {Key: "country", Label: "Country", Description: "Country (select from dropdown)", ExampleValue: "India", AlwaysRequired: true},
        {Key: "gstin", Label: "GSTIN", Description: "15-character GST Identification Number", FormatRule: "Format: 22AAAAA0000A1Z5", ExampleValue: "27AAPFU0939F1ZV"},
        {Key: "pan", Label: "PAN", Description: "10-character Permanent Account Number", FormatRule: "Format: ABCDE1234F", ExampleValue: "ABCDE1234F"},
    }
}

// BaseInstallAtFields returns the ordered list of fields for Install At addresses.
// It prepends a "Ship To Reference" column that links to a Ship To site_name.
func BaseInstallAtFields() []AddressField {
    installFields := []AddressField{
        {Key: "ship_to_reference", Label: "Ship To Reference", Description: "Must match an existing Ship To 'Site Name' in this project", FormatRule: "Exact match required", ExampleValue: "Mumbai HQ", AlwaysRequired: true},
    }
    return append(installFields, BaseShipToFields()...)
}

// IndianStates returns the list of Indian states and union territories.
var IndianStates = []string{
    "Andhra Pradesh", "Arunachal Pradesh", "Assam", "Bihar",
    "Chhattisgarh", "Goa", "Gujarat", "Haryana", "Himachal Pradesh",
    "Jharkhand", "Karnataka", "Kerala", "Madhya Pradesh", "Maharashtra",
    "Manipur", "Meghalaya", "Mizoram", "Nagaland", "Odisha", "Punjab",
    "Rajasthan", "Sikkim", "Tamil Nadu", "Telangana", "Tripura",
    "Uttar Pradesh", "Uttarakhand", "West Bengal",
    "Andaman and Nicobar Islands", "Chandigarh",
    "Dadra and Nagar Haveli and Daman and Diu", "Delhi",
    "Jammu and Kashmir", "Ladakh", "Lakshadweep", "Puducherry",
}

// Countries returns the common country list (India first, then alphabetical).
var Countries = []string{
    "India", "Afghanistan", "Australia", "Bangladesh", "Bhutan",
    "Canada", "China", "France", "Germany", "Indonesia",
    "Japan", "Malaysia", "Nepal", "Pakistan", "Singapore",
    "Sri Lanka", "Thailand", "United Arab Emirates",
    "United Kingdom", "United States",
}
```

### Step 2: Add Collections (`collections/setup.go`)

Add the following collections inside the `Setup` function. These are needed by Phase 10-12 but defining them now ensures the template handler can look up required fields.

```go
// projects collection
projects := ensureCollection(app, "projects", func(c *core.Collection) {
    c.Fields.Add(&core.TextField{Name: "name", Required: true})
    c.Fields.Add(&core.TextField{Name: "reference_number", Required: false})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})

// project_address_settings - per-project, per-type required field configuration
ensureCollection(app, "project_address_settings", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{
        Name:         "project",
        Required:     true,
        CollectionId: projects.Id,
        CascadeDelete: true,
        MaxSelect:    1,
    })
    c.Fields.Add(&core.SelectField{
        Name:     "address_type",
        Required: true,
        Values:   []string{"ship_to", "install_at"},
        MaxSelect: 1,
    })
    // JSON field storing array of field keys that are required
    // e.g. ["phone","email","gstin","pin_code"]
    c.Fields.Add(&core.JSONField{Name: "required_fields", MaxSize: 10000})
})

// ship_to_addresses
shipTo := ensureCollection(app, "ship_to_addresses", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{
        Name:         "project",
        Required:     true,
        CollectionId: projects.Id,
        CascadeDelete: true,
        MaxSelect:    1,
    })
    c.Fields.Add(&core.TextField{Name: "site_name", Required: true})
    c.Fields.Add(&core.TextField{Name: "contact_person"})
    c.Fields.Add(&core.TextField{Name: "phone"})
    c.Fields.Add(&core.EmailField{Name: "email"})
    c.Fields.Add(&core.TextField{Name: "address_line1", Required: true})
    c.Fields.Add(&core.TextField{Name: "address_line2"})
    c.Fields.Add(&core.TextField{Name: "city", Required: true})
    c.Fields.Add(&core.TextField{Name: "state", Required: true})
    c.Fields.Add(&core.TextField{Name: "pin_code"})
    c.Fields.Add(&core.TextField{Name: "country", Required: true})
    c.Fields.Add(&core.TextField{Name: "gstin"})
    c.Fields.Add(&core.TextField{Name: "pan"})
})

// install_at_addresses
ensureCollection(app, "install_at_addresses", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{
        Name:         "project",
        Required:     true,
        CollectionId: projects.Id,
        CascadeDelete: true,
        MaxSelect:    1,
    })
    c.Fields.Add(&core.RelationField{
        Name:         "ship_to_parent",
        Required:     false,
        CollectionId: shipTo.Id,
        MaxSelect:    1,
    })
    c.Fields.Add(&core.TextField{Name: "site_name", Required: true})
    c.Fields.Add(&core.TextField{Name: "contact_person"})
    c.Fields.Add(&core.TextField{Name: "phone"})
    c.Fields.Add(&core.EmailField{Name: "email"})
    c.Fields.Add(&core.TextField{Name: "address_line1", Required: true})
    c.Fields.Add(&core.TextField{Name: "address_line2"})
    c.Fields.Add(&core.TextField{Name: "city", Required: true})
    c.Fields.Add(&core.TextField{Name: "state", Required: true})
    c.Fields.Add(&core.TextField{Name: "pin_code"})
    c.Fields.Add(&core.TextField{Name: "country", Required: true})
    c.Fields.Add(&core.TextField{Name: "gstin"})
    c.Fields.Add(&core.TextField{Name: "pan"})
})
```

### Step 3: Template Generation Service (`services/address_template.go`)

```go
package services

import (
    "bytes"
    "fmt"

    "github.com/pocketbase/pocketbase"
    "github.com/xuri/excelize/v2"
)

// GenerateAddressTemplate creates a downloadable .xlsx template for the given
// project and address type ("ship_to" or "install_at").
func GenerateAddressTemplate(app *pocketbase.PocketBase, projectID, addressType string) ([]byte, error) {
    // 1. Determine fields for the address type
    var fields []AddressField
    if addressType == "install_at" {
        fields = BaseInstallAtFields()
    } else {
        fields = BaseShipToFields()
    }

    // 2. Fetch project-specific required field overrides
    requiredSet, err := getProjectRequiredFields(app, projectID, addressType)
    if err != nil {
        return nil, fmt.Errorf("fetch required fields: %w", err)
    }

    // 3. Build the workbook
    f := excelize.NewFile()
    defer f.Close()

    sheetName := "Addresses"
    defaultSheet := f.GetSheetName(0)
    f.SetSheetName(defaultSheet, sheetName)

    // --- Styles ---
    requiredHeaderStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
        Fill: excelize.Fill{Type: "pattern", Color: []string{"#1D4ED8"}, Pattern: 1}, // blue-700
        Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
        Border: thinBorders(),
    })

    optionalHeaderStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
        Fill: excelize.Fill{Type: "pattern", Color: []string{"#6B7280"}, Pattern: 1}, // gray-500
        Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
        Border: thinBorders(),
    })

    // 4. Write header row and set column widths
    columns := columnLetters(len(fields))
    for i, field := range fields {
        cell := fmt.Sprintf("%s1", columns[i])

        // Header text: append " *" to required fields
        headerText := field.Label
        isRequired := field.AlwaysRequired || requiredSet[field.Key]
        if isRequired {
            headerText += " *"
        }
        f.SetCellValue(sheetName, cell, headerText)

        // Apply style
        if isRequired {
            f.SetCellStyle(sheetName, cell, cell, requiredHeaderStyle)
        } else {
            f.SetCellStyle(sheetName, cell, cell, optionalHeaderStyle)
        }

        // Column width
        width := float64(len(field.Label)) * 1.3
        if width < 15 {
            width = 15
        }
        f.SetColWidth(sheetName, columns[i], columns[i], width)
    }

    // 5. Add data validation dropdowns for State and Country
    for i, field := range fields {
        col := columns[i]
        rangeRef := fmt.Sprintf("%s2:%s1048576", col, col) // entire column below header

        switch field.Key {
        case "state":
            dv := excelize.NewDataValidation(true)
            dv.Sqref = rangeRef
            dv.SetDropList(IndianStates)
            f.AddDataValidation(sheetName, dv)
        case "country":
            dv := excelize.NewDataValidation(true)
            dv.Sqref = rangeRef
            dv.SetDropList(Countries)
            f.AddDataValidation(sheetName, dv)
        }
    }

    // 6. Freeze header row
    f.SetPanes(sheetName, &excelize.Panes{
        Freeze:      true,
        Split:       false,
        XSplit:      0,
        YSplit:      1,
        TopLeftCell: "A2",
        ActivePane:  "bottomLeft",
    })

    // 7. Create hidden Instructions sheet
    addInstructionsSheet(f, fields, requiredSet, addressType)

    // 8. Write to buffer
    var buf bytes.Buffer
    if err := f.Write(&buf); err != nil {
        return nil, fmt.Errorf("write excel template: %w", err)
    }
    return buf.Bytes(), nil
}

// getProjectRequiredFields loads the project_address_settings record and
// returns a set of field keys that are marked required.
func getProjectRequiredFields(app *pocketbase.PocketBase, projectID, addressType string) (map[string]bool, error) {
    result := make(map[string]bool)

    col, err := app.FindCollectionByNameOrId("project_address_settings")
    if err != nil {
        return result, nil // collection doesn't exist yet, no custom settings
    }

    records, err := app.FindRecordsByFilter(
        col,
        "project = {:projectId} && address_type = {:addrType}",
        "", 1, 0,
        map[string]any{
            "projectId": projectID,
            "addrType":  addressType,
        },
    )
    if err != nil || len(records) == 0 {
        return result, nil
    }

    // required_fields is stored as a JSON array of strings
    raw := records[0].Get("required_fields")
    if arr, ok := raw.([]any); ok {
        for _, v := range arr {
            if s, ok := v.(string); ok {
                result[s] = true
            }
        }
    }
    return result, nil
}

// addInstructionsSheet creates a hidden sheet with field descriptions.
func addInstructionsSheet(f *excelize.File, fields []AddressField, requiredSet map[string]bool, addressType string) {
    instSheet := "Instructions"
    f.NewSheet(instSheet)

    titleStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true, Size: 14},
    })
    headerStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true, Size: 11},
        Fill: excelize.Fill{Type: "pattern", Color: []string{"#E5E7EB"}, Pattern: 1},
    })

    typeName := "Ship To"
    if addressType == "install_at" {
        typeName = "Install At"
    }
    f.SetCellValue(instSheet, "A1", fmt.Sprintf("%s Address Import - Instructions", typeName))
    f.SetCellStyle(instSheet, "A1", "A1", titleStyle)

    // Column headers
    instructionHeaders := []string{"Field Name", "Required?", "Format Rule", "Description", "Example"}
    for i, h := range instructionHeaders {
        cell := fmt.Sprintf("%s3", columnLetters(5)[i])
        f.SetCellValue(instSheet, cell, h)
        f.SetCellStyle(instSheet, cell, cell, headerStyle)
    }

    cols := columnLetters(5)
    for i, field := range fields {
        row := fmt.Sprintf("%d", i+4)
        reqLabel := "Optional"
        if field.AlwaysRequired || requiredSet[field.Key] {
            reqLabel = "Required"
        }
        f.SetCellValue(instSheet, cols[0]+row, field.Label)
        f.SetCellValue(instSheet, cols[1]+row, reqLabel)
        f.SetCellValue(instSheet, cols[2]+row, field.FormatRule)
        f.SetCellValue(instSheet, cols[3]+row, field.Description)
        f.SetCellValue(instSheet, cols[4]+row, field.ExampleValue)
    }

    // Set column widths
    widths := []float64{20, 12, 30, 45, 25}
    for i, w := range widths {
        f.SetColWidth(instSheet, cols[i], cols[i], w)
    }

    // Hide the Instructions sheet
    f.SetSheetVisible(instSheet, false)
}

// columnLetters returns Excel column letters for n columns: A, B, ... Z, AA, AB ...
func columnLetters(n int) []string {
    cols := make([]string, n)
    for i := 0; i < n; i++ {
        name, _ := excelize.ColumnNumberToName(i + 1)
        cols[i] = name
    }
    return cols
}
```

### Step 4: HTTP Handler (`handlers/address_template.go`)

```go
package handlers

import (
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"

    "projectcreation/services"
)

// HandleAddressTemplateDownload serves the Excel template for address import.
// Route: GET /projects/{projectId}/addresses/{type}/template
func HandleAddressTemplateDownload(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectID := e.Request.PathValue("projectId")
        addressType := e.Request.PathValue("type") // "ship_to" or "install_at"

        if projectID == "" {
            return e.String(http.StatusBadRequest, "Missing project ID")
        }
        if addressType != "ship_to" && addressType != "install_at" {
            return e.String(http.StatusBadRequest, "Invalid address type. Must be ship_to or install_at")
        }

        // Verify project exists
        if _, err := app.FindRecordById("projects", projectID); err != nil {
            return e.String(http.StatusNotFound, "Project not found")
        }

        xlsxBytes, err := services.GenerateAddressTemplate(app, projectID, addressType)
        if err != nil {
            log.Printf("address_template: failed to generate: %v", err)
            return e.String(http.StatusInternalServerError, "Failed to generate template")
        }

        typeName := "ShipTo"
        if addressType == "install_at" {
            typeName = "InstallAt"
        }
        filename := fmt.Sprintf("%s_Template_%d.xlsx", typeName, time.Now().Year())

        e.Response.Header().Set("Content-Type",
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
        e.Response.Header().Set("Content-Disposition",
            fmt.Sprintf(`attachment; filename="%s"`, filename))
        e.Response.Write(xlsxBytes)
        return nil
    }
}
```

### Step 5: Register Route (`main.go`)

Add inside the `OnServe` block:

```go
// Address template download
se.Router.GET("/projects/{projectId}/addresses/{type}/template",
    handlers.HandleAddressTemplateDownload(app))
```

---

## Dependencies on Other Phases

| Dependency | Detail |
|-----------|--------|
| **None (self-contained)** | Phase 10 introduces the new collections and the template service. |
| Phase 11 depends on Phase 10 | The field definitions (`address_fields.go`) and collections created here are reused by the validation engine. |
| Phase 12 depends on Phase 10 | The collections (`ship_to_addresses`, `install_at_addresses`) are the insert targets. |

---

## Testing / Verification Steps

1. **Unit test: field list completeness**
   - `TestBaseShipToFields` -- verify 12 fields returned, no duplicates.
   - `TestBaseInstallAtFields` -- verify 13 fields, first field is `ship_to_reference`.

2. **Unit test: template generation**
   - Call `GenerateAddressTemplate(app, projectID, "ship_to")` with a seeded project that has custom required fields `["phone", "gstin"]`.
   - Open the resulting bytes with Excelize and assert:
     - Sheet "Addresses" exists with 12 header columns.
     - Columns for `site_name`, `address_line1`, `city`, `state`, `country` have blue headers (always required).
     - Columns for `phone`, `gstin` also have blue headers (project-required).
     - Remaining columns have grey headers.
     - Data validation on the `state` column contains all 36 Indian states/UTs.
     - Hidden sheet "Instructions" exists with correct row count.

3. **Integration test: HTTP handler**
   - `GET /projects/{id}/addresses/ship_to/template` returns 200 with correct Content-Type and Content-Disposition.
   - `GET /projects/{id}/addresses/install_at/template` returns xlsx whose first header column is "Ship To Reference *".
   - `GET /projects/{id}/addresses/invalid_type/template` returns 400.
   - `GET /projects/nonexistent/addresses/ship_to/template` returns 404.

4. **Manual QA**
   - Download both templates in a browser.
   - Open in Excel/Google Sheets: confirm dropdowns work, colors are correct, Instructions sheet is hidden but accessible.

---

## Acceptance Criteria

- [ ] `GET /projects/{projectId}/addresses/ship_to/template` downloads a valid `.xlsx` file.
- [ ] `GET /projects/{projectId}/addresses/install_at/template` downloads a valid `.xlsx` file with "Ship To Reference" as the first column.
- [ ] Required fields (always-required + project-configured) have blue headers with " *" suffix.
- [ ] Optional fields have grey headers.
- [ ] State column has a dropdown with all Indian states and UTs.
- [ ] Country column has a dropdown with the predefined country list.
- [ ] Header row is frozen (stays visible on scroll).
- [ ] Hidden "Instructions" sheet contains field descriptions, format rules, and example values.
- [ ] Template opens without errors in Microsoft Excel, Google Sheets, and LibreOffice Calc.
- [ ] No regressions in existing BOQ export functionality.
