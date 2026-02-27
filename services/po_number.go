package services

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase"
)

// GetFiscalYear returns the Indian fiscal year string for a given date.
// Indian fiscal year runs April to March.
// Jan 2026 → "25-26", May 2026 → "26-27"
func GetFiscalYear(t time.Time) string {
	year := t.Year()
	month := t.Month()

	var startYear int
	if month >= time.April {
		startYear = year
	} else {
		startYear = year - 1
	}
	endYear := startYear + 1

	return fmt.Sprintf("%02d-%02d", startYear%100, endYear%100)
}

// formatPONumber constructs the PO number string from components.
// Uses "-" as separator to avoid conflicts with reference numbers that contain "/".
func formatPONumber(projectRef, fiscalYear string, sequence int) string {
	return fmt.Sprintf("FSS-PO-%s-%s-%03d", projectRef, fiscalYear, sequence)
}

// GeneratePONumber creates the next PO number for a project.
// Format: FSS-PO-{project_ref}-{fiscal_year}-{sequence}
// - project_ref: project's reference_number (falls back to project ID if empty)
// - fiscal_year: Indian fiscal year (Apr-Mar), e.g., "25-26"
// - sequence: 3-digit zero-padded, per project per fiscal year
func GeneratePONumber(app *pocketbase.PocketBase, projectId string, now time.Time) (string, error) {
	// Get project to find reference number
	project, err := app.FindRecordById("projects", projectId)
	if err != nil {
		return "", fmt.Errorf("project not found: %w", err)
	}

	projectRef := project.GetString("reference_number")
	if projectRef == "" {
		projectRef = projectId
	}

	fiscalYear := GetFiscalYear(now)

	// Count existing POs for this project with matching fiscal year prefix
	prefix := fmt.Sprintf("FSS-PO-%s-%s-", projectRef, fiscalYear)

	existingPOs, err := app.FindRecordsByFilter(
		"purchase_orders",
		"project = {:projectId} && po_number ~ {:prefix}",
		"",
		0,
		0,
		map[string]any{
			"projectId": projectId,
			"prefix":    prefix + "%",
		},
	)
	if err != nil {
		// If collection doesn't exist or no records, start at 1
		existingPOs = nil
	}

	nextSeq := len(existingPOs) + 1

	return formatPONumber(projectRef, fiscalYear, nextSeq), nil
}
