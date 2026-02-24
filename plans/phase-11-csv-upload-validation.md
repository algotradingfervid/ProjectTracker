# Phase 11: CSV/Excel Upload & Validation Engine

## Overview & Objectives

Allow users to upload a CSV or Excel file containing address rows, validate every row against the project's required-field configuration and field-format rules, and present a clear pass/fail report. No data is written to PocketBase in this phase -- the goal is **validation only** with a preview before committing (Phase 12).

### Key goals

1. Multipart file upload endpoint accepting `.csv` and `.xlsx` files.
2. Row-by-row validation engine that checks:
   - Required fields (from `project_address_settings` + always-required fields).
   - Format rules: PIN (6 digits), GSTIN (15-char regex), PAN (10-char regex), Email, Phone (10 digits).
   - For Install At: Ship To Reference must match an existing Ship To `site_name` in the project.
3. Structured error report: `{row_number, field_name, error_message}`.
4. Templ-based upload page with drag-and-drop zone.
5. After validation, show green/red summary and an error table with "Download Error Report" button.
6. "Confirm Import" button is only enabled when there are zero errors.

---

## Files to Create / Modify

| Action | Path | Purpose |
|--------|------|---------|
| **Create** | `services/address_validation.go` | Validation engine: parse file, validate rows, return report |
| **Create** | `handlers/address_import.go` | Upload + validation handler, error report download handler |
| **Create** | `templates/address_import.templ` | Upload form + validation results UI |
| **Modify** | `main.go` | Register POST route and GET routes for the import page |

---

## Detailed Implementation Steps

### Step 1: Validation Data Types (`services/address_validation.go` -- top of file)

```go
package services

import (
    "encoding/csv"
    "fmt"
    "io"
    "mime/multipart"
    "regexp"
    "strings"

    "github.com/pocketbase/pocketbase"
    "github.com/xuri/excelize/v2"
)

// ValidationError represents a single field-level error on one row.
type ValidationError struct {
    Row     int    `json:"row"`
    Field   string `json:"field"`
    Message string `json:"message"`
}

// ValidationResult is returned after parsing and validating an uploaded file.
type ValidationResult struct {
    TotalRows  int               `json:"total_rows"`
    ValidRows  int               `json:"valid_rows"`
    ErrorRows  int               `json:"error_rows"`
    Errors     []ValidationError `json:"errors"`
    // ParsedRows holds the raw data keyed by field Key, used later in Phase 12 commit.
    ParsedRows []map[string]string `json:"-"`
    // FileName is stored so the commit endpoint can reference the same file.
    FileName   string             `json:"-"`
}

// Pre-compiled regexps for field format validation.
var (
    rePIN   = regexp.MustCompile(`^\d{6}$`)
    rePhone = regexp.MustCompile(`^\d{10}$`)
    reEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
    reGSTIN = regexp.MustCompile(`^\d{2}[A-Z]{5}\d{4}[A-Z]\d[Z][A-Z0-9]$`)
    rePAN   = regexp.MustCompile(`^[A-Z]{5}\d{4}[A-Z]$`)
)
```

### Step 2: File Parsing Functions

```go
// parseCSV reads a CSV file and returns rows as []map[string]string keyed by
// the header values. The header row is matched to AddressField.Label (case-insensitive).
func parseCSV(file io.Reader) ([]string, [][]string, error) {
    reader := csv.NewReader(file)
    reader.TrimLeadingSpace = true
    reader.LazyQuotes = true

    allRows, err := reader.ReadAll()
    if err != nil {
        return nil, nil, fmt.Errorf("failed to parse CSV: %w", err)
    }
    if len(allRows) < 2 {
        return nil, nil, fmt.Errorf("file must contain a header row and at least one data row")
    }

    headers := allRows[0]
    dataRows := allRows[1:]
    return headers, dataRows, nil
}

// parseExcel reads an xlsx file and returns headers + data rows from the first sheet.
func parseExcel(file multipart.File) ([]string, [][]string, error) {
    f, err := excelize.OpenReader(file)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to open Excel file: %w", err)
    }
    defer f.Close()

    sheetName := f.GetSheetName(0)
    rows, err := f.GetRows(sheetName)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to read sheet: %w", err)
    }
    if len(rows) < 2 {
        return nil, nil, fmt.Errorf("file must contain a header row and at least one data row")
    }

    headers := rows[0]
    dataRows := rows[1:]
    return headers, dataRows, nil
}
```

### Step 3: Header Mapping

Map the uploaded file's header labels to canonical field keys. This makes the system resilient to minor header differences (e.g. extra spaces, case changes).

```go
// mapHeadersToFields maps uploaded column headers to AddressField keys.
// Returns: ordered list of field keys (one per column), and a list of
// any unrecognized columns.
func mapHeadersToFields(headers []string, fields []AddressField) ([]string, []string) {
    // Build a label -> key lookup (case-insensitive, trimmed, strip trailing " *")
    labelToKey := make(map[string]string, len(fields))
    for _, f := range fields {
        normalized := strings.ToLower(strings.TrimSpace(f.Label))
        labelToKey[normalized] = f.Key
    }

    mapped := make([]string, len(headers))
    var unrecognized []string

    for i, h := range headers {
        norm := strings.ToLower(strings.TrimSpace(h))
        // Strip " *" suffix that our template adds
        norm = strings.TrimSuffix(norm, " *")
        norm = strings.TrimSpace(norm)

        if key, ok := labelToKey[norm]; ok {
            mapped[i] = key
        } else {
            mapped[i] = "" // unmapped column
            unrecognized = append(unrecognized, h)
        }
    }
    return mapped, unrecognized
}
```

### Step 4: Core Validation Engine

```go
// ValidateAddressFile parses and validates an uploaded address file.
func ValidateAddressFile(
    app *pocketbase.PocketBase,
    file multipart.File,
    fileName string,
    projectID string,
    addressType string,
) (*ValidationResult, error) {
    // 1. Determine fields and required set
    var fields []AddressField
    if addressType == "install_at" {
        fields = BaseInstallAtFields()
    } else {
        fields = BaseShipToFields()
    }

    requiredSet, err := getProjectRequiredFields(app, projectID, addressType)
    if err != nil {
        return nil, fmt.Errorf("load required fields: %w", err)
    }

    // 2. Parse file based on extension
    var headers []string
    var dataRows [][]string

    lowerName := strings.ToLower(fileName)
    if strings.HasSuffix(lowerName, ".csv") {
        headers, dataRows, err = parseCSV(file)
    } else if strings.HasSuffix(lowerName, ".xlsx") {
        headers, dataRows, err = parseExcel(file)
    } else {
        return nil, fmt.Errorf("unsupported file format: must be .csv or .xlsx")
    }
    if err != nil {
        return nil, err
    }

    // 3. Map headers to field keys
    columnKeys, _ := mapHeadersToFields(headers, fields)

    // 4. Build required field key set (always-required + project-configured)
    isRequired := make(map[string]bool)
    for _, f := range fields {
        if f.AlwaysRequired || requiredSet[f.Key] {
            isRequired[f.Key] = true
        }
    }

    // 5. For Install At, load existing Ship To site names for reference validation
    var shipToNames map[string]bool
    if addressType == "install_at" {
        shipToNames, err = loadShipToSiteNames(app, projectID)
        if err != nil {
            return nil, fmt.Errorf("load ship to names: %w", err)
        }
    }

    // 6. Validate each row
    result := &ValidationResult{
        TotalRows:  len(dataRows),
        FileName:   fileName,
        ParsedRows: make([]map[string]string, 0, len(dataRows)),
    }

    for rowIdx, row := range dataRows {
        rowNum := rowIdx + 2 // 1-indexed, +1 for header row
        rowData := make(map[string]string)
        var rowErrors []ValidationError

        // Map columns to values
        for colIdx, key := range columnKeys {
            if key == "" {
                continue // unmapped column
            }
            value := ""
            if colIdx < len(row) {
                value = strings.TrimSpace(row[colIdx])
            }
            rowData[key] = value
        }

        // Check required fields
        for key := range isRequired {
            if rowData[key] == "" {
                // Find label for the field
                label := key
                for _, f := range fields {
                    if f.Key == key {
                        label = f.Label
                        break
                    }
                }
                rowErrors = append(rowErrors, ValidationError{
                    Row:     rowNum,
                    Field:   label,
                    Message: fmt.Sprintf("%s is required", label),
                })
            }
        }

        // Field-format validations (only if value is non-empty)
        rowErrors = append(rowErrors, validateFieldFormats(rowNum, rowData)...)

        // Ship To Reference validation for Install At
        if addressType == "install_at" {
            ref := rowData["ship_to_reference"]
            if ref != "" && !shipToNames[ref] {
                rowErrors = append(rowErrors, ValidationError{
                    Row:     rowNum,
                    Field:   "Ship To Reference",
                    Message: fmt.Sprintf("No Ship To address with site name %q found in this project", ref),
                })
            }
        }

        if len(rowErrors) > 0 {
            result.Errors = append(result.Errors, rowErrors...)
        }
        result.ParsedRows = append(result.ParsedRows, rowData)
    }

    // Compute summary
    errorRowSet := make(map[int]bool)
    for _, e := range result.Errors {
        errorRowSet[e.Row] = true
    }
    result.ErrorRows = len(errorRowSet)
    result.ValidRows = result.TotalRows - result.ErrorRows

    return result, nil
}

// validateFieldFormats checks format-specific rules for non-empty values.
func validateFieldFormats(rowNum int, data map[string]string) []ValidationError {
    var errs []ValidationError

    if v := data["pin_code"]; v != "" && !rePIN.MatchString(v) {
        errs = append(errs, ValidationError{Row: rowNum, Field: "PIN Code", Message: "PIN Code must be exactly 6 digits"})
    }
    if v := data["phone"]; v != "" && !rePhone.MatchString(v) {
        errs = append(errs, ValidationError{Row: rowNum, Field: "Phone", Message: "Phone must be exactly 10 digits"})
    }
    if v := data["email"]; v != "" && !reEmail.MatchString(v) {
        errs = append(errs, ValidationError{Row: rowNum, Field: "Email", Message: "Invalid email format"})
    }
    if v := data["gstin"]; v != "" && !reGSTIN.MatchString(strings.ToUpper(v)) {
        errs = append(errs, ValidationError{Row: rowNum, Field: "GSTIN", Message: "GSTIN must be 15 characters in format 22AAAAA0000A1Z5"})
    }
    if v := data["pan"]; v != "" && !rePAN.MatchString(strings.ToUpper(v)) {
        errs = append(errs, ValidationError{Row: rowNum, Field: "PAN", Message: "PAN must be 10 characters in format ABCDE1234F"})
    }

    return errs
}

// loadShipToSiteNames fetches all Ship To site_name values for a project.
func loadShipToSiteNames(app *pocketbase.PocketBase, projectID string) (map[string]bool, error) {
    col, err := app.FindCollectionByNameOrId("ship_to_addresses")
    if err != nil {
        return nil, err
    }

    records, err := app.FindRecordsByFilter(col,
        "project = {:projectId}", "", 0, 0,
        map[string]any{"projectId": projectID},
    )
    if err != nil {
        return make(map[string]bool), nil
    }

    names := make(map[string]bool, len(records))
    for _, r := range records {
        names[r.GetString("site_name")] = true
    }
    return names, nil
}
```

### Step 5: Generate Error Report Excel

```go
// GenerateErrorReport creates a downloadable .xlsx file from validation errors.
func GenerateErrorReport(errors []ValidationError) ([]byte, error) {
    f := excelize.NewFile()
    defer f.Close()

    sheet := "Errors"
    defaultSheet := f.GetSheetName(0)
    f.SetSheetName(defaultSheet, sheet)

    // Header style
    headerStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
        Fill: excelize.Fill{Type: "pattern", Color: []string{"#DC2626"}, Pattern: 1}, // red-600
        Alignment: &excelize.Alignment{Horizontal: "center"},
        Border: thinBorders(),
    })

    // Headers
    f.SetCellValue(sheet, "A1", "Row #")
    f.SetCellValue(sheet, "B1", "Field")
    f.SetCellValue(sheet, "C1", "Error")
    f.SetCellStyle(sheet, "A1", "C1", headerStyle)
    f.SetColWidth(sheet, "A", "A", 8)
    f.SetColWidth(sheet, "B", "B", 22)
    f.SetColWidth(sheet, "C", "C", 55)

    for i, e := range errors {
        row := fmt.Sprintf("%d", i+2)
        f.SetCellValue(sheet, "A"+row, e.Row)
        f.SetCellValue(sheet, "B"+row, e.Field)
        f.SetCellValue(sheet, "C"+row, e.Message)
    }

    var buf bytes.Buffer
    if err := f.Write(&buf); err != nil {
        return nil, fmt.Errorf("write error report: %w", err)
    }
    return buf.Bytes(), nil
}
```

### Step 6: HTTP Handler (`handlers/address_import.go`)

```go
package handlers

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"

    "projectcreation/services"
    "projectcreation/templates"
)

// HandleAddressImportPage renders the upload form.
// Route: GET /projects/{projectId}/addresses/{type}/import
func HandleAddressImportPage(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectID := e.Request.PathValue("projectId")
        addressType := e.Request.PathValue("type")

        if addressType != "ship_to" && addressType != "install_at" {
            return e.String(http.StatusBadRequest, "Invalid address type")
        }

        // Verify project exists
        project, err := app.FindRecordById("projects", projectID)
        if err != nil {
            return e.String(http.StatusNotFound, "Project not found")
        }

        component := templates.AddressImportPage(
            project.GetString("name"),
            projectID,
            addressType,
        )
        return component.Render(e.Request.Context(), e.Response)
    }
}

// HandleAddressValidate receives a file upload, validates it, and returns
// the validation results as an HTMX partial.
// Route: POST /projects/{projectId}/addresses/{type}/import
func HandleAddressValidate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectID := e.Request.PathValue("projectId")
        addressType := e.Request.PathValue("type")

        if addressType != "ship_to" && addressType != "install_at" {
            return e.String(http.StatusBadRequest, "Invalid address type")
        }

        // Parse multipart form (max 10MB)
        if err := e.Request.ParseMultipartForm(10 << 20); err != nil {
            return e.String(http.StatusBadRequest, "File too large or invalid form data")
        }

        file, header, err := e.Request.FormFile("file")
        if err != nil {
            return e.String(http.StatusBadRequest, "No file provided")
        }
        defer file.Close()

        // Validate the file
        result, err := services.ValidateAddressFile(app, file, header.Filename, projectID, addressType)
        if err != nil {
            log.Printf("address_validate: %v", err)
            return e.String(http.StatusBadRequest, err.Error())
        }

        // Store parsed result in a temp cache for Phase 12 commit
        // Using a simple approach: store the validation result JSON in a temp PocketBase record
        // or in-memory cache. For now, we serialize to JSON and embed in a hidden form field.
        resultJSON, _ := json.Marshal(result)

        // Return HTMX partial with results
        component := templates.AddressValidationResults(
            projectID,
            addressType,
            result.TotalRows,
            result.ValidRows,
            result.ErrorRows,
            result.Errors,
            string(resultJSON),
            header.Filename,
        )
        return component.Render(e.Request.Context(), e.Response)
    }
}

// HandleAddressErrorReport downloads the error report as an Excel file.
// Route: POST /projects/{projectId}/addresses/{type}/import/errors
func HandleAddressErrorReport(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        addressType := e.Request.PathValue("type")

        // Parse errors from the posted JSON
        var errors []services.ValidationError
        decoder := json.NewDecoder(e.Request.Body)
        if err := decoder.Decode(&errors); err != nil {
            return e.String(http.StatusBadRequest, "Invalid error data")
        }

        xlsxBytes, err := services.GenerateErrorReport(errors)
        if err != nil {
            log.Printf("error_report: %v", err)
            return e.String(http.StatusInternalServerError, "Failed to generate error report")
        }

        typeName := "ShipTo"
        if addressType == "install_at" {
            typeName = "InstallAt"
        }
        filename := fmt.Sprintf("%s_Errors_%s.xlsx", typeName,
            time.Now().Format("2006-01-02"))

        e.Response.Header().Set("Content-Type",
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
        e.Response.Header().Set("Content-Disposition",
            fmt.Sprintf(`attachment; filename="%s"`, filename))
        e.Response.Write(xlsxBytes)
        return nil
    }
}
```

### Step 7: Templ Template (`templates/address_import.templ`)

```go
package templates

import (
    "fmt"
    "projectcreation/services"
)

// AddressImportPage renders the initial upload form with drag-and-drop zone.
templ AddressImportPage(projectName, projectID, addressType string) {
    @Layout(fmt.Sprintf("Import %s Addresses", addressTypeName(addressType))) {
        <div class="max-w-4xl mx-auto p-6">
            <div class="mb-6">
                <h1 class="text-2xl font-bold">
                    Import { addressTypeName(addressType) } Addresses
                </h1>
                <p class="text-base-content/60 mt-1">
                    Project: { projectName }
                </p>
            </div>

            <!-- Download template link -->
            <div class="alert alert-info mb-6">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"></path>
                </svg>
                <span>
                    Need the template?
                    <a
                        href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s/template", projectID, addressType)) }
                        class="link link-primary font-semibold"
                    >
                        Download Excel Template
                    </a>
                </span>
            </div>

            <!-- Upload form with drag-and-drop -->
            <div
                x-data="{
                    dragging: false,
                    fileName: '',
                    handleDrop(e) {
                        this.dragging = false;
                        const file = e.dataTransfer.files[0];
                        if (file) {
                            this.fileName = file.name;
                            $refs.fileInput.files = e.dataTransfer.files;
                            htmx.trigger($refs.uploadForm, 'submit');
                        }
                    }
                }"
                class="mb-6"
            >
                <form
                    x-ref="uploadForm"
                    hx-post={ fmt.Sprintf("/projects/%s/addresses/%s/import", projectID, addressType) }
                    hx-target="#validation-results"
                    hx-swap="innerHTML"
                    hx-encoding="multipart/form-data"
                    hx-indicator="#upload-spinner"
                >
                    <div
                        class="border-2 border-dashed rounded-xl p-12 text-center transition-colors"
                        :class="dragging ? 'border-primary bg-primary/5' : 'border-base-300'"
                        @dragover.prevent="dragging = true"
                        @dragleave.prevent="dragging = false"
                        @drop.prevent="handleDrop($event)"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 mx-auto mb-4 text-base-content/30" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"></path>
                        </svg>
                        <p class="text-lg font-medium mb-2">
                            Drop your CSV or Excel file here
                        </p>
                        <p class="text-base-content/50 mb-4">or</p>
                        <label class="btn btn-primary">
                            Browse Files
                            <input
                                x-ref="fileInput"
                                type="file"
                                name="file"
                                accept=".csv,.xlsx"
                                class="hidden"
                                @change="fileName = $event.target.files[0]?.name || ''; htmx.trigger($refs.uploadForm, 'submit')"
                            />
                        </label>
                        <p x-show="fileName" x-text="'Selected: ' + fileName" class="mt-3 text-sm text-success"></p>
                    </div>
                </form>

                <!-- Loading spinner -->
                <div id="upload-spinner" class="htmx-indicator flex justify-center mt-4">
                    <span class="loading loading-spinner loading-lg text-primary"></span>
                    <span class="ml-2">Validating file...</span>
                </div>
            </div>

            <!-- Validation results target -->
            <div id="validation-results"></div>
        </div>
    }
}

// AddressValidationResults renders the validation summary and error table.
templ AddressValidationResults(
    projectID, addressType string,
    totalRows, validRows, errorRows int,
    errors []services.ValidationError,
    resultJSON string,
    fileName string,
) {
    <div class="space-y-6">
        <!-- Summary cards -->
        <div class="grid grid-cols-3 gap-4">
            <div class="stat bg-base-200 rounded-xl">
                <div class="stat-title">Total Rows</div>
                <div class="stat-value text-2xl">{ fmt.Sprint(totalRows) }</div>
            </div>
            <div class="stat bg-success/10 rounded-xl">
                <div class="stat-title">Valid</div>
                <div class="stat-value text-2xl text-success">{ fmt.Sprint(validRows) }</div>
            </div>
            <div class={ "stat rounded-xl", templ.KV("bg-error/10", errorRows > 0), templ.KV("bg-base-200", errorRows == 0) }>
                <div class="stat-title">Errors</div>
                <div class={ "stat-value text-2xl", templ.KV("text-error", errorRows > 0) }>
                    { fmt.Sprint(errorRows) }
                </div>
            </div>
        </div>

        if errorRows > 0 {
            <!-- Error table -->
            <div class="card bg-base-100 shadow">
                <div class="card-body">
                    <div class="flex justify-between items-center mb-4">
                        <h3 class="card-title text-error">Validation Errors</h3>
                        <button
                            class="btn btn-outline btn-sm"
                            x-data
                            @click={
                                templ.ComponentScript{
                                    Call: fmt.Sprintf(`
                                        fetch('/projects/%s/addresses/%s/import/errors', {
                                            method: 'POST',
                                            headers: {'Content-Type': 'application/json'},
                                            body: JSON.stringify(%s)
                                        })
                                        .then(r => r.blob())
                                        .then(blob => {
                                            const url = URL.createObjectURL(blob);
                                            const a = document.createElement('a');
                                            a.href = url;
                                            a.download = 'error_report.xlsx';
                                            a.click();
                                            URL.revokeObjectURL(url);
                                        })
                                    `, projectID, addressType, errorsToJSON(errors)),
                                }.Call
                            }
                        >
                            Download Error Report
                        </button>
                    </div>
                    <div class="overflow-x-auto max-h-96">
                        <table class="table table-xs table-zebra">
                            <thead>
                                <tr>
                                    <th>Row #</th>
                                    <th>Field</th>
                                    <th>Error</th>
                                </tr>
                            </thead>
                            <tbody>
                                for _, err := range errors {
                                    <tr>
                                        <td>{ fmt.Sprint(err.Row) }</td>
                                        <td>{ err.Field }</td>
                                        <td class="text-error">{ err.Message }</td>
                                    </tr>
                                }
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        }

        <!-- Confirm Import button -->
        <div class="flex justify-end gap-3">
            <a
                href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s/import", projectID, addressType)) }
                class="btn btn-ghost"
            >
                Upload Different File
            </a>
            if errorRows == 0 {
                <form
                    hx-post={ fmt.Sprintf("/projects/%s/addresses/%s/import/commit", projectID, addressType) }
                    hx-target="#validation-results"
                    hx-swap="innerHTML"
                    hx-indicator="#commit-spinner"
                    hx-encoding="multipart/form-data"
                >
                    <input type="hidden" name="file_name" value={ fileName }/>
                    <input type="hidden" name="result_json" value={ resultJSON }/>
                    <button type="submit" class="btn btn-primary">
                        Confirm Import ({ fmt.Sprint(validRows) } rows)
                    </button>
                </form>
            } else {
                <button class="btn btn-primary" disabled="disabled">
                    Confirm Import (fix errors first)
                </button>
            }
        </div>

        <!-- Commit loading spinner -->
        <div id="commit-spinner" class="htmx-indicator flex justify-center">
            <span class="loading loading-spinner loading-lg text-primary"></span>
            <span class="ml-2">Importing addresses...</span>
        </div>
    </div>
}

// addressTypeName returns a human-readable name for the address type.
func addressTypeName(t string) string {
    if t == "install_at" {
        return "Install At"
    }
    return "Ship To"
}

// errorsToJSON is a helper for embedding errors as JSON in a script tag.
func errorsToJSON(errors []services.ValidationError) string {
    // Minimal JSON serialization for the template
    b, _ := json.Marshal(errors)
    return string(b)
}
```

### Step 8: Register Routes (`main.go`)

Add inside the `OnServe` block:

```go
// Address import - upload & validate
se.Router.GET("/projects/{projectId}/addresses/{type}/import",
    handlers.HandleAddressImportPage(app))
se.Router.POST("/projects/{projectId}/addresses/{type}/import",
    handlers.HandleAddressValidate(app))

// Address import - download error report
se.Router.POST("/projects/{projectId}/addresses/{type}/import/errors",
    handlers.HandleAddressErrorReport(app))
```

---

## Dependencies on Other Phases

| Dependency | Detail |
|-----------|--------|
| **Phase 10** | Uses `AddressField` definitions from `services/address_fields.go`, `getProjectRequiredFields()`, and the PocketBase collections created in Phase 10. |
| Phase 12 depends on Phase 11 | The `ValidationResult` and parsed data flow into the commit endpoint. |

---

## Testing / Verification Steps

### Unit Tests (`services/address_validation_test.go`)

1. **TestValidateFieldFormats_PIN**
   - Valid: `"400001"` -- no error.
   - Invalid: `"40001"` (5 digits), `"ABCDEF"` (letters) -- error returned.

2. **TestValidateFieldFormats_GSTIN**
   - Valid: `"27AAPFU0939F1ZV"` -- no error.
   - Invalid: `"27AAPFU0939F1Z"` (14 chars), `"XXAAPFU0939F1ZV"` (bad format) -- error.

3. **TestValidateFieldFormats_PAN**
   - Valid: `"ABCDE1234F"` -- no error.
   - Invalid: `"ABCDE123F"` (9 chars), `"12345ABCDE"` (wrong order) -- error.

4. **TestMapHeadersToFields**
   - Headers `["Site Name *", "City", "Unknown Col"]` correctly map to `["site_name", "city", ""]`.

5. **TestValidateAddressFile_RequiredFields**
   - Upload a CSV missing `site_name` on row 3. Verify error: `{Row: 3, Field: "Site Name", Message: "..."}`.

6. **TestValidateAddressFile_ShipToReference**
   - Seed a Ship To address with `site_name = "HQ"`.
   - Upload Install At CSV with reference `"Branch"` (does not exist). Verify error on that row.

7. **TestValidateAddressFile_HappyPath**
   - All rows valid. `ErrorRows == 0`, `ValidRows == TotalRows`.

### Integration Tests

8. **TestHTTPValidateEndpoint**
   - POST multipart form with a valid CSV to `/projects/{id}/addresses/ship_to/import`.
   - Assert 200 response, HTML contains "Valid" count.

9. **TestHTTPValidateEndpoint_NoFile**
   - POST without file attachment. Assert 400.

10. **TestHTTPErrorReportDownload**
    - POST JSON errors to `/projects/{id}/addresses/ship_to/import/errors`.
    - Assert response is a valid xlsx with correct row count.

### Manual QA

11. Open the import page in a browser, drag-drop a CSV file, and verify the validation results render correctly.
12. Click "Download Error Report" and verify the xlsx opens with correct error data.
13. Verify the "Confirm Import" button is disabled when errors exist and enabled when clean.

---

## Acceptance Criteria

- [ ] `POST /projects/{projectId}/addresses/{type}/import` accepts `.csv` and `.xlsx` files up to 10MB.
- [ ] Unsupported file types return a clear error message.
- [ ] Required fields (always-required + project-configured) are validated -- missing values produce row-level errors.
- [ ] PIN Code validation: exactly 6 digits.
- [ ] GSTIN validation: 15-character format `22AAAAA0000A1Z5`.
- [ ] PAN validation: 10-character format `ABCDE1234F`.
- [ ] Email validation: standard email format.
- [ ] Phone validation: exactly 10 digits.
- [ ] For Install At: Ship To Reference is validated against existing Ship To `site_name` values in the project.
- [ ] Validation results show green (valid count) and red (error count) summary cards.
- [ ] Error table shows Row #, Field, Error message for every validation error.
- [ ] "Download Error Report" button generates and downloads an `.xlsx` file with all errors.
- [ ] "Confirm Import" button is enabled only when there are 0 errors.
- [ ] Upload form supports drag-and-drop and file browser selection.
- [ ] Loading indicator appears during validation.
- [ ] File with only a header row and no data returns a clear error message.
