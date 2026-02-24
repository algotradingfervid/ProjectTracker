# Phase 6: Sidebar Update -- Project-Specific Navigation

## Overview & Objectives

Modify `templates/sidebar.templ` to become context-aware. When no project is selected, the sidebar shows a minimal navigation with just a "Projects" link. When a project is active, the sidebar shows project-specific navigation items: BOQ and an expandable Addresses accordion with five sub-links (Bill From, Ship From, Bill To, Ship To, Install At), each with a count badge.

### Goals

1. **No project selected** -- Sidebar shows: Dashboard, Projects link, Settings.
2. **Project active** -- Sidebar shows: Dashboard, Project Details (active, with sub-nav: BOQ, Addresses accordion), Settings.
3. Addresses accordion uses Alpine.js for expand/collapse.
4. Each address type sub-link shows a count badge (number of addresses of that type).
5. Active state highlighting for the current page/tab.
6. Route links become project-scoped: `/projects/{id}/boq`, `/projects/{id}/addresses/bill_from`, etc.

---

## Files to Create / Modify

| Action | Path | Purpose |
|--------|------|---------|
| **Modify** | `templates/sidebar.templ` | Add `SidebarData` struct, parameterized rendering |
| **Modify** | `templates/page.templ` | Already updated in Phase 5 to pass `SidebarData` |
| **Create** | `handlers/sidebar_helpers.go` | Helper to build `SidebarData` from request context |
| **Modify** | `main.go` | Add project-scoped routes for BOQ and addresses |

---

## Detailed Implementation Steps

### Step 1: Define `SidebarData` Struct

```go
// templates/sidebar.templ (top of file)
package templates

type AddressTypeCounts struct {
    BillFrom  int
    ShipFrom  int
    BillTo    int
    ShipTo    int
    InstallAt int
    Total     int
}

type SidebarData struct {
    ActiveProject       *ActiveProject    // nil = no project selected
    ActivePath          string            // current URL path for highlighting, e.g. "/projects/abc/boq"
    AddressCounts       AddressTypeCounts
    BOQCount            int
    ShipToIsInstallAt   bool              // hides "Install At" when true
}
```

### Step 2: Modify `templates/sidebar.templ`

Replace the entire file. The original `Sidebar()` function is preserved as a backward-compatible wrapper.

```templ
package templates

import "fmt"
import "strconv"

type AddressTypeCounts struct {
    BillFrom  int
    ShipFrom  int
    BillTo    int
    ShipTo    int
    InstallAt int
    Total     int
}

type SidebarData struct {
    ActiveProject     *ActiveProject
    ActivePath        string
    AddressCounts     AddressTypeCounts
    BOQCount          int
    ShipToIsInstallAt bool
}

// Backward-compatible: renders sidebar without project context
templ Sidebar() {
    @SidebarWithProject(SidebarData{})
}

// Project-aware sidebar
templ SidebarWithProject(data SidebarData) {
    <aside class="w-[260px] min-h-screen flex flex-col" style="background-color: var(--bg-sidebar); padding: 32px 24px;">
        <div class="flex flex-col" style="gap: 32px;">
            <div class="flex flex-col">
                <!-- Dashboard -->
                <a
                    href="/"
                    hx-get="/"
                    hx-target="#main-content"
                    hx-push-url="true"
                    class="flex items-center"
                    style={ sidebarLinkStyle(data.ActivePath == "/") + " gap: 12px; padding: 14px 0;" }
                >
                    <svg style={ sidebarIconStyle(data.ActivePath == "/") + " width: 20px; height: 20px;" } xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <rect width="7" height="7" x="3" y="3" rx="1"></rect>
                        <rect width="7" height="7" x="14" y="3" rx="1"></rect>
                        <rect width="7" height="7" x="14" y="14" rx="1"></rect>
                        <rect width="7" height="7" x="3" y="14" rx="1"></rect>
                    </svg>
                    <span style={ sidebarLabelStyle(data.ActivePath == "/") }>DASHBOARD</span>
                </a>

                if data.ActiveProject != nil {
                    <!-- PROJECT DETAILS (Active project selected) -->
                    @SidebarProjectSection(data)
                } else {
                    <!-- Projects link (no project selected) -->
                    @SidebarProjectsLink(data)
                }

                <!-- Settings -->
                <a
                    href="/settings"
                    class="flex items-center"
                    style="gap: 12px; padding: 14px 0; border-top: 1px solid var(--border-dark);"
                >
                    <svg style="width: 20px; height: 20px; color: #666666;" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"></path>
                        <circle cx="12" cy="12" r="3"></circle>
                    </svg>
                    <span style="color: #666666; font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 500; letter-spacing: 1px;">SETTINGS</span>
                </a>
            </div>
        </div>
    </aside>
}

// SidebarProjectsLink renders a simple "Projects" link when no project is active
templ SidebarProjectsLink(data SidebarData) {
    <a
        href="/projects"
        hx-get="/projects"
        hx-target="#main-content"
        hx-push-url="true"
        class="flex items-center"
        style="gap: 12px; padding: 14px 0; border-top: 1px solid var(--border-dark);"
    >
        <svg style="width: 20px; height: 20px; color: #666666;" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="m6 14 1.5-2.9A2 2 0 0 1 9.24 10H20a2 2 0 0 1 1.94 2.5l-1.54 6a2 2 0 0 1-1.95 1.5H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3.9a2 2 0 0 1 1.69.9l.81 1.2a2 2 0 0 0 1.67.9H18a2 2 0 0 1 2 2v2"></path>
        </svg>
        <span style="color: #666666; font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 500; letter-spacing: 1px;">PROJECTS</span>
    </a>
}

// SidebarProjectSection renders the full project navigation when a project is active
templ SidebarProjectSection(data SidebarData) {
    <div class="flex flex-col">
        <!-- Project Details Header (always active-styled) -->
        <div class="flex items-center" style="gap: 12px; padding: 14px 0; border-top: 2px solid var(--terracotta);">
            <svg style="width: 20px; height: 20px; color: var(--terracotta);" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m6 14 1.5-2.9A2 2 0 0 1 9.24 10H20a2 2 0 0 1 1.94 2.5l-1.54 6a2 2 0 0 1-1.95 1.5H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3.9a2 2 0 0 1 1.69.9l.81 1.2a2 2 0 0 0 1.67.9H18a2 2 0 0 1 2 2v2"></path>
            </svg>
            <span style="color: var(--text-light); font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 600; letter-spacing: 1px;">
                PROJECT DETAILS
            </span>
        </div>

        <!-- Sub Navigation -->
        <div class="flex flex-col" style="padding-left: 32px;">
            <!-- BOQ Link -->
            @SidebarSubLink(
                fmt.Sprintf("/projects/%s/boq", data.ActiveProject.ID),
                "BOQ",
                isPathActive(data.ActivePath, fmt.Sprintf("/projects/%s/boq", data.ActiveProject.ID)),
                data.BOQCount,
            )

            <!-- Addresses Accordion (Alpine.js) -->
            <div
                x-data={ fmt.Sprintf(`{ addressOpen: %t }`, isAddressPath(data.ActivePath, data.ActiveProject.ID)) }
            >
                <!-- Accordion Trigger -->
                <button
                    @click="addressOpen = !addressOpen"
                    class="w-full flex items-center justify-between"
                    style="background: none; border: none; cursor: pointer; padding: 10px 0; text-align: left;"
                >
                    <div class="flex items-center" style="gap: 8px;">
                        <div
                            style={ subnavDotStyle(isAnyAddressActive(data.ActivePath, data.ActiveProject.ID)) }
                        ></div>
                        <span style={ subnavLabelStyle(isAnyAddressActive(data.ActivePath, data.ActiveProject.ID)) }>
                            ADDRESSES
                        </span>
                        <!-- Total count badge -->
                        if data.AddressCounts.Total > 0 {
                            <span style="padding: 1px 6px; font-family: 'Space Grotesk', sans-serif; font-size: 9px; font-weight: 600; color: var(--text-light); background-color: var(--border-dark); border-radius: 2px;">
                                { strconv.Itoa(data.AddressCounts.Total) }
                            </span>
                        }
                    </div>
                    <!-- Chevron -->
                    <svg
                        class="transition-transform duration-200"
                        :class="{ 'rotate-180': addressOpen }"
                        style="width: 12px; height: 12px; color: #666666;"
                        xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
                    >
                        <path d="m6 9 6 6 6-6"></path>
                    </svg>
                </button>

                <!-- Accordion Body -->
                <div
                    x-show="addressOpen"
                    x-transition:enter="transition ease-out duration-200"
                    x-transition:enter-start="opacity-0 -translate-y-1"
                    x-transition:enter-end="opacity-100 translate-y-0"
                    x-transition:leave="transition ease-in duration-150"
                    x-transition:leave-start="opacity-100 translate-y-0"
                    x-transition:leave-end="opacity-0 -translate-y-1"
                    x-cloak
                    class="flex flex-col"
                    style="padding-left: 14px;"
                >
                    @SidebarAddressSubLink(
                        fmt.Sprintf("/projects/%s/addresses/bill_from", data.ActiveProject.ID),
                        "Bill From",
                        data.AddressCounts.BillFrom,
                        data.ActivePath,
                    )
                    @SidebarAddressSubLink(
                        fmt.Sprintf("/projects/%s/addresses/ship_from", data.ActiveProject.ID),
                        "Ship From",
                        data.AddressCounts.ShipFrom,
                        data.ActivePath,
                    )
                    @SidebarAddressSubLink(
                        fmt.Sprintf("/projects/%s/addresses/bill_to", data.ActiveProject.ID),
                        "Bill To",
                        data.AddressCounts.BillTo,
                        data.ActivePath,
                    )
                    @SidebarAddressSubLink(
                        fmt.Sprintf("/projects/%s/addresses/ship_to", data.ActiveProject.ID),
                        "Ship To",
                        data.AddressCounts.ShipTo,
                        data.ActivePath,
                    )
                    if !data.ShipToIsInstallAt {
                        @SidebarAddressSubLink(
                            fmt.Sprintf("/projects/%s/addresses/install_at", data.ActiveProject.ID),
                            "Install At",
                            data.AddressCounts.InstallAt,
                            data.ActivePath,
                        )
                    }
                </div>
            </div>
        </div>
    </div>
}

// SidebarSubLink renders a single sub-navigation link with an optional count badge
templ SidebarSubLink(href string, label string, isActive bool, count int) {
    <a
        href={ templ.SafeURL(href) }
        hx-get={ href }
        hx-target="#main-content"
        hx-push-url="true"
        class="flex items-center justify-between"
        style="padding: 10px 0;"
    >
        <div class="flex items-center" style="gap: 8px;">
            <div style={ subnavDotStyle(isActive) }></div>
            <span style={ subnavLabelStyle(isActive) }>{ label }</span>
        </div>
        if count > 0 {
            <span style="padding: 1px 6px; font-family: 'Space Grotesk', sans-serif; font-size: 9px; font-weight: 600; color: var(--text-light); background-color: var(--border-dark); border-radius: 2px;">
                { strconv.Itoa(count) }
            </span>
        }
    </a>
}

// SidebarAddressSubLink renders an address type link inside the accordion
templ SidebarAddressSubLink(href string, label string, count int, activePath string) {
    <a
        href={ templ.SafeURL(href) }
        hx-get={ href }
        hx-target="#main-content"
        hx-push-url="true"
        class="flex items-center justify-between"
        style="padding: 8px 0;"
    >
        <div class="flex items-center" style="gap: 8px;">
            <div style={ addressDotStyle(activePath == href) }></div>
            <span style={ addressLabelStyle(activePath == href) }>{ label }</span>
        </div>
        <span style={ addressCountStyle(count) }>
            { strconv.Itoa(count) }
        </span>
    </a>
}

// --- Go helper functions for dynamic styling ---

func sidebarLinkStyle(active bool) string {
    if active {
        return "text-decoration: none;"
    }
    return "text-decoration: none;"
}

func sidebarIconStyle(active bool) string {
    if active {
        return "color: var(--terracotta);"
    }
    return "color: #666666;"
}

func sidebarLabelStyle(active bool) string {
    if active {
        return "color: var(--text-light); font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 600; letter-spacing: 1px;"
    }
    return "color: #666666; font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 500; letter-spacing: 1px;"
}

func subnavDotStyle(active bool) string {
    if active {
        return "width: 6px; height: 6px; background-color: var(--terracotta); flex-shrink: 0;"
    }
    return "width: 6px; height: 6px; background-color: #666666; flex-shrink: 0;"
}

func subnavLabelStyle(active bool) string {
    if active {
        return "color: var(--text-light); font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px;"
    }
    return "color: #666666; font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 500; letter-spacing: 1px;"
}

func addressDotStyle(active bool) string {
    if active {
        return "width: 4px; height: 4px; background-color: var(--terracotta); flex-shrink: 0;"
    }
    return "width: 4px; height: 4px; background-color: #555555; flex-shrink: 0;"
}

func addressLabelStyle(active bool) string {
    if active {
        return "color: var(--text-light); font-family: 'Inter', sans-serif; font-size: 11px; font-weight: 600;"
    }
    return "color: #666666; font-family: 'Inter', sans-serif; font-size: 11px; font-weight: 400;"
}

func addressCountStyle(count int) string {
    base := "padding: 1px 6px; font-family: 'Space Grotesk', sans-serif; font-size: 9px; font-weight: 600; border-radius: 2px;"
    if count > 0 {
        return base + " color: var(--text-light); background-color: var(--border-dark);"
    }
    return base + " color: #555555; background-color: transparent;"
}

// Path matching helpers

func isPathActive(currentPath, targetPath string) bool {
    return currentPath == targetPath || strings.HasPrefix(currentPath, targetPath+"/")
}

func isAddressPath(currentPath, projectID string) bool {
    prefix := fmt.Sprintf("/projects/%s/addresses", projectID)
    return strings.HasPrefix(currentPath, prefix)
}

func isAnyAddressActive(currentPath, projectID string) bool {
    return isAddressPath(currentPath, projectID)
}
```

Note: Add `"strings"` to the import block.

### Step 3: Sidebar Data Builder -- `handlers/sidebar_helpers.go`

```go
package handlers

import (
    "net/http"

    "github.com/pocketbase/pocketbase"

    "projectcreation/templates"
)

// BuildSidebarData constructs the SidebarData from the current request context.
// It reads the active project from middleware context and queries address counts.
func BuildSidebarData(r *http.Request, app *pocketbase.PocketBase) templates.SidebarData {
    activeProj := GetActiveProject(r)
    if activeProj == nil {
        return templates.SidebarData{
            ActivePath: r.URL.Path,
        }
    }

    data := templates.SidebarData{
        ActiveProject: activeProj,
        ActivePath:    r.URL.Path,
    }

    // Load project record for config
    projRec, err := app.FindRecordById("projects", activeProj.ID)
    if err == nil {
        data.ShipToIsInstallAt = projRec.GetBool("ship_to_is_install_at")
    }

    // Count BOQs for this project
    boqCol, _ := app.FindCollectionByNameOrId("boqs")
    if boqCol != nil {
        boqs, _ := app.FindRecordsByFilter(boqCol, "project = {:pid}", "", 0, 0, map[string]any{"pid": activeProj.ID})
        data.BOQCount = len(boqs)
    }

    // Count addresses by type
    addrCol, _ := app.FindCollectionByNameOrId("addresses")
    if addrCol != nil {
        types := map[string]*int{
            "bill_from":  &data.AddressCounts.BillFrom,
            "ship_from":  &data.AddressCounts.ShipFrom,
            "bill_to":    &data.AddressCounts.BillTo,
            "ship_to":    &data.AddressCounts.ShipTo,
            "install_at": &data.AddressCounts.InstallAt,
        }
        for addrType, countPtr := range types {
            records, err := app.FindRecordsByFilter(
                addrCol,
                "project = {:pid} && type = {:type}",
                "", 0, 0,
                map[string]any{"pid": activeProj.ID, "type": addrType},
            )
            if err == nil {
                *countPtr = len(records)
            }
        }
        data.AddressCounts.Total = data.AddressCounts.BillFrom +
            data.AddressCounts.ShipFrom +
            data.AddressCounts.BillTo +
            data.AddressCounts.ShipTo +
            data.AddressCounts.InstallAt
    }

    return data
}
```

### Step 4: Update Handler Pattern for All Page Renders

Every handler that renders a full page should now call `BuildSidebarData` and `GetHeaderData` to pass to `PageShellWithProject`. Here is the generic pattern:

```go
// In any handler that returns a full page:
func HandleSomePage(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        // ... load page-specific data ...

        var component templ.Component
        if e.Request.Header.Get("HX-Request") == "true" {
            component = templates.SomePageContent(pageData)
        } else {
            headerData := GetHeaderData(e.Request)
            sidebarData := BuildSidebarData(e.Request, app)
            component = templates.SomePageWithProject(pageData, headerData, sidebarData)
        }
        return component.Render(e.Request.Context(), e.Response)
    }
}
```

### Step 5: Add Project-Scoped Routes in `main.go`

```go
// Project-scoped BOQ routes
se.Router.GET("/projects/{id}/boq", handlers.HandleProjectBOQList(app))

// Project-scoped address routes (5 types, same handler with different type param)
se.Router.GET("/projects/{id}/addresses/bill_from", handlers.HandleAddressList(app, "bill_from"))
se.Router.GET("/projects/{id}/addresses/ship_from", handlers.HandleAddressList(app, "ship_from"))
se.Router.GET("/projects/{id}/addresses/bill_to", handlers.HandleAddressList(app, "bill_to"))
se.Router.GET("/projects/{id}/addresses/ship_to", handlers.HandleAddressList(app, "ship_to"))
se.Router.GET("/projects/{id}/addresses/install_at", handlers.HandleAddressList(app, "install_at"))
```

---

## Dependencies on Other Phases

| Dependency | Phase | Notes |
|-----------|-------|-------|
| `ActiveProject` struct and `HeaderData` | Phase 5 | Defined in header.templ, used by sidebar |
| `ActiveProjectMiddleware` | Phase 5 | Populates request context with active project |
| `projects` and `addresses` collections | Phase 4 | Must exist for count queries |
| `PageShellWithProject` | Phase 5 | Sidebar is rendered inside the shell |
| `boqs.project` relation field | Phase 4 | Needed for BOQ count per project |

---

## Testing / Verification Steps

1. **No project selected** -- Navigate to `/projects`:
   - Sidebar shows Dashboard, Projects link, Settings.
   - No "Project Details" section visible.
   - No addresses accordion visible.

2. **Select a project** via header dropdown:
   - Sidebar updates to show "PROJECT DETAILS" section with terracotta top border.
   - BOQ sub-link appears with correct count badge.
   - Addresses accordion appears.

3. **Expand Addresses accordion**:
   - Click "Addresses" -- verify smooth expand animation.
   - All 5 address types visible: Bill From, Ship From, Bill To, Ship To, Install At.
   - Each shows a count badge (0 if no addresses exist).
   - Click again -- verify smooth collapse animation.

4. **Auto-expand on address page**:
   - Navigate directly to `/projects/{id}/addresses/bill_from`.
   - Verify accordion is auto-expanded (Alpine `x-data` initializes to true).
   - Verify "Bill From" link has active styling (terracotta dot, white text).

5. **Active state highlighting**:
   - Navigate to `/projects/{id}/boq` -- verify BOQ dot is terracotta.
   - Navigate to `/projects/{id}/addresses/ship_to` -- verify Ship To is highlighted and accordion is expanded.

6. **Count badges**:
   - Add 2 "Bill From" addresses via PocketBase admin.
   - Reload sidebar -- verify Bill From badge shows "2" and Total shows "2".

7. **Ship To = Install At toggle**:
   - Enable the toggle in project edit (Phase 4).
   - Verify "Install At" link disappears from the sidebar.
   - Disable the toggle -- verify "Install At" reappears.

8. **HTMX navigation**:
   - Click sidebar links -- verify `#main-content` swaps without full reload.
   - Verify `hx-push-url` updates the browser URL.

9. **Backward compatibility**:
   - Call `Sidebar()` (no args) from old templates -- verify it renders the "no project" state without errors.

10. **Responsive scroll**:
    - Add many address entries -- verify sidebar does not overflow (content scrolls within `min-h-screen` container).

---

## Acceptance Criteria

- [ ] `Sidebar()` (no-arg) remains backward-compatible
- [ ] `SidebarWithProject(data)` renders context-aware navigation
- [ ] When no project is selected: Dashboard, Projects link, Settings are shown
- [ ] When project is active: Dashboard, Project Details (BOQ + Addresses accordion), Settings are shown
- [ ] Project Details section has terracotta top border matching existing design
- [ ] BOQ link navigates to `/projects/{id}/boq` via HTMX
- [ ] Addresses accordion uses Alpine.js `x-show` with enter/leave transitions
- [ ] Accordion auto-expands when current URL is an address sub-page
- [ ] 5 address type links: Bill From, Ship From, Bill To, Ship To, Install At
- [ ] Install At link is hidden when `ShipToIsInstallAt` is true
- [ ] Each address type link shows a count badge (number from PocketBase query)
- [ ] Total addresses count badge on the "Addresses" accordion trigger
- [ ] Active link has terracotta dot + white text; inactive has gray dot + gray text
- [ ] Address sub-links use smaller dots (4px) and Inter font, matching visual hierarchy
- [ ] Count badges use Space Grotesk 9px, `--border-dark` background, `--text-light` color
- [ ] Chevron on accordion rotates 180deg when open
- [ ] All sidebar links use both `href` (fallback) and `hx-get` + `hx-target="#main-content"` + `hx-push-url="true"`
- [ ] `BuildSidebarData` helper queries address counts efficiently (one query per type)
- [ ] Sidebar renders correctly on full page load and on HTMX partial swaps
