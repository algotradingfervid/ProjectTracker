# Phase 4: Project Templates

## Overview & Objectives

Create the template layer for the new **Project** entity. This phase builds four templ files that let users list, create, edit, and configure projects. All templates follow the established design language: `--bg-card` card surfaces, Space Grotesk headers, Inter body text, inline `style=` attributes matching `boq_list.templ` / `boq_create.templ` / `boq_edit.templ` patterns, and HTMX for navigation with Alpine.js for client-side interactivity.

### Goals

1. `project_list.templ` -- Card-grid listing of all projects with status badges and quick actions.
2. `project_create.templ` -- Form for creating a new project (Name, Client, Reference, Status).
3. `project_edit.templ` -- Edit form with all project fields including the "Ship To = Install At" toggle.
4. `project_settings.templ` -- Per-project settings page for configuring required address fields by type.

---

## Files to Create / Modify

| Action | Path | Purpose |
|--------|------|---------|
| **Create** | `templates/project_list.templ` | Card grid + stats row |
| **Create** | `templates/project_create.templ` | New project form |
| **Create** | `templates/project_edit.templ` | Edit project form |
| **Create** | `templates/project_settings.templ` | Required-field config per address type |
| **Create** | `handlers/project_list.go` | Handler for GET /projects |
| **Create** | `handlers/project_create.go` | Handlers for GET /projects/create, POST /projects |
| **Create** | `handlers/project_edit.go` | Handlers for GET /projects/{id}/edit, POST /projects/{id}/save |
| **Create** | `handlers/project_settings.go` | Handler for GET/POST /projects/{id}/settings |
| **Modify** | `collections/setup.go` | Add `projects` and `addresses` collections |
| **Modify** | `main.go` | Register new routes |

---

## Detailed Implementation Steps

### Step 1: Add PocketBase Collections

Add to `collections/setup.go` inside the `Setup` function, **before** the `boqs` collection block so that the `projects` collection ID is available for the relation field on `boqs`.

```go
// --- projects collection ---
projects := ensureCollection(app, "projects", func(c *core.Collection) {
    c.Fields.Add(&core.TextField{Name: "name", Required: true})
    c.Fields.Add(&core.TextField{Name: "client", Required: false})
    c.Fields.Add(&core.TextField{Name: "reference", Required: false})
    c.Fields.Add(&core.SelectField{
        Name:      "status",
        Required:  true,
        Values:    []string{"draft", "active", "completed", "archived"},
        MaxSelect: 1,
    })
    c.Fields.Add(&core.BoolField{Name: "ship_to_is_install_at"})
    c.Fields.Add(&core.JSONField{Name: "required_address_fields"}) // stores per-type config
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})

// --- addresses collection ---
ensureCollection(app, "addresses", func(c *core.Collection) {
    c.Fields.Add(&core.RelationField{
        Name:          "project",
        Required:      true,
        CollectionId:  projects.Id,
        CascadeDelete: true,
        MaxSelect:     1,
    })
    c.Fields.Add(&core.SelectField{
        Name:      "type",
        Required:  true,
        Values:    []string{"bill_from", "ship_from", "bill_to", "ship_to", "install_at"},
        MaxSelect: 1,
    })
    c.Fields.Add(&core.TextField{Name: "company_name", Required: false})
    c.Fields.Add(&core.TextField{Name: "contact_person", Required: false})
    c.Fields.Add(&core.TextField{Name: "phone", Required: false})
    c.Fields.Add(&core.TextField{Name: "email", Required: false})
    c.Fields.Add(&core.TextField{Name: "address_line1", Required: false})
    c.Fields.Add(&core.TextField{Name: "address_line2", Required: false})
    c.Fields.Add(&core.TextField{Name: "city", Required: false})
    c.Fields.Add(&core.TextField{Name: "state", Required: false})
    c.Fields.Add(&core.TextField{Name: "pincode", Required: false})
    c.Fields.Add(&core.TextField{Name: "gst_number", Required: false})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})
```

Also add a `project` relation field to the existing `boqs` collection so BOQs belong to a project:

```go
// After boqs collection is created, add optional project relation
// (optional so existing data is not broken)
boqs.Fields.Add(&core.RelationField{
    Name:         "project",
    Required:     false,
    CollectionId: projects.Id,
    MaxSelect:    1,
})
```

### Step 2: Data Structs (top of each templ file)

These Go structs live at the top of the respective templ files, following the same pattern as `BOQListItem` / `BOQListData` in `boq_list.templ`.

```go
// templates/project_list.templ
package templates

type ProjectListItem struct {
    ID           string
    Name         string
    Client       string
    Reference    string
    Status       string // "draft" | "active" | "completed" | "archived"
    BOQCount     int
    AddressCount int
    CreatedDate  string
}

type ProjectListData struct {
    Items        []ProjectListItem
    TotalCount   int
    ActiveCount  int
    DraftCount   int
}
```

```go
// templates/project_create.templ
package templates

type ProjectCreateData struct {
    Name      string
    Client    string
    Reference string
    Status    string
    Errors    map[string]string
}
```

```go
// templates/project_edit.templ
package templates

type ProjectEditData struct {
    ID                 string
    Name               string
    Client             string
    Reference          string
    Status             string
    ShipToIsInstallAt  bool
    CreatedDate        string
    Errors             map[string]string
}
```

### Step 3: `project_list.templ` -- Card Grid Layout

This follows the same Content/Page split pattern as `BOQListContent` / `BOQListPage`.

```templ
templ ProjectListContent(data ProjectListData) {
    <!-- Page Header -->
    <div class="flex justify-between items-center">
        <div>
            <h1
                style="font-family: 'Space Grotesk', sans-serif; font-size: 36px; font-weight: 700; color: var(--text-primary); margin: 0;"
            >
                Projects
            </h1>
            <p
                style="font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-secondary); margin-top: 8px;"
            >
                Manage all your projects and their BOQs
            </p>
        </div>
        <div class="flex items-center" style="gap: 12px;">
            <!-- Search -->
            <div
                class="flex items-center"
                style="background-color: var(--bg-card); padding: 10px 14px; gap: 10px;"
            >
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"></circle><path d="m21 21-4.3-4.3"></path></svg>
                <span style="font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-muted);">Search projects...</span>
            </div>
            <!-- New Project -->
            <a
                href="/projects/create"
                hx-get="/projects/create"
                hx-target="#main-content"
                hx-push-url="true"
                class="flex items-center hover:opacity-90"
                style="background-color: var(--bg-sidebar); padding: 10px 16px; gap: 8px; text-decoration: none;"
            >
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--text-light)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"></path><path d="M12 5v14"></path></svg>
                <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-light);">NEW PROJECT</span>
            </a>
        </div>
    </div>

    <!-- Stats Cards Row -->
    <div class="flex" style="gap: 20px; margin-top: 32px;">
        <div class="flex-1" style="background-color: var(--bg-card); padding: 24px;">
            <div style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">
                TOTAL PROJECTS
            </div>
            <div style="font-family: 'Space Grotesk', sans-serif; font-size: 32px; font-weight: 700; color: var(--text-primary); margin-top: 12px;">
                { strconv.Itoa(data.TotalCount) }
            </div>
        </div>
        <div class="flex-1" style="background-color: var(--bg-card); padding: 24px;">
            <div style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">
                ACTIVE
            </div>
            <div style="font-family: 'Space Grotesk', sans-serif; font-size: 32px; font-weight: 700; color: var(--success); margin-top: 12px;">
                { strconv.Itoa(data.ActiveCount) }
            </div>
        </div>
        <div class="flex-1" style="background-color: var(--bg-card); padding: 24px;">
            <div style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">
                DRAFTS
            </div>
            <div style="font-family: 'Space Grotesk', sans-serif; font-size: 32px; font-weight: 700; color: var(--terracotta); margin-top: 12px;">
                { strconv.Itoa(data.DraftCount) }
            </div>
        </div>
    </div>

    <!-- Project Cards Grid -->
    <div class="grid grid-cols-3" style="gap: 24px; margin-top: 32px;">
        if len(data.Items) == 0 {
            <div class="col-span-3 flex justify-center items-center" style="padding: 64px 0; color: var(--text-muted); font-family: 'Inter', sans-serif; font-size: 14px;">
                No projects found. Create your first project to get started.
            </div>
        } else {
            for _, item := range data.Items {
                @ProjectCard(item)
            }
        }
    </div>
}

templ ProjectCard(item ProjectListItem) {
    <div
        style="background-color: var(--bg-card); display: flex; flex-direction: column;"
    >
        <!-- Card Header -->
        <div style="padding: 20px 24px; border-bottom: 1px solid var(--border-light);">
            <div class="flex items-start justify-between">
                <div style="flex: 1; min-width: 0;">
                    <h3
                        style="font-family: 'Space Grotesk', sans-serif; font-size: 16px; font-weight: 700; color: var(--text-primary); margin: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;"
                    >
                        { item.Name }
                    </h3>
                    if item.Client != "" {
                        <p style="font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary); margin-top: 4px;">
                            { item.Client }
                        </p>
                    }
                </div>
                <!-- Status Badge -->
                @ProjectStatusBadge(item.Status)
            </div>
        </div>

        <!-- Card Body -->
        <div style="padding: 16px 24px; flex: 1;">
            <!-- Meta Row -->
            <div class="flex" style="gap: 24px;">
                <div>
                    <div style="font-family: 'Space Grotesk', sans-serif; font-size: 10px; font-weight: 600; letter-spacing: 1px; color: var(--text-muted);">
                        BOQS
                    </div>
                    <div style="font-family: 'Space Grotesk', sans-serif; font-size: 20px; font-weight: 700; color: var(--text-primary); margin-top: 4px;">
                        { strconv.Itoa(item.BOQCount) }
                    </div>
                </div>
                <div>
                    <div style="font-family: 'Space Grotesk', sans-serif; font-size: 10px; font-weight: 600; letter-spacing: 1px; color: var(--text-muted);">
                        ADDRESSES
                    </div>
                    <div style="font-family: 'Space Grotesk', sans-serif; font-size: 20px; font-weight: 700; color: var(--text-primary); margin-top: 4px;">
                        { strconv.Itoa(item.AddressCount) }
                    </div>
                </div>
            </div>
            if item.Reference != "" {
                <div style="margin-top: 12px; font-family: 'Inter', sans-serif; font-size: 12px; color: var(--text-muted);">
                    Ref: { item.Reference }
                </div>
            }
            <div style="margin-top: 4px; font-family: 'Inter', sans-serif; font-size: 12px; color: var(--text-muted);">
                Created { item.CreatedDate }
            </div>
        </div>

        <!-- Card Actions -->
        <div class="flex" style="border-top: 1px solid var(--border-light);">
            <!-- View -->
            <a
                hx-get={ "/projects/" + item.ID }
                hx-target="#main-content"
                hx-push-url="true"
                class="flex-1 flex items-center justify-center"
                style="padding: 12px; gap: 6px; cursor: pointer; text-decoration: none; border-right: 1px solid var(--border-light);"
            >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--text-secondary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0"></path><circle cx="12" cy="12" r="3"></circle></svg>
                <span style="font-family: 'Space Grotesk', sans-serif; font-size: 10px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">VIEW</span>
            </a>
            <!-- Edit -->
            <a
                hx-get={ "/projects/" + item.ID + "/edit" }
                hx-target="#main-content"
                hx-push-url="true"
                class="flex-1 flex items-center justify-center"
                style="padding: 12px; gap: 6px; cursor: pointer; text-decoration: none; border-right: 1px solid var(--border-light);"
            >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--text-secondary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z"></path></svg>
                <span style="font-family: 'Space Grotesk', sans-serif; font-size: 10px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">EDIT</span>
            </a>
            <!-- Delete -->
            <button
                hx-delete={ "/projects/" + item.ID }
                hx-target="#main-content"
                hx-push-url="/projects"
                hx-confirm="Are you sure you want to delete this project? All BOQs and addresses will be permanently removed."
                class="flex-1 flex items-center justify-center"
                style="padding: 12px; gap: 6px; cursor: pointer; background: none; border: none;"
            >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--error)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"></path><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"></path><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"></path></svg>
                <span style="font-family: 'Space Grotesk', sans-serif; font-size: 10px; font-weight: 600; letter-spacing: 1px; color: var(--error);">DELETE</span>
            </button>
        </div>
    </div>
}

templ ProjectStatusBadge(status string) {
    switch status {
        case "active":
            <span style="padding: 4px 10px; font-family: 'Space Grotesk', sans-serif; font-size: 10px; font-weight: 600; letter-spacing: 1px; color: var(--success); background-color: #E8F0EB; text-transform: uppercase;">
                ACTIVE
            </span>
        case "draft":
            <span style="padding: 4px 10px; font-family: 'Space Grotesk', sans-serif; font-size: 10px; font-weight: 600; letter-spacing: 1px; color: var(--text-muted); background-color: var(--bg-page); text-transform: uppercase;">
                DRAFT
            </span>
        case "completed":
            <span style="padding: 4px 10px; font-family: 'Space Grotesk', sans-serif; font-size: 10px; font-weight: 600; letter-spacing: 1px; color: var(--terracotta); background-color: #F0DDD7; text-transform: uppercase;">
                COMPLETED
            </span>
        case "archived":
            <span style="padding: 4px 10px; font-family: 'Space Grotesk', sans-serif; font-size: 10px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); background-color: var(--border-light); text-transform: uppercase;">
                ARCHIVED
            </span>
    }
}

templ ProjectListPage(data ProjectListData) {
    @PageShell("Projects -- Fervid Smart Solutions") {
        @ProjectListContent(data)
    }
}
```

### Step 4: `project_create.templ` -- Create Project Form

Follows the same card-with-header pattern as `BOQCreatePage`.

```templ
templ ProjectCreateContent(data ProjectCreateData) {
    <!-- Breadcrumbs -->
    <div class="flex items-center" style="gap: 6px; margin-bottom: 16px;">
        <a
            hx-get="/projects"
            hx-target="#main-content"
            hx-push-url="true"
            style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 500; color: var(--text-secondary); text-decoration: none; cursor: pointer; text-transform: uppercase; letter-spacing: 0.5px;"
        >
            PROJECTS
        </a>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; color: var(--text-muted);">/</span>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; color: var(--terracotta); text-transform: uppercase; letter-spacing: 0.5px;">
            NEW PROJECT
        </span>
    </div>

    <!-- Page Header -->
    <div>
        <h1 style="font-family: 'Space Grotesk', sans-serif; font-size: 32px; font-weight: 700; color: var(--text-primary); margin: 0;">
            Create New Project
        </h1>
        <p style="font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-secondary); margin-top: 8px;">
            Set up a new project to organize BOQs and addresses
        </p>
    </div>

    <!-- Validation Error Banner -->
    if len(data.Errors) > 0 {
        <div style="background-color: #FEE2E2; border: 1px solid #EF4444; padding: 12px 16px; margin-top: 24px;">
            for _, msg := range data.Errors {
                <div style="font-family: 'Inter', sans-serif; font-size: 13px; color: #DC2626;">
                    { msg }
                </div>
            }
        </div>
    }

    <!-- Form -->
    <form method="POST" action="/projects" style="margin-top: 32px;">
        <div style="background-color: var(--bg-card);">
            <!-- Card Header -->
            <div style="background-color: #E2DED6; padding: 16px 24px;">
                <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">
                    PROJECT DETAILS
                </span>
            </div>
            <!-- Card Body -->
            <div style="padding: 24px;">
                <!-- Row 1: Name + Client -->
                <div class="flex" style="gap: 24px; margin-bottom: 16px;">
                    <div class="flex-1">
                        <label for="name" style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); display: block; margin-bottom: 6px;">
                            PROJECT NAME <span style="color: var(--terracotta);">*</span>
                        </label>
                        <input
                            type="text" id="name" name="name" value={ data.Name }
                            placeholder="e.g. Interior Fit-Out -- Block A" required
                            style="width: 100%; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid var(--border-light); border-radius: 0; outline: none; box-sizing: border-box;"
                        />
                    </div>
                    <div class="flex-1">
                        <label for="client" style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); display: block; margin-bottom: 6px;">
                            CLIENT
                        </label>
                        <input
                            type="text" id="client" name="client" value={ data.Client }
                            placeholder="Client company name"
                            style="width: 100%; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid var(--border-light); border-radius: 0; outline: none; box-sizing: border-box;"
                        />
                    </div>
                </div>
                <!-- Row 2: Reference + Status -->
                <div class="flex" style="gap: 24px;">
                    <div style="width: 300px; min-width: 300px;">
                        <label for="reference" style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); display: block; margin-bottom: 6px;">
                            REFERENCE NUMBER
                        </label>
                        <input
                            type="text" id="reference" name="reference" value={ data.Reference }
                            placeholder="e.g. PRJ-001"
                            style="width: 100%; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid var(--border-light); border-radius: 0; outline: none; box-sizing: border-box;"
                        />
                    </div>
                    <div style="width: 220px; min-width: 220px;">
                        <label for="status" style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); display: block; margin-bottom: 6px;">
                            STATUS
                        </label>
                        <select
                            id="status" name="status"
                            style="width: 100%; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid var(--border-light); border-radius: 0; outline: none; box-sizing: border-box; appearance: auto;"
                        >
                            <option value="draft" selected?={ data.Status == "draft" || data.Status == "" }>Draft</option>
                            <option value="active" selected?={ data.Status == "active" }>Active</option>
                            <option value="completed" selected?={ data.Status == "completed" }>Completed</option>
                            <option value="archived" selected?={ data.Status == "archived" }>Archived</option>
                        </select>
                    </div>
                </div>
            </div>
        </div>

        <!-- Action Buttons -->
        <div class="flex justify-end" style="gap: 12px; margin-top: 24px;">
            <a href="/projects"
                hx-get="/projects"
                hx-target="#main-content"
                hx-push-url="true"
                class="flex items-center justify-center"
                style="padding: 12px 24px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; font-weight: 600; letter-spacing: 1px; color: var(--text-primary); background-color: var(--bg-card); border: none; text-decoration: none;"
            >CANCEL</a>
            <button type="submit" class="flex items-center justify-center"
                style="padding: 12px 24px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; font-weight: 600; letter-spacing: 1px; color: var(--text-light); background-color: var(--terracotta); border: none; cursor: pointer;"
            >CREATE PROJECT</button>
        </div>
    </form>
}

templ ProjectCreatePage(data ProjectCreateData) {
    @PageShell("Create Project -- Fervid Smart Solutions") {
        @ProjectCreateContent(data)
    }
}
```

### Step 5: `project_edit.templ` -- Edit Form with Ship To Toggle

The key addition is the "Ship To = Install At" toggle using a DaisyUI-styled checkbox with Alpine.js.

```templ
templ ProjectEditContent(data ProjectEditData) {
    <!-- Breadcrumbs -->
    <div class="flex items-center" style="gap: 6px; margin-bottom: 16px;">
        <a
            hx-get="/projects"
            hx-target="#main-content"
            hx-push-url="true"
            style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 500; color: var(--text-secondary); text-decoration: none; cursor: pointer; text-transform: uppercase; letter-spacing: 0.5px;"
        >
            PROJECTS
        </a>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; color: var(--text-muted);">/</span>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 500; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px;">
            { data.Name }
        </span>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; color: var(--text-muted);">/</span>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; color: var(--terracotta); text-transform: uppercase; letter-spacing: 0.5px;">
            EDIT
        </span>
    </div>

    <!-- Page Header -->
    <div class="flex justify-between items-start">
        <div>
            <h1 style="font-family: 'Space Grotesk', sans-serif; font-size: 32px; font-weight: 700; color: var(--text-primary); margin: 0;">
                Edit Project
            </h1>
            <p style="font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary); margin-top: 8px;">
                Created { data.CreatedDate }
            </p>
        </div>
    </div>

    <!-- Validation Error Banner -->
    if len(data.Errors) > 0 {
        <div style="background-color: #FEE2E2; border: 1px solid #EF4444; padding: 12px 16px; margin-top: 24px;">
            for _, msg := range data.Errors {
                <div style="font-family: 'Inter', sans-serif; font-size: 13px; color: #DC2626;">
                    { msg }
                </div>
            }
        </div>
    }

    <!-- Form -->
    <form
        method="POST"
        action={ templ.SafeURL("/projects/" + data.ID + "/save") }
        x-data={ fmt.Sprintf(`{ shipToIsInstallAt: %t }`, data.ShipToIsInstallAt) }
        style="margin-top: 32px;"
    >
        <!-- Project Details Card -->
        <div style="background-color: var(--bg-card);">
            <div style="background-color: #E2DED6; padding: 16px 24px;">
                <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">
                    PROJECT DETAILS
                </span>
            </div>
            <div style="padding: 24px;">
                <!-- Row 1: Name + Client -->
                <div class="flex" style="gap: 24px; margin-bottom: 16px;">
                    <div class="flex-1">
                        <label for="name" style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); display: block; margin-bottom: 6px;">
                            PROJECT NAME <span style="color: var(--terracotta);">*</span>
                        </label>
                        <input
                            type="text" id="name" name="name" value={ data.Name }
                            placeholder="e.g. Interior Fit-Out -- Block A" required
                            style="width: 100%; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid var(--border-light); border-radius: 0; outline: none; box-sizing: border-box;"
                        />
                    </div>
                    <div class="flex-1">
                        <label for="client" style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); display: block; margin-bottom: 6px;">
                            CLIENT
                        </label>
                        <input
                            type="text" id="client" name="client" value={ data.Client }
                            placeholder="Client company name"
                            style="width: 100%; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid var(--border-light); border-radius: 0; outline: none; box-sizing: border-box;"
                        />
                    </div>
                </div>
                <!-- Row 2: Reference + Status -->
                <div class="flex" style="gap: 24px; margin-bottom: 16px;">
                    <div style="width: 300px; min-width: 300px;">
                        <label for="reference" style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); display: block; margin-bottom: 6px;">
                            REFERENCE NUMBER
                        </label>
                        <input
                            type="text" id="reference" name="reference" value={ data.Reference }
                            placeholder="e.g. PRJ-001"
                            style="width: 100%; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid var(--border-light); border-radius: 0; outline: none; box-sizing: border-box;"
                        />
                    </div>
                    <div style="width: 220px; min-width: 220px;">
                        <label for="status" style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); display: block; margin-bottom: 6px;">
                            STATUS
                        </label>
                        <select
                            id="status" name="status"
                            style="width: 100%; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid var(--border-light); border-radius: 0; outline: none; box-sizing: border-box; appearance: auto;"
                        >
                            <option value="draft" selected?={ data.Status == "draft" }>Draft</option>
                            <option value="active" selected?={ data.Status == "active" }>Active</option>
                            <option value="completed" selected?={ data.Status == "completed" }>Completed</option>
                            <option value="archived" selected?={ data.Status == "archived" }>Archived</option>
                        </select>
                    </div>
                </div>
            </div>
        </div>

        <!-- Address Configuration Card -->
        <div style="background-color: var(--bg-card); margin-top: 24px;">
            <div style="background-color: #E2DED6; padding: 16px 24px;">
                <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">
                    ADDRESS CONFIGURATION
                </span>
            </div>
            <div style="padding: 24px;">
                <!-- Ship To = Install At Toggle -->
                <div class="flex items-center" style="gap: 16px;">
                    <label class="flex items-center cursor-pointer" style="gap: 12px;">
                        <input
                            type="checkbox"
                            name="ship_to_is_install_at"
                            x-model="shipToIsInstallAt"
                            class="toggle toggle-sm"
                            style="--tglbg: var(--terracotta); border-color: var(--border-light);"
                            value="true"
                        />
                        <div>
                            <span style="font-family: 'Space Grotesk', sans-serif; font-size: 13px; font-weight: 600; color: var(--text-primary);">
                                Ship To address is the same as Install At
                            </span>
                            <p style="font-family: 'Inter', sans-serif; font-size: 12px; color: var(--text-muted); margin-top: 2px;">
                                When enabled, the Install At tab will be hidden and Ship To address will be used for installation
                            </p>
                        </div>
                    </label>
                </div>
            </div>
        </div>

        <!-- Action Buttons -->
        <div class="flex justify-end" style="gap: 12px; margin-top: 24px;">
            <a
                hx-get={ "/projects/" + data.ID }
                hx-target="#main-content"
                hx-push-url="true"
                class="flex items-center justify-center"
                style="padding: 12px 24px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; font-weight: 600; letter-spacing: 1px; color: var(--text-primary); background-color: var(--bg-card); border: none; text-decoration: none; cursor: pointer;"
            >CANCEL</a>
            <button type="submit" class="flex items-center justify-center"
                style="padding: 12px 24px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; font-weight: 600; letter-spacing: 1px; color: var(--text-light); background-color: var(--terracotta); border: none; cursor: pointer;"
            >SAVE CHANGES</button>
        </div>
    </form>
}

templ ProjectEditPage(data ProjectEditData) {
    @PageShell("Edit Project -- Fervid Smart Solutions") {
        @ProjectEditContent(data)
    }
}
```

### Step 6: `project_settings.templ` -- Required Fields Configuration

This page allows configuring which address fields are required per address type (Bill From, Ship From, Bill To, Ship To, Install At). Uses Alpine.js for interactive toggles.

```templ
// Data struct
type AddressFieldConfig struct {
    Field    string // e.g. "company_name", "contact_person", etc.
    Label    string // e.g. "Company Name"
    Required bool
}

type AddressTypeConfig struct {
    Type   string               // e.g. "bill_from"
    Label  string               // e.g. "Bill From"
    Fields []AddressFieldConfig
}

type ProjectSettingsData struct {
    ProjectID   string
    ProjectName string
    AddressTypes []AddressTypeConfig
    Errors      map[string]string
}

templ ProjectSettingsContent(data ProjectSettingsData) {
    <!-- Breadcrumbs -->
    <div class="flex items-center" style="gap: 6px; margin-bottom: 16px;">
        <a hx-get="/projects" hx-target="#main-content" hx-push-url="true"
            style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 500; color: var(--text-secondary); text-decoration: none; cursor: pointer; text-transform: uppercase; letter-spacing: 0.5px;">
            PROJECTS
        </a>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; color: var(--text-muted);">/</span>
        <a hx-get={ "/projects/" + data.ProjectID } hx-target="#main-content" hx-push-url="true"
            style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 500; color: var(--text-secondary); text-decoration: none; cursor: pointer; text-transform: uppercase; letter-spacing: 0.5px;">
            { data.ProjectName }
        </a>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; color: var(--text-muted);">/</span>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; color: var(--terracotta); text-transform: uppercase; letter-spacing: 0.5px;">
            SETTINGS
        </span>
    </div>

    <h1 style="font-family: 'Space Grotesk', sans-serif; font-size: 32px; font-weight: 700; color: var(--text-primary); margin: 0;">
        Project Settings
    </h1>
    <p style="font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-secondary); margin-top: 8px;">
        Configure required fields for each address type
    </p>

    <form method="POST" action={ templ.SafeURL("/projects/" + data.ProjectID + "/settings") } style="margin-top: 32px;">
        for _, addrType := range data.AddressTypes {
            <div style="background-color: var(--bg-card); margin-bottom: 24px;">
                <div style="background-color: #E2DED6; padding: 16px 24px;">
                    <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">
                        { addrType.Label }
                    </span>
                </div>
                <div style="padding: 16px 24px;">
                    for _, field := range addrType.Fields {
                        <label class="flex items-center cursor-pointer" style="padding: 8px 0; gap: 12px;">
                            <input
                                type="checkbox"
                                name={ addrType.Type + "." + field.Field }
                                checked?={ field.Required }
                                class="checkbox checkbox-sm"
                                style="border-color: var(--border-light); --chkbg: var(--terracotta); --chkfg: var(--text-light);"
                                value="true"
                            />
                            <span style="font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-primary);">
                                { field.Label }
                            </span>
                        </label>
                    }
                </div>
            </div>
        }

        <div class="flex justify-end" style="gap: 12px;">
            <a
                hx-get={ "/projects/" + data.ProjectID }
                hx-target="#main-content"
                hx-push-url="true"
                style="padding: 12px 24px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; font-weight: 600; letter-spacing: 1px; color: var(--text-primary); background-color: var(--bg-card); text-decoration: none;"
            >CANCEL</a>
            <button type="submit"
                style="padding: 12px 24px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; font-weight: 600; letter-spacing: 1px; color: var(--text-light); background-color: var(--terracotta); border: none; cursor: pointer;"
            >SAVE SETTINGS</button>
        </div>
    </form>
}

templ ProjectSettingsPage(data ProjectSettingsData) {
    @PageShell("Project Settings -- Fervid Smart Solutions") {
        @ProjectSettingsContent(data)
    }
}
```

### Step 7: Handler Skeleton -- `handlers/project_list.go`

```go
package handlers

import (
    "log"

    "github.com/a-h/templ"
    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"

    "projectcreation/templates"
)

func HandleProjectList(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        col, err := app.FindCollectionByNameOrId("projects")
        if err != nil {
            log.Printf("project_list: could not find projects collection: %v", err)
            return e.String(500, "Internal error")
        }

        records, err := app.FindAllRecords(col)
        if err != nil {
            log.Printf("project_list: could not query projects: %v", err)
            return e.String(500, "Internal error")
        }

        addrCol, _ := app.FindCollectionByNameOrId("addresses")
        boqCol, _ := app.FindCollectionByNameOrId("boqs")

        var items []templates.ProjectListItem
        var activeCount, draftCount int

        for _, rec := range records {
            status := rec.GetString("status")
            if status == "active" {
                activeCount++
            } else if status == "draft" {
                draftCount++
            }

            boqCount := 0
            if boqCol != nil {
                boqs, _ := app.FindRecordsByFilter(boqCol, "project = {:pid}", "", 0, 0, map[string]any{"pid": rec.Id})
                boqCount = len(boqs)
            }

            addrCount := 0
            if addrCol != nil {
                addrs, _ := app.FindRecordsByFilter(addrCol, "project = {:pid}", "", 0, 0, map[string]any{"pid": rec.Id})
                addrCount = len(addrs)
            }

            createdDate := "---"
            if dt := rec.GetDateTime("created"); !dt.IsZero() {
                createdDate = dt.Time().Format("02 Jan 2006")
            }

            items = append(items, templates.ProjectListItem{
                ID:           rec.Id,
                Name:         rec.GetString("name"),
                Client:       rec.GetString("client"),
                Reference:    rec.GetString("reference"),
                Status:       status,
                BOQCount:     boqCount,
                AddressCount: addrCount,
                CreatedDate:  createdDate,
            })
        }

        data := templates.ProjectListData{
            Items:       items,
            TotalCount:  len(records),
            ActiveCount: activeCount,
            DraftCount:  draftCount,
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

### Step 8: Route Registration in `main.go`

Add these routes inside the existing `OnServe` block:

```go
// Project routes
se.Router.GET("/projects", handlers.HandleProjectList(app))
se.Router.GET("/projects/create", handlers.HandleProjectCreateForm(app))
se.Router.POST("/projects", handlers.HandleProjectSave(app))
se.Router.GET("/projects/{id}/edit", handlers.HandleProjectEdit(app))
se.Router.POST("/projects/{id}/save", handlers.HandleProjectUpdate(app))
se.Router.DELETE("/projects/{id}", handlers.HandleProjectDelete(app))
se.Router.GET("/projects/{id}/settings", handlers.HandleProjectSettings(app))
se.Router.POST("/projects/{id}/settings", handlers.HandleProjectSettingsSave(app))
se.Router.GET("/projects/{id}", handlers.HandleProjectView(app))
```

---

## Dependencies on Other Phases

| Dependency | Phase | Notes |
|-----------|-------|-------|
| PocketBase `projects` collection | Phase 3 (Data Model) or created inline here | Must exist before handlers run |
| PocketBase `addresses` collection | Phase 3 or created inline here | Needed for address counts |
| `boqs.project` relation field | Phase 3 or added here | Needed for BOQ counts per project |
| None from Phase 5 or 6 | -- | This phase is self-contained for templates |

---

## Testing / Verification Steps

1. **Run `templ generate`** -- confirm all four `.templ` files compile without errors.
2. **Run `go build`** -- confirm handlers compile and link correctly.
3. **Start the server** -- navigate to `/projects`:
   - Verify the stats cards render (Total, Active, Drafts).
   - Verify empty state message appears when no projects exist.
4. **Create a project** at `/projects/create`:
   - Submit with empty name -- verify validation error banner.
   - Submit with valid data -- verify redirect to project list with new card.
5. **Verify project card** shows Name, Client, Status badge (correct color), BOQ count, Address count, Created date.
6. **Click Edit** -- verify edit form pre-fills all fields.
7. **Toggle "Ship To = Install At"** -- verify the toggle state persists after save.
8. **Delete a project** -- verify confirmation dialog, then card removal.
9. **Project Settings** -- verify all 5 address types render with their checkbox fields.
10. **HTMX navigation** -- verify clicking cards / buttons updates `#main-content` without full-page reload.

---

## Acceptance Criteria

- [ ] `/projects` renders a responsive card grid with project data
- [ ] Each card shows: Name, Client, Status badge (color-coded), BOQ count, Address count, Reference, Created date
- [ ] Card action bar has View, Edit, Delete buttons
- [ ] Delete triggers `hx-confirm` dialog before executing
- [ ] `/projects/create` form validates required `name` field server-side
- [ ] `/projects/{id}/edit` pre-populates all fields including the Ship To toggle
- [ ] Ship To = Install At toggle uses DaisyUI `toggle` component styled with `--terracotta`
- [ ] `/projects/{id}/settings` shows checkboxes for all address fields grouped by type
- [ ] All templates follow existing patterns: inline `style=` with CSS variables, Space Grotesk headers, Inter body
- [ ] HTMX partial rendering works (HX-Request header check with Content/Page split)
- [ ] All breadcrumbs use `hx-get` with `hx-target="#main-content"` for SPA-like navigation
