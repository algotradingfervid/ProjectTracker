package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// HandleAddressImportPage renders the upload form.
// Route: GET /projects/{projectId}/addresses/{type}/import
func HandleAddressImportPage(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		addressType := e.Request.PathValue("type")

		// Normalize slug to db type: "ship-to" -> "ship_to", "install-at" -> "install_at"
		dbType := slugToDBType(addressType)
		if dbType != "ship_to" && dbType != "install_at" {
			return ErrorToast(e, http.StatusBadRequest, "Import is only available for Ship To and Install At addresses")
		}

		project, err := app.FindRecordById("projects", projectID)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Project not found")
		}

		data := templates.AddressImportData{
			ProjectID:   projectID,
			ProjectName: project.GetString("name"),
			AddressType: dbType,
			AddressSlug: addressType,
		}

		isHTMX := e.Request.Header.Get("HX-Request") == "true"
		if isHTMX {
			return templates.AddressImportContent(data).Render(e.Request.Context(), e.Response)
		}

		sidebarData := GetSidebarData(e.Request)
		activeProject := GetActiveProject(e.Request)
		headerData := templates.HeaderData{ActiveProject: activeProject}
		return templates.AddressImportPage(data, headerData, sidebarData).Render(e.Request.Context(), e.Response)
	}
}

// HandleAddressValidate receives a file upload, validates it, and returns
// the validation results as an HTMX partial.
// Route: POST /projects/{projectId}/addresses/{type}/import
func HandleAddressValidate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		addressType := e.Request.PathValue("type")

		dbType := slugToDBType(addressType)
		if dbType != "ship_to" && dbType != "install_at" {
			return ErrorToast(e, http.StatusBadRequest, "Invalid address type for import")
		}

		// Parse multipart form (max 10MB)
		if err := e.Request.ParseMultipartForm(10 << 20); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "File too large or invalid form data")
		}

		file, header, err := e.Request.FormFile("file")
		if err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Please select a file to upload")
		}
		defer file.Close()

		// Validate the file
		result, err := services.ValidateAddressFile(app, file, header.Filename, projectID, dbType)
		if err != nil {
			log.Printf("address_validate: %v", err)
			return ErrorToast(e, http.StatusBadRequest, err.Error())
		}

		// Serialize parsed rows for the commit form
		var parsedRowsJSON string
		if result.ErrorRows == 0 {
			b, err := json.Marshal(result.ParsedRows)
			if err != nil {
				log.Printf("address_validate: marshal parsed rows: %v", err)
			} else {
				parsedRowsJSON = string(b)
			}
		}

		// Return HTMX partial with results
		component := templates.AddressValidationResults(
			projectID,
			dbType,
			addressType,
			result,
			parsedRowsJSON,
		)
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleAddressErrorReport downloads the error report as an Excel file.
// Route: POST /projects/{projectId}/addresses/{type}/import/errors
func HandleAddressErrorReport(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		addressType := e.Request.PathValue("type")
		dbType := slugToDBType(addressType)

		// Parse errors from the posted JSON
		var errors []services.ValidationError
		decoder := json.NewDecoder(e.Request.Body)
		if err := decoder.Decode(&errors); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid error data")
		}

		xlsxBytes, err := services.GenerateErrorReport(errors)
		if err != nil {
			log.Printf("error_report: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		typeName := "ShipTo"
		if dbType == "install_at" {
			typeName = "InstallAt"
		}
		filename := fmt.Sprintf("%s_Errors_%s.xlsx", typeName,
			time.Now().Format("2006-01-02"))

		e.Response.Header().Set("Content-Type",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		e.Response.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, filename))
		e.Response.Write(xlsxBytes)
		return nil
	}
}

// HandleAddressImportCommit re-validates and batch-inserts the uploaded addresses.
// Route: POST /projects/{projectId}/addresses/{type}/import/commit
func HandleAddressImportCommit(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		addressType := e.Request.PathValue("type")

		dbType := slugToDBType(addressType)
		if dbType != "ship_to" && dbType != "install_at" {
			return ErrorToast(e, http.StatusBadRequest, "Invalid address type")
		}

		// Verify project exists
		if _, err := app.FindRecordById("projects", projectID); err != nil {
			return ErrorToast(e, http.StatusNotFound, "Project not found")
		}

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		// Parse the serialized rows from the hidden form field
		parsedJSON := e.Request.FormValue("parsed_rows_json")
		if parsedJSON == "" {
			return ErrorToast(e, http.StatusBadRequest,
				"File data missing. Please re-upload and try again.")
		}

		var parsedRows []map[string]string
		if err := json.Unmarshal([]byte(parsedJSON), &parsedRows); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid parsed data")
		}

		// Commit the import
		importResult, err := services.CommitAddressImport(app, projectID, dbType, parsedRows)
		if err != nil {
			log.Printf("address_import_commit: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		// Render result
		if importResult.Failed > 0 {
			component := templates.AddressImportFailure(
				projectID, addressType, importResult,
			)
			return component.Render(e.Request.Context(), e.Response)
		}

		// Success
		SetToast(e, "success", fmt.Sprintf("%d addresses imported successfully", importResult.Imported))
		component := templates.AddressImportSuccess(
			projectID, addressType, importResult.Imported,
		)
		return component.Render(e.Request.Context(), e.Response)
	}
}

// slugToDBType converts URL slug to database type.
// "ship-to" -> "ship_to", "install-at" -> "install_at"
func slugToDBType(slug string) string {
	switch slug {
	case "ship-to":
		return "ship_to"
	case "install-at":
		return "install_at"
	default:
		return slug
	}
}
