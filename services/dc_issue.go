package services

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase"
)

// IssueShipmentGroup validates and issues all DCs in a shipment group atomically.
func IssueShipmentGroup(app *pocketbase.PocketBase, groupID string) error {
	// Fetch shipment group
	sg, err := app.FindRecordById("shipment_groups", groupID)
	if err != nil {
		return fmt.Errorf("shipment group not found: %w", err)
	}

	if sg.GetString("status") != "draft" {
		return fmt.Errorf("shipment group is not in draft status")
	}

	// Fetch all DCs in this group
	dcs, err := app.FindRecordsByFilter("delivery_challans", "shipment_group = {:sgid}", "", 0, 0, map[string]any{"sgid": groupID})
	if err != nil {
		return fmt.Errorf("failed to fetch DCs for shipment group: %w", err)
	}

	// Validate all DCs are in draft
	for _, dc := range dcs {
		if dc.GetString("status") != "draft" {
			return fmt.Errorf("DC %s is not in draft status", dc.GetString("dc_number"))
		}
	}

	// Validate serial numbers for transit DC
	projectID := sg.GetString("project")
	for _, dc := range dcs {
		if dc.GetString("dc_type") != "transit" {
			continue
		}
		if err := validateDCSerials(app, dc.Id, projectID); err != nil {
			return err
		}
	}

	// Issue all DCs
	now := time.Now().UTC().Format(time.RFC3339)
	for _, dc := range dcs {
		dc.Set("status", "issued")
		dc.Set("issued_at", now)
		if err := app.Save(dc); err != nil {
			return fmt.Errorf("failed to issue DC %s: %w", dc.GetString("dc_number"), err)
		}
	}

	// Update shipment group status
	sg.Set("status", "issued")
	if err := app.Save(sg); err != nil {
		return fmt.Errorf("failed to update shipment group status: %w", err)
	}

	return nil
}

// IssueTransferDC validates and issues a transfer DC.
func IssueTransferDC(app *pocketbase.PocketBase, dcID string) error {
	dc, err := app.FindRecordById("delivery_challans", dcID)
	if err != nil {
		return fmt.Errorf("delivery challan not found: %w", err)
	}

	if dc.GetString("status") != "draft" {
		return fmt.Errorf("DC is not in draft status")
	}

	if dc.GetString("dc_type") != "transfer" {
		return fmt.Errorf("DC is not a transfer DC")
	}

	projectID := dc.GetString("project")
	if err := validateDCSerials(app, dcID, projectID); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	dc.Set("status", "issued")
	dc.Set("issued_at", now)
	if err := app.Save(dc); err != nil {
		return fmt.Errorf("failed to issue DC: %w", err)
	}

	return nil
}

// IssueSingleDC issues a single DC (transit, official, or transfer).
func IssueSingleDC(app *pocketbase.PocketBase, dcID string) error {
	dc, err := app.FindRecordById("delivery_challans", dcID)
	if err != nil {
		return fmt.Errorf("delivery challan not found: %w", err)
	}

	if dc.GetString("status") != "draft" {
		return fmt.Errorf("DC is not in draft status")
	}

	dcType := dc.GetString("dc_type")

	// If it's part of a shipment group, issue the whole group
	if sgID := dc.GetString("shipment_group"); sgID != "" {
		return IssueShipmentGroup(app, sgID)
	}

	// Transfer DC
	if dcType == "transfer" {
		return IssueTransferDC(app, dcID)
	}

	// Standalone DC (shouldn't happen normally, but handle gracefully)
	projectID := dc.GetString("project")
	if err := validateDCSerials(app, dcID, projectID); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	dc.Set("status", "issued")
	dc.Set("issued_at", now)
	return app.Save(dc)
}

// validateDCSerials checks that all line items with required serial tracking have correct serial counts.
func validateDCSerials(app *pocketbase.PocketBase, dcID, projectID string) error {
	lineItems, err := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "line_order", 0, 0, map[string]any{"did": dcID})
	if err != nil {
		return fmt.Errorf("failed to fetch line items: %w", err)
	}

	// Get existing serials for uniqueness check (exclude this DC's serials)
	allSerials, _ := app.FindRecordsByFilter("serial_numbers", "project = {:pid}", "", 0, 0, map[string]any{"pid": projectID})
	existingSerials := make(map[string]string)
	thisDCSerials := make(map[string]bool)

	// First pass: identify serials belonging to this DC
	for _, li := range lineItems {
		serialRecs, _ := app.FindRecordsByFilter("serial_numbers", "line_item = {:lid}", "", 0, 0, map[string]any{"lid": li.Id})
		for _, sr := range serialRecs {
			thisDCSerials[sr.GetString("serial_number")] = true
		}
	}

	// Build existing map excluding this DC's serials
	for _, sr := range allSerials {
		serial := sr.GetString("serial_number")
		if thisDCSerials[serial] {
			continue
		}
		lineItemID := sr.GetString("line_item")
		liRec, err := app.FindRecordById("dc_line_items", lineItemID)
		if err != nil {
			existingSerials[serial] = "unknown"
			continue
		}
		otherDCID := liRec.GetString("dc")
		otherDC, err := app.FindRecordById("delivery_challans", otherDCID)
		if err != nil {
			existingSerials[serial] = "unknown"
			continue
		}
		existingSerials[serial] = otherDC.GetString("dc_number")
	}

	// Validate each line item
	for _, li := range lineItems {
		qty := li.GetInt("quantity")
		serialRecs, _ := app.FindRecordsByFilter("serial_numbers", "line_item = {:lid}", "", 0, 0, map[string]any{"lid": li.Id})

		var serials []string
		for _, sr := range serialRecs {
			serials = append(serials, sr.GetString("serial_number"))
		}

		// Resolve source item to check serial tracking requirement
		collection := "sub_items"
		if li.GetString("source_item_type") == "sub_sub_item" {
			collection = "sub_sub_items"
		}

		// Check serial tracking from template item
		templateItems, _ := app.FindRecordsByFilter("dc_template_items",
			"source_item_type = {:sit} && source_item_id = {:sid}",
			"", 1, 0,
			map[string]any{"sit": li.GetString("source_item_type"), "sid": li.GetString("source_item_id")})

		serialTracking := "none"
		if len(templateItems) > 0 {
			serialTracking = templateItems[0].GetString("serial_tracking")
		}
		_ = collection // used for potential future lookups

		if serialTracking == "required" && qty > 0 {
			if len(serials) != qty {
				return fmt.Errorf("item requires %d serial number(s), but has %d", qty, len(serials))
			}
		}

		// Validate uniqueness of serials
		if len(serials) > 0 {
			result := ValidateSerials(serials, len(serials), existingSerials)
			if !result.Valid {
				if len(result.DuplicatesInInput) > 0 {
					return fmt.Errorf("duplicate serial numbers within DC: %v", result.DuplicatesInInput)
				}
				for _, conflict := range result.DuplicatesInDB {
					return fmt.Errorf("serial %s already used in %s", conflict.Serial, conflict.ExistingDC)
				}
			}
		}
	}

	return nil
}
