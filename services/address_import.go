package services

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const importBatchSize = 100

// ImportResult holds the outcome of a batch import operation.
type ImportResult struct {
	TotalRows  int              `json:"total_rows"`
	Imported   int              `json:"imported"`
	Failed     int              `json:"failed"`
	Errors     []ImportRowError `json:"errors,omitempty"`
	RolledBack bool             `json:"rolled_back"`
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
func CommitAddressImport(
	app *pocketbase.PocketBase,
	projectID string,
	addressType string,
	parsedRows []map[string]string,
) (*ImportResult, error) {
	// 1. Re-validate all rows before committing
	revalidationErrors := revalidateImportRows(app, projectID, addressType, parsedRows)
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

	// 2. For Install At, build a company_name -> record_id lookup for Ship To references
	var shipToLookup map[string]string
	if addressType == "install_at" {
		var err error
		shipToLookup, err = buildShipToLookup(app, projectID)
		if err != nil {
			return nil, fmt.Errorf("build ship_to lookup: %w", err)
		}
	}

	// 3. Get the addresses collection
	col, err := app.FindCollectionByNameOrId("addresses")
	if err != nil {
		return nil, fmt.Errorf("addresses collection not found: %w", err)
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
			record.Set("address_type", addressType)

			// Set common address fields
			for _, key := range importAddressFieldKeys() {
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

// revalidateImportRows performs the same validation as Phase 11 but using the
// already-parsed row data. This catches cases where project settings
// changed between the initial validation and the commit.
func revalidateImportRows(
	app *pocketbase.PocketBase,
	projectID string,
	addressType string,
	parsedRows []map[string]string,
) []ValidationError {
	var fields []TemplateField
	if addressType == "install_at" {
		fields = InstallAtTemplateFields()
	} else {
		fields = ShipToTemplateFields()
	}

	requiredSet := GetRequiredFields(app, projectID, addressType)

	isRequired := make(map[string]bool)
	for _, f := range fields {
		if f.AlwaysRequired || requiredSet[f.Key] {
			isRequired[f.Key] = true
		}
	}

	// Build key -> label lookup
	keyToLabel := make(map[string]string, len(fields))
	for _, f := range fields {
		keyToLabel[f.Key] = f.Label
	}

	var shipToNames map[string]bool
	if addressType == "install_at" {
		shipToNames, _ = loadShipToCompanyNames(app, projectID)
	}

	var allErrors []ValidationError

	for rowIdx, rowData := range parsedRows {
		rowNum := rowIdx + 2

		// Check required fields
		for key := range isRequired {
			if rowData[key] == "" {
				label := keyToLabel[key]
				if label == "" {
					label = key
				}
				allErrors = append(allErrors, ValidationError{
					Row:     rowNum,
					Field:   label,
					Message: fmt.Sprintf("%s is required", label),
				})
			}
		}

		// Format validations
		allErrors = append(allErrors, validateImportFieldFormats(rowNum, rowData)...)

		// Ship To Reference for Install At
		if addressType == "install_at" {
			ref := rowData["ship_to_reference"]
			if ref != "" && !shipToNames[ref] {
				allErrors = append(allErrors, ValidationError{
					Row:     rowNum,
					Field:   "Ship To Reference",
					Message: fmt.Sprintf("No Ship To address with company name %q found", ref),
				})
			}
		}
	}

	return allErrors
}

// buildShipToLookup returns a map of company_name -> record_id for all Ship To
// addresses in the project.
func buildShipToLookup(app *pocketbase.PocketBase, projectID string) (map[string]string, error) {
	col, err := app.FindCollectionByNameOrId("addresses")
	if err != nil {
		return nil, err
	}

	records, err := app.FindRecordsByFilter(col,
		"project = {:projectId} && address_type = 'ship_to'", "", 0, 0,
		map[string]any{"projectId": projectID},
	)
	if err != nil {
		return make(map[string]string), nil
	}

	lookup := make(map[string]string, len(records))
	for _, r := range records {
		name := r.GetString("company_name")
		if name != "" {
			lookup[name] = r.Id
		}
	}
	return lookup, nil
}

// importAddressFieldKeys returns all field keys that map to PocketBase columns
// (excludes ship_to_reference which is a virtual lookup field).
func importAddressFieldKeys() []string {
	return []string{
		"company_name", "contact_person", "phone", "email",
		"address_line_1", "address_line_2", "city", "state",
		"pin_code", "country", "landmark", "district",
		"fax", "website", "gstin", "pan", "cin",
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
