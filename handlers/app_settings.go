package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"

	"projectcreation/collections"
	"projectcreation/templates"
)

// HandleAppSettings renders the global app settings page (GET /settings).
func HandleAppSettings(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		record, err := collections.GetAppSettings(app)
		if err != nil {
			log.Printf("app_settings: could not load settings: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Could not load settings")
		}

		data := templates.AppSettingsData{
			CompanyName: record.GetString("company_name"),
			LogoURL:     collections.GetLogoURL(app),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.AppSettingsContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.AppSettingsPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleAppSettingsSave handles the POST /settings form submission.
func HandleAppSettingsSave(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Parse multipart form (for logo file upload)
		if err := e.Request.ParseMultipartForm(5 << 20); err != nil {
			// Fallback to regular form parsing if not multipart
			if parseErr := e.Request.ParseForm(); parseErr != nil {
				return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
			}
		}

		record, err := collections.GetAppSettings(app)
		if err != nil {
			log.Printf("app_settings_save: could not load settings: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Could not load settings")
		}

		// Validate company name
		companyName := strings.TrimSpace(e.Request.FormValue("company_name"))
		if companyName == "" {
			data := templates.AppSettingsData{
				CompanyName: e.Request.FormValue("company_name"),
				LogoURL:     collections.GetLogoURL(app),
				Errors:      map[string]string{"company_name": "Company name is required"},
			}
			var component templ.Component
			if e.Request.Header.Get("HX-Request") == "true" {
				component = templates.AppSettingsContent(data)
			} else {
				headerData := GetHeaderData(e.Request)
				sidebarData := GetSidebarData(e.Request)
				component = templates.AppSettingsPage(data, headerData, sidebarData)
			}
			return component.Render(e.Request.Context(), e.Response)
		}

		record.Set("company_name", companyName)

		// Handle logo removal
		if e.Request.FormValue("remove_logo") == "true" {
			record.Set("logo", "")
		}

		// Handle logo upload
		if e.Request.MultipartForm != nil {
			file, header, fileErr := e.Request.FormFile("logo")
			if fileErr == nil && header != nil && header.Size > 0 {
				defer file.Close()
				f, fErr := filesystem.NewFileFromMultipart(header)
				if fErr != nil {
					log.Printf("app_settings_save: could not process logo file: %v", fErr)
					return ErrorToast(e, http.StatusBadRequest, "Could not process uploaded file")
				}
				record.Set("logo", f)
			}
		}

		if err := app.Save(record); err != nil {
			log.Printf("app_settings_save: could not save settings: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Could not save settings")
		}

		SetToast(e, "success", "Settings saved")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", "/settings")
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, "/settings")
	}
}
