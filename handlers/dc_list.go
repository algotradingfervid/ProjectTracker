package handlers

import (
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

// HandleDCList renders the delivery challan list for a project.
func HandleDCList(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")

		// Query parameters for filtering
		typeFilter := e.Request.URL.Query().Get("type")
		statusFilter := e.Request.URL.Query().Get("status")
		search := e.Request.URL.Query().Get("search")
		pageStr := e.Request.URL.Query().Get("page")
		page, _ := strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
		perPage := 25

		// Build filter
		filter := "project = {:pid}"
		params := map[string]any{"pid": projectId}

		if typeFilter != "" && typeFilter != "all" {
			filter += " && dc_type = {:dctype}"
			params["dctype"] = typeFilter
		}
		if statusFilter != "" && statusFilter != "all" {
			filter += " && status = {:status}"
			params["status"] = statusFilter
		}
		if search != "" {
			filter += " && dc_number ~ {:search}"
			params["search"] = search
		}

		dcCol, err := app.FindCollectionByNameOrId("delivery_challans")
		if err != nil {
			return ErrorToast(e, http.StatusInternalServerError, "Collection not found")
		}

		// Get total count
		allRecords, _ := app.FindRecordsByFilter(dcCol, filter, "", 0, 0, params)
		total := len(allRecords)

		// Get paginated records
		offset := (page - 1) * perPage
		records, _ := app.FindRecordsByFilter(dcCol, filter, "-created", perPage, offset, params)

		var dcItems []templates.DCListItem
		for _, rec := range records {
			item := templates.DCListItem{
				ID:          rec.Id,
				DCNumber:    rec.GetString("dc_number"),
				DCType:      rec.GetString("dc_type"),
				Status:      rec.GetString("status"),
				ChallanDate: rec.GetString("challan_date"),
				Created:     rec.GetString("created"),
			}

			// Resolve template name
			if templateID := rec.GetString("template"); templateID != "" {
				if tRec, err := app.FindRecordById("dc_templates", templateID); err == nil {
					item.TemplateName = tRec.GetString("name")
				}
			}

			// Resolve ship-to display name
			if shipToID := rec.GetString("ship_to_address"); shipToID != "" {
				if aRec, err := app.FindRecordById("addresses", shipToID); err == nil {
					data := readAddressData(aRec)
					item.ShipTo = data["company_name"]
					if item.ShipTo == "" {
						item.ShipTo = data["contact_person"]
					}
					if city := data["city"]; city != "" {
						item.ShipTo += ", " + city
					}
				}
			}

			// Count line items
			lineItems, _ := app.FindRecordsByFilter("dc_line_items", "dc = {:did}", "", 0, 0, map[string]any{"did": rec.Id})
			item.ItemCount = len(lineItems)

			dcItems = append(dcItems, item)
		}

		totalPages := (total + perPage - 1) / perPage

		data := templates.DCListData{
			ProjectID:    projectId,
			DCs:          dcItems,
			TypeFilter:   typeFilter,
			StatusFilter: statusFilter,
			Search:       search,
			Total:        total,
			Page:         page,
			TotalPages:   totalPages,
			PerPage:      perPage,
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.DCListContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.DCListPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

