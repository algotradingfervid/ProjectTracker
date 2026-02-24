# Phase 3: Project CRUD Handlers & Routes

## Overview & Objectives

Implement the full CRUD (Create, Read/List, Edit/Update, Delete) handler layer for the `projects` entity. This phase adds four new handler files following the existing pattern established by `handlers/boq_*.go`, registers routes in `main.go`, and provides the Templ template data structures needed to render project pages.

**Goals:**
1. `HandleProjectList` -- list all projects as cards with summary data.
2. `HandleProjectCreate` + `HandleProjectSave` -- render creation form and save new projects.
3. `HandleProjectEdit` + `HandleProjectUpdate` -- render edit form and save changes.
4. `HandleProjectDelete` -- delete project with cascade cleanup (addresses, settings, and optionally BOQs).
5. Register all routes in `main.go`.
6. Define Templ data types for project views.

---

## Files to Create / Modify

| Action | Path | Purpose |
|--------|------|---------|
| Create | `handlers/project_list.go` | List all projects handler |
| Create | `handlers/project_create.go` | Create project form + save handler |
| Create | `handlers/project_edit.go` | Edit project form + update handler |
| Create | `handlers/project_delete.go` | Delete project handler |
| Modify | `main.go` | Register project routes |
| Create | `templates/project_list.templ` | Project list page template (data types + component) |
| Create | `templates/project_create.templ` | Project creation form template |
| Create | `templates/project_edit.templ` | Project edit form template |

---

## Detailed Implementation Steps

### Step 1 -- Define Templ data types

These go in the respective `.templ` files. Listed here for reference:

```go
// templates/project_list.templ (data types section)

// ProjectListItem holds summary data for one project card in the list view.
type ProjectListItem struct {
    ID                    string
    Name                  string
    ClientName            string
    ReferenceNumber       string
    Status                string   // "active", "completed", "on_hold"
    StatusBadgeClass      string   // DaisyUI badge class
    BOQCount              int
    AddressCount          int
    ShipToEqualsInstallAt bool
    CreatedDate           string
}

// ProjectListData holds the full data for the project list page.
type ProjectListData struct {
    Items       []ProjectListItem
    TotalCount  int
}

// ProjectCreateData holds data for the project creation form.
type ProjectCreateData struct {
    Name                  string
    ClientName            string
    ReferenceNumber       string
    Status                string
    ShipToEqualsInstallAt bool
    StatusOptions         []string
    Errors                map[string]string
}

// ProjectEditData holds data for the project edit form.
type ProjectEditData struct {
    ID                    string
    Name                  string
    ClientName            string
    ReferenceNumber       string
    Status                string
    ShipToEqualsInstallAt bool
    StatusOptions         []string
    BOQCount              int
    AddressCount          int
    CreatedDate           string
    Errors                map[string]string
}
```

### Step 2 -- `handlers/project_list.go`

```go
package handlers

import (
    "log"

    "github.com/a-h/templ"
    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"

    "projectcreation/templates"
)

// statusBadgeClass maps project status to a DaisyUI badge CSS class.
func statusBadgeClass(status string) string {
    switch status {
    case "active":
        return "badge-success"
    case "completed":
        return "badge-info"
    case "on_hold":
        return "badge-warning"
    default:
        return "badge-ghost"
    }
}

// HandleProjectList returns a handler that renders the project list page.
func HandleProjectList(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectsCol, err := app.FindCollectionByNameOrId("projects")
        if err != nil {
            log.Printf("project_list: could not find projects collection: %v", err)
            return e.String(500, "Internal error")
        }

        records, err := app.FindAllRecords(projectsCol)
        if err != nil {
            log.Printf("project_list: could not query projects: %v", err)
            return e.String(500, "Internal error")
        }

        boqsCol, err := app.FindCollectionByNameOrId("boqs")
        if err != nil {
            log.Printf("project_list: could not find boqs collection: %v", err)
            return e.String(500, "Internal error")
        }

        addressesCol, err := app.FindCollectionByNameOrId("addresses")
        if err != nil {
            log.Printf("project_list: could not find addresses collection: %v", err)
            // Non-fatal: addresses collection may not exist yet
            addressesCol = nil
        }

        var items []templates.ProjectListItem

        for _, rec := range records {
            projectID := rec.Id

            // Count BOQs for this project
            boqs, err := app.FindRecordsByFilter(
                boqsCol,
                "project = {:projectId}",
                "", 0, 0,
                map[string]any{"projectId": projectID},
            )
            if err != nil {
                log.Printf("project_list: could not count BOQs for project %s: %v", projectID, err)
                boqs = nil
            }

            // Count addresses for this project
            var addressCount int
            if addressesCol != nil {
                addresses, err := app.FindRecordsByFilter(
                    addressesCol,
                    "project = {:projectId}",
                    "", 0, 0,
                    map[string]any{"projectId": projectID},
                )
                if err != nil {
                    log.Printf("project_list: could not count addresses for project %s: %v", projectID, err)
                } else {
                    addressCount = len(addresses)
                }
            }

            createdDate := "—"
            if dt := rec.GetDateTime("created"); !dt.IsZero() {
                createdDate = dt.Time().Format("02 Jan 2006")
            }

            status := rec.GetString("status")

            items = append(items, templates.ProjectListItem{
                ID:                    projectID,
                Name:                  rec.GetString("name"),
                ClientName:            rec.GetString("client_name"),
                ReferenceNumber:       rec.GetString("reference_number"),
                Status:                status,
                StatusBadgeClass:      statusBadgeClass(status),
                BOQCount:              len(boqs),
                AddressCount:          addressCount,
                ShipToEqualsInstallAt: rec.GetBool("ship_to_equals_install_at"),
                CreatedDate:           createdDate,
            })
        }

        data := templates.ProjectListData{
            Items:      items,
            TotalCount: len(records),
        }

        var component templ.Component
        if e.Request.Header.Get("HX-Request") == "true" {
            component = templates.ProjectListContent(data)
        } else {
            component = templates.ProjectListPage(data)
        }
        return component.Render(e.Request.Context(), e.Response)
    }
}
```

### Step 3 -- `handlers/project_create.go`

```go
package handlers

import (
    "log"
    "net/http"
    "strings"

    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"

    "projectcreation/collections"
    "projectcreation/templates"
)

// ProjectStatusOptions is the list of valid project statuses for form dropdowns.
var ProjectStatusOptions = []string{"active", "completed", "on_hold"}

// HandleProjectCreate returns a handler that renders the project creation form.
func HandleProjectCreate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        data := templates.ProjectCreateData{
            Status:                "active", // default
            ShipToEqualsInstallAt: true,     // default
            StatusOptions:         ProjectStatusOptions,
            Errors:                make(map[string]string),
        }
        component := templates.ProjectCreatePage(data)
        return component.Render(e.Request.Context(), e.Response)
    }
}

// HandleProjectSave returns a handler that processes the project creation form.
func HandleProjectSave(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        if err := e.Request.ParseForm(); err != nil {
            log.Printf("project_create: could not parse form: %v", err)
            return e.String(http.StatusBadRequest, "Invalid form data")
        }

        name := strings.TrimSpace(e.Request.FormValue("name"))
        clientName := strings.TrimSpace(e.Request.FormValue("client_name"))
        refNumber := strings.TrimSpace(e.Request.FormValue("reference_number"))
        status := strings.TrimSpace(e.Request.FormValue("status"))
        shipToEqualsInstallAt := e.Request.FormValue("ship_to_equals_install_at") == "on" ||
            e.Request.FormValue("ship_to_equals_install_at") == "true"

        // Validate
        errors := make(map[string]string)
        if name == "" {
            errors["name"] = "Project name is required"
        }

        // Validate status is one of the allowed values
        validStatus := false
        for _, s := range ProjectStatusOptions {
            if status == s {
                validStatus = true
                break
            }
        }
        if !validStatus {
            status = "active" // fallback
        }

        // Check for duplicate name
        if name != "" {
            existing, _ := app.FindRecordsByFilter(
                "projects",
                "name = {:name}",
                "", 1, 0,
                map[string]any{"name": name},
            )
            if len(existing) > 0 {
                errors["name"] = "A project with this name already exists"
            }
        }

        // If validation errors, re-render form with errors
        if len(errors) > 0 {
            data := templates.ProjectCreateData{
                Name:                  name,
                ClientName:            clientName,
                ReferenceNumber:       refNumber,
                Status:                status,
                ShipToEqualsInstallAt: shipToEqualsInstallAt,
                StatusOptions:         ProjectStatusOptions,
                Errors:                errors,
            }
            component := templates.ProjectCreatePage(data)
            return component.Render(e.Request.Context(), e.Response)
        }

        // Create project record
        projectsCol, err := app.FindCollectionByNameOrId("projects")
        if err != nil {
            log.Printf("project_create: could not find projects collection: %v", err)
            return e.String(http.StatusInternalServerError, "Internal error")
        }

        record := core.NewRecord(projectsCol)
        record.Set("name", name)
        record.Set("client_name", clientName)
        record.Set("reference_number", refNumber)
        record.Set("status", status)
        record.Set("ship_to_equals_install_at", shipToEqualsInstallAt)

        if err := app.Save(record); err != nil {
            log.Printf("project_create: could not save project: %v", err)
            return e.String(http.StatusInternalServerError, "Internal error")
        }

        // Create default address settings for this project
        if err := collections.MigrateDefaultAddressSettings(app); err != nil {
            log.Printf("project_create: failed to create default address settings: %v", err)
            // Non-fatal: continue
        }

        // Redirect to project list (or project view page in future)
        if e.Request.Header.Get("HX-Request") == "true" {
            e.Response.Header().Set("HX-Redirect", "/projects")
            return e.String(http.StatusOK, "")
        }
        return e.Redirect(http.StatusFound, "/projects")
    }
}
```

### Step 4 -- `handlers/project_edit.go`

```go
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

// HandleProjectEdit returns a handler that renders the project edit form.
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

        // Count related BOQs
        boqs, _ := app.FindRecordsByFilter(
            "boqs",
            "project = {:projectId}",
            "", 0, 0,
            map[string]any{"projectId": projectID},
        )

        // Count related addresses
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
            component = templates.ProjectEditPage(data)
        }
        return component.Render(e.Request.Context(), e.Response)
    }
}

// HandleProjectUpdate returns a handler that saves project edits.
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

        // Validate
        errors := make(map[string]string)
        if name == "" {
            errors["name"] = "Project name is required"
        }

        // Validate status
        validStatus := false
        for _, s := range ProjectStatusOptions {
            if status == s {
                validStatus = true
                break
            }
        }
        if !validStatus {
            status = record.GetString("status") // keep existing
        }

        // Check for duplicate name (exclude current project)
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
            // Re-render edit form with errors
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
                component = templates.ProjectEditPage(data)
            }
            return component.Render(e.Request.Context(), e.Response)
        }

        // Update project
        record.Set("name", name)
        record.Set("client_name", clientName)
        record.Set("reference_number", refNumber)
        record.Set("status", status)
        record.Set("ship_to_equals_install_at", shipToEqualsInstallAt)

        if err := app.Save(record); err != nil {
            log.Printf("project_update: could not save project %s: %v", projectID, err)
            return e.String(http.StatusInternalServerError, "Failed to save project")
        }

        // Redirect to project list
        if e.Request.Header.Get("HX-Request") == "true" {
            e.Response.Header().Set("HX-Redirect", "/projects")
            return e.String(http.StatusOK, "")
        }
        return e.Redirect(http.StatusFound, "/projects")
    }
}
```

### Step 5 -- `handlers/project_delete.go`

```go
package handlers

import (
    "log"
    "net/http"

    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"
)

// HandleProjectDelete returns a handler that deletes a project and cleans up related data.
//
// Cascade behavior:
// - addresses: auto-deleted via CascadeDelete on the relation field.
// - project_address_settings: auto-deleted via CascadeDelete on the relation field.
// - boqs: NOT auto-deleted (CascadeDelete=false). Instead, we unlink them (set project="")
//   or delete them based on query param ?delete_boqs=true.
func HandleProjectDelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectID := e.Request.PathValue("id")
        if projectID == "" {
            return e.String(http.StatusBadRequest, "Missing project ID")
        }

        // Find the project
        projectRecord, err := app.FindRecordById("projects", projectID)
        if err != nil {
            log.Printf("project_delete: could not find project %s: %v", projectID, err)
            return e.String(http.StatusNotFound, "Project not found")
        }

        // Handle BOQs: either delete them or unlink them
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
            log.Printf("project_delete: could not query BOQs for project %s: %v", projectID, err)
            boqs = nil
        }

        for _, boq := range boqs {
            if deleteBoqs {
                // Delete BOQ and all its items (cascade handles main_boq_items -> sub_items -> sub_sub_items)
                if err := app.Delete(boq); err != nil {
                    log.Printf("project_delete: failed to delete BOQ %s: %v", boq.Id, err)
                }
            } else {
                // Unlink BOQ from project (orphan it)
                boq.Set("project", "")
                if err := app.Save(boq); err != nil {
                    log.Printf("project_delete: failed to unlink BOQ %s: %v", boq.Id, err)
                }
            }
        }

        // Delete the project (cascade handles addresses + project_address_settings)
        if err := app.Delete(projectRecord); err != nil {
            log.Printf("project_delete: failed to delete project %s: %v", projectID, err)
            return e.String(http.StatusInternalServerError, "Failed to delete project")
        }

        log.Printf("project_delete: deleted project %s (delete_boqs=%v, boq_count=%d)\n",
            projectID, deleteBoqs, len(boqs))

        // Redirect to project list
        if e.Request.Header.Get("HX-Request") == "true" {
            e.Response.Header().Set("HX-Redirect", "/projects")
            return e.String(http.StatusOK, "")
        }
        return e.Redirect(http.StatusFound, "/projects")
    }
}
```

### Step 6 -- Register routes in `main.go`

Add the following routes in the `app.OnServe().BindFunc` block, **before** the BOQ routes:

```go
// ── Project routes ───────────────────────────────────────────────
se.Router.GET("/projects", handlers.HandleProjectList(app))
se.Router.GET("/projects/create", handlers.HandleProjectCreate(app))
se.Router.POST("/projects", handlers.HandleProjectSave(app))
se.Router.GET("/projects/{id}/edit", handlers.HandleProjectEdit(app))
se.Router.POST("/projects/{id}/save", handlers.HandleProjectUpdate(app))
se.Router.DELETE("/projects/{id}", handlers.HandleProjectDelete(app))
```

Update the home redirect:

```go
// Redirect home to projects list (was /boq)
se.Router.GET("/", func(e *core.RequestEvent) error {
    return e.Redirect(http.StatusFound, "/projects")
})
```

### Step 7 -- Complete `main.go` with all routes

```go
func main() {
    app := pocketbase.New()

    app.OnServe().BindFunc(func(se *core.ServeEvent) error {
        collections.Setup(app)
        if err := collections.Seed(app); err != nil {
            log.Printf("Warning: seed data failed: %v", err)
        }
        if err := collections.MigrateOrphanBOQsToProjects(app); err != nil {
            log.Printf("Warning: project migration failed: %v", err)
        }
        if err := collections.MigrateDefaultAddressSettings(app); err != nil {
            log.Printf("Warning: address settings migration failed: %v", err)
        }
        return se.Next()
    })

    app.OnServe().BindFunc(func(se *core.ServeEvent) error {
        se.Router.GET("/static/{path...}", apis.Static(os.DirFS("./static"), false))

        // ── Project CRUD ─────────────────────────────────────────
        se.Router.GET("/projects", handlers.HandleProjectList(app))
        se.Router.GET("/projects/create", handlers.HandleProjectCreate(app))
        se.Router.POST("/projects", handlers.HandleProjectSave(app))
        se.Router.GET("/projects/{id}/edit", handlers.HandleProjectEdit(app))
        se.Router.POST("/projects/{id}/save", handlers.HandleProjectUpdate(app))
        se.Router.DELETE("/projects/{id}", handlers.HandleProjectDelete(app))

        // ── BOQ routes (existing, unchanged) ─────────────────────
        se.Router.GET("/boq/create", handlers.HandleBOQCreate(app))
        se.Router.POST("/boq", handlers.HandleBOQSave(app))
        se.Router.GET("/boq/{id}/edit", handlers.HandleBOQEdit(app))
        se.Router.GET("/boq/{id}/view", handlers.HandleBOQViewMode(app))
        se.Router.POST("/boq/{id}/save", handlers.HandleBOQUpdate(app))
        se.Router.DELETE("/boq/{id}", handlers.HandleBOQDelete(app))
        se.Router.GET("/boq/{id}/export/excel", handlers.HandleBOQExportExcel(app))
        se.Router.GET("/boq/{id}/export/pdf", handlers.HandleBOQExportPDF(app))
        se.Router.POST("/boq/{id}/main-items", handlers.HandleAddMainItem(app))
        se.Router.POST("/boq/{id}/main-item/{mainItemId}/subitems", handlers.HandleAddSubItem(app))
        se.Router.POST("/boq/{id}/subitem/{subItemId}/subsubitems", handlers.HandleAddSubSubItem(app))
        se.Router.DELETE("/boq/{id}/main-item/{itemId}", handlers.HandleDeleteMainItem(app))
        se.Router.DELETE("/boq/{id}/subitem/{subItemId}", handlers.HandleDeleteSubItem(app))
        se.Router.DELETE("/boq/{id}/subsubitem/{subSubItemId}", handlers.HandleDeleteSubSubItem(app))
        se.Router.GET("/boq/{id}/main-item/{itemId}/subitems", handlers.HandleExpandMainItem(app))
        se.Router.PATCH("/boq/{id}/main-item/{itemId}", handlers.HandlePatchMainItem(app))
        se.Router.PATCH("/boq/{id}/subitem/{subItemId}", handlers.HandlePatchSubItem(app))
        se.Router.PATCH("/boq/{id}/subsubitem/{subSubItemId}", handlers.HandlePatchSubSubItem(app))
        se.Router.GET("/boq/{id}", handlers.HandleBOQView(app))
        se.Router.GET("/boq", handlers.HandleBOQList(app))

        // ── Home redirect ────────────────────────────────────────
        se.Router.GET("/", func(e *core.RequestEvent) error {
            return e.Redirect(http.StatusFound, "/projects")
        })

        return se.Next()
    })

    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
```

---

## Route Summary

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/projects` | HandleProjectList | List all projects (card layout) |
| GET | `/projects/create` | HandleProjectCreate | Render project creation form |
| POST | `/projects` | HandleProjectSave | Save new project |
| GET | `/projects/{id}/edit` | HandleProjectEdit | Render project edit form |
| POST | `/projects/{id}/save` | HandleProjectUpdate | Save project edits |
| DELETE | `/projects/{id}` | HandleProjectDelete | Delete project (with ?delete_boqs=true option) |

---

## Handler Pattern Summary

All handlers follow the same established pattern from the existing BOQ handlers:

1. **Signature:** `func HandleXxx(app *pocketbase.PocketBase) func(*core.RequestEvent) error`
2. **Path params:** Extracted via `e.Request.PathValue("id")`
3. **Form parsing:** `e.Request.ParseForm()` for POST handlers
4. **Record lookup:** `app.FindRecordById("collection", id)`
5. **Validation:** Build `errors map[string]string`, re-render form if non-empty
6. **Record creation:** `core.NewRecord(col)` + `record.Set(...)` + `app.Save(record)`
7. **HTMX support:** Check `e.Request.Header.Get("HX-Request") == "true"` for partial vs full page
8. **Redirect pattern:** Use `HX-Redirect` header for HTMX, `e.Redirect` for standard requests
9. **Error logging:** `log.Printf("handler_name: message: %v", err)` with consistent prefix

---

## Delete Cascade Behavior

```
DELETE /projects/{id}?delete_boqs=false (default)
    |
    ├── BOQs: project field set to "" (orphaned, preserved)
    ├── addresses: auto-deleted (CascadeDelete=true on addresses.project)
    └── project_address_settings: auto-deleted (CascadeDelete=true)

DELETE /projects/{id}?delete_boqs=true
    |
    ├── BOQs: deleted (cascade deletes main_boq_items -> sub_items -> sub_sub_items)
    ├── addresses: auto-deleted (CascadeDelete=true)
    └── project_address_settings: auto-deleted (CascadeDelete=true)
```

---

## Dependencies on Other Phases

- **Depends on Phase 1:** `projects` collection must exist.
- **Depends on Phase 2:** `addresses` and `project_address_settings` collections must exist (used for counts in list/edit and cascade in delete). Handlers gracefully handle missing collections.
- **Templates:** The `.templ` files referenced here need to be created. The data types are defined above; the actual Templ markup (HTML/HTMX/DaisyUI) will be implemented as a follow-on task.

---

## Testing / Verification Steps

1. **Project creation:**
   - Navigate to `/projects/create`.
   - Submit with empty name -- verify validation error is shown.
   - Submit with a valid name -- verify redirect to `/projects` and project appears in list.
   - Submit with a duplicate name -- verify duplicate error message.
   - Verify default address settings (5 records) are created for the new project.

2. **Project listing:**
   - Navigate to `/projects`.
   - Verify all projects are displayed as cards.
   - Verify BOQ count and address count are accurate.
   - Verify status badge colors (green for active, blue for completed, yellow for on hold).
   - Test HTMX partial rendering: fetch with `HX-Request: true` header and verify only content partial is returned.

3. **Project editing:**
   - Navigate to `/projects/{id}/edit`.
   - Verify form is pre-populated with existing data.
   - Change the name and save -- verify update persists.
   - Try changing to a duplicate name -- verify error.
   - Toggle `ship_to_equals_install_at` -- verify it saves correctly.

4. **Project deletion (default -- preserve BOQs):**
   - Create a project with a BOQ.
   - Delete the project via `DELETE /projects/{id}`.
   - Verify the project is gone from the list.
   - Verify the BOQ still exists but has `project = ""`.
   - Verify all addresses for the project are deleted.

5. **Project deletion (with BOQ cascade):**
   - Create a project with a BOQ and addresses.
   - Delete via `DELETE /projects/{id}?delete_boqs=true`.
   - Verify the project, all BOQs, all BOQ items, and all addresses are deleted.

6. **HTMX integration:**
   - All handlers should return partial content when `HX-Request: true`.
   - All redirect handlers should use `HX-Redirect` header for HTMX requests.
   - Delete handler should trigger `HX-Redirect` to `/projects`.

7. **Compilation check:**
   ```bash
   go build ./...
   ```
   Must compile without errors after all handler files are added.

---

## Acceptance Criteria

- [ ] `handlers/project_list.go` exists and renders project cards with BOQ count, address count, status badge, and created date.
- [ ] `handlers/project_create.go` exists with both form render and save handlers, including validation (required name, duplicate check).
- [ ] `handlers/project_edit.go` exists with both form render and update handlers, including validation.
- [ ] `handlers/project_delete.go` exists with configurable BOQ handling (delete vs orphan).
- [ ] All six routes are registered in `main.go` under `/projects/*`.
- [ ] Home route (`/`) redirects to `/projects` instead of `/boq`.
- [ ] All handlers follow the established pattern: closure over `app`, `PathValue` for IDs, `HX-Request` check for partial rendering, `HX-Redirect` for HTMX redirects.
- [ ] Validation errors re-render the form with error messages and preserve user input.
- [ ] Default address settings are created when a new project is saved.
- [ ] Project deletion cascades to addresses and settings, with BOQs handled per query param.
- [ ] `ProjectStatusOptions` is defined as a shared variable for form dropdowns.
- [ ] Code compiles cleanly with `go build ./...`.
