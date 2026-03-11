package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// ShipmentItemParams holds item data for DC creation.
type ShipmentItemParams struct {
	SourceItemType string
	SourceItemID   string
	Rate           float64
	TaxPercentage  float64
	SerialTracking string
	Quantities     []int    // per-destination quantities
	TotalQty       int
	Serials        []string // serial numbers for transit DC
}

// ShipmentParams holds all parameters for creating a direct shipment.
type ShipmentParams struct {
	ProjectID      string
	TemplateID     string
	ChallanDate    string
	TransporterID  string
	VehicleID      string
	EwayBillNumber string
	DocketNumber   string
	ReverseCharge  bool
	BillFromID     string
	DispatchFromID string
	BillToID       string
	ShipToIDs      []string
	TaxType        string
	Items          []ShipmentItemParams
}

// ShipmentResult contains IDs of created records.
type ShipmentResult struct {
	ShipmentGroupID string
	TransitDCID     string
	TransitDCNumber string
	OfficialDCIDs   []string
	OfficialDCNums  []string
}

// TransferDCParams holds parameters for creating a transfer DC.
type TransferDCParams struct {
	ProjectID      string
	TemplateID     string
	ChallanDate    string
	TransporterID  string
	VehicleID      string
	EwayBillNumber string
	DocketNumber   string
	ReverseCharge  bool
	BillFromID     string
	DispatchFromID string
	BillToID       string
	ShipToIDs      []string
	HubAddressID   string
	TaxType        string
	Items          []ShipmentItemParams
}

// TransferDCResult contains IDs of created records.
type TransferDCResult struct {
	DCID     string
	DCNumber string
}

// CreateDirectShipment creates a shipment group with 1 transit DC + N official DCs.
func CreateDirectShipment(app *pocketbase.PocketBase, params ShipmentParams) (*ShipmentResult, error) {
	result := &ShipmentResult{}
	docDate, err := time.Parse("2006-01-02", params.ChallanDate)
	if err != nil {
		docDate = time.Now()
	}

	// 1. Create Shipment Group
	sgCol, err := app.FindCollectionByNameOrId("shipment_groups")
	if err != nil {
		return nil, fmt.Errorf("shipment_groups collection not found: %w", err)
	}

	sgRec := core.NewRecord(sgCol)
	sgRec.Set("project", params.ProjectID)
	if params.TemplateID != "" {
		sgRec.Set("template", params.TemplateID)
	}
	sgRec.Set("num_locations", len(params.ShipToIDs))
	sgRec.Set("tax_type", params.TaxType)
	sgRec.Set("reverse_charge", params.ReverseCharge)
	sgRec.Set("status", "draft")
	if err := app.Save(sgRec); err != nil {
		return nil, fmt.Errorf("failed to create shipment group: %w", err)
	}
	result.ShipmentGroupID = sgRec.Id

	// 2. Generate Transit DC number
	transitDCNum, err := NextDocNumber(app, params.ProjectID, "tdc", docDate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate transit DC number: %w", err)
	}
	result.TransitDCNumber = transitDCNum

	// 3. Create Transit DC
	dcCol, err := app.FindCollectionByNameOrId("delivery_challans")
	if err != nil {
		return nil, fmt.Errorf("delivery_challans collection not found: %w", err)
	}

	transitDC := core.NewRecord(dcCol)
	transitDC.Set("project", params.ProjectID)
	transitDC.Set("dc_number", transitDCNum)
	transitDC.Set("dc_type", "transit")
	transitDC.Set("status", "draft")
	if params.TemplateID != "" {
		transitDC.Set("template", params.TemplateID)
	}
	if params.BillFromID != "" {
		transitDC.Set("bill_from_address", params.BillFromID)
	}
	if params.DispatchFromID != "" {
		transitDC.Set("dispatch_from_address", params.DispatchFromID)
	}
	if params.BillToID != "" {
		transitDC.Set("bill_to_address", params.BillToID)
	}
	// Transit DC uses the first ship_to as primary
	if len(params.ShipToIDs) > 0 && params.ShipToIDs[0] != "" {
		transitDC.Set("ship_to_address", params.ShipToIDs[0])
	}
	transitDC.Set("challan_date", params.ChallanDate)
	transitDC.Set("shipment_group", sgRec.Id)
	if err := app.Save(transitDC); err != nil {
		return nil, fmt.Errorf("failed to create transit DC: %w", err)
	}
	result.TransitDCID = transitDC.Id

	// 4. Create transit DC line items with pricing and serials
	lineItemCol, err := app.FindCollectionByNameOrId("dc_line_items")
	if err != nil {
		return nil, fmt.Errorf("dc_line_items collection not found: %w", err)
	}

	serialCol, err := app.FindCollectionByNameOrId("serial_numbers")
	if err != nil {
		return nil, fmt.Errorf("serial_numbers collection not found: %w", err)
	}

	for i, item := range params.Items {
		if item.TotalQty == 0 {
			continue
		}

		taxableAmount := item.Rate * float64(item.TotalQty)
		taxAmount := taxableAmount * item.TaxPercentage / 100.0
		totalAmount := taxableAmount + taxAmount

		lineItem := core.NewRecord(lineItemCol)
		lineItem.Set("dc", transitDC.Id)
		lineItem.Set("source_item_type", item.SourceItemType)
		lineItem.Set("source_item_id", item.SourceItemID)
		lineItem.Set("quantity", item.TotalQty)
		lineItem.Set("rate", item.Rate)
		lineItem.Set("tax_percentage", item.TaxPercentage)
		lineItem.Set("taxable_amount", taxableAmount)
		lineItem.Set("tax_amount", taxAmount)
		lineItem.Set("total_amount", totalAmount)
		lineItem.Set("line_order", i+1)
		if err := app.Save(lineItem); err != nil {
			return nil, fmt.Errorf("failed to create transit DC line item: %w", err)
		}

		// Create serial number records
		for _, serial := range item.Serials {
			serialRec := core.NewRecord(serialCol)
			serialRec.Set("project", params.ProjectID)
			serialRec.Set("line_item", lineItem.Id)
			serialRec.Set("serial_number", serial)
			if err := app.Save(serialRec); err != nil {
				return nil, fmt.Errorf("failed to save serial number %s: %w", serial, err)
			}
		}
	}

	// 5. Create transit details
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
			detail.Set("dc", transitDC.Id)
			if params.TransporterID != "" {
				detail.Set("transporter", params.TransporterID)
			}
			detail.Set("vehicle_number", vehicleNumber)
			detail.Set("eway_bill_number", params.EwayBillNumber)
			detail.Set("docket_number", params.DocketNumber)
			if err := app.Save(detail); err != nil {
				return nil, fmt.Errorf("failed to create transit details: %w", err)
			}
		}
	}

	// 6. Create Official DCs (one per destination with qty > 0)
	for destIdx, shipToID := range params.ShipToIDs {
		if shipToID == "" {
			continue
		}

		// Check if this destination has any qty
		hasQty := false
		for _, item := range params.Items {
			if destIdx < len(item.Quantities) && item.Quantities[destIdx] > 0 {
				hasQty = true
				break
			}
		}
		if !hasQty {
			continue
		}

		odcNum, err := NextDocNumber(app, params.ProjectID, "odc", docDate)
		if err != nil {
			return nil, fmt.Errorf("failed to generate official DC number: %w", err)
		}

		odc := core.NewRecord(dcCol)
		odc.Set("project", params.ProjectID)
		odc.Set("dc_number", odcNum)
		odc.Set("dc_type", "official")
		odc.Set("status", "draft")
		if params.TemplateID != "" {
			odc.Set("template", params.TemplateID)
		}
		if params.BillFromID != "" {
			odc.Set("bill_from_address", params.BillFromID)
		}
		if params.DispatchFromID != "" {
			odc.Set("dispatch_from_address", params.DispatchFromID)
		}
		if params.BillToID != "" {
			odc.Set("bill_to_address", params.BillToID)
		}
		odc.Set("ship_to_address", shipToID)
		odc.Set("challan_date", params.ChallanDate)
		odc.Set("shipment_group", sgRec.Id)
		if err := app.Save(odc); err != nil {
			return nil, fmt.Errorf("failed to create official DC: %w", err)
		}

		result.OfficialDCIDs = append(result.OfficialDCIDs, odc.Id)
		result.OfficialDCNums = append(result.OfficialDCNums, odcNum)

		// Create line items for official DC (per-location qty, no pricing/serials)
		for lineIdx, item := range params.Items {
			if destIdx >= len(item.Quantities) || item.Quantities[destIdx] == 0 {
				continue
			}
			qty := item.Quantities[destIdx]

			odcLineItem := core.NewRecord(lineItemCol)
			odcLineItem.Set("dc", odc.Id)
			odcLineItem.Set("source_item_type", item.SourceItemType)
			odcLineItem.Set("source_item_id", item.SourceItemID)
			odcLineItem.Set("quantity", qty)
			odcLineItem.Set("line_order", lineIdx+1)
			if err := app.Save(odcLineItem); err != nil {
				return nil, fmt.Errorf("failed to create official DC line item: %w", err)
			}
		}
	}

	return result, nil
}

// CreateTransferDC creates a transfer DC with destination plan.
func CreateTransferDC(app *pocketbase.PocketBase, params TransferDCParams) (*TransferDCResult, error) {
	result := &TransferDCResult{}
	docDate, err := time.Parse("2006-01-02", params.ChallanDate)
	if err != nil {
		docDate = time.Now()
	}

	// 1. Generate Transfer DC number
	stdcNum, err := NextDocNumber(app, params.ProjectID, "stdc", docDate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate transfer DC number: %w", err)
	}
	result.DCNumber = stdcNum

	// 2. Create Delivery Challan record (type=transfer)
	dcCol, err := app.FindCollectionByNameOrId("delivery_challans")
	if err != nil {
		return nil, fmt.Errorf("delivery_challans collection not found: %w", err)
	}

	dc := core.NewRecord(dcCol)
	dc.Set("project", params.ProjectID)
	dc.Set("dc_number", stdcNum)
	dc.Set("dc_type", "transfer")
	dc.Set("status", "draft")
	if params.TemplateID != "" {
		dc.Set("template", params.TemplateID)
	}
	if params.BillFromID != "" {
		dc.Set("bill_from_address", params.BillFromID)
	}
	if params.DispatchFromID != "" {
		dc.Set("dispatch_from_address", params.DispatchFromID)
	}
	if params.BillToID != "" {
		dc.Set("bill_to_address", params.BillToID)
	}
	if len(params.ShipToIDs) > 0 && params.ShipToIDs[0] != "" {
		dc.Set("ship_to_address", params.ShipToIDs[0])
	}
	dc.Set("challan_date", params.ChallanDate)
	if err := app.Save(dc); err != nil {
		return nil, fmt.Errorf("failed to create transfer DC: %w", err)
	}
	result.DCID = dc.Id

	// 3. Create line items with pricing and serials
	lineItemCol, err := app.FindCollectionByNameOrId("dc_line_items")
	if err != nil {
		return nil, fmt.Errorf("dc_line_items collection not found: %w", err)
	}

	serialCol, err := app.FindCollectionByNameOrId("serial_numbers")
	if err != nil {
		return nil, fmt.Errorf("serial_numbers collection not found: %w", err)
	}

	for i, item := range params.Items {
		if item.TotalQty == 0 {
			continue
		}

		taxableAmount := item.Rate * float64(item.TotalQty)
		taxAmount := taxableAmount * item.TaxPercentage / 100.0
		totalAmount := taxableAmount + taxAmount

		lineItem := core.NewRecord(lineItemCol)
		lineItem.Set("dc", dc.Id)
		lineItem.Set("source_item_type", item.SourceItemType)
		lineItem.Set("source_item_id", item.SourceItemID)
		lineItem.Set("quantity", item.TotalQty)
		lineItem.Set("rate", item.Rate)
		lineItem.Set("tax_percentage", item.TaxPercentage)
		lineItem.Set("taxable_amount", taxableAmount)
		lineItem.Set("tax_amount", taxAmount)
		lineItem.Set("total_amount", totalAmount)
		lineItem.Set("line_order", i+1)
		if err := app.Save(lineItem); err != nil {
			return nil, fmt.Errorf("failed to create transfer DC line item: %w", err)
		}

		// Create serial number records
		for _, serial := range item.Serials {
			serialRec := core.NewRecord(serialCol)
			serialRec.Set("project", params.ProjectID)
			serialRec.Set("line_item", lineItem.Id)
			serialRec.Set("serial_number", serial)
			if err := app.Save(serialRec); err != nil {
				return nil, fmt.Errorf("failed to save serial number %s: %w", serial, err)
			}
		}
	}

	// 4. Create Transfer DC metadata
	transferDCCol, err := app.FindCollectionByNameOrId("transfer_dcs")
	if err != nil {
		return nil, fmt.Errorf("transfer_dcs collection not found: %w", err)
	}

	vehicleNumber := ""
	if params.VehicleID != "" {
		if vRec, err := app.FindRecordById("transporter_vehicles", params.VehicleID); err == nil {
			vehicleNumber = vRec.GetString("vehicle_number")
		}
	}

	transferDC := core.NewRecord(transferDCCol)
	transferDC.Set("dc", dc.Id)
	if params.HubAddressID != "" {
		transferDC.Set("hub_address", params.HubAddressID)
	}
	if params.TemplateID != "" {
		transferDC.Set("template", params.TemplateID)
	}
	transferDC.Set("tax_type", params.TaxType)
	transferDC.Set("reverse_charge", params.ReverseCharge)
	if params.TransporterID != "" {
		transferDC.Set("transporter", params.TransporterID)
	}
	transferDC.Set("vehicle_number", vehicleNumber)
	transferDC.Set("eway_bill_number", params.EwayBillNumber)
	transferDC.Set("docket_number", params.DocketNumber)
	transferDC.Set("num_destinations", len(params.ShipToIDs))
	transferDC.Set("num_split", 0)
	if err := app.Save(transferDC); err != nil {
		return nil, fmt.Errorf("failed to create transfer DC metadata: %w", err)
	}

	// 5. Create destination records with per-product quantities
	destCol, err := app.FindCollectionByNameOrId("transfer_dc_destinations")
	if err != nil {
		return nil, fmt.Errorf("transfer_dc_destinations collection not found: %w", err)
	}

	destQtyCol, err := app.FindCollectionByNameOrId("transfer_dc_dest_quantities")
	if err != nil {
		return nil, fmt.Errorf("transfer_dc_dest_quantities collection not found: %w", err)
	}

	for destIdx, shipToID := range params.ShipToIDs {
		if shipToID == "" {
			continue
		}

		destRec := core.NewRecord(destCol)
		destRec.Set("transfer_dc", transferDC.Id)
		destRec.Set("ship_to_address", shipToID)
		destRec.Set("is_split", false)
		if err := app.Save(destRec); err != nil {
			return nil, fmt.Errorf("failed to create transfer DC destination: %w", err)
		}

		// Per-item quantities for this destination
		for _, item := range params.Items {
			if destIdx >= len(item.Quantities) || item.Quantities[destIdx] == 0 {
				continue
			}

			destQty := core.NewRecord(destQtyCol)
			destQty.Set("destination", destRec.Id)
			destQty.Set("source_item_type", item.SourceItemType)
			destQty.Set("source_item_id", item.SourceItemID)
			destQty.Set("quantity", item.Quantities[destIdx])
			if err := app.Save(destQty); err != nil {
				return nil, fmt.Errorf("failed to create destination quantity: %w", err)
			}
		}
	}

	// 6. Create transit details
	if params.TransporterID != "" || params.EwayBillNumber != "" || params.DocketNumber != "" {
		transitDetailCol, err := app.FindCollectionByNameOrId("dc_transit_details")
		if err == nil {
			detail := core.NewRecord(transitDetailCol)
			detail.Set("dc", dc.Id)
			if params.TransporterID != "" {
				detail.Set("transporter", params.TransporterID)
			}
			detail.Set("vehicle_number", vehicleNumber)
			detail.Set("eway_bill_number", params.EwayBillNumber)
			detail.Set("docket_number", params.DocketNumber)
			_ = app.Save(detail)
		}
	}

	return result, nil
}

// ParseShipmentItemsFromForm extracts item data from wizard form submission.
func ParseShipmentItemsFromForm(form map[string][]string, numDest int) []ShipmentItemParams {
	itemKeys := form["item_keys"]
	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, k := range itemKeys {
		if !seen[k] {
			seen[k] = true
			unique = append(unique, k)
		}
	}

	var items []ShipmentItemParams
	for _, key := range unique {
		getVal := func(name string) string {
			vals := form[name]
			if len(vals) > 0 {
				return vals[0]
			}
			return ""
		}

		rateStr := getVal(fmt.Sprintf("item_rate_%s", key))
		taxStr := getVal(fmt.Sprintf("item_tax_%s", key))
		rate := parseFormFloat(rateStr)
		taxPct := parseFormFloat(taxStr)

		quantities := make([]int, numDest)
		totalQty := 0
		for i := 0; i < numDest; i++ {
			qtyStr := getVal(fmt.Sprintf("qty_%s_dest_%d", key, i))
			qty := parseFormInt(qtyStr)
			quantities[i] = qty
			totalQty += qty
		}

		serialsRaw := getVal(fmt.Sprintf("serials_%s", key))
		serials := parseSerials(serialsRaw)

		items = append(items, ShipmentItemParams{
			SourceItemType: getVal(fmt.Sprintf("item_type_%s", key)),
			SourceItemID:   getVal(fmt.Sprintf("item_id_%s", key)),
			Rate:           rate,
			TaxPercentage:  taxPct,
			SerialTracking: getVal(fmt.Sprintf("item_serial_tracking_%s", key)),
			Quantities:     quantities,
			TotalQty:       totalQty,
			Serials:        serials,
		})
	}

	return items
}

func parseFormFloat(s string) float64 {
	s = strings.TrimSpace(s)
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func parseFormInt(s string) int {
	s = strings.TrimSpace(s)
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func parseSerials(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, ",", "\n")
	parts := strings.Split(raw, "\n")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

