package handlers

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// HandleProjectActivate sets the active project cookie and returns a full page
// redirect via HX-Redirect so the entire shell (header + sidebar) re-renders.
func HandleProjectActivate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("id")

		// Verify project exists
		_, err := app.FindRecordById("projects", projectID)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Project not found")
		}

		// Set cookie (30-day expiry, HttpOnly)
		http.SetCookie(e.Response, &http.Cookie{
			Name:     "active_project",
			Value:    projectID,
			Path:     "/",
			MaxAge:   60 * 60 * 24 * 30,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		SetToast(e, "success", "Project activated")

		// Tell HTMX to do a full page redirect so header + sidebar re-render
		e.Response.Header().Set("HX-Redirect", "/projects/"+projectID)
		return e.String(200, "OK")
	}
}

// HandleProjectDeactivate clears the active project cookie and redirects to /projects.
func HandleProjectDeactivate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		http.SetCookie(e.Response, &http.Cookie{
			Name:   "active_project",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})

		SetToast(e, "success", "Project deactivated")

		e.Response.Header().Set("HX-Redirect", "/projects")
		return e.String(200, "OK")
	}
}
