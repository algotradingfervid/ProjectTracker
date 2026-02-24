package handlers

import (
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func HandleProjectDelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("id")
		if projectID == "" {
			return e.String(http.StatusBadRequest, "Missing project ID")
		}

		projectRecord, err := app.FindRecordById("projects", projectID)
		if err != nil {
			log.Printf("project_delete: could not find project %s: %v", projectID, err)
			return e.String(http.StatusNotFound, "Project not found")
		}

		deleteBoqs := e.Request.URL.Query().Get("delete_boqs") == "true"

		boqsCol, err := app.FindCollectionByNameOrId("boqs")
		if err != nil {
			log.Printf("project_delete: could not find boqs collection: %v", err)
			return e.String(http.StatusInternalServerError, "Internal error")
		}

		boqs, err := app.FindRecordsByFilter(
			boqsCol,
			"project = {:projectId}",
			"", 0, 0,
			map[string]any{"projectId": projectID},
		)
		if err != nil {
			boqs = nil
		}

		for _, boq := range boqs {
			if deleteBoqs {
				if err := app.Delete(boq); err != nil {
					log.Printf("project_delete: failed to delete BOQ %s: %v", boq.Id, err)
				}
			} else {
				boq.Set("project", "")
				if err := app.Save(boq); err != nil {
					log.Printf("project_delete: failed to unlink BOQ %s: %v", boq.Id, err)
				}
			}
		}

		if err := app.Delete(projectRecord); err != nil {
			log.Printf("project_delete: failed to delete project %s: %v", projectID, err)
			return e.String(http.StatusInternalServerError, "Failed to delete project")
		}

		log.Printf("project_delete: deleted project %s (delete_boqs=%v, boq_count=%d)\n",
			projectID, deleteBoqs, len(boqs))

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", "/projects")
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, "/projects")
	}
}
