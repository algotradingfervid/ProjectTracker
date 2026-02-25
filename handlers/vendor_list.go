package handlers

import (
	"log"
	"strings"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

func HandleVendorList(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		searchQuery := strings.TrimSpace(e.Request.URL.Query().Get("q"))

		var records []*core.Record
		var err error

		if searchQuery != "" {
			records, err = app.FindRecordsByFilter(
				"vendors",
				"name ~ {:q} || city ~ {:q} || gstin ~ {:q}",
				"name",
				0, 0,
				map[string]any{"q": searchQuery},
			)
		} else {
			records, err = app.FindRecordsByFilter(
				"vendors",
				"1=1",
				"name",
				0, 0,
				nil,
			)
		}
		if err != nil {
			log.Printf("vendor_list: could not query vendors: %v", err)
			records = nil
		}

		// If project context, get linked vendor IDs
		linkedVendorIDs := make(map[string]bool)
		if projectID != "" {
			links, err := app.FindRecordsByFilter(
				"project_vendors",
				"project = {:projectId}",
				"", 0, 0,
				map[string]any{"projectId": projectID},
			)
			if err == nil {
				for _, link := range links {
					linkedVendorIDs[link.GetString("vendor")] = true
				}
			}
		}

		var items []templates.VendorListItem
		for _, rec := range records {
			items = append(items, templates.VendorListItem{
				ID:          rec.Id,
				Name:        rec.GetString("name"),
				City:        rec.GetString("city"),
				GSTIN:       rec.GetString("gstin"),
				ContactName: rec.GetString("contact_name"),
				Phone:       rec.GetString("phone"),
				Email:       rec.GetString("email"),
				IsLinked:    linkedVendorIDs[rec.Id],
			})
		}

		data := templates.VendorListData{
			Vendors:     items,
			SearchQuery: searchQuery,
			ProjectID:   projectID,
			TotalCount:  len(items),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.VendorListContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.VendorListPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}
