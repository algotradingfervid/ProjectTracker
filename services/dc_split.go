package services

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// SplitParams holds parameters for creating a split from a transfer DC.
type SplitParams struct {
	ProjectID      string
	TransferDCID   string // delivery_challans record ID
	DestinationIDs []string // transfer_dc_destinations IDs to split
	TransporterID  string
	VehicleID      string
	EwayBillNumber string
	DocketNumber   string
	// SerialAssignments maps line_item_id -> list of serial numbers for this split
	SerialAssignments map[string][]string
}

// SplitResult contains IDs of created records from a split.
type SplitResult struct {
	SplitID         string
	ShipmentGroupID string
	TransitDCID     string
	TransitDCNumber string
	OfficialDCIDs   []string
	OfficialDCNums  []string
}

// CreateSplit creates a child shipment group from selected transfer DC destinations.
func CreateSplit(app *pocketbase.PocketBase, params SplitParams) (*SplitResult, error) {
	result := &SplitResult{}

	// 1. Validate transfer DC exists and is issued/splitting
	dc, err := app.FindRecordById("delivery_challans", params.TransferDCID)
	if err != nil {
		return nil, fmt.Errorf("delivery challan not found: %w", err)
	}
	status := dc.GetString("status")
	if status != "issued" && status != "splitting" {
		return nil, fmt.Errorf("transfer DC must be in issued or splitting status, got %s", status)
	}
	if dc.GetString("dc_type") != "transfer" {
		return nil, fmt.Errorf("DC is not a transfer DC")
	}

	projectID := dc.GetString("project")

	// 2. Fetch transfer_dcs metadata
	transferRecs, err := app.FindRecordsByFilter("transfer_dcs", "dc = {:did}", "", 1, 0, map[string]any{"did": params.TransferDCID})
	if err != nil || len(transferRecs) == 0 {
		return nil, fmt.Errorf("transfer DC metadata not found")
	}
	transferDC := transferRecs[0]
	numDest := transferDC.GetInt("num_destinations")
	numSplit := transferDC.GetInt("num_split")

	// 3. Validate selected destinations are not already split
	for _, destID := range params.DestinationIDs {
		destRec, err := app.FindRecordById("transfer_dc_destinations", destID)
		if err != nil {
			return nil, fmt.Errorf("destination %s not found: %w", destID, err)
		}
		if destRec.GetBool("is_split") {
			return nil, fmt.Errorf("destination %s is already split", destID)
		}
		if destRec.GetString("transfer_dc") != transferDC.Id {
			return nil, fmt.Errorf("destination %s does not belong to this transfer DC", destID)
		}
	}

	// 4. Fetch parent DC line items for pricing info
	parentLineItems, err := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "line_order", 0, 0, map[string]any{"did": params.TransferDCID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch parent line items: %w", err)
	}

	// 5. Compute per-item total quantities across selected destinations
	type itemQtyInfo struct {
		SourceItemType string
		SourceItemID   string
		Rate           float64
		TaxPercentage  float64
		TotalQty       int
		PerDestQty     map[string]int // destID -> qty
		LineOrder      int
		ParentLineID   string
	}
	var itemInfos []itemQtyInfo

	for _, li := range parentLineItems {
		info := itemQtyInfo{
			SourceItemType: li.GetString("source_item_type"),
			SourceItemID:   li.GetString("source_item_id"),
			Rate:           li.GetFloat("rate"),
			TaxPercentage:  li.GetFloat("tax_percentage"),
			LineOrder:      li.GetInt("line_order"),
			ParentLineID:   li.Id,
			PerDestQty:     make(map[string]int),
		}

		for _, destID := range params.DestinationIDs {
			destQtyRecs, _ := app.FindRecordsByFilter("transfer_dc_dest_quantities",
				"destination = {:did} && source_item_type = {:sit} && source_item_id = {:sid}",
				"", 1, 0,
				map[string]any{"did": destID, "sit": info.SourceItemType, "sid": info.SourceItemID})
			if len(destQtyRecs) > 0 {
				qty := destQtyRecs[0].GetInt("quantity")
				info.PerDestQty[destID] = qty
				info.TotalQty += qty
			}
		}

		if info.TotalQty > 0 {
			itemInfos = append(itemInfos, info)
		}
	}

	// 6. Resolve addresses from transfer DC
	billFromID := dc.GetString("bill_from_address")
	dispatchFromID := dc.GetString("dispatch_from_address")
	billToID := dc.GetString("bill_to_address")
	templateID := dc.GetString("template")

	docDate := time.Now()
	if cd := dc.GetString("challan_date"); cd != "" {
		if t, err := time.Parse("2006-01-02", cd); err == nil {
			docDate = t
		}
	}

	// 7. Create child Shipment Group
	sgCol, err := app.FindCollectionByNameOrId("shipment_groups")
	if err != nil {
		return nil, fmt.Errorf("shipment_groups collection not found: %w", err)
	}

	sgRec := core.NewRecord(sgCol)
	sgRec.Set("project", projectID)
	if templateID != "" {
		sgRec.Set("template", templateID)
	}
	sgRec.Set("num_locations", len(params.DestinationIDs))
	sgRec.Set("tax_type", transferDC.GetString("tax_type"))
	sgRec.Set("reverse_charge", transferDC.GetBool("reverse_charge"))
	sgRec.Set("status", "draft")
	sgRec.Set("transfer_dc", transferDC.Id)
	if err := app.Save(sgRec); err != nil {
		return nil, fmt.Errorf("failed to create shipment group: %w", err)
	}
	result.ShipmentGroupID = sgRec.Id

	// 8. Generate Transit DC number
	transitDCNum, err := NextDocNumber(app, projectID, "tdc", docDate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate transit DC number: %w", err)
	}
	result.TransitDCNumber = transitDCNum

	// 9. Create Transit DC
	dcCol, err := app.FindCollectionByNameOrId("delivery_challans")
	if err != nil {
		return nil, fmt.Errorf("delivery_challans collection not found: %w", err)
	}

	transitDCRec := core.NewRecord(dcCol)
	transitDCRec.Set("project", projectID)
	transitDCRec.Set("dc_number", transitDCNum)
	transitDCRec.Set("dc_type", "transit")
	transitDCRec.Set("status", "draft")
	if templateID != "" {
		transitDCRec.Set("template", templateID)
	}
	if billFromID != "" {
		transitDCRec.Set("bill_from_address", billFromID)
	}
	if dispatchFromID != "" {
		transitDCRec.Set("dispatch_from_address", dispatchFromID)
	}
	if billToID != "" {
		transitDCRec.Set("bill_to_address", billToID)
	}
	// Set first destination's ship_to as primary
	if len(params.DestinationIDs) > 0 {
		if firstDest, err := app.FindRecordById("transfer_dc_destinations", params.DestinationIDs[0]); err == nil {
			transitDCRec.Set("ship_to_address", firstDest.GetString("ship_to_address"))
		}
	}
	transitDCRec.Set("challan_date", docDate.Format("2006-01-02"))
	transitDCRec.Set("shipment_group", sgRec.Id)
	if err := app.Save(transitDCRec); err != nil {
		return nil, fmt.Errorf("failed to create transit DC: %w", err)
	}
	result.TransitDCID = transitDCRec.Id

	// 10. Create transit DC line items with pricing and assigned serials
	lineItemCol, err := app.FindCollectionByNameOrId("dc_line_items")
	if err != nil {
		return nil, fmt.Errorf("dc_line_items collection not found: %w", err)
	}

	serialCol, err := app.FindCollectionByNameOrId("serial_numbers")
	if err != nil {
		return nil, fmt.Errorf("serial_numbers collection not found: %w", err)
	}

	for _, info := range itemInfos {
		taxableAmount := info.Rate * float64(info.TotalQty)
		taxAmount := taxableAmount * info.TaxPercentage / 100.0
		totalAmount := taxableAmount + taxAmount

		lineItem := core.NewRecord(lineItemCol)
		lineItem.Set("dc", transitDCRec.Id)
		lineItem.Set("source_item_type", info.SourceItemType)
		lineItem.Set("source_item_id", info.SourceItemID)
		lineItem.Set("quantity", info.TotalQty)
		lineItem.Set("rate", info.Rate)
		lineItem.Set("tax_percentage", info.TaxPercentage)
		lineItem.Set("taxable_amount", taxableAmount)
		lineItem.Set("tax_amount", taxAmount)
		lineItem.Set("total_amount", totalAmount)
		lineItem.Set("line_order", info.LineOrder)
		if err := app.Save(lineItem); err != nil {
			return nil, fmt.Errorf("failed to create transit DC line item: %w", err)
		}

		// Create serial number records from assigned serials
		if serials, ok := params.SerialAssignments[info.ParentLineID]; ok {
			for _, serial := range serials {
				serialRec := core.NewRecord(serialCol)
				serialRec.Set("project", projectID)
				serialRec.Set("line_item", lineItem.Id)
				serialRec.Set("serial_number", serial)
				if err := app.Save(serialRec); err != nil {
					return nil, fmt.Errorf("failed to save serial number %s: %w", serial, err)
				}
			}
		}
	}

	// 11. Create transit details
	if params.TransporterID != "" || params.EwayBillNumber != "" || params.DocketNumber != "" {
		transitDetailCol, err := app.FindCollectionByNameOrId("dc_transit_details")
		if err == nil {
			vehicleNumber := ""
			if params.VehicleID != "" {
				if vRec, err := app.FindRecordById("transporter_vehicles", params.VehicleID); err == nil {
					vehicleNumber = vRec.GetString("vehicle_number")
				}
			}
			detail := core.NewRecord(transitDetailCol)
			detail.Set("dc", transitDCRec.Id)
			if params.TransporterID != "" {
				detail.Set("transporter", params.TransporterID)
			}
			detail.Set("vehicle_number", vehicleNumber)
			detail.Set("eway_bill_number", params.EwayBillNumber)
			detail.Set("docket_number", params.DocketNumber)
			_ = app.Save(detail)
		}
	}

	// 12. Create Official DCs (one per selected destination)
	for _, destID := range params.DestinationIDs {
		destRec, err := app.FindRecordById("transfer_dc_destinations", destID)
		if err != nil {
			continue
		}
		shipToID := destRec.GetString("ship_to_address")

		odcNum, err := NextDocNumber(app, projectID, "odc", docDate)
		if err != nil {
			return nil, fmt.Errorf("failed to generate official DC number: %w", err)
		}

		odc := core.NewRecord(dcCol)
		odc.Set("project", projectID)
		odc.Set("dc_number", odcNum)
		odc.Set("dc_type", "official")
		odc.Set("status", "draft")
		if templateID != "" {
			odc.Set("template", templateID)
		}
		if billFromID != "" {
			odc.Set("bill_from_address", billFromID)
		}
		if dispatchFromID != "" {
			odc.Set("dispatch_from_address", dispatchFromID)
		}
		if billToID != "" {
			odc.Set("bill_to_address", billToID)
		}
		odc.Set("ship_to_address", shipToID)
		odc.Set("challan_date", docDate.Format("2006-01-02"))
		odc.Set("shipment_group", sgRec.Id)
		if err := app.Save(odc); err != nil {
			return nil, fmt.Errorf("failed to create official DC: %w", err)
		}

		result.OfficialDCIDs = append(result.OfficialDCIDs, odc.Id)
		result.OfficialDCNums = append(result.OfficialDCNums, odcNum)

		// Create line items for official DC (per-location qty, no pricing/serials)
		for _, info := range itemInfos {
			qty := info.PerDestQty[destID]
			if qty == 0 {
				continue
			}

			odcLineItem := core.NewRecord(lineItemCol)
			odcLineItem.Set("dc", odc.Id)
			odcLineItem.Set("source_item_type", info.SourceItemType)
			odcLineItem.Set("source_item_id", info.SourceItemID)
			odcLineItem.Set("quantity", qty)
			odcLineItem.Set("line_order", info.LineOrder)
			if err := app.Save(odcLineItem); err != nil {
				return nil, fmt.Errorf("failed to create official DC line item: %w", err)
			}
		}
	}

	// 13. Create transfer_dc_splits record
	splitCol, err := app.FindCollectionByNameOrId("transfer_dc_splits")
	if err != nil {
		return nil, fmt.Errorf("transfer_dc_splits collection not found: %w", err)
	}

	splitRec := core.NewRecord(splitCol)
	splitRec.Set("transfer_dc", transferDC.Id)
	splitRec.Set("shipment_group", sgRec.Id)
	splitRec.Set("split_number", numSplit+1)
	if err := app.Save(splitRec); err != nil {
		return nil, fmt.Errorf("failed to create split record: %w", err)
	}
	result.SplitID = splitRec.Id

	// Link split back to shipment group
	sgRec.Set("split", splitRec.Id)
	_ = app.Save(sgRec)

	// 14. Mark destinations as split
	for _, destID := range params.DestinationIDs {
		destRec, err := app.FindRecordById("transfer_dc_destinations", destID)
		if err != nil {
			continue
		}
		destRec.Set("is_split", true)
		destRec.Set("split_group", sgRec.Id)
		_ = app.Save(destRec)
	}

	// 15. Update transfer DC metadata
	newNumSplit := numSplit + 1
	transferDC.Set("num_split", newNumSplit)
	_ = app.Save(transferDC)

	// 16. Update parent DC status based on split progress
	if newNumSplit >= numDest {
		dc.Set("status", "split")
	} else {
		dc.Set("status", "splitting")
	}
	_ = app.Save(dc)

	return result, nil
}

// UndoSplit reverses a split operation, deleting the child shipment group
// and resetting destination split flags.
func UndoSplit(app *pocketbase.PocketBase, splitID string) error {
	// 1. Find the split record
	splitRec, err := app.FindRecordById("transfer_dc_splits", splitID)
	if err != nil {
		return fmt.Errorf("split record not found: %w", err)
	}

	transferDCID := splitRec.GetString("transfer_dc")
	sgID := splitRec.GetString("shipment_group")

	// 2. Find the transfer DC metadata
	transferDC, err := app.FindRecordById("transfer_dcs", transferDCID)
	if err != nil {
		return fmt.Errorf("transfer DC metadata not found: %w", err)
	}

	// 3. Find parent DC record
	parentDCID := transferDC.GetString("dc")
	parentDC, err := app.FindRecordById("delivery_challans", parentDCID)
	if err != nil {
		return fmt.Errorf("parent DC not found: %w", err)
	}

	// 4. Verify shipment group is in draft (can't undo issued splits)
	sg, err := app.FindRecordById("shipment_groups", sgID)
	if err != nil {
		return fmt.Errorf("shipment group not found: %w", err)
	}
	if sg.GetString("status") != "draft" {
		return fmt.Errorf("cannot undo split: shipment group is already %s", sg.GetString("status"))
	}

	// 5. Reset destinations that reference this shipment group
	destRecs, _ := app.FindRecordsByFilter("transfer_dc_destinations",
		"transfer_dc = {:tid} && split_group = {:sgid}",
		"", 0, 0,
		map[string]any{"tid": transferDCID, "sgid": sgID})
	for _, dest := range destRecs {
		dest.Set("is_split", false)
		dest.Set("split_group", "")
		_ = app.Save(dest)
	}

	// 6. Delete child DCs in the shipment group (cascade will handle line items, serials)
	childDCs, _ := app.FindRecordsByFilter("delivery_challans", "shipment_group = {:sgid}", "", 0, 0, map[string]any{"sgid": sgID})
	for _, childDC := range childDCs {
		// Delete associated serial numbers and transit details first
		childLineItems, _ := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "", 0, 0, map[string]any{"did": childDC.Id})
		for _, li := range childLineItems {
			childSerials, _ := app.FindRecordsByFilter("serial_numbers", "line_item = {:lid}", "", 0, 0, map[string]any{"lid": li.Id})
			for _, sr := range childSerials {
				_ = app.Delete(sr)
			}
			_ = app.Delete(li)
		}
		transitDetails, _ := app.FindRecordsByFilter("dc_transit_details", "dc = {:did}", "", 0, 0, map[string]any{"did": childDC.Id})
		for _, td := range transitDetails {
			_ = app.Delete(td)
		}
		_ = app.Delete(childDC)
	}

	// 7. Delete the split record
	_ = app.Delete(splitRec)

	// 8. Delete the shipment group
	_ = app.Delete(sg)

	// 9. Decrement num_split on transfer DC
	numSplit := transferDC.GetInt("num_split")
	if numSplit > 0 {
		numSplit--
	}
	transferDC.Set("num_split", numSplit)
	_ = app.Save(transferDC)

	// 10. Recompute parent DC status
	if numSplit == 0 {
		parentDC.Set("status", "issued")
	} else {
		numDest := transferDC.GetInt("num_destinations")
		if numSplit >= numDest {
			parentDC.Set("status", "split")
		} else {
			parentDC.Set("status", "splitting")
		}
	}
	_ = app.Save(parentDC)

	return nil
}
