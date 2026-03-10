package handlers

import (
	"log"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

func HandleTransporterList(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")

		records, err := app.FindRecordsByFilter(
			"transporters",
			"project = {:projectId}",
			"-created",
			0, 0,
			map[string]any{"projectId": projectId},
		)
		if err != nil {
			log.Printf("transporter_list: could not query transporters: %v", err)
			records = nil
		}

		var items []templates.TransporterListItem
		for _, rec := range records {
			vehicles, _ := app.FindRecordsByFilter(
				"transporter_vehicles",
				"transporter = {:tid}",
				"", 0, 0,
				map[string]any{"tid": rec.Id},
			)

			items = append(items, templates.TransporterListItem{
				ID:            rec.Id,
				CompanyName:   rec.GetString("company_name"),
				ContactPerson: rec.GetString("contact_person"),
				Phone:         rec.GetString("phone"),
				GSTNumber:     rec.GetString("gst_number"),
				IsActive:      rec.GetBool("is_active"),
				VehicleCount:  len(vehicles),
			})
		}

		data := templates.TransporterListData{
			ProjectID:    projectId,
			Transporters: items,
			Total:        len(items),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.TransporterListContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.TransporterListPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}
