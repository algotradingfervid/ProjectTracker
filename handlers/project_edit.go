package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

func HandleProjectEdit(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("id")
		if projectID == "" {
			return e.String(http.StatusBadRequest, "Missing project ID")
		}

		record, err := app.FindRecordById("projects", projectID)
		if err != nil {
			log.Printf("project_edit: could not find project %s: %v", projectID, err)
			return e.String(http.StatusNotFound, "Project not found")
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

		data := templates.ProjectEditData{
			ID:                    projectID,
			Name:                  record.GetString("name"),
			ClientName:            record.GetString("client_name"),
			ReferenceNumber:       record.GetString("reference_number"),
			Status:                record.GetString("status"),
			ShipToEqualsInstallAt: record.GetBool("ship_to_equals_install_at"),
			StatusOptions:         ProjectStatusOptions,
			BOQCount:              len(boqs),
			AddressCount:          addressCount,
			CreatedDate:           createdDate,
			Errors:                make(map[string]string),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.ProjectEditContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.ProjectEditPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

func HandleProjectUpdate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("id")
		if projectID == "" {
			return e.String(http.StatusBadRequest, "Missing project ID")
		}

		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "Invalid form data")
		}

		record, err := app.FindRecordById("projects", projectID)
		if err != nil {
			log.Printf("project_update: could not find project %s: %v", projectID, err)
			return e.String(http.StatusNotFound, "Project not found")
		}

		name := strings.TrimSpace(e.Request.FormValue("name"))
		clientName := strings.TrimSpace(e.Request.FormValue("client_name"))
		refNumber := strings.TrimSpace(e.Request.FormValue("reference_number"))
		status := strings.TrimSpace(e.Request.FormValue("status"))
		shipToEqualsInstallAt := e.Request.FormValue("ship_to_equals_install_at") == "on" ||
			e.Request.FormValue("ship_to_equals_install_at") == "true"

		errors := make(map[string]string)
		if name == "" {
			errors["name"] = "Project name is required"
		}

		validStatus := false
		for _, s := range ProjectStatusOptions {
			if status == s {
				validStatus = true
				break
			}
		}
		if !validStatus {
			status = record.GetString("status")
		}

		if name != "" {
			existing, _ := app.FindRecordsByFilter(
				"projects",
				"name = {:name} && id != {:id}",
				"", 1, 0,
				map[string]any{"name": name, "id": projectID},
			)
			if len(existing) > 0 {
				errors["name"] = "A project with this name already exists"
			}
		}

		if len(errors) > 0 {
			boqs, _ := app.FindRecordsByFilter("boqs", "project = {:projectId}", "", 0, 0, map[string]any{"projectId": projectID})
			var addressCount int
			addresses, err := app.FindRecordsByFilter("addresses", "project = {:projectId}", "", 0, 0, map[string]any{"projectId": projectID})
			if err == nil {
				addressCount = len(addresses)
			}

			createdDate := "—"
			if dt := record.GetDateTime("created"); !dt.IsZero() {
				createdDate = dt.Time().Format("02 Jan 2006")
			}

			data := templates.ProjectEditData{
				ID:                    projectID,
				Name:                  name,
				ClientName:            clientName,
				ReferenceNumber:       refNumber,
				Status:                status,
				ShipToEqualsInstallAt: shipToEqualsInstallAt,
				StatusOptions:         ProjectStatusOptions,
				BOQCount:              len(boqs),
				AddressCount:          addressCount,
				CreatedDate:           createdDate,
				Errors:                errors,
			}

			var component templ.Component
			if e.Request.Header.Get("HX-Request") == "true" {
				component = templates.ProjectEditContent(data)
			} else {
				headerData := GetHeaderData(e.Request)
				sidebarData := GetSidebarData(e.Request)
				component = templates.ProjectEditPage(data, headerData, sidebarData)
			}
			return component.Render(e.Request.Context(), e.Response)
		}

		record.Set("name", name)
		record.Set("client_name", clientName)
		record.Set("reference_number", refNumber)
		record.Set("status", status)
		record.Set("ship_to_equals_install_at", shipToEqualsInstallAt)

		if err := app.Save(record); err != nil {
			log.Printf("project_update: could not save project %s: %v", projectID, err)
			return e.String(http.StatusInternalServerError, "Failed to save project")
		}

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", "/projects")
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, "/projects")
	}
}
