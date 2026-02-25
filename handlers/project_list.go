package handlers

import (
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

func statusBadgeClass(status string) string {
	switch status {
	case "active":
		return "badge-success"
	case "completed":
		return "badge-info"
	case "on_hold":
		return "badge-warning"
	default:
		return "badge-ghost"
	}
}

func HandleProjectList(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectsCol, err := app.FindCollectionByNameOrId("projects")
		if err != nil {
			log.Printf("project_list: could not find projects collection: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		records, err := app.FindAllRecords(projectsCol)
		if err != nil {
			log.Printf("project_list: could not query projects: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		boqsCol, err := app.FindCollectionByNameOrId("boqs")
		if err != nil {
			log.Printf("project_list: could not find boqs collection: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		addressesCol, _ := app.FindCollectionByNameOrId("addresses")

		var items []templates.ProjectListItem

		for _, rec := range records {
			projectID := rec.Id

			boqs, err := app.FindRecordsByFilter(
				boqsCol,
				"project = {:projectId}",
				"", 0, 0,
				map[string]any{"projectId": projectID},
			)
			if err != nil {
				boqs = nil
			}

			var addressCount int
			if addressesCol != nil {
				addresses, err := app.FindRecordsByFilter(
					addressesCol,
					"project = {:projectId}",
					"", 0, 0,
					map[string]any{"projectId": projectID},
				)
				if err == nil {
					addressCount = len(addresses)
				}
			}

			createdDate := "—"
			if dt := rec.GetDateTime("created"); !dt.IsZero() {
				createdDate = dt.Time().Format("02 Jan 2006")
			}

			status := rec.GetString("status")

			items = append(items, templates.ProjectListItem{
				ID:                    projectID,
				Name:                  rec.GetString("name"),
				ClientName:            rec.GetString("client_name"),
				ReferenceNumber:       rec.GetString("reference_number"),
				Status:                status,
				StatusBadgeClass:      statusBadgeClass(status),
				BOQCount:              len(boqs),
				AddressCount:          addressCount,
				ShipToEqualsInstallAt: rec.GetBool("ship_to_equals_install_at"),
				CreatedDate:           createdDate,
			})
		}

		data := templates.ProjectListData{
			Items:      items,
			TotalCount: len(records),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.ProjectListContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.ProjectListPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}
