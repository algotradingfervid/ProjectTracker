package collections

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// MigrateOrphanBOQsToProjects finds all BOQ records that have no project
// assigned and creates a project for each one, linking them together.
// Safe to call on every startup -- returns early if nothing to migrate.
func MigrateOrphanBOQsToProjects(app *pocketbase.PocketBase) error {
	boqsCol, err := app.FindCollectionByNameOrId("boqs")
	if err != nil {
		return fmt.Errorf("migrate: could not find boqs collection: %w", err)
	}

	projectsCol, err := app.FindCollectionByNameOrId("projects")
	if err != nil {
		return fmt.Errorf("migrate: could not find projects collection: %w", err)
	}

	orphanBOQs, err := app.FindRecordsByFilter(
		boqsCol,
		"project = ''",
		"",
		0,
		0,
		nil,
	)
	if err != nil {
		return fmt.Errorf("migrate: could not query orphan BOQs: %w", err)
	}

	if len(orphanBOQs) == 0 {
		return nil
	}

	log.Printf("migrate: found %d orphan BOQ(s) without a project -- creating projects...\n", len(orphanBOQs))

	for _, boq := range orphanBOQs {
		boqTitle := boq.GetString("title")
		boqRef := boq.GetString("reference_number")

		projectRecord := core.NewRecord(projectsCol)
		projectRecord.Set("name", boqTitle)
		projectRecord.Set("client_name", "")
		projectRecord.Set("reference_number", boqRef)
		projectRecord.Set("status", "active")
		projectRecord.Set("ship_to_equals_install_at", true)

		if err := app.Save(projectRecord); err != nil {
			log.Printf("migrate: failed to create project for BOQ %q (%s): %v\n", boqTitle, boq.Id, err)
			continue
		}

		boq.Set("project", projectRecord.Id)
		if err := app.Save(boq); err != nil {
			log.Printf("migrate: failed to link BOQ %s to project %s: %v\n", boq.Id, projectRecord.Id, err)
			continue
		}

		log.Printf("migrate: BOQ %q -> Project %q (%s)\n", boqTitle, projectRecord.Get("name"), projectRecord.Id)
	}

	log.Println("migrate: orphan BOQ migration complete.")
	return nil
}
