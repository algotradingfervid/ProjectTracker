package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

type contextKey string

const ActiveProjectKey contextKey = "activeProject"
const HeaderDataKey contextKey = "headerData"
const SidebarDataKey contextKey = "sidebarData"

// GetActiveProject extracts the active project from the request context.
func GetActiveProject(r *http.Request) *templates.ActiveProject {
	if val, ok := r.Context().Value(ActiveProjectKey).(*templates.ActiveProject); ok {
		return val
	}
	return nil
}

// GetHeaderData extracts the pre-built HeaderData from the request context.
func GetHeaderData(r *http.Request) templates.HeaderData {
	if val, ok := r.Context().Value(HeaderDataKey).(templates.HeaderData); ok {
		return val
	}
	return templates.HeaderData{}
}

// GetSidebarData extracts the pre-built SidebarData from the request context.
func GetSidebarData(r *http.Request) templates.SidebarData {
	if val, ok := r.Context().Value(SidebarDataKey).(templates.SidebarData); ok {
		return val
	}
	return templates.SidebarData{}
}

// ActiveProjectMiddleware reads the "active_project" cookie, loads the project
// record, builds HeaderData with the full project list, and stores both in the
// request context so handlers and templates can use them.
func ActiveProjectMiddleware(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var activeProj *templates.ActiveProject

		// Read cookie
		cookie, err := e.Request.Cookie("active_project")
		if err == nil && cookie.Value != "" {
			rec, err := app.FindRecordById("projects", cookie.Value)
			if err == nil {
				activeProj = &templates.ActiveProject{
					ID:   rec.Id,
					Name: rec.GetString("name"),
				}
			} else {
				log.Printf("middleware: active project %s not found, clearing cookie", cookie.Value)
				http.SetCookie(e.Response, &http.Cookie{
					Name:   "active_project",
					Value:  "",
					Path:   "/",
					MaxAge: -1,
				})
			}
		}

		// Build full project list for the header dropdown
		projectsCol, _ := app.FindCollectionByNameOrId("projects")
		var selectorItems []templates.ProjectSelectorItem
		if projectsCol != nil {
			records, _ := app.FindAllRecords(projectsCol)
			for _, rec := range records {
				isActive := activeProj != nil && rec.Id == activeProj.ID
				selectorItems = append(selectorItems, templates.ProjectSelectorItem{
					ID:       rec.Id,
					Name:     rec.GetString("name"),
					Client:   rec.GetString("client"),
					IsActive: isActive,
				})
			}
		}

		headerData := templates.HeaderData{
			ActiveProject: activeProj,
			Projects:      selectorItems,
		}

		// Store in context
		ctx := context.WithValue(e.Request.Context(), ActiveProjectKey, activeProj)
		ctx = context.WithValue(ctx, HeaderDataKey, headerData)
		e.Request = e.Request.WithContext(ctx)

		// Build sidebar data (needs activeProj in context first)
		sidebarData := BuildSidebarData(e.Request, app)
		ctx = context.WithValue(e.Request.Context(), SidebarDataKey, sidebarData)
		e.Request = e.Request.WithContext(ctx)

		return e.Next()
	}
}
