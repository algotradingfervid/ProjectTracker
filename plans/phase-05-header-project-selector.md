# Phase 5: Header Project Selector

## Overview & Objectives

Transform the static project name in the header into a fully functional **project selector dropdown**. When a user selects a project, it becomes the "active project" context for the entire application -- the sidebar updates, routes become project-scoped, and all handlers gain awareness of which project is selected.

### Goals

1. Replace the hard-coded "Interior Fit-Out -- Block A" text in `header.templ` with a dynamic dropdown.
2. Dropdown lists all projects plus an "All Projects" option at the bottom.
3. Selecting a project stores the active project ID (via cookie) and triggers a page context switch.
4. Display the active project name in the header when one is selected.
5. Modify the handler pattern to extract and pass the active project ID.
6. "All Projects" navigates to `/projects` list page.

---

## Files to Create / Modify

| Action | Path | Purpose |
|--------|------|---------|
| **Modify** | `templates/header.templ` | Replace static text with Alpine.js dropdown |
| **Modify** | `templates/page.templ` | Update `PageShell` to accept and pass active project |
| **Create** | `handlers/middleware.go` | Middleware to extract active project from cookie |
| **Create** | `handlers/project_switch.go` | POST /projects/{id}/activate endpoint |
| **Modify** | `main.go` | Register new routes, apply middleware |
| **Modify** | `templates/layout.templ` | No changes needed (Alpine.js already loaded) |

---

## Detailed Implementation Steps

### Step 1: Define Active Project Context Types

Create a shared type that all templates can use to know about the active project.

Add to a new file or at the top of `header.templ`:

```go
// templates/header.templ (or a shared types file)
package templates

type ActiveProject struct {
    ID   string
    Name string
}

type ProjectSelectorItem struct {
    ID       string
    Name     string
    Client   string
    IsActive bool
}

type HeaderData struct {
    ActiveProject *ActiveProject          // nil when no project selected
    Projects      []ProjectSelectorItem
}
```

### Step 2: Modify `templates/header.templ`

Replace the current static header with a parameterized version that accepts `HeaderData` and renders an Alpine.js dropdown.

```templ
package templates

type ActiveProject struct {
    ID   string
    Name string
}

type ProjectSelectorItem struct {
    ID       string
    Name     string
    Client   string
    IsActive bool
}

type HeaderData struct {
    ActiveProject *ActiveProject
    Projects      []ProjectSelectorItem
}

templ TopHeader() {
    @TopHeaderWithProject(HeaderData{})
}

templ TopHeaderWithProject(data HeaderData) {
    <header class="w-full h-14 flex items-center justify-between px-6" style="background-color: var(--bg-sidebar);">
        <!-- Left: Company logo + name (260px to align with sidebar) -->
        <div class="w-[260px] flex items-center gap-2.5">
            <div class="w-8 h-8 flex items-center justify-center" style="background-color: var(--terracotta);">
                <span class="text-lg font-bold" style="color: var(--text-light); font-family: 'Space Grotesk', sans-serif;">A</span>
            </div>
            <span class="text-sm font-bold tracking-[2px]" style="color: var(--text-light); font-family: 'Space Grotesk', sans-serif;">
                FERVID SMART SOLUTIONS
            </span>
        </div>

        <!-- Center: Project selector dropdown -->
        <div
            class="flex items-center"
            x-data="{ open: false }"
            @click.outside="open = false"
            @keydown.escape.window="open = false"
        >
            <!-- Trigger Button -->
            <button
                @click="open = !open"
                class="flex items-center"
                style="gap: 10px; padding: 8px 16px; background-color: #2a2a2a; border: none; cursor: pointer;"
            >
                <!-- Folder icon -->
                <svg class="w-4 h-4" style="color: var(--terracotta);" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="m6 14 1.5-2.9A2 2 0 0 1 9.24 10H20a2 2 0 0 1 1.94 2.5l-1.54 6a2 2 0 0 1-1.95 1.5H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3.9a2 2 0 0 1 1.69.9l.81 1.2a2 2 0 0 0 1.67.9H18a2 2 0 0 1 2 2v2"></path>
                </svg>
                <!-- Active project name or placeholder -->
                if data.ActiveProject != nil {
                    <span class="text-xs font-medium" style="color: var(--text-light); font-family: 'Space Grotesk', sans-serif;">
                        { data.ActiveProject.Name }
                    </span>
                } else {
                    <span class="text-xs font-medium" style="color: #666666; font-family: 'Space Grotesk', sans-serif;">
                        Select Project
                    </span>
                }
                <!-- Chevron -->
                <svg
                    class="w-3.5 h-3.5 transition-transform duration-200"
                    :class="{ 'rotate-180': open }"
                    style="color: #666666;"
                    xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
                >
                    <path d="m6 9 6 6 6-6"></path>
                </svg>
            </button>

            <!-- Dropdown Panel -->
            <div
                x-show="open"
                x-transition:enter="transition ease-out duration-150"
                x-transition:enter-start="opacity-0 -translate-y-1"
                x-transition:enter-end="opacity-100 translate-y-0"
                x-transition:leave="transition ease-in duration-100"
                x-transition:leave-start="opacity-100 translate-y-0"
                x-transition:leave-end="opacity-0 -translate-y-1"
                x-cloak
                class="absolute z-50"
                style="top: 48px; left: 50%; transform: translateX(-50%); width: 320px; background-color: var(--bg-sidebar); border: 1px solid var(--border-dark); box-shadow: 0 8px 24px rgba(0,0,0,0.3);"
            >
                <!-- Dropdown Header -->
                <div style="padding: 12px 16px; border-bottom: 1px solid var(--border-dark);">
                    <span style="font-family: 'Space Grotesk', sans-serif; font-size: 10px; font-weight: 600; letter-spacing: 1px; color: var(--text-muted);">
                        SWITCH PROJECT
                    </span>
                </div>

                <!-- Project List (scrollable) -->
                <div style="max-height: 280px; overflow-y: auto;">
                    if len(data.Projects) == 0 {
                        <div style="padding: 16px; text-align: center;">
                            <span style="font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-muted);">
                                No projects yet
                            </span>
                        </div>
                    } else {
                        for _, proj := range data.Projects {
                            <button
                                hx-post={ "/projects/" + proj.ID + "/activate" }
                                hx-target="body"
                                hx-push-url={ "/projects/" + proj.ID }
                                @click="open = false"
                                class="w-full flex items-center"
                                style={
                                    "padding: 10px 16px; gap: 12px; border: none; cursor: pointer; text-align: left;" +
                                    projectItemBg(proj.IsActive)
                                }
                            >
                                <!-- Active indicator dot -->
                                if proj.IsActive {
                                    <div style="width: 6px; height: 6px; background-color: var(--terracotta); border-radius: 50%; flex-shrink: 0;"></div>
                                } else {
                                    <div style="width: 6px; height: 6px; flex-shrink: 0;"></div>
                                }
                                <div style="flex: 1; min-width: 0;">
                                    <div style="font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 600; color: var(--text-light); overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
                                        { proj.Name }
                                    </div>
                                    if proj.Client != "" {
                                        <div style="font-family: 'Inter', sans-serif; font-size: 11px; color: var(--text-muted); margin-top: 2px;">
                                            { proj.Client }
                                        </div>
                                    }
                                </div>
                            </button>
                        }
                    }
                </div>

                <!-- All Projects Link (footer) -->
                <a
                    hx-get="/projects"
                    hx-target="#main-content"
                    hx-push-url="true"
                    @click="open = false"
                    class="flex items-center justify-center"
                    style="padding: 12px 16px; gap: 8px; border-top: 1px solid var(--border-dark); cursor: pointer; text-decoration: none;"
                >
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#666666" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="7" height="7" x="3" y="3" rx="1"></rect><rect width="7" height="7" x="14" y="3" rx="1"></rect><rect width="7" height="7" x="14" y="14" rx="1"></rect><rect width="7" height="7" x="3" y="14" rx="1"></rect></svg>
                    <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: #666666;">
                        ALL PROJECTS
                    </span>
                </a>
            </div>
        </div>

        <!-- Right: Bell + Avatar (260px to align with sidebar) -->
        <div class="w-[260px] flex items-center justify-end gap-4">
            <button style="color: #666666; background: none; border: none; cursor: pointer;">
                <svg class="w-[18px] h-[18px]" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M10.268 21a2 2 0 0 0 3.464 0"></path>
                    <path d="M3.262 15.326A1 1 0 0 0 4 17h16a1 1 0 0 0 .74-1.673C19.41 13.956 18 12.499 18 8A6 6 0 0 0 6 8c0 4.499-1.411 5.956-2.738 7.326"></path>
                </svg>
            </button>
            <div class="w-8 h-8 flex items-center justify-center" style="background-color: var(--border-dark);">
                <span class="text-[11px] font-semibold" style="color: var(--text-muted); font-family: 'Space Grotesk', sans-serif;">PM</span>
            </div>
        </div>
    </header>
}

func projectItemBg(isActive bool) string {
    if isActive {
        return " background-color: #2a2a2a;"
    }
    return " background-color: transparent;"
}
```

### Step 3: Update `templates/page.templ` to Pass Project Context

The `PageShell` needs a variant that accepts `HeaderData` so the header can render the dropdown with project data.

```templ
package templates

// Backward-compatible: existing pages without project context
templ PageShell(title string) {
    @PageShellWithProject(title, HeaderData{}, SidebarData{})
}

// New: project-aware shell
templ PageShellWithProject(title string, headerData HeaderData, sidebarData SidebarData) {
    @Layout(title) {
        <div class="flex flex-col h-screen">
            @TopHeaderWithProject(headerData)
            <div class="flex flex-1 overflow-hidden">
                @SidebarWithProject(sidebarData)
                <main id="main-content" class="flex-1 overflow-y-auto" style="background-color: var(--bg-page); padding: 40px 48px;">
                    { children... }
                </main>
            </div>
        </div>
    }
}
```

### Step 4: Active Project Cookie Middleware -- `handlers/middleware.go`

```go
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
const HeaderDataKey    contextKey = "headerData"

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
                // Clear invalid cookie
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

        return e.Next()
    }
}
```

### Step 5: Project Activation Endpoint -- `handlers/project_switch.go`

```go
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
            return e.String(404, "Project not found")
        }

        // Set cookie (30-day expiry, HttpOnly)
        http.SetCookie(e.Response, &http.Cookie{
            Name:     "active_project",
            Value:    projectID,
            Path:     "/",
            MaxAge:   60 * 60 * 24 * 30, // 30 days
            HttpOnly: true,
            SameSite: http.SameSiteLaxMode,
        })

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

        e.Response.Header().Set("HX-Redirect", "/projects")
        return e.String(200, "OK")
    }
}
```

### Step 6: Register Routes and Middleware in `main.go`

```go
// Inside the OnServe handler, BEFORE all other routes:

// Apply middleware globally
se.Router.BindFunc(handlers.ActiveProjectMiddleware(app))

// Project activation routes
se.Router.POST("/projects/{id}/activate", handlers.HandleProjectActivate(app))
se.Router.POST("/projects/deactivate", handlers.HandleProjectDeactivate(app))
```

### Step 7: Update Existing Handlers to Be Project-Aware

Every handler that renders a full page (not just HTMX partials) should use the project-aware `PageShellWithProject`. Example update pattern for `HandleBOQList`:

```go
func HandleBOQList(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        // ... existing data loading logic ...

        headerData := handlers.GetHeaderData(e.Request)
        sidebarData := buildSidebarData(e.Request, app) // helper from Phase 6

        var component templ.Component
        if e.Request.Header.Get("HX-Request") == "true" {
            component = templates.BOQListContent(data)
        } else {
            component = templates.BOQListPageWithProject(data, headerData, sidebarData)
        }
        return component.Render(e.Request.Context(), e.Response)
    }
}
```

This pattern is applied incrementally -- existing handlers keep working with the no-arg `PageShell` until they are updated.

---

## Dependencies on Other Phases

| Dependency | Phase | Notes |
|-----------|-------|-------|
| `projects` collection | Phase 4 | Must exist for dropdown data |
| `templates/sidebar.templ` `SidebarData` type | Phase 6 | `PageShellWithProject` references it, but can use empty struct initially |
| `templates/project_list.templ` | Phase 4 | "All Projects" link navigates there |

---

## Testing / Verification Steps

1. **Header renders dropdown trigger** -- verify the button shows "Select Project" when no cookie is set.
2. **Click dropdown** -- verify it opens with a smooth transition, lists all projects.
3. **Click outside** -- verify dropdown closes.
4. **Press Escape** -- verify dropdown closes.
5. **Select a project** -- verify:
   - Cookie `active_project` is set (check browser dev tools).
   - Page redirects to `/projects/{id}`.
   - Header now shows the selected project name.
   - The selected project has a terracotta dot in the dropdown.
6. **Click "All Projects"** -- verify navigation to `/projects` list page.
7. **No projects exist** -- verify "No projects yet" message in dropdown.
8. **Invalid cookie** -- manually set cookie to a non-existent ID, reload, verify it is cleared and dropdown shows "Select Project".
9. **Multiple tabs** -- switch project in one tab, reload another, verify cookie is shared.
10. **Middleware context** -- add a debug log in a handler to print `GetActiveProject(r)` and verify it is populated.

---

## Acceptance Criteria

- [ ] Header project selector replaces static text with a dynamic Alpine.js dropdown
- [ ] Dropdown lists all projects with Name and Client subtitle
- [ ] Active project has a terracotta indicator dot and darker background
- [ ] "All Projects" link at the bottom navigates to `/projects`
- [ ] Selecting a project sets an `active_project` HttpOnly cookie (30-day expiry)
- [ ] Selecting a project triggers `HX-Redirect` for a full page reload (header + sidebar re-render)
- [ ] `TopHeader()` (no-arg) remains backward-compatible for existing pages
- [ ] `TopHeaderWithProject(data)` renders the full dropdown
- [ ] `PageShellWithProject` passes header and sidebar data through the layout
- [ ] Middleware runs on all routes and populates request context with `ActiveProject` and `HeaderData`
- [ ] Chevron icon rotates 180deg when dropdown is open
- [ ] Dropdown panel has max-height 280px with overflow scroll for many projects
- [ ] Dropdown closes on click outside and on Escape key
