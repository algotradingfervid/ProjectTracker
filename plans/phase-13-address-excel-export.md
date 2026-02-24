# Phase 13: Address Excel Export

## Overview & Objectives

Add the ability to export all addresses of a given type (Bill From, Ship From, Bill To, Ship To, Install At) for a project as a formatted Excel file. The export follows the same Excelize patterns established in `services/export_excel.go` for BOQ exports, providing formatted headers, auto-sized columns, a frozen header row, and a proper file download via `Content-Disposition`.

For "Install At" addresses, an extra human-readable column "Ship To Parent (Company Name)" is included so users can see which Ship To address each Install At is linked to.

---

## Files to Create/Modify

| Action | Path |
|--------|------|
| **Create** | `services/address_export.go` |
| **Create** | `handlers/address_export.go` |
| **Modify** | `main.go` (register route) |

---

## Detailed Implementation Steps

### Step 1: Define Address Export Data Structures

Create `services/address_export.go` with the data structures and Excel generation function.

```go
package services

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// AddressExportColumn defines a column in the address export spreadsheet.
type AddressExportColumn struct {
	Header string
	Field  string  // field name on the PocketBase record
	Width  float64 // column width in Excel units
}

// AddressExportData holds all data needed for address export.
type AddressExportData struct {
	ProjectName string
	AddressType string            // "bill_from", "ship_from", "bill_to", "ship_to", "install_at"
	TypeLabel   string            // "Bill From", "Ship From", etc.
	Columns     []AddressExportColumn
	Rows        []map[string]string // each row is field -> value
}
```

### Step 2: Define Columns Per Address Type

Each address type shares a common set of fields. Install At gets an additional "Ship To Parent (Company Name)" column.

```go
// GetAddressColumns returns the export columns for a given address type.
func GetAddressColumns(addressType string) []AddressExportColumn {
	// Common columns shared by all address types
	common := []AddressExportColumn{
		{Header: "Company Name", Field: "company_name", Width: 30},
		{Header: "Contact Person", Field: "contact_person", Width: 25},
		{Header: "Email", Field: "email", Width: 30},
		{Header: "Phone", Field: "phone", Width: 18},
		{Header: "Address Line 1", Field: "address_line_1", Width: 35},
		{Header: "Address Line 2", Field: "address_line_2", Width: 35},
		{Header: "City", Field: "city", Width: 20},
		{Header: "State", Field: "state", Width: 20},
		{Header: "PIN Code", Field: "pin_code", Width: 12},
		{Header: "Country", Field: "country", Width: 18},
		{Header: "GSTIN", Field: "gstin", Width: 20},
	}

	if addressType == "install_at" {
		// Prepend the Ship To Parent reference column for Install At
		parentCol := AddressExportColumn{
			Header: "Ship To Parent (Company Name)",
			Field:  "_ship_to_parent_name", // virtual field resolved at export time
			Width:  35,
		}
		return append([]AddressExportColumn{parentCol}, common...)
	}

	return common
}
```

### Step 3: Implement `GenerateAddressExcel`

Follow the same style as `GenerateExcel` in `services/export_excel.go`:

```go
// GenerateAddressExcel creates an Excel file from address export data.
func GenerateAddressExcel(data AddressExportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := data.TypeLabel + " Addresses"
	if len(sheetName) > 31 {
		sheetName = sheetName[:31]
	}

	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheetName); err != nil {
		return nil, fmt.Errorf("set sheet name: %w", err)
	}

	// --- Styles (matching existing BOQ export patterns) ---

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16},
	})

	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 11},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#333333"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: thinBorders(),
	})

	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Size: 10},
		Border: thinBorders(),
		Alignment: &excelize.Alignment{
			Vertical: "center",
			WrapText: true,
		},
	})

	// --- Column widths ---
	for i, col := range data.Columns {
		colLetter := colName(i)
		f.SetColWidth(sheetName, colLetter, colLetter, col.Width)
	}

	// --- Row 1: Title ---
	lastCol := colName(len(data.Columns) - 1)
	f.MergeCell(sheetName, "A1", lastCol+"1")
	f.SetCellValue(sheetName, "A1", data.ProjectName+" - "+data.TypeLabel+" Addresses")
	f.SetCellStyle(sheetName, "A1", lastCol+"1", titleStyle)

	// --- Row 2: Subtitle with count ---
	f.MergeCell(sheetName, "A2", lastCol+"2")
	f.SetCellValue(sheetName, "A2", fmt.Sprintf("Total: %d addresses", len(data.Rows)))
	f.SetCellStyle(sheetName, "A2", lastCol+"2", subtitleStyle)

	// --- Row 4: Column headers ---
	for i, col := range data.Columns {
		cell := fmt.Sprintf("%s4", colName(i))
		f.SetCellValue(sheetName, cell, col.Header)
	}
	f.SetCellStyle(sheetName, "A4", lastCol+"4", headerStyle)

	// --- Freeze header row (freeze pane below row 4) ---
	f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      4,
		TopLeftCell: "A5",
		ActivePane:  "bottomLeft",
	})

	// --- Data rows starting at row 5 ---
	for rowIdx, rowData := range data.Rows {
		rowNum := rowIdx + 5
		rowStr := fmt.Sprintf("%d", rowNum)
		for colIdx, col := range data.Columns {
			cell := fmt.Sprintf("%s%s", colName(colIdx), rowStr)
			f.SetCellValue(sheetName, cell, rowData[col.Field])
		}
		f.SetCellStyle(sheetName, "A"+rowStr, lastCol+rowStr, dataStyle)
	}

	// --- Write to buffer ---
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write excel: %w", err)
	}

	return buf.Bytes(), nil
}

// colName converts a 0-based column index to an Excel column letter (A, B, ..., Z, AA, ...).
func colName(index int) string {
	name := ""
	for index >= 0 {
		name = string(rune('A'+index%26)) + name
		index = index/26 - 1
	}
	return name
}
```

### Step 4: Create the Export Handler

Create `handlers/address_export.go`:

```go
package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
)

// addressTypeLabels maps URL type slugs to human-readable labels.
var addressTypeLabels = map[string]string{
	"bill_from":  "Bill From",
	"ship_from":  "Ship From",
	"bill_to":    "Bill To",
	"ship_to":    "Ship To",
	"install_at": "Install At",
}

// HandleAddressExportExcel exports all addresses of a given type for a project.
// Route: GET /projects/{projectId}/addresses/{type}/export
func HandleAddressExportExcel(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		addrType := e.Request.PathValue("type")

		if projectID == "" || addrType == "" {
			return e.String(http.StatusBadRequest, "Missing project ID or address type")
		}

		typeLabel, ok := addressTypeLabels[addrType]
		if !ok {
			return e.String(http.StatusBadRequest, "Invalid address type")
		}

		// Fetch project record for the name
		project, err := app.FindRecordById("projects", projectID)
		if err != nil {
			log.Printf("address_export: project not found %s: %v", projectID, err)
			return e.String(http.StatusNotFound, "Project not found")
		}
		projectName := project.GetString("name")

		// Fetch addresses collection
		collectionName := addrType + "_addresses"
		addrCol, err := app.FindCollectionByNameOrId(collectionName)
		if err != nil {
			log.Printf("address_export: collection %s not found: %v", collectionName, err)
			return e.String(http.StatusInternalServerError, "Address collection not found")
		}

		// Query all addresses for this project
		records, err := app.FindRecordsByFilter(
			addrCol,
			"project = {:projectId}",
			"created",
			0, 0,
			map[string]any{"projectId": projectID},
		)
		if err != nil {
			log.Printf("address_export: query failed: %v", err)
			records = nil
		}

		// Build columns and rows
		columns := services.GetAddressColumns(addrType)
		var rows []map[string]string

		for _, rec := range records {
			row := make(map[string]string)
			for _, col := range columns {
				if col.Field == "_ship_to_parent_name" {
					// Resolve Ship To parent company name
					parentID := rec.GetString("ship_to_parent")
					if parentID != "" {
						parentRec, err := app.FindRecordById("ship_to_addresses", parentID)
						if err == nil {
							row[col.Field] = parentRec.GetString("company_name")
						} else {
							row[col.Field] = "(unknown)"
						}
					} else {
						row[col.Field] = ""
					}
				} else {
					row[col.Field] = rec.GetString(col.Field)
				}
			}
			rows = append(rows, row)
		}

		exportData := services.AddressExportData{
			ProjectName: projectName,
			AddressType: addrType,
			TypeLabel:   typeLabel,
			Columns:     columns,
			Rows:        rows,
		}

		xlsxBytes, err := services.GenerateAddressExcel(exportData)
		if err != nil {
			log.Printf("address_export: generate failed: %v", err)
			return e.String(http.StatusInternalServerError, "Failed to generate Excel file")
		}

		// Filename: {ProjectName}_{AddressType}_Addresses.xlsx
		filename := fmt.Sprintf("%s_%s_Addresses.xlsx",
			sanitizeFilename(projectName),
			strings.ReplaceAll(typeLabel, " ", ""),
		)

		e.Response.Header().Set("Content-Type",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		e.Response.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, filename))
		e.Response.Write(xlsxBytes)
		return nil
	}
}
```

### Step 5: Register the Route in `main.go`

Add the following line in the `app.OnServe().BindFunc` block in `main.go`:

```go
// Address export
se.Router.GET("/projects/{projectId}/addresses/{type}/export",
    handlers.HandleAddressExportExcel(app))
```

---

## Dependencies on Other Phases

- **Phase 10 (assumed)**: The `projects` collection and address collections (`bill_from_addresses`, `ship_from_addresses`, `bill_to_addresses`, `ship_to_addresses`, `install_at_addresses`) must already exist in `collections/setup.go`.
- **Phase 12 (assumed)**: Address CRUD handlers and templates should be in place so there is data to export.
- The `sanitizeFilename` helper already exists in `handlers/export.go` and can be reused directly since it is in the same package.
- The `thinBorders()` helper already exists in `services/export_excel.go` and is reused.

---

## Testing / Verification Steps

1. **Manual verification**:
   - Create a project with several addresses of each type.
   - For Install At addresses, link some to Ship To parents.
   - Navigate to `GET /projects/{projectId}/addresses/ship_to/export` and verify the file downloads.
   - Open the downloaded `.xlsx` and confirm:
     - Title row shows `{ProjectName} - Ship To Addresses`.
     - Subtitle shows correct count.
     - Header row is frozen (scroll down and headers stay visible).
     - Columns are properly sized and headers are formatted (white text, dark background).
     - All address records appear as data rows.
   - Repeat for `install_at` type and verify the "Ship To Parent (Company Name)" column is populated correctly.

2. **Edge cases**:
   - Export with zero addresses: file should download with headers only and no data rows.
   - Invalid address type in URL: returns 400 Bad Request.
   - Invalid project ID: returns 404 Not Found.
   - Install At address with no `ship_to_parent` set: column value should be empty string, not an error.

3. **Regression**:
   - Existing BOQ Excel export (`/boq/{id}/export/excel`) still works unchanged.
   - No compilation errors after adding new files.

---

## Acceptance Criteria

- [ ] `GET /projects/{projectId}/addresses/{type}/export` returns a valid `.xlsx` file download for each of the 5 address types.
- [ ] Filename follows the pattern `{ProjectName}_{AddressType}_Addresses.xlsx` (e.g., `MyProject_ShipTo_Addresses.xlsx`).
- [ ] Excel file has a title row, subtitle with count, formatted header row with dark background and white bold text.
- [ ] Header row is frozen so it remains visible when scrolling.
- [ ] Column widths are set appropriately (not default narrow columns).
- [ ] All fields from the address record are exported as columns.
- [ ] For Install At type, a "Ship To Parent (Company Name)" column is included and correctly resolved.
- [ ] Exporting a type with zero addresses produces a valid file with headers and no data rows.
- [ ] Invalid type or project ID returns appropriate HTTP error status codes.
- [ ] Content-Disposition header is set for file download (not inline display).
- [ ] Existing BOQ export functionality is unaffected.
