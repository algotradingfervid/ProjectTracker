package handlers

import (
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// HandleDCDelete handles DELETE request for a delivery challan (draft only).
func HandleDCDelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		dcId := e.Request.PathValue("id")

		dc, err := app.FindRecordById("delivery_challans", dcId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Delivery challan not found")
		}
		if dc.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Delivery challan not found in this project")
		}
		if dc.GetString("status") != "draft" {
			return ErrorToast(e, http.StatusBadRequest, "Only draft DCs can be deleted")
		}

		// For transit/official DCs in a shipment group, delete the whole group
		if sgID := dc.GetString("shipment_group"); sgID != "" {
			return deleteShipmentGroup(app, e, projectId, sgID)
		}

		// For transfer DCs, delete associated metadata first
		if dc.GetString("dc_type") == "transfer" {
			if err := deleteTransferDCMetadata(app, dcId); err != nil {
				return ErrorToast(e, http.StatusInternalServerError, "Failed to delete transfer DC data: "+err.Error())
			}
		}

		// Delete the DC (cascade will handle line items and serials)
		if err := app.Delete(dc); err != nil {
			return ErrorToast(e, http.StatusInternalServerError, "Failed to delete DC: "+err.Error())
		}

		SetToast(e, "success", "Delivery challan deleted")
		redirectURL := fmt.Sprintf("/projects/%s/dcs/", projectId)
		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}

// deleteShipmentGroup deletes all DCs in a shipment group and the group itself.
func deleteShipmentGroup(app *pocketbase.PocketBase, e *core.RequestEvent, projectId, sgID string) error {
	sg, err := app.FindRecordById("shipment_groups", sgID)
	if err != nil {
		return ErrorToast(e, http.StatusNotFound, "Shipment group not found")
	}

	if sg.GetString("status") != "draft" {
		return ErrorToast(e, http.StatusBadRequest, "Only draft shipment groups can be deleted")
	}

	// Find and delete all DCs in the group
	dcs, _ := app.FindRecordsByFilter("delivery_challans", "shipment_group = {:sgid}", "", 0, 0, map[string]any{"sgid": sgID})
	for _, dc := range dcs {
		// Delete transit details
		transitDetails, _ := app.FindRecordsByFilter("dc_transit_details", "dc = {:did}", "", 0, 0, map[string]any{"did": dc.Id})
		for _, td := range transitDetails {
			_ = app.Delete(td)
		}
		// Delete serial numbers for line items
		lineItems, _ := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "", 0, 0, map[string]any{"did": dc.Id})
		for _, li := range lineItems {
			serials, _ := app.FindRecordsByFilter("serial_numbers", "line_item = {:lid}", "", 0, 0, map[string]any{"lid": li.Id})
			for _, s := range serials {
				_ = app.Delete(s)
			}
			_ = app.Delete(li)
		}
		_ = app.Delete(dc)
	}

	// Delete the shipment group
	_ = app.Delete(sg)

	SetToast(e, "success", "Shipment group and all DCs deleted")
	redirectURL := fmt.Sprintf("/projects/%s/dcs/", projectId)
	if e.Request.Header.Get("HX-Request") == "true" {
		e.Response.Header().Set("HX-Redirect", redirectURL)
		return e.String(http.StatusOK, "")
	}
	return e.Redirect(http.StatusFound, redirectURL)
}

// deleteTransferDCMetadata deletes transfer_dcs, destinations, and dest_quantities records.
func deleteTransferDCMetadata(app *pocketbase.PocketBase, dcID string) error {
	transferRecs, _ := app.FindRecordsByFilter("transfer_dcs", "dc = {:did}", "", 0, 0, map[string]any{"did": dcID})
	for _, tdc := range transferRecs {
		// Delete destinations and their quantities
		dests, _ := app.FindRecordsByFilter("transfer_dc_destinations", "transfer_dc = {:tid}", "", 0, 0, map[string]any{"tid": tdc.Id})
		for _, dest := range dests {
			qtys, _ := app.FindRecordsByFilter("transfer_dc_dest_quantities", "destination = {:did}", "", 0, 0, map[string]any{"did": dest.Id})
			for _, q := range qtys {
				_ = app.Delete(q)
			}
			_ = app.Delete(dest)
		}
		_ = app.Delete(tdc)
	}

	// Delete transit details
	transitDetails, _ := app.FindRecordsByFilter("dc_transit_details", "dc = {:did}", "", 0, 0, map[string]any{"did": dcID})
	for _, td := range transitDetails {
		_ = app.Delete(td)
	}

	// Delete serial numbers for line items
	lineItems, _ := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "", 0, 0, map[string]any{"did": dcID})
	for _, li := range lineItems {
		serials, _ := app.FindRecordsByFilter("serial_numbers", "line_item = {:lid}", "", 0, 0, map[string]any{"lid": li.Id})
		for _, s := range serials {
			_ = app.Delete(s)
		}
		_ = app.Delete(li)
	}

	return nil
}
