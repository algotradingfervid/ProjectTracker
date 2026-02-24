# Phase 12: CSV Import Commit (Batch Insert into PocketBase)

## Overview & Objectives

After Phase 11 validates an uploaded file with zero errors, the user clicks "Confirm Import". This phase implements the commit endpoint that **re-validates** the file (in case project settings changed between validation and commit), batch-inserts all rows into PocketBase, and handles progress tracking, transactions, and error recovery for imports that can range from 50 to 1000+ rows.

### Key goals

1. Re-validate before committing (settings may have changed).
2. Batch insert into `ship_to_addresses` or `install_at_addresses` collections.
3. For Install At: resolve `ship_to_reference` to a `ship_to_parent` relation ID.
4. Chunked processing for large files (batches of 100 rows).
5. All-or-nothing transaction semantics: if any row fails, roll back all inserts in that batch.
6. On success: redirect to the address list with a toast showing the count imported.
7. On failure: show which rows failed and why.

---

## Files to Create / Modify

| Action | Path | Purpose |
|--------|------|---------|
| **Create** | `services/address_import.go` | Batch insert logic with chunking, transaction support, and re-validation |
| **Modify** | `handlers/address_import.go` | Add the commit handler |
| **Modify** | `main.go` | Register the commit POST route |
| **Modify** | `templates/address_import.templ` | Add import success and failure result partials |

---

## Detailed Implementation Steps

### Step 1: Import Service (`services/address_import.go`)

```go
package services

import (
    "fmt"
    "log"
    "strings"

    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"
)

const importBatchSize = 100

// ImportResult holds the outcome of a batch import operation.
type ImportResult struct {
    TotalRows    int              `json:"total_rows"`
    Imported     int              `json:"imported"`
    Failed       int              `json:"failed"`
    Errors       []ImportRowError `json:"errors,omitempty"`
    // RolledBack is true if the entire import was rolled back due to errors.
    RolledBack   bool             `json:"rolled_back"`
}

// ImportRowError represents a failure to insert a specific row.
type ImportRowError struct {
    Row     int    `json:"row"`
    Field   string `json:"field"`
    Message string `json:"message"`
}

// CommitAddressImport re-validates and batch-inserts parsed address rows
// into PocketBase. It processes rows in chunks of importBatchSize.
//
// Strategy: Process in chunks. Within each chunk, if any insert fails,
// roll back the entire chunk and record errors. Continue with next chunk.
// This gives a good balance between atomicity and partial progress.
func CommitAddressImport(
    app *pocketbase.PocketBase,
    projectID string,
    addressType string,
    parsedRows []map[string]string,
) (*ImportResult, error) {
    // 1. Re-validate all rows before committing
    revalidationErrors := revalidateRows(app, projectID, addressType, parsedRows)
    if len(revalidationErrors) > 0 {
        errorRowSet := make(map[int]bool)
        for _, e := range revalidationErrors {
            errorRowSet[e.Row] = true
        }
        return &ImportResult{
            TotalRows:  len(parsedRows),
            Imported:   0,
            Failed:     len(errorRowSet),
            Errors:     toImportRowErrors(revalidationErrors),
            RolledBack: true,
        }, nil
    }

    // 2. For Install At, build a site_name -> record_id lookup for Ship To references
    var shipToLookup map[string]string
    if addressType == "install_at" {
        var err error
        shipToLookup, err = buildShipToLookup(app, projectID)
        if err != nil {
            return nil, fmt.Errorf("build ship_to lookup: %w", err)
        }
    }

    // 3. Determine target collection
    collectionName := "ship_to_addresses"
    if addressType == "install_at" {
        collectionName = "install_at_addresses"
    }

    col, err := app.FindCollectionByNameOrId(collectionName)
    if err != nil {
        return nil, fmt.Errorf("collection %s not found: %w", collectionName, err)
    }

    // 4. Process in chunks
    result := &ImportResult{
        TotalRows: len(parsedRows),
    }

    for chunkStart := 0; chunkStart < len(parsedRows); chunkStart += importBatchSize {
        chunkEnd := chunkStart + importBatchSize
        if chunkEnd > len(parsedRows) {
            chunkEnd = len(parsedRows)
        }
        chunk := parsedRows[chunkStart:chunkEnd]

        chunkErrors := insertChunk(app, col, projectID, addressType, chunk, chunkStart, shipToLookup)
        if len(chunkErrors) > 0 {
            result.Errors = append(result.Errors, chunkErrors...)
            result.Failed += len(chunk) // entire chunk failed
            result.RolledBack = true
        } else {
            result.Imported += len(chunk)
        }
    }

    return result, nil
}

// insertChunk inserts a batch of rows within a RunInTransaction block.
// If any row fails, the entire chunk is rolled back and errors are returned.
func insertChunk(
    app *pocketbase.PocketBase,
    col *core.Collection,
    projectID string,
    addressType string,
    rows []map[string]string,
    startOffset int,
    shipToLookup map[string]string,
) []ImportRowError {
    var chunkErrors []ImportRowError

    err := app.RunInTransaction(func(txApp core.App) error {
        for i, rowData := range rows {
            rowNum := startOffset + i + 2 // 1-indexed + header row

            record := core.NewRecord(col)
            record.Set("project", projectID)

            // Set common address fields
            for _, key := range addressFieldKeys() {
                if key == "ship_to_reference" {
                    continue // handled separately for Install At
                }
                if val, ok := rowData[key]; ok && val != "" {
                    record.Set(key, val)
                }
            }

            // For Install At: resolve Ship To Reference to relation ID
            if addressType == "install_at" {
                ref := rowData["ship_to_reference"]
                if ref != "" {
                    if shipToID, ok := shipToLookup[ref]; ok {
                        record.Set("ship_to_parent", shipToID)
                    } else {
                        chunkErrors = append(chunkErrors, ImportRowError{
                            Row:     rowNum,
                            Field:   "Ship To Reference",
                            Message: fmt.Sprintf("Ship To %q not found", ref),
                        })
                        return fmt.Errorf("ship_to_reference lookup failed at row %d", rowNum)
                    }
                }
            }

            if err := txApp.Save(record); err != nil {
                chunkErrors = append(chunkErrors, ImportRowError{
                    Row:     rowNum,
                    Field:   "",
                    Message: fmt.Sprintf("Failed to save: %s", err.Error()),
                })
                return fmt.Errorf("save failed at row %d: %w", rowNum, err)
            }
        }
        return nil
    })

    if err != nil {
        log.Printf("address_import: chunk insert rolled back: %v", err)
        // If chunkErrors is empty but transaction failed, add a generic error
        if len(chunkErrors) == 0 {
            chunkErrors = append(chunkErrors, ImportRowError{
                Row:     startOffset + 2,
                Field:   "",
                Message: fmt.Sprintf("Transaction failed: %s", err.Error()),
            })
        }
    }

    return chunkErrors
}

// revalidateRows performs the same validation as Phase 11 but using the
// already-parsed row data. This catches cases where project settings
// changed between the initial validation and the commit.
func revalidateRows(
    app *pocketbase.PocketBase,
    projectID string,
    addressType string,
    parsedRows []map[string]string,
) []ValidationError {
    var fields []AddressField
    if addressType == "install_at" {
        fields = BaseInstallAtFields()
    } else {
        fields = BaseShipToFields()
    }

    requiredSet, _ := getProjectRequiredFields(app, projectID, addressType)

    isRequired := make(map[string]bool)
    for _, f := range fields {
        if f.AlwaysRequired || requiredSet[f.Key] {
            isRequired[f.Key] = true
        }
    }

    var shipToNames map[string]bool
    if addressType == "install_at" {
        shipToNames, _ = loadShipToSiteNames(app, projectID)
    }

    var allErrors []ValidationError

    for rowIdx, rowData := range parsedRows {
        rowNum := rowIdx + 2

        // Check required fields
        for key := range isRequired {
            if rowData[key] == "" {
                label := key
                for _, f := range fields {
                    if f.Key == key {
                        label = f.Label
                        break
                    }
                }
                allErrors = append(allErrors, ValidationError{
                    Row:     rowNum,
                    Field:   label,
                    Message: fmt.Sprintf("%s is required", label),
                })
            }
        }

        // Format validations
        allErrors = append(allErrors, validateFieldFormats(rowNum, rowData)...)

        // Ship To Reference for Install At
        if addressType == "install_at" {
            ref := rowData["ship_to_reference"]
            if ref != "" && !shipToNames[ref] {
                allErrors = append(allErrors, ValidationError{
                    Row:     rowNum,
                    Field:   "Ship To Reference",
                    Message: fmt.Sprintf("No Ship To address with site name %q found", ref),
                })
            }
        }
    }

    return allErrors
}

// buildShipToLookup returns a map of site_name -> record_id for all Ship To
// addresses in the project.
func buildShipToLookup(app *pocketbase.PocketBase, projectID string) (map[string]string, error) {
    col, err := app.FindCollectionByNameOrId("ship_to_addresses")
    if err != nil {
        return nil, err
    }

    records, err := app.FindRecordsByFilter(col,
        "project = {:projectId}", "", 0, 0,
        map[string]any{"projectId": projectID},
    )
    if err != nil {
        return make(map[string]string), nil
    }

    lookup := make(map[string]string, len(records))
    for _, r := range records {
        lookup[r.GetString("site_name")] = r.Id
    }
    return lookup, nil
}

// addressFieldKeys returns all field keys that map to PocketBase columns
// (excludes ship_to_reference which is a virtual lookup field).
func addressFieldKeys() []string {
    return []string{
        "site_name", "contact_person", "phone", "email",
        "address_line1", "address_line2", "city", "state",
        "pin_code", "country", "gstin", "pan",
    }
}

// toImportRowErrors converts ValidationErrors to ImportRowErrors.
func toImportRowErrors(ve []ValidationError) []ImportRowError {
    result := make([]ImportRowError, len(ve))
    for i, e := range ve {
        result[i] = ImportRowError{
            Row:     e.Row,
            Field:   e.Field,
            Message: e.Message,
        }
    }
    return result
}
```

### Step 2: Commit Handler (add to `handlers/address_import.go`)

```go
// HandleAddressImportCommit re-validates and batch-inserts the uploaded addresses.
// Route: POST /projects/{projectId}/addresses/{type}/import/commit
func HandleAddressImportCommit(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectID := e.Request.PathValue("projectId")
        addressType := e.Request.PathValue("type")

        if addressType != "ship_to" && addressType != "install_at" {
            return e.String(http.StatusBadRequest, "Invalid address type")
        }

        // Verify project exists
        if _, err := app.FindRecordById("projects", projectID); err != nil {
            return e.String(http.StatusNotFound, "Project not found")
        }

        // The validated+parsed data was passed from Phase 11 as a hidden form field.
        // Re-parse it here.
        resultJSON := e.Request.FormValue("result_json")
        if resultJSON == "" {
            return e.String(http.StatusBadRequest, "Missing validation data. Please re-upload the file.")
        }

        var validationResult services.ValidationResult
        if err := json.Unmarshal([]byte(resultJSON), &validationResult); err != nil {
            return e.String(http.StatusBadRequest, "Invalid validation data")
        }

        // The ParsedRows are not included in JSON serialization (json:"-").
        // We need to re-parse from the hidden field that stores the raw parsed data.
        // Alternative approach: re-upload the file in the commit form.
        //
        // For robustness, the commit form also accepts the file again.
        // Let's check if the file is provided; if so, re-parse and re-validate.
        var parsedRows []map[string]string

        file, header, err := e.Request.FormFile("file")
        if err == nil {
            defer file.Close()
            // Re-validate from file
            reResult, err := services.ValidateAddressFile(app, file, header.Filename, projectID, addressType)
            if err != nil {
                return e.String(http.StatusBadRequest, err.Error())
            }
            if reResult.ErrorRows > 0 {
                // Settings changed, re-validation found errors
                component := templates.AddressValidationResults(
                    projectID, addressType,
                    reResult.TotalRows, reResult.ValidRows, reResult.ErrorRows,
                    reResult.Errors, "", header.Filename,
                )
                return component.Render(e.Request.Context(), e.Response)
            }
            parsedRows = reResult.ParsedRows
        } else {
            // Fall back to parsing from the JSON result
            // We need to store ParsedRows properly. Use a separate hidden field.
            parsedJSON := e.Request.FormValue("parsed_rows_json")
            if parsedJSON == "" {
                return e.String(http.StatusBadRequest,
                    "File data missing. Please re-upload and try again.")
            }
            if err := json.Unmarshal([]byte(parsedJSON), &parsedRows); err != nil {
                return e.String(http.StatusBadRequest, "Invalid parsed data")
            }
        }

        // Commit the import
        importResult, err := services.CommitAddressImport(app, projectID, addressType, parsedRows)
        if err != nil {
            log.Printf("address_import_commit: %v", err)
            return e.String(http.StatusInternalServerError, "Import failed: "+err.Error())
        }

        // Render result
        if importResult.Failed > 0 {
            component := templates.AddressImportFailure(
                projectID, addressType, importResult,
            )
            return component.Render(e.Request.Context(), e.Response)
        }

        // Success -- render success message (HTMX will swap into results div)
        component := templates.AddressImportSuccess(
            projectID, addressType, importResult.Imported,
        )
        return component.Render(e.Request.Context(), e.Response)
    }
}
```

### Step 3: Success/Failure Templates (add to `templates/address_import.templ`)

```go
// AddressImportSuccess renders after a successful import.
templ AddressImportSuccess(projectID, addressType string, count int) {
    <div class="space-y-6">
        <div class="alert alert-success shadow-lg">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
            <div>
                <h3 class="font-bold">Import Successful</h3>
                <p>{ fmt.Sprint(count) } { addressTypeName(addressType) } addresses imported successfully.</p>
            </div>
        </div>

        <div class="flex justify-end gap-3">
            <a
                href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s/import", projectID, addressType)) }
                class="btn btn-ghost"
            >
                Import More
            </a>
            <a
                href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s", projectID, addressType)) }
                class="btn btn-primary"
            >
                View Address List
            </a>
        </div>
    </div>
}

// AddressImportFailure renders when some rows fail during commit.
templ AddressImportFailure(projectID, addressType string, result *services.ImportResult) {
    <div class="space-y-6">
        <div class="alert alert-error shadow-lg">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
            <div>
                <h3 class="font-bold">Import Failed</h3>
                if result.RolledBack {
                    <p>
                        The import was rolled back due to errors.
                        { fmt.Sprint(result.Failed) } rows failed out of { fmt.Sprint(result.TotalRows) }.
                        { fmt.Sprint(result.Imported) } rows were successfully imported before the failure.
                    </p>
                } else {
                    <p>
                        { fmt.Sprint(result.Imported) } rows imported,
                        { fmt.Sprint(result.Failed) } rows failed.
                    </p>
                }
            </div>
        </div>

        <!-- Summary cards -->
        <div class="grid grid-cols-3 gap-4">
            <div class="stat bg-base-200 rounded-xl">
                <div class="stat-title">Total Rows</div>
                <div class="stat-value text-2xl">{ fmt.Sprint(result.TotalRows) }</div>
            </div>
            <div class="stat bg-success/10 rounded-xl">
                <div class="stat-title">Imported</div>
                <div class="stat-value text-2xl text-success">{ fmt.Sprint(result.Imported) }</div>
            </div>
            <div class="stat bg-error/10 rounded-xl">
                <div class="stat-title">Failed</div>
                <div class="stat-value text-2xl text-error">{ fmt.Sprint(result.Failed) }</div>
            </div>
        </div>

        if len(result.Errors) > 0 {
            <!-- Error details table -->
            <div class="card bg-base-100 shadow">
                <div class="card-body">
                    <h3 class="card-title text-error">Failed Rows</h3>
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
                                for _, rowErr := range result.Errors {
                                    <tr>
                                        <td>{ fmt.Sprint(rowErr.Row) }</td>
                                        <td>{ rowErr.Field }</td>
                                        <td class="text-error">{ rowErr.Message }</td>
                                    </tr>
                                }
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        }

        <div class="flex justify-end gap-3">
            <a
                href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s/import", projectID, addressType)) }
                class="btn btn-primary"
            >
                Re-upload File
            </a>
        </div>
    </div>
}
```

### Step 4: Register Route (`main.go`)

Add inside the `OnServe` block:

```go
// Address import - commit
se.Router.POST("/projects/{projectId}/addresses/{type}/import/commit",
    handlers.HandleAddressImportCommit(app))
```

### Step 5: Improved Commit Flow with File Re-upload

To handle the problem where `ParsedRows` cannot be serialized to the hidden form field (it uses `json:"-"`), update the Phase 11 validation results template to include the file in the commit form. This requires the commit form to re-submit the original file:

Update `AddressValidationResults` in `templates/address_import.templ` -- replace the confirm import form:

```go
if errorRows == 0 {
    <!-- Re-upload the file in the commit request for re-validation -->
    <div
        x-data="{ file: null }"
        x-init="
            // Grab the file from the original upload form
            const fileInput = document.querySelector('input[name=file]');
            if (fileInput && fileInput.files[0]) {
                file = fileInput.files[0];
            }
        "
    >
        <form
            hx-post={ fmt.Sprintf("/projects/%s/addresses/%s/import/commit", projectID, addressType) }
            hx-target="#validation-results"
            hx-swap="innerHTML"
            hx-indicator="#commit-spinner"
            hx-encoding="multipart/form-data"
            x-ref="commitForm"
        >
            <input type="hidden" name="parsed_rows_json" value={ parsedRowsJSON }/>
            <button type="submit" class="btn btn-primary">
                Confirm Import ({ fmt.Sprint(validRows) } rows)
            </button>
        </form>
    </div>
}
```

Where `parsedRowsJSON` is a new parameter passed from the handler containing the JSON-serialized parsed rows.

### Step 6: Progress Tracking for Large Imports

For imports with 1000+ rows, add SSE (Server-Sent Events) progress updates. This is an optional enhancement:

```go
// ProgressTracker provides real-time progress for large imports.
// It can be used with SSE or polling.
type ProgressTracker struct {
    TotalRows   int `json:"total_rows"`
    Processed   int `json:"processed"`
    Percentage  int `json:"percentage"`
    CurrentChunk int `json:"current_chunk"`
    TotalChunks int `json:"total_chunks"`
}

// For the initial implementation, we use a simpler approach:
// The HTMX indicator shows a spinner during the request.
// For very large files (1000+), the handler streams progress
// via chunked transfer encoding or uses polling.

// Simple polling approach for Phase 12 v1:
// 1. Commit endpoint immediately returns a "processing" response with a job ID
// 2. Background goroutine processes the import
// 3. Client polls GET /projects/{id}/addresses/{type}/import/status/{jobId}
// 4. When complete, returns the final result

// This can be deferred to a future iteration if 1000-row imports
// complete within acceptable HTTP timeout (typically < 30 seconds).
```

---

## Dependencies on Other Phases

| Dependency | Detail |
|-----------|--------|
| **Phase 10** | Collections (`ship_to_addresses`, `install_at_addresses`, `project_address_settings`) must exist. `AddressField` definitions and `getProjectRequiredFields()` are used for re-validation. |
| **Phase 11** | `ValidateAddressFile()`, `ValidationResult`, `ValidationError` types, and the upload+validation UI are prerequisites. The commit handler extends the same `handlers/address_import.go` file. |

---

## Testing / Verification Steps

### Unit Tests (`services/address_import_test.go`)

1. **TestCommitAddressImport_ShipTo_HappyPath**
   - Seed a project with address settings.
   - Provide 5 valid parsed rows.
   - Assert `ImportResult.Imported == 5`, `Failed == 0`.
   - Query PocketBase and confirm 5 records in `ship_to_addresses`.

2. **TestCommitAddressImport_InstallAt_WithReference**
   - Seed a project with 2 Ship To addresses (`"HQ"`, `"Branch"`).
   - Provide 3 Install At rows referencing those Ship To names.
   - Assert all 3 imported, `ship_to_parent` relation is correctly set.

3. **TestCommitAddressImport_InstallAt_BadReference**
   - Provide an Install At row with `ship_to_reference = "NonExistent"`.
   - Assert the chunk is rolled back and error mentions the bad reference.

4. **TestCommitAddressImport_RevalidationCatchesChanges**
   - Validate a file successfully.
   - Then change `project_address_settings` to require `gstin`.
   - Call `CommitAddressImport` -- assert re-validation catches the missing GSTIN.

5. **TestCommitAddressImport_TransactionRollback**
   - Provide 3 valid rows and 1 row with data that causes a PocketBase save failure (e.g. duplicate unique constraint if applicable).
   - Assert the entire chunk of 4 rows is rolled back. No partial inserts.

6. **TestCommitAddressImport_LargeFile_Chunking**
   - Provide 250 parsed rows (all valid).
   - Assert `Imported == 250` (processed in 3 chunks of 100, 100, 50).

7. **TestBuildShipToLookup**
   - Seed 3 Ship To addresses.
   - Call `buildShipToLookup`. Assert map has 3 entries with correct IDs.

### Integration Tests

8. **TestHTTPCommitEndpoint_Success**
   - Upload and validate a clean CSV (Phase 11 endpoint).
   - POST to commit endpoint with the parsed data.
   - Assert 200, response HTML contains "Import Successful".
   - Query the PocketBase collection to confirm records exist.

9. **TestHTTPCommitEndpoint_RevalidationFailure**
   - Validate a CSV, then modify project settings, then commit.
   - Assert response shows re-validation errors instead of success.

10. **TestHTTPCommitEndpoint_MissingData**
    - POST to commit without `parsed_rows_json` or file.
    - Assert 400 error with clear message.

### Manual QA

11. Upload a 50-row Ship To CSV, validate (0 errors), click "Confirm Import".
    - Verify success message and count.
    - Navigate to address list and confirm all 50 rows appear.

12. Upload a 100-row Install At CSV with Ship To References.
    - Verify `ship_to_parent` relations are correctly set in PocketBase admin.

13. Upload a file, wait, then change project required fields in PocketBase admin, then click "Confirm Import".
    - Verify re-validation catches the new requirements and shows errors.

14. Test with a 500+ row file and observe that the import completes within 30 seconds.

---

## Acceptance Criteria

- [ ] `POST /projects/{projectId}/addresses/{type}/import/commit` re-validates all rows before inserting.
- [ ] If re-validation finds errors (settings changed), the user sees the error report and no data is inserted.
- [ ] Ship To addresses are inserted into the `ship_to_addresses` collection with correct project relation.
- [ ] Install At addresses are inserted into `install_at_addresses` with correct `ship_to_parent` relation resolved from `ship_to_reference`.
- [ ] If an Install At row references a non-existent Ship To site name, the chunk is rolled back and the error is reported.
- [ ] Rows are processed in chunks of 100 for memory efficiency.
- [ ] Within each chunk, inserts are transactional: if any row fails, the entire chunk is rolled back.
- [ ] On success: the UI shows a green "Import Successful" alert with the count of imported rows and a link to the address list.
- [ ] On partial failure: the UI shows imported count, failed count, and an error table listing which rows failed.
- [ ] A loading indicator is shown during the commit operation.
- [ ] A 500-row import completes within 30 seconds.
- [ ] A 1000-row import completes within 60 seconds.
- [ ] No orphaned or duplicate records are created on failure.
- [ ] The "Import More" button returns to the upload page for another file.
- [ ] The "View Address List" button navigates to the correct address list page.
