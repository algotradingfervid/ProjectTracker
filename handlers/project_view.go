package handlers

import (
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

func HandleProjectView(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("id")
		if projectID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing project ID")
		}

		record, err := app.FindRecordById("projects", projectID)
		if err != nil {
			log.Printf("project_view: could not find project %s: %v", projectID, err)
			return ErrorToast(e, http.StatusNotFound, "Project not found")
		}

		// Auto-activate this project so sidebar links point to the correct project
		http.SetCookie(e.Response, &http.Cookie{
			Name:     "active_project",
			Value:    projectID,
			Path:     "/",
			MaxAge:   60 * 60 * 24 * 30,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		// If HTMX request and active project differs, force full page reload
		// so the sidebar re-renders with the correct project context
		if e.Request.Header.Get("HX-Request") == "true" {
			activeProject := GetActiveProject(e.Request)
			if activeProject == nil || activeProject.ID != projectID {
				e.Response.Header().Set("HX-Redirect", "/projects/"+projectID)
				return e.String(http.StatusOK, "")
			}
		}

		boqs, _ := app.FindRecordsByFilter(
			"boqs",
			"project = {:projectId}",
			"", 0, 0,
			map[string]any{"projectId": projectID},
		)

		var addressCount int
		addresses, err := app.FindRecordsByFilter(
			"addresses",
			"project = {:projectId}",
			"", 0, 0,
			map[string]any{"projectId": projectID},
		)
		if err == nil {
			addressCount = len(addresses)
		}

		createdDate := "—"
		if dt := record.GetDateTime("created"); !dt.IsZero() {
			createdDate = dt.Time().Format("02 Jan 2006")
		}

		data := templates.ProjectViewData{
			ID:                    projectID,
			Name:                  record.GetString("name"),
			ClientName:            record.GetString("client_name"),
			ReferenceNumber:       record.GetString("reference_number"),
			Status:                record.GetString("status"),
			ShipToEqualsInstallAt: record.GetBool("ship_to_equals_install_at"),
			BOQCount:              len(boqs),
			AddressCount:          addressCount,
			CreatedDate:           createdDate,
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.ProjectViewContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.ProjectViewPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}
