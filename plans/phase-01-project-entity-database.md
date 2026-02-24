# Phase 1: Project Entity & Database Schema

## Overview & Objectives

Introduce a `projects` collection that serves as the top-level container for BOQs and (in later phases) addresses. Every BOQ will belong to exactly one project. Existing BOQs are automatically migrated into auto-created projects so that no data is lost.

**Goals:**
1. Create the `projects` PocketBase collection with all required fields.
2. Add a `project` relation field to the existing `boqs` collection.
3. Migrate every existing orphan BOQ into its own auto-created project.
4. Keep the existing `ensureCollection` pattern from `collections/setup.go`.

---

## Files to Create / Modify

| Action | Path | Purpose |
|--------|------|---------|
| Modify | `collections/setup.go` | Add `projects` collection, add `project` relation to `boqs` |
| Create | `collections/migrate_projects.go` | One-time migration: wrap orphan BOQs in projects |
| Modify | `main.go` | Call migration function on startup (after Setup) |

---

## Detailed Implementation Steps

### Step 1 -- Add `projects` collection in `collections/setup.go`

Insert the `projects` collection **before** the `boqs` collection so its ID is available for the relation field.

```go
// collections/setup.go  (updated Setup function)

func Setup(app *pocketbase.PocketBase) {
	// ── Projects (new) ───────────────────────────────────────────────
	projects := ensureCollection(app, "projects", func(c *core.Collection) {
		c.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})
		c.Fields.Add(&core.TextField{
			Name:     "client_name",
			Required: false,
		})
		c.Fields.Add(&core.TextField{
			Name:     "reference_number",
			Required: false,
		})
		c.Fields.Add(&core.SelectField{
			Name:      "status",
			Required:  true,
			Values:    []string{"active", "completed", "on_hold"},
			MaxSelect: 1,
		})
		c.Fields.Add(&core.BoolField{
			Name: "ship_to_equals_install_at",
		})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	})

	// ── BOQs (existing -- add project relation) ──────────────────────
	boqs := ensureCollection(app, "boqs", func(c *core.Collection) {
		c.Fields.Add(&core.TextField{Name: "title", Required: true})
		c.Fields.Add(&core.TextField{Name: "reference_number", Required: false})
		c.Fields.Add(&core.RelationField{
			Name:          "project",
			Required:      false,       // false during migration; flip to true later
			CollectionId:  projects.Id,
			CascadeDelete: false,       // deleting project should NOT auto-delete BOQs without confirmation
			MaxSelect:     1,
		})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	})

	// ... rest of existing collections unchanged (main_boq_items, sub_items, sub_sub_items)
	mainBOQItems := ensureCollection(app, "main_boq_items", func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{
			Name:          "boq",
			Required:      true,
			CollectionId:  boqs.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})
		c.Fields.Add(&core.NumberField{Name: "sort_order", Required: true})
		c.Fields.Add(&core.TextField{Name: "description", Required: true})
		c.Fields.Add(&core.NumberField{Name: "qty", Required: true})
		c.Fields.Add(&core.TextField{Name: "uom", Required: true})
		c.Fields.Add(&core.NumberField{Name: "quoted_price", Required: true})
		c.Fields.Add(&core.NumberField{Name: "budgeted_price", Required: false})
		c.Fields.Add(&core.TextField{Name: "hsn_code", Required: false})
		c.Fields.Add(&core.NumberField{Name: "gst_percent", Required: true})
	})

	subItems := ensureCollection(app, "sub_items", func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{
			Name:          "main_item",
			Required:      true,
			CollectionId:  mainBOQItems.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})
		c.Fields.Add(&core.NumberField{Name: "sort_order", Required: true})
		c.Fields.Add(&core.SelectField{
			Name:      "type",
			Required:  true,
			Values:    []string{"product", "service"},
			MaxSelect: 1,
		})
		c.Fields.Add(&core.TextField{Name: "description", Required: true})
		c.Fields.Add(&core.NumberField{Name: "qty_per_unit", Required: true})
		c.Fields.Add(&core.TextField{Name: "uom", Required: true})
		c.Fields.Add(&core.NumberField{Name: "unit_price", Required: true})
		c.Fields.Add(&core.NumberField{Name: "budgeted_price", Required: false})
		c.Fields.Add(&core.TextField{Name: "hsn_code", Required: false})
		c.Fields.Add(&core.NumberField{Name: "gst_percent", Required: true})
	})

	ensureCollection(app, "sub_sub_items", func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{
			Name:          "sub_item",
			Required:      true,
			CollectionId:  subItems.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})
		c.Fields.Add(&core.NumberField{Name: "sort_order", Required: true})
		c.Fields.Add(&core.SelectField{
			Name:      "type",
			Required:  true,
			Values:    []string{"product", "service"},
			MaxSelect: 1,
		})
		c.Fields.Add(&core.TextField{Name: "description", Required: true})
		c.Fields.Add(&core.NumberField{Name: "qty_per_unit", Required: true})
		c.Fields.Add(&core.TextField{Name: "uom", Required: true})
		c.Fields.Add(&core.NumberField{Name: "unit_price", Required: true})
		c.Fields.Add(&core.NumberField{Name: "budgeted_price", Required: false})
		c.Fields.Add(&core.TextField{Name: "hsn_code", Required: false})
		c.Fields.Add(&core.NumberField{Name: "gst_percent", Required: true})
	})
}
```

**Key decisions:**
- `ship_to_equals_install_at` is a `BoolField` -- when `true`, the UI (Phase 2+) will auto-clone Ship To addresses as Install At.
- `status` uses a `SelectField` with three values: `active`, `completed`, `on_hold`.
- The `project` relation on `boqs` is `Required: false` initially to allow the migration to run. A follow-up step can set it to `true` once all BOQs have a project.
- `CascadeDelete: false` on the `boqs.project` relation because project deletion should be handled explicitly in the handler (Phase 3) with a confirmation step that decides what to do with child BOQs.

**Important note on `ensureCollection`:** The existing helper only creates a collection if it does not exist -- it does **not** add new fields to an existing collection. To add the `project` field to an already-existing `boqs` collection, we need an `ensureField` helper or field-migration logic.

### Step 1b -- Add `ensureField` helper to `collections/setup.go`

```go
// ensureField adds a field to an existing collection if it doesn't already exist.
// It is a no-op if the field is already present.
func ensureField(app *pocketbase.PocketBase, collectionName string, field core.Field) {
	col, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		log.Printf("ensureField: collection %q not found, skipping field add.\n", collectionName)
		return
	}

	// Check if field already exists by name
	existingField := col.Fields.GetByName(field.GetName())
	if existingField != nil {
		return // field already present
	}

	col.Fields.Add(field)
	if err := app.Save(col); err != nil {
		log.Printf("ensureField: failed to add field %q to %q: %v\n", field.GetName(), collectionName, err)
	} else {
		log.Printf("ensureField: added field %q to collection %q\n", field.GetName(), collectionName)
	}
}
```

After `ensureCollection` runs for `boqs`, call:

```go
// Add project relation to existing boqs collection (idempotent)
ensureField(app, "boqs", &core.RelationField{
    Name:          "project",
    Required:      false,
    CollectionId:  projects.Id,
    CascadeDelete: false,
    MaxSelect:     1,
})
```

### Step 2 -- Create migration file `collections/migrate_projects.go`

```go
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

	// Find BOQs where project is empty
	orphanBOQs, err := app.FindRecordsByFilter(
		boqsCol,
		"project = ''",
		"",    // no sort
		0,     // no limit
		0,     // no offset
		nil,   // no params
	)
	if err != nil {
		return fmt.Errorf("migrate: could not query orphan BOQs: %w", err)
	}

	if len(orphanBOQs) == 0 {
		return nil // nothing to migrate
	}

	log.Printf("migrate: found %d orphan BOQ(s) without a project -- creating projects...\n", len(orphanBOQs))

	for _, boq := range orphanBOQs {
		boqTitle := boq.GetString("title")
		boqRef := boq.GetString("reference_number")

		// Create a project named after the BOQ
		projectRecord := core.NewRecord(projectsCol)
		projectRecord.Set("name", boqTitle)
		projectRecord.Set("client_name", "")
		projectRecord.Set("reference_number", boqRef)
		projectRecord.Set("status", "active")
		projectRecord.Set("ship_to_equals_install_at", true) // default

		if err := app.Save(projectRecord); err != nil {
			log.Printf("migrate: failed to create project for BOQ %q (%s): %v\n", boqTitle, boq.Id, err)
			continue
		}

		// Link BOQ to the new project
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
```

### Step 3 -- Update `main.go` to call migration

```go
// main.go  (updated OnServe block)
app.OnServe().BindFunc(func(se *core.ServeEvent) error {
    collections.Setup(app)
    if err := collections.Seed(app); err != nil {
        log.Printf("Warning: seed data failed: %v", err)
    }
    // Migrate orphan BOQs into projects (idempotent)
    if err := collections.MigrateOrphanBOQsToProjects(app); err != nil {
        log.Printf("Warning: project migration failed: %v", err)
    }
    return se.Next()
})
```

### Step 4 -- Update seed data (optional enhancement)

Update `collections/seed.go` to create a project and assign the seed BOQ to it:

```go
// In Seed(), after creating the BOQ record:

// Create a project for the seed BOQ
projectsCol, err := app.FindCollectionByNameOrId("projects")
if err != nil {
    return fmt.Errorf("seed: could not find projects collection: %w", err)
}

projectRecord := core.NewRecord(projectsCol)
projectRecord.Set("name", "Interior Fit-Out — Block A")
projectRecord.Set("client_name", "Acme Constructions Pvt. Ltd.")
projectRecord.Set("reference_number", "PO-2025-001")
projectRecord.Set("status", "active")
projectRecord.Set("ship_to_equals_install_at", true)
if err := app.Save(projectRecord); err != nil {
    return fmt.Errorf("seed: save project: %w", err)
}

// Link BOQ to project
boqRecord.Set("project", projectRecord.Id)
if err := app.Save(boqRecord); err != nil {
    return fmt.Errorf("seed: link boq to project: %w", err)
}
```

---

## Project Collection Schema Summary

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `name` | TextField | Yes | Project display name |
| `client_name` | TextField | No | Client / customer name |
| `reference_number` | TextField | No | PO number or external reference |
| `status` | SelectField | Yes | Values: `active`, `completed`, `on_hold` |
| `ship_to_equals_install_at` | BoolField | No | Default `false`; when true, Ship To addresses are auto-cloned as Install At |
| `created` | AutodateField | -- | Set on create |
| `updated` | AutodateField | -- | Set on create + update |

**BOQ collection change:**

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `project` | RelationField | No (initially) | Points to `projects` collection, MaxSelect=1 |

---

## Dependencies on Other Phases

- **None** -- this is the foundational phase.
- Phase 2 depends on the `projects` collection ID for address relations.
- Phase 3 depends on this phase for project CRUD handlers.

---

## Testing / Verification Steps

1. **Fresh database start:**
   - Delete `pb_data/` directory.
   - Run `go run main.go serve`.
   - Verify `projects` collection is created in the PocketBase admin UI (`/_/`).
   - Verify `boqs` collection has a `project` relation field.
   - Verify seed data creates both a project and a BOQ linked to it.

2. **Migration with existing data:**
   - Start with an existing database that has BOQs but no projects.
   - Run the application.
   - Verify that each existing BOQ now has a corresponding project.
   - Verify the project name matches the BOQ title.
   - Verify the project status is `active`.

3. **Idempotency:**
   - Restart the application multiple times.
   - Verify no duplicate projects are created.
   - Verify the migration log says nothing to migrate on subsequent runs.

4. **Field verification via PocketBase Admin:**
   - Navigate to `http://localhost:8090/_/` (PocketBase admin).
   - Open the `projects` collection and verify all fields are present with correct types.
   - Open the `boqs` collection and verify the `project` relation field exists.
   - Create a project manually and verify the status dropdown has the three options.

5. **Data integrity check:**
   ```
   # In PocketBase admin, run API queries:
   GET /api/collections/projects/records
   GET /api/collections/boqs/records?filter=(project='')
   # Second query should return 0 records after migration
   ```

---

## Acceptance Criteria

- [ ] `projects` collection exists with fields: name, client_name, reference_number, status, ship_to_equals_install_at, created, updated.
- [ ] `boqs` collection has a `project` relation field pointing to `projects`.
- [ ] All pre-existing BOQs are migrated into auto-created projects (one project per BOQ).
- [ ] Migration is idempotent -- running the app again does not create duplicates.
- [ ] Seed data creates both a project and a BOQ, linked together.
- [ ] `ensureField` helper is available for future use when adding fields to existing collections.
- [ ] Application starts without errors on both fresh and existing databases.
