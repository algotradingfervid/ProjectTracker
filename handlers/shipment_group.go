package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// HandleShipmentGroupDetail renders the shipment group detail view.
func HandleShipmentGroupDetail(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		sgId := e.Request.PathValue("id")

		sg, err := app.FindRecordById("shipment_groups", sgId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Shipment group not found")
		}
		if sg.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Shipment group not found in this project")
		}

		// Fetch all DCs in this shipment group
		dcs, _ := app.FindRecordsByFilter("delivery_challans", "shipment_group = {:sgid}", "dc_type,dc_number", 0, 0, map[string]any{"sgid": sgId})

		var transitDC *templates.ShipmentGroupDCItem
		var officialDCs []templates.ShipmentGroupDCItem

		for _, dc := range dcs {
			dcType := dc.GetString("dc_type")

			// Resolve ship-to address display
			shipToDisplay := ""
			if shipToID := dc.GetString("ship_to_address"); shipToID != "" {
				if aRec, err := app.FindRecordById("addresses", shipToID); err == nil {
					data := readAddressData(aRec)
					shipToDisplay = data["company_name"]
					if shipToDisplay == "" {
						shipToDisplay = data["contact_person"]
					}
					if city := data["city"]; city != "" {
						shipToDisplay += ", " + city
					}
				}
			}

			// Count line items
			lineItems, _ := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "", 0, 0, map[string]any{"did": dc.Id})
			var totalQty int
			var grandTotal float64
			for _, li := range lineItems {
				totalQty += li.GetInt("quantity")
				grandTotal += li.GetFloat("total_amount")
			}

			item := templates.ShipmentGroupDCItem{
				ID:          dc.Id,
				DCNumber:    dc.GetString("dc_number"),
				DCType:      dcType,
				Status:      dc.GetString("status"),
				ChallanDate: dc.GetString("challan_date"),
				ShipTo:      shipToDisplay,
				ItemCount:   len(lineItems),
				TotalQty:    strconv.Itoa(totalQty),
				GrandTotal:  services.FormatINR(grandTotal),
			}

			if dcType == "transit" {
				transitDC = &item
			} else {
				officialDCs = append(officialDCs, item)
			}
		}

		data := templates.ShipmentGroupDetailData{
			ProjectID:    projectId,
			GroupID:      sgId,
			Status:       sg.GetString("status"),
			NumLocations: sg.GetInt("num_locations"),
			Created:      sg.GetString("created"),
			TransitDC:    transitDC,
			OfficialDCs:  officialDCs,
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.ShipmentGroupDetailContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.ShipmentGroupDetailPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleShipmentGroupIssueAll issues all DCs in a shipment group.
func HandleShipmentGroupIssueAll(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		sgId := e.Request.PathValue("id")

		sg, err := app.FindRecordById("shipment_groups", sgId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Shipment group not found")
		}
		if sg.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Shipment group not found in this project")
		}

		if err := services.IssueShipmentGroup(app, sgId); err != nil {
			return ErrorToast(e, http.StatusBadRequest, fmt.Sprintf("Failed to issue: %s", err.Error()))
		}

		redirectURL := fmt.Sprintf("/projects/%s/shipment-groups/%s", projectId, sgId)
		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}

// HandleShipmentGroupDelete deletes a draft shipment group and its DCs.
func HandleShipmentGroupDelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		sgId := e.Request.PathValue("id")

		sg, err := app.FindRecordById("shipment_groups", sgId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Shipment group not found")
		}
		if sg.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Shipment group not found in this project")
		}
		if sg.GetString("status") != "draft" {
			return ErrorToast(e, http.StatusBadRequest, "Only draft shipment groups can be deleted")
		}

		// Delete all DCs in the group (cascade should handle line items, serials, etc.)
		dcs, _ := app.FindRecordsByFilter("delivery_challans", "shipment_group = {:sgid}", "", 0, 0, map[string]any{"sgid": sgId})
		for _, dc := range dcs {
			if err := app.Delete(dc); err != nil {
				return ErrorToast(e, http.StatusInternalServerError, fmt.Sprintf("Failed to delete DC %s: %s", dc.GetString("dc_number"), err.Error()))
			}
		}

		// Delete the shipment group
		if err := app.Delete(sg); err != nil {
			return ErrorToast(e, http.StatusInternalServerError, "Failed to delete shipment group")
		}

		redirectURL := fmt.Sprintf("/projects/%s/dcs/", projectId)
		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}
