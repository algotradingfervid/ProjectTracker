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

	// Find or create sequence record within a transaction for atomicity
	seqCol, err := app.FindCollectionByNameOrId("number_sequences")
	if err != nil {
		return "", fmt.Errorf("number_sequences collection not found: %w", err)
	}

	var nextNum int
	err = app.RunInTransaction(func(txApp core.App) error {
		records, err := txApp.FindRecordsByFilter(
			seqCol,
			"project = {:pid} && sequence_type = {:type} && financial_year = {:fy}",
			"", 1, 0,
			map[string]any{"pid": projectID, "type": seqType, "fy": fy},
		)
		if err != nil {
			return fmt.Errorf("failed to query sequences: %w", err)
		}

		if len(records) > 0 {
			rec := records[0]
			nextNum = rec.GetInt("last_number") + 1
			rec.Set("last_number", nextNum)
			if err := txApp.Save(rec); err != nil {
				return fmt.Errorf("failed to update sequence: %w", err)
			}
		} else {
			nextNum = seqStart
			rec := core.NewRecord(seqCol)
			rec.Set("project", projectID)
			rec.Set("sequence_type", seqType)
			rec.Set("financial_year", fy)
			rec.Set("last_number", nextNum)
			if err := txApp.Save(rec); err != nil {
				return fmt.Errorf("failed to create sequence: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return FormatDocNumber(format, sep, prefix, seqType, fy, nextNum, padding, projRef), nil
}
