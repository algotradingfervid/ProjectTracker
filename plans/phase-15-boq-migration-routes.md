# Phase 15: BOQ Migration & Route Restructuring

## Overview & Objectives

Migrate all BOQ routes from the current flat structure (`/boq/...`) to project-scoped routes (`/projects/{projectId}/boq/...`). This involves:

1. Updating every BOQ route and handler to accept and validate a `projectId` parameter.
2. Updating all templates to include project context in breadcrumbs and navigation.
3. Adding redirect handlers from old `/boq/...` routes to the new project-scoped URLs.
4. Creating a data migration that runs on startup to auto-create projects for any orphaned BOQs (BOQs without a project relation).
5. Updating sidebar links to use project-scoped URLs.
6. Filtering the BOQ list to show only BOQs belonging to the active project.

---

## Files to Create/Modify

| Action | Path |
|--------|------|
| **Modify** | `main.go` (update all route registrations, add redirects) |
| **Modify** | `collections/setup.go` (add `project` relation field to `boqs` collection) |
| **Create** | `collections/migrate.go` (data migration for orphaned BOQs) |
| **Modify** | `handlers/boq_list.go` (accept projectId, filter by project) |
| **Modify** | `handlers/boq_create.go` (accept projectId, set project on new BOQ) |
| **Modify** | `handlers/boq_edit.go` (accept projectId, validate BOQ belongs to project) |
| **Modify** | `handlers/boq_view.go` (accept projectId, validate BOQ belongs to project) |
| **Modify** | `handlers/boq_delete.go` (accept projectId, validate, update redirect URL) |
| **Modify** | `handlers/export.go` (accept projectId, validate, update redirect URLs) |
| **Modify** | `handlers/items.go` (accept projectId in all item routes) |
| **Modify** | `templates/sidebar.templ` (update BOQ link to use project-scoped URL) |
| **Modify** | `templates/boq_list.templ` (update links, add project breadcrumb) |
| **Modify** | `templates/boq_view.templ` (update links, add project breadcrumb) |
| **Modify** | `templates/boq_edit.templ` (update links, add project breadcrumb) |
| **Modify** | `templates/boq_create.templ` (update form action, add project breadcrumb) |
| **Modify** | `templates/page.templ` (pass projectId to Sidebar) |

---

## Detailed Implementation Steps

### Step 1: Add `project` Relation Field to `boqs` Collection

Update `collections/setup.go` to add a `project` relation to the `boqs` collection. The field is NOT required initially because existing BOQs won't have it yet (the migration handles that).

```go
// In Setup(), after the projects collection is ensured:
projects := ensureCollection(app, "projects", func(c *core.Collection) {
    c.Fields.Add(&core.TextField{Name: "name", Required: true})
    // ... other project fields
})

boqs := ensureCollection(app, "boqs", func(c *core.Collection) {
    c.Fields.Add(&core.TextField{Name: "title", Required: true})
    c.Fields.Add(&core.TextField{Name: "reference_number", Required: false})
    // Add project relation - not required to allow migration
    c.Fields.Add(&core.RelationField{
        Name:         "project",
        Required:     false,
        CollectionId: projects.Id,
        MaxSelect:    1,
    })
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})
```

**Note**: The `ensureCollection` function skips creation if the collection already exists. To add the new `project` field to an existing `boqs` collection, add a separate field-migration step (see Step 2).

### Step 2: Ensure `project` Field Exists on Existing `boqs` Collection

Add a helper that adds the field if it is missing:

```go
// In collections/setup.go or a new collections/migrate_fields.go

// EnsureBoqProjectField adds the 'project' relation field to the boqs
// collection if it doesn't already exist.
func EnsureBoqProjectField(app *pocketbase.PocketBase, projectsColId string) {
    boqsCol, err := app.FindCollectionByNameOrId("boqs")
    if err != nil {
        log.Printf("migrate: boqs collection not found: %v", err)
        return
    }

    // Check if field already exists
    if boqsCol.Fields.GetByName("project") != nil {
        return
    }

    boqsCol.Fields.Add(&core.RelationField{
        Name:         "project",
        Required:     false,
        CollectionId: projectsColId,
        MaxSelect:    1,
    })

    if err := app.Save(boqsCol); err != nil {
        log.Printf("migrate: failed to add project field to boqs: %v", err)
    } else {
        log.Println("migrate: added 'project' field to boqs collection")
    }
}
```

### Step 3: Create Data Migration for Orphaned BOQs

Create `collections/migrate.go`:

```go
package collections

import (
	"log"

	"github.com/pocketbase/pocketbase"
)

// MigrateOrphanedBOQs finds all BOQ records that have no project assigned
// and creates a project for each one, linking them. This is idempotent:
// running it multiple times will not create duplicate projects.
func MigrateOrphanedBOQs(app *pocketbase.PocketBase) error {
	boqsCol, err := app.FindCollectionByNameOrId("boqs")
	if err != nil {
		return err
	}

	projectsCol, err := app.FindCollectionByNameOrId("projects")
	if err != nil {
		return err
	}

	// Find all BOQs where project is empty
	// PocketBase filter: project = "" means the relation field is unset
	orphaned, err := app.FindRecordsByFilter(
		boqsCol,
		"project = ''",
		"created",
		0, 0,
		nil,
	)
	if err != nil {
		log.Printf("migrate: could not query orphaned BOQs: %v", err)
		return err
	}

	if len(orphaned) == 0 {
		log.Println("migrate: no orphaned BOQs found, nothing to do")
		return nil
	}

	log.Printf("migrate: found %d orphaned BOQ(s), creating projects...", len(orphaned))

	for _, boq := range orphaned {
		boqTitle := boq.GetString("title")
		if boqTitle == "" {
			boqTitle = "Untitled Project"
		}

		// Create a new project with the same name as the BOQ title
		projectRecord := core.NewRecord(projectsCol)
		projectRecord.Set("name", boqTitle)

		if err := app.Save(projectRecord); err != nil {
			log.Printf("migrate: failed to create project for BOQ %s: %v", boq.Id, err)
			continue
		}

		// Link the BOQ to the new project
		boq.Set("project", projectRecord.Id)
		if err := app.Save(boq); err != nil {
			log.Printf("migrate: failed to link BOQ %s to project %s: %v",
				boq.Id, projectRecord.Id, err)
			continue
		}

		log.Printf("migrate: BOQ '%s' -> Project '%s' (id=%s)",
			boqTitle, boqTitle, projectRecord.Id)
	}

	return nil
}
```

### Step 4: Run Migration on Startup

Update `main.go` to call the migration after setup:

```go
app.OnServe().BindFunc(func(se *core.ServeEvent) error {
    collections.Setup(app)
    if err := collections.Seed(app); err != nil {
        log.Printf("Warning: seed data failed: %v", err)
    }

    // Ensure the project field exists on boqs collection
    projectsCol, _ := app.FindCollectionByNameOrId("projects")
    if projectsCol != nil {
        collections.EnsureBoqProjectField(app, projectsCol.Id)
    }

    // Migrate orphaned BOQs (idempotent)
    if err := collections.MigrateOrphanedBOQs(app); err != nil {
        log.Printf("Warning: BOQ migration failed: %v", err)
    }

    return se.Next()
})
```

### Step 5: Update All BOQ Routes in `main.go`

Replace all existing BOQ routes with project-scoped versions:

```go
// ── Project-scoped BOQ routes ──

// BOQ creation
se.Router.GET("/projects/{projectId}/boq/create", handlers.HandleBOQCreate(app))
se.Router.POST("/projects/{projectId}/boq", handlers.HandleBOQSave(app))

// BOQ edit mode
se.Router.GET("/projects/{projectId}/boq/{id}/edit", handlers.HandleBOQEdit(app))
se.Router.GET("/projects/{projectId}/boq/{id}/view", handlers.HandleBOQViewMode(app))
se.Router.POST("/projects/{projectId}/boq/{id}/save", handlers.HandleBOQUpdate(app))

// BOQ delete
se.Router.DELETE("/projects/{projectId}/boq/{id}", handlers.HandleBOQDelete(app))

// BOQ export
se.Router.GET("/projects/{projectId}/boq/{id}/export/excel", handlers.HandleBOQExportExcel(app))
se.Router.GET("/projects/{projectId}/boq/{id}/export/pdf", handlers.HandleBOQExportPDF(app))

// BOQ edit - add items
se.Router.POST("/projects/{projectId}/boq/{id}/main-items", handlers.HandleAddMainItem(app))
se.Router.POST("/projects/{projectId}/boq/{id}/main-item/{mainItemId}/subitems", handlers.HandleAddSubItem(app))
se.Router.POST("/projects/{projectId}/boq/{id}/subitem/{subItemId}/subsubitems", handlers.HandleAddSubSubItem(app))

// BOQ edit - delete items
se.Router.DELETE("/projects/{projectId}/boq/{id}/main-item/{itemId}", handlers.HandleDeleteMainItem(app))
se.Router.DELETE("/projects/{projectId}/boq/{id}/subitem/{subItemId}", handlers.HandleDeleteSubItem(app))
se.Router.DELETE("/projects/{projectId}/boq/{id}/subsubitem/{subSubItemId}", handlers.HandleDeleteSubSubItem(app))

// BOQ edit - expand/collapse
se.Router.GET("/projects/{projectId}/boq/{id}/main-item/{itemId}/subitems", handlers.HandleExpandMainItem(app))

// BOQ edit - patch fields
se.Router.PATCH("/projects/{projectId}/boq/{id}/main-item/{itemId}", handlers.HandlePatchMainItem(app))
se.Router.PATCH("/projects/{projectId}/boq/{id}/subitem/{subItemId}", handlers.HandlePatchSubItem(app))
se.Router.PATCH("/projects/{projectId}/boq/{id}/subsubitem/{subSubItemId}", handlers.HandlePatchSubSubItem(app))

// BOQ view (must be after specific routes)
se.Router.GET("/projects/{projectId}/boq/{id}", handlers.HandleBOQView(app))

// BOQ list for a project
se.Router.GET("/projects/{projectId}/boq", handlers.HandleBOQList(app))
```

### Step 6: Add Legacy Redirect Routes

Keep the old routes as redirects for backward compatibility:

```go
// ── Legacy BOQ redirects ──
// These redirect old /boq/... URLs to the new project-scoped URLs.

se.Router.GET("/boq", func(e *core.RequestEvent) error {
    // Redirect to the first available project's BOQ list
    projectsCol, err := app.FindCollectionByNameOrId("projects")
    if err != nil {
        return e.String(http.StatusInternalServerError, "Projects not found")
    }
    projects, err := app.FindRecordsByFilter(projectsCol, "", "-created", 1, 0, nil)
    if err != nil || len(projects) == 0 {
        return e.String(http.StatusNotFound, "No projects found")
    }
    return e.Redirect(http.StatusFound,
        fmt.Sprintf("/projects/%s/boq", projects[0].Id))
})

se.Router.GET("/boq/{id}", func(e *core.RequestEvent) error {
    boqID := e.Request.PathValue("id")
    boq, err := app.FindRecordById("boqs", boqID)
    if err != nil {
        return e.String(http.StatusNotFound, "BOQ not found")
    }
    projectID := boq.GetString("project")
    if projectID == "" {
        return e.String(http.StatusNotFound, "BOQ has no project")
    }
    return e.Redirect(http.StatusFound,
        fmt.Sprintf("/projects/%s/boq/%s", projectID, boqID))
})

// Redirect home to first project's BOQ list
se.Router.GET("/", func(e *core.RequestEvent) error {
    projectsCol, _ := app.FindCollectionByNameOrId("projects")
    if projectsCol != nil {
        projects, _ := app.FindRecordsByFilter(projectsCol, "", "-created", 1, 0, nil)
        if len(projects) > 0 {
            return e.Redirect(http.StatusFound,
                fmt.Sprintf("/projects/%s/boq", projects[0].Id))
        }
    }
    return e.Redirect(http.StatusFound, "/boq")
})
```

### Step 7: Update Handlers to Accept and Validate `projectId`

Each handler needs to:
1. Extract `projectId` from the URL path.
2. Validate the project exists.
3. When loading a BOQ, verify it belongs to the project.
4. Pass `projectId` to templates.

Example update for `handlers/boq_list.go`:

```go
func HandleBOQList(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectID := e.Request.PathValue("projectId")
        if projectID == "" {
            return e.String(http.StatusBadRequest, "Missing project ID")
        }

        // Validate project exists
        project, err := app.FindRecordById("projects", projectID)
        if err != nil {
            return e.String(http.StatusNotFound, "Project not found")
        }
        projectName := project.GetString("name")

        boqsCol, err := app.FindCollectionByNameOrId("boqs")
        if err != nil {
            return e.String(500, "Internal error")
        }

        // Filter BOQs by project
        records, err := app.FindRecordsByFilter(
            boqsCol,
            "project = {:projectId}",
            "-created",
            0, 0,
            map[string]any{"projectId": projectID},
        )
        if err != nil {
            records = nil
        }

        // ... rest of existing logic to build items ...

        data := templates.BOQListData{
            ProjectID:   projectID,
            ProjectName: projectName,
            Items:       items,
            // ... other fields
        }

        // ... render with HX-Request check
    }
}
```

Example validation helper (add to a new `handlers/helpers.go` or existing handler file):

```go
// validateBOQBelongsToProject checks that the BOQ with the given ID
// has a project relation matching projectID. Returns the BOQ record or an error.
func validateBOQBelongsToProject(app *pocketbase.PocketBase, boqID, projectID string) (*core.Record, error) {
    boq, err := app.FindRecordById("boqs", boqID)
    if err != nil {
        return nil, fmt.Errorf("BOQ not found: %w", err)
    }

    if boq.GetString("project") != projectID {
        return nil, fmt.Errorf("BOQ %s does not belong to project %s", boqID, projectID)
    }

    return boq, nil
}
```

Apply this pattern to `boq_view.go`, `boq_edit.go`, `boq_delete.go`, `export.go`, and `items.go`.

### Step 8: Update `HandleBOQDelete` Redirect

```go
// In HandleBOQDelete, update the redirect URL:
if e.Request.Header.Get("HX-Request") == "true" {
    e.Response.Header().Set("HX-Redirect",
        fmt.Sprintf("/projects/%s/boq", projectID))
    return e.String(http.StatusOK, "")
}
return e.Redirect(http.StatusFound,
    fmt.Sprintf("/projects/%s/boq", projectID))
```

### Step 9: Update Template Data Structs

Add `ProjectID` and `ProjectName` fields to template data structs that need them:

```go
// In templates/boq_list.templ
type BOQListData struct {
    ProjectID        string   // NEW
    ProjectName      string   // NEW
    Items            []BOQListItem
    TotalBOQs        int
    SumQuoted        string
    SumBudgeted      string
    Margin           string
    IsPositiveMargin bool
}
```

### Step 10: Update Template Links

All template links referencing `/boq/...` must be updated to `/projects/{ProjectID}/boq/...`.

Example in `boq_list.templ`:

```go
// Before:
hx-get={ "/boq/" + item.ID }
// After:
hx-get={ fmt.Sprintf("/projects/%s/boq/%s", data.ProjectID, item.ID) }

// Before:
<a href="/boq/create" ...>
// After:
<a href={ templ.SafeURL(fmt.Sprintf("/projects/%s/boq/create", data.ProjectID)) } ...>
```

### Step 11: Update Sidebar

Update `templates/sidebar.templ` to accept a `projectId` parameter and use it in the BOQ link:

```go
templ Sidebar(projectID string) {
    // ...
    <a href={ templ.SafeURL(fmt.Sprintf("/projects/%s/boq", projectID)) }
       hx-get={ fmt.Sprintf("/projects/%s/boq", projectID) }
       hx-target="#main-content"
       hx-push-url="true"
       class="flex items-center"
       style="gap: 8px; padding: 10px 0;">
        <div style="width: 6px; height: 6px; background-color: var(--terracotta);"></div>
        <span style="...">BOQ</span>
    </a>
    // ...
}
```

Update `templates/page.templ` to pass `projectID`:

```go
templ PageShell(title string, projectID string) {
    @Layout(title) {
        <div class="flex flex-col h-screen">
            @TopHeader()
            <div class="flex flex-1 overflow-hidden">
                @Sidebar(projectID)
                <main id="main-content" class="flex-1 overflow-y-auto" style="...">
                    { children... }
                </main>
            </div>
        </div>
    }
}
```

### Step 12: Add Breadcrumbs

Add a breadcrumb component to BOQ pages:

```go
templ BOQBreadcrumb(projectID string, projectName string, extra ...string) {
    <nav class="flex items-center gap-2 mb-4"
         style="font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
        <a href={ templ.SafeURL(fmt.Sprintf("/projects/%s", projectID)) }
           style="color: var(--text-secondary); text-decoration: none;">
            { projectName }
        </a>
        <span>/</span>
        <a href={ templ.SafeURL(fmt.Sprintf("/projects/%s/boq", projectID)) }
           style="color: var(--text-secondary); text-decoration: none;">
            BOQ
        </a>
        for _, crumb := range extra {
            <span>/</span>
            <span style="color: var(--text-primary);">{ crumb }</span>
        }
    </nav>
}
```

---

## Dependencies on Other Phases

- **Phase 10 (assumed)**: The `projects` collection must already be defined in `collections/setup.go`. If not, this phase needs to create it.
- **Phase 13**: The address export route (`/projects/{projectId}/addresses/...`) already uses the project-scoped pattern, so no conflict.
- **Phase 14**: Address delete routes also use the project-scoped pattern.
- No dependency on Phase 16 (project settings).

---

## Testing / Verification Steps

1. **Data migration**:
   - Start with existing BOQs that have no `project` field.
   - Run the application. Check logs for migration messages.
   - Verify in PocketBase admin that each BOQ now has a `project` relation set.
   - Verify a project was created for each orphaned BOQ with a matching name.
   - Restart the application. Verify migration is idempotent (no duplicate projects created).

2. **New routes**:
   - Navigate to `/projects/{projectId}/boq` and verify the BOQ list shows only BOQs for that project.
   - Create a new BOQ at `/projects/{projectId}/boq/create` and verify it is linked to the project.
   - View, edit, and delete BOQs using the new URLs.
   - Export BOQs (Excel and PDF) using the new URLs.

3. **Legacy redirects**:
   - Navigate to `/boq` and verify redirect to `/projects/{someProjectId}/boq`.
   - Navigate to `/boq/{id}` and verify redirect to `/projects/{projectId}/boq/{id}`.
   - Navigate to `/` and verify redirect to the first project's BOQ list.

4. **Validation**:
   - Try accessing `/projects/{projectA}/boq/{boqBelongingToProjectB}` and verify a 403 or 404 error.
   - Try accessing `/projects/nonexistent/boq` and verify a 404 error.

5. **Sidebar links**:
   - Click the BOQ link in the sidebar and verify it navigates to the correct project-scoped URL.
   - HTMX partial navigation (via `hx-get`) should work correctly.

6. **Breadcrumbs**:
   - On BOQ list, view, edit, and create pages, verify breadcrumbs show: `{ProjectName} / BOQ / {PageTitle}`.
   - Breadcrumb links navigate correctly.

7. **Regression**:
   - All existing BOQ operations (CRUD, export, item add/delete/patch) work with the new URLs.
   - No 404 errors for any previously working functionality.

---

## Acceptance Criteria

- [ ] All BOQ routes are scoped under `/projects/{projectId}/boq/...`.
- [ ] Every BOQ handler extracts and validates `projectId`, confirming the BOQ belongs to the project.
- [ ] Old `/boq/...` routes return 302 redirects to the corresponding project-scoped URLs.
- [ ] The `/` route redirects to the most recently created project's BOQ list.
- [ ] Data migration runs on startup and creates a project for each BOQ that has no project set.
- [ ] Migration is idempotent: running multiple times does not create duplicate projects.
- [ ] Migration uses the BOQ title as the project name.
- [ ] BOQ list page filters to show only BOQs belonging to the active project.
- [ ] Creating a new BOQ automatically sets the `project` relation to the current project.
- [ ] Sidebar BOQ link uses the project-scoped URL.
- [ ] Breadcrumbs on all BOQ pages show project name and navigation hierarchy.
- [ ] Templates pass `projectId` through all links, form actions, and HTMX attributes.
- [ ] All BOQ item operations (add, delete, patch, expand) work with the new route structure.
- [ ] BOQ export (Excel and PDF) works with the new route structure.
- [ ] No broken links or 404 errors when navigating through the application.
