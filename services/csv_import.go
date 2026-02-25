package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
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
	ParsedRows []map[string]string `json:"-"`
	FileName   string              `json:"-"`
}

// parseCSV reads a CSV file and returns headers + data rows.
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

// mapHeadersToFields maps uploaded column headers to TemplateField keys.
// Returns ordered list of field keys (one per column) and any unrecognized columns.
func mapHeadersToFields(headers []string, fields []TemplateField) ([]string, []string) {
	labelToKey := make(map[string]string, len(fields))
	for _, f := range fields {
		normalized := strings.ToLower(strings.TrimSpace(f.Label))
		labelToKey[normalized] = f.Key
	}

	mapped := make([]string, len(headers))
	var unrecognized []string

	for i, h := range headers {
		norm := strings.ToLower(strings.TrimSpace(h))
		// Strip trailing " *" that our template adds for required fields
		norm = strings.TrimSuffix(norm, " *")
		norm = strings.TrimSpace(norm)

		if key, ok := labelToKey[norm]; ok {
			mapped[i] = key
		} else {
			mapped[i] = ""
			unrecognized = append(unrecognized, h)
		}
	}
	return mapped, unrecognized
}

// ValidateAddressFile parses and validates an uploaded address file.
func ValidateAddressFile(
	app *pocketbase.PocketBase,
	file multipart.File,
	fileName string,
	projectID string,
	addressType string,
) (*ValidationResult, error) {
	// 1. Determine fields for the address type
	var fields []TemplateField
	if addressType == "install_at" {
		fields = InstallAtTemplateFields()
	} else {
		fields = ShipToTemplateFields()
	}

	// 2. Get project-required fields
	requiredSet := GetRequiredFields(app, projectID, addressType)

	// 3. Parse file based on extension
	var headers []string
	var dataRows [][]string
	var err error

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

	// 4. Map headers to field keys
	columnKeys, _ := mapHeadersToFields(headers, fields)

	// 5. Build required field key set (always-required + project-configured)
	isRequired := make(map[string]bool)
	for _, f := range fields {
		if f.AlwaysRequired || requiredSet[f.Key] {
			isRequired[f.Key] = true
		}
	}

	// 6. For Install At, load existing Ship To company_name values for reference validation
	var shipToNames map[string]bool
	if addressType == "install_at" {
		shipToNames, err = loadShipToCompanyNames(app, projectID)
		if err != nil {
			return nil, fmt.Errorf("load ship to names: %w", err)
		}
	}

	// 7. Validate each row
	result := &ValidationResult{
		TotalRows:  len(dataRows),
		FileName:   fileName,
		ParsedRows: make([]map[string]string, 0, len(dataRows)),
	}

	// Build field key -> label lookup
	keyToLabel := make(map[string]string, len(fields))
	for _, f := range fields {
		keyToLabel[f.Key] = f.Label
	}

	for rowIdx, row := range dataRows {
		rowNum := rowIdx + 2 // 1-indexed, +1 for header row
		rowData := make(map[string]string)
		var rowErrors []ValidationError

		// Map columns to values
		for colIdx, key := range columnKeys {
			if key == "" {
				continue
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
				label := keyToLabel[key]
				if label == "" {
					label = key
				}
				rowErrors = append(rowErrors, ValidationError{
					Row:     rowNum,
					Field:   label,
					Message: fmt.Sprintf("%s is required", label),
				})
			}
		}

		// Field-format validations (only if value is non-empty)
		rowErrors = append(rowErrors, validateImportFieldFormats(rowNum, rowData)...)

		// Ship To Reference validation for Install At
		if addressType == "install_at" {
			ref := rowData["ship_to_reference"]
			if ref != "" && !shipToNames[ref] {
				rowErrors = append(rowErrors, ValidationError{
					Row:     rowNum,
					Field:   "Ship To Reference",
					Message: fmt.Sprintf("No Ship To address with company name %q found in this project", ref),
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

// validateImportFieldFormats checks format-specific rules for non-empty values.
// Reuses the existing individual validator functions from address_validation.go.
func validateImportFieldFormats(rowNum int, data map[string]string) []ValidationError {
	var errs []ValidationError

	if v := data["pin_code"]; v != "" && !ValidatePINCode(v) {
		errs = append(errs, ValidationError{Row: rowNum, Field: "PIN Code", Message: "PIN Code must be exactly 6 digits"})
	}
	if v := data["phone"]; v != "" && !ValidatePhone(v) {
		errs = append(errs, ValidationError{Row: rowNum, Field: "Phone", Message: "Phone must be 10 digits starting with 6-9"})
	}
	if v := data["email"]; v != "" && !ValidateEmail(v) {
		errs = append(errs, ValidationError{Row: rowNum, Field: "Email", Message: "Invalid email format"})
	}
	if v := data["gstin"]; v != "" && !ValidateGSTIN(v) {
		errs = append(errs, ValidationError{Row: rowNum, Field: "GSTIN", Message: "GSTIN must be 15 characters in format 22AAAAA0000A1Z5"})
	}
	if v := data["pan"]; v != "" && !ValidatePAN(v) {
		errs = append(errs, ValidationError{Row: rowNum, Field: "PAN", Message: "PAN must be 10 characters in format ABCDE1234F"})
	}
	if v := data["cin"]; v != "" && !ValidateCIN(v) {
		errs = append(errs, ValidationError{Row: rowNum, Field: "CIN", Message: "CIN must be 21 characters in format U12345AB1234ABC123456"})
	}

	return errs
}

// loadShipToCompanyNames fetches all Ship To company_name values for a project.
func loadShipToCompanyNames(app *pocketbase.PocketBase, projectID string) (map[string]bool, error) {
	col, err := app.FindCollectionByNameOrId("addresses")
	if err != nil {
		return make(map[string]bool), nil
	}

	records, err := app.FindRecordsByFilter(col,
		"project = {:projectId} && address_type = 'ship_to'", "", 0, 0,
		map[string]any{"projectId": projectID},
	)
	if err != nil {
		return make(map[string]bool), nil
	}

	names := make(map[string]bool, len(records))
	for _, r := range records {
		name := r.GetString("company_name")
		if name != "" {
			names[name] = true
		}
	}
	return names, nil
}

// GenerateErrorReport creates a downloadable .xlsx file from validation errors.
func GenerateErrorReport(errors []ValidationError) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Errors"
	defaultSheet := f.GetSheetName(0)
	f.SetSheetName(defaultSheet, sheet)

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#DC2626"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Border:    thinBorders(),
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
