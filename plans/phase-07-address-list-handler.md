# Phase 7: Address List Handler

## Overview & Objectives

Create a generic, reusable handler for listing addresses of any type (bill_from, ship_from, bill_to, ship_to, install_at) within a project. The handler supports pagination, search/filter, sorting, and returns both full-page and HTMX partial responses following the existing pattern established by `HandleBOQList` and `HandleBOQView`.

### Key Goals

- Single generic `HandleAddressList` handler function parameterized by address type
- Query addresses from PocketBase filtered by `project_id` and `address_type`
- Server-side pagination with configurable page size
- Search across company name, city, state, and contact person
- Sort by any column with ascending/descending order
- Return address count for sidebar badge display
- HTMX partial vs full page rendering based on `HX-Request` header

---

## Files to Create/Modify

| File | Action | Purpose |
|------|--------|---------|
| `handlers/address_list.go` | **Create** | Generic address list handler |
| `main.go` | **Modify** | Register routes for all 5 address types |

---

## Detailed Implementation Steps

### Step 1: Define Address List Data Types

Create the view model structs in `handlers/address_list.go`. These mirror the pattern used by `BOQListItem` and `BOQListData` in `templates/boq_list.templ`.

```go
package handlers

import (
    "fmt"
    "log"
    "math"
    "strconv"
    "strings"

    "github.com/a-h/templ"
    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"

    "projectcreation/templates"
)
```

### Step 2: Define Constants and Label Mapping

```go
// AddressType represents the valid address types in the system.
type AddressType string

const (
    AddressTypeBillFrom  AddressType = "bill_from"
    AddressTypeShipFrom  AddressType = "ship_from"
    AddressTypeBillTo    AddressType = "bill_to"
    AddressTypeShipTo    AddressType = "ship_to"
    AddressTypeInstallAt AddressType = "install_at"
)

// addressTypeLabels maps address type slugs to human-readable labels.
var addressTypeLabels = map[AddressType]string{
    AddressTypeBillFrom:  "Bill From",
    AddressTypeShipFrom:  "Ship From",
    AddressTypeBillTo:    "Bill To",
    AddressTypeShipTo:    "Ship To",
    AddressTypeInstallAt: "Install At",
}

// ValidAddressTypes is the ordered list of all address types.
var ValidAddressTypes = []AddressType{
    AddressTypeBillFrom,
    AddressTypeShipFrom,
    AddressTypeBillTo,
    AddressTypeShipTo,
    AddressTypeInstallAt,
}

// addressTypeLabel returns the human-readable label for an address type.
func addressTypeLabel(at AddressType) string {
    if label, ok := addressTypeLabels[at]; ok {
        return label
    }
    return string(at)
}
```

### Step 3: Define Default Pagination Constants

```go
const (
    defaultPageSize = 20
    maxPageSize     = 100
    defaultPage     = 1
    defaultSortBy   = "company_name"
    defaultSortOrder = "asc"
)
```

### Step 4: Parse Query Parameters Helper

Extract pagination, search, and sort parameters from the request. This follows the same `e.Request` pattern used in `HandleBOQCreate` and `HandleBOQUpdate`.

```go
// addressListParams holds parsed query parameters for the address list.
type addressListParams struct {
    Page      int
    PageSize  int
    Search    string
    SortBy    string
    SortOrder string // "asc" or "desc"
}

// parseAddressListParams extracts and validates query parameters from the request.
func parseAddressListParams(e *core.RequestEvent) addressListParams {
    params := addressListParams{
        Page:      defaultPage,
        PageSize:  defaultPageSize,
        SortBy:    defaultSortBy,
        SortOrder: defaultSortOrder,
    }

    if p := e.Request.URL.Query().Get("page"); p != "" {
        if v, err := strconv.Atoi(p); err == nil && v > 0 {
            params.Page = v
        }
    }

    if ps := e.Request.URL.Query().Get("page_size"); ps != "" {
        if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= maxPageSize {
            params.PageSize = v
        }
    }

    params.Search = strings.TrimSpace(e.Request.URL.Query().Get("search"))

    if sb := e.Request.URL.Query().Get("sort_by"); sb != "" {
        // Whitelist allowed sort columns to prevent injection
        allowedSorts := map[string]bool{
            "company_name": true, "contact_person": true, "city": true,
            "state": true, "pin_code": true, "gstin": true, "pan": true,
            "email": true, "phone": true, "created": true, "updated": true,
            "country": true, "district": true,
        }
        if allowedSorts[sb] {
            params.SortBy = sb
        }
    }

    if so := e.Request.URL.Query().Get("sort_order"); so == "desc" {
        params.SortOrder = "desc"
    }

    return params
}
```

### Step 5: Build PocketBase Filter String

Construct the filter expression combining project ID, address type, and optional search terms.

```go
// buildAddressFilter constructs a PocketBase filter string and bind params.
func buildAddressFilter(projectID string, addressType AddressType, search string) (string, map[string]any) {
    filter := "project = {:projectId} && address_type = {:addressType}"
    params := map[string]any{
        "projectId":   projectID,
        "addressType": string(addressType),
    }

    if search != "" {
        filter += " && (company_name ~ {:search} || city ~ {:search} || state ~ {:search} || contact_person ~ {:search})"
        params["search"] = search
    }

    return filter, params
}
```

### Step 6: Build Sort String

Convert sort parameters to PocketBase sort format (prefix with `-` for descending).

```go
// buildSortString returns the PocketBase sort expression.
func buildSortString(sortBy, sortOrder string) string {
    if sortOrder == "desc" {
        return "-" + sortBy
    }
    return sortBy
}
```

### Step 7: Implement HandleAddressList

The main handler function, following the closure pattern used by `HandleBOQList(app)` and `HandleBOQView(app)`.

```go
// HandleAddressList returns a handler that lists addresses of a given type for a project.
// It supports pagination, search, sort, and both full-page and HTMX partial rendering.
func HandleAddressList(app *pocketbase.PocketBase, addressType AddressType) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectID := e.Request.PathValue("projectId")
        if projectID == "" {
            return e.String(400, "Missing project ID")
        }

        // 1. Verify project exists
        projectRecord, err := app.FindRecordById("projects", projectID)
        if err != nil {
            log.Printf("address_list: could not find project %s: %v", projectID, err)
            return e.String(404, "Project not found")
        }

        // 2. Parse query parameters
        params := parseAddressListParams(e)

        // 3. Build filter and sort
        filter, filterParams := buildAddressFilter(projectID, addressType, params.Search)
        sortStr := buildSortString(params.SortBy, params.SortOrder)

        // 4. Get total count for pagination (query with no limit)
        addressesCol, err := app.FindCollectionByNameOrId("addresses")
        if err != nil {
            log.Printf("address_list: could not find addresses collection: %v", err)
            return e.String(500, "Internal error")
        }

        allRecords, err := app.FindRecordsByFilter(addressesCol, filter, sortStr, 0, 0, filterParams)
        if err != nil {
            log.Printf("address_list: could not count addresses for project %s type %s: %v", projectID, addressType, err)
            allRecords = nil
        }
        totalCount := len(allRecords)

        // 5. Calculate pagination
        totalPages := int(math.Ceil(float64(totalCount) / float64(params.PageSize)))
        if totalPages < 1 {
            totalPages = 1
        }
        if params.Page > totalPages {
            params.Page = totalPages
        }

        // 6. Fetch paginated records
        offset := (params.Page - 1) * params.PageSize
        records, err := app.FindRecordsByFilter(
            addressesCol, filter, sortStr, params.PageSize, offset, filterParams,
        )
        if err != nil {
            log.Printf("address_list: could not query addresses for project %s type %s: %v", projectID, addressType, err)
            records = nil
        }

        // 7. Map records to view models
        var items []templates.AddressListItem
        for i, rec := range records {
            items = append(items, templates.AddressListItem{
                ID:            rec.Id,
                Index:         offset + i + 1,
                CompanyName:   rec.GetString("company_name"),
                ContactPerson: rec.GetString("contact_person"),
                AddressLine1:  rec.GetString("address_line_1"),
                AddressLine2:  rec.GetString("address_line_2"),
                City:          rec.GetString("city"),
                State:         rec.GetString("state"),
                PinCode:       rec.GetString("pin_code"),
                Country:       rec.GetString("country"),
                Phone:         rec.GetString("phone"),
                Email:         rec.GetString("email"),
                GSTIN:         rec.GetString("gstin"),
                PAN:           rec.GetString("pan"),
                CIN:           rec.GetString("cin"),
                Website:       rec.GetString("website"),
                Fax:           rec.GetString("fax"),
                Landmark:      rec.GetString("landmark"),
                District:      rec.GetString("district"),
            })
        }

        // 8. Build page number list for pagination controls
        var pageNumbers []int
        for p := 1; p <= totalPages; p++ {
            pageNumbers = append(pageNumbers, p)
        }

        // 9. Build template data
        data := templates.AddressListData{
            ProjectID:    projectID,
            ProjectName:  projectRecord.GetString("name"),
            AddressType:  string(addressType),
            AddressLabel: addressTypeLabel(addressType),
            Items:        items,
            TotalCount:   totalCount,
            Page:         params.Page,
            PageSize:     params.PageSize,
            TotalPages:   totalPages,
            PageNumbers:  pageNumbers,
            HasPrev:      params.Page > 1,
            HasNext:      params.Page < totalPages,
            Search:       params.Search,
            SortBy:       params.SortBy,
            SortOrder:    params.SortOrder,
        }

        // 10. Render: partial for HTMX, full page otherwise
        var component templ.Component
        if e.Request.Header.Get("HX-Request") == "true" {
            component = templates.AddressListContent(data)
        } else {
            component = templates.AddressListPage(data)
        }
        return component.Render(e.Request.Context(), e.Response)
    }
}
```

### Step 8: Implement HandleAddressCount for Sidebar Badges

A lightweight endpoint that returns just the count for a given address type, useful for sidebar badge updates via HTMX.

```go
// HandleAddressCount returns a handler that returns the address count for a given type.
// This is used by the sidebar to display badge counts via HTMX.
func HandleAddressCount(app *pocketbase.PocketBase, addressType AddressType) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectID := e.Request.PathValue("projectId")
        if projectID == "" {
            return e.String(400, "Missing project ID")
        }

        addressesCol, err := app.FindCollectionByNameOrId("addresses")
        if err != nil {
            return e.String(500, "Internal error")
        }

        filter := "project = {:projectId} && address_type = {:addressType}"
        filterParams := map[string]any{
            "projectId":   projectID,
            "addressType": string(addressType),
        }

        records, err := app.FindRecordsByFilter(addressesCol, filter, "", 0, 0, filterParams)
        if err != nil {
            log.Printf("address_count: error querying count for project %s type %s: %v", projectID, addressType, err)
            return e.String(200, "0")
        }

        return e.String(200, fmt.Sprintf("%d", len(records)))
    }
}
```

### Step 9: Register Routes in main.go

Add route registrations following the existing pattern in `main.go`. Each address type gets its own list route and count route.

```go
// Address list routes - one per address type
addressTypes := []struct {
    slug    string
    addrType handlers.AddressType
}{
    {"bill-from", handlers.AddressTypeBillFrom},
    {"ship-from", handlers.AddressTypeShipFrom},
    {"bill-to", handlers.AddressTypeBillTo},
    {"ship-to", handlers.AddressTypeShipTo},
    {"install-at", handlers.AddressTypeInstallAt},
}

for _, at := range addressTypes {
    // List page
    se.Router.GET(
        "/projects/{projectId}/addresses/"+at.slug,
        handlers.HandleAddressList(app, at.addrType),
    )
    // Count endpoint for sidebar badges
    se.Router.GET(
        "/projects/{projectId}/addresses/"+at.slug+"/count",
        handlers.HandleAddressCount(app, at.addrType),
    )
}
```

**Route mapping:**

| HTTP Method | Route | Handler | Description |
|-------------|-------|---------|-------------|
| GET | `/projects/{projectId}/addresses/bill-from` | `HandleAddressList(app, AddressTypeBillFrom)` | Bill From list |
| GET | `/projects/{projectId}/addresses/ship-from` | `HandleAddressList(app, AddressTypeShipFrom)` | Ship From list |
| GET | `/projects/{projectId}/addresses/bill-to` | `HandleAddressList(app, AddressTypeBillTo)` | Bill To list |
| GET | `/projects/{projectId}/addresses/ship-to` | `HandleAddressList(app, AddressTypeShipTo)` | Ship To list |
| GET | `/projects/{projectId}/addresses/install-at` | `HandleAddressList(app, AddressTypeInstallAt)` | Install At list |
| GET | `/projects/{projectId}/addresses/{type}/count` | `HandleAddressCount(app, ...)` | Badge count |

**Query parameters supported on list routes:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `page` | `1` | Current page number |
| `page_size` | `20` | Records per page (max 100) |
| `search` | `""` | Search across company_name, city, state, contact_person |
| `sort_by` | `company_name` | Column to sort by (whitelisted) |
| `sort_order` | `asc` | Sort direction: `asc` or `desc` |

### Step 10: Define Template Data Structs

These structs should be placed in `templates/address_list.templ` (created in Phase 8) but are referenced by the handler. Define them here for completeness:

```go
// In templates/address_list.templ (package templates)

type AddressListItem struct {
    ID            string
    Index         int
    CompanyName   string
    ContactPerson string
    AddressLine1  string
    AddressLine2  string
    City          string
    State         string
    PinCode       string
    Country       string
    Phone         string
    Email         string
    GSTIN         string
    PAN           string
    CIN           string
    Website       string
    Fax           string
    Landmark      string
    District      string
}

type AddressListData struct {
    ProjectID    string
    ProjectName  string
    AddressType  string
    AddressLabel string
    Items        []AddressListItem
    TotalCount   int
    Page         int
    PageSize     int
    TotalPages   int
    PageNumbers  []int
    HasPrev      bool
    HasNext      bool
    Search       string
    SortBy       string
    SortOrder    string
}
```

---

## Dependencies on Other Phases

| Dependency | Phase | Required For |
|-----------|-------|--------------|
| `addresses` PocketBase collection | Phase 5/6 (DB setup) | Querying address records |
| `projects` PocketBase collection | Phase 1-3 (Project setup) | Verifying project exists, fetching project name |
| `project_address_settings` collection | Phase 5/6 | Required fields configuration (read-only in this phase) |
| `templates/address_list.templ` | Phase 8 | Template rendering (struct types + templ components) |

---

## Testing / Verification Steps

### Manual Testing

1. **Start the server** and ensure no compilation errors:
   ```bash
   go run main.go serve
   ```

2. **Test list endpoint with no addresses:**
   ```
   GET /projects/{validProjectId}/addresses/bill-from
   ```
   Expect: 200 OK, empty state displayed

3. **Test with invalid project ID:**
   ```
   GET /projects/nonexistent/addresses/bill-from
   ```
   Expect: 404 "Project not found"

4. **Test pagination parameters:**
   ```
   GET /projects/{id}/addresses/bill-to?page=2&page_size=10
   ```
   Expect: Correct offset/limit applied, pagination metadata correct

5. **Test search:**
   ```
   GET /projects/{id}/addresses/ship-from?search=Mumbai
   ```
   Expect: Only addresses with "Mumbai" in company_name, city, state, or contact_person

6. **Test sorting:**
   ```
   GET /projects/{id}/addresses/bill-from?sort_by=city&sort_order=desc
   ```
   Expect: Results sorted by city descending

7. **Test sort injection prevention:**
   ```
   GET /projects/{id}/addresses/bill-from?sort_by=invalid_column
   ```
   Expect: Falls back to default sort (company_name asc)

8. **Test HTMX partial rendering:**
   ```
   GET /projects/{id}/addresses/bill-from
   Headers: HX-Request: true
   ```
   Expect: Only `AddressListContent` partial returned (no full page shell)

9. **Test count endpoint:**
   ```
   GET /projects/{id}/addresses/bill-from/count
   ```
   Expect: Plain text number (e.g., "5")

### Automated Test Ideas

```go
func TestParseAddressListParams(t *testing.T) {
    // Test default values
    // Test valid page/page_size
    // Test invalid page (negative, zero)
    // Test page_size exceeding max
    // Test search trimming
    // Test sort_by whitelist
    // Test sort_order validation
}

func TestBuildAddressFilter(t *testing.T) {
    // Test without search
    // Test with search - verify filter includes all 4 fields
}

func TestBuildSortString(t *testing.T) {
    // Test asc sort
    // Test desc sort (should prefix with -)
}
```

---

## Acceptance Criteria

- [ ] `handlers/address_list.go` compiles without errors
- [ ] `HandleAddressList` accepts `AddressType` parameter and returns a `func(*core.RequestEvent) error` closure
- [ ] All 5 address type routes are registered in `main.go`
- [ ] Count endpoints return plain text integer for each address type
- [ ] Pagination defaults: page=1, page_size=20
- [ ] Search filters across company_name, city, state, contact_person using PocketBase `~` (contains) operator
- [ ] Sort column is validated against a whitelist; invalid columns fall back to default
- [ ] Sort order accepts only "asc" or "desc"; invalid values default to "asc"
- [ ] HTMX requests (`HX-Request: true`) return partial content; regular requests return full page with layout
- [ ] Handler returns 404 for invalid project IDs
- [ ] Handler returns 400 for missing project ID path parameter
- [ ] `AddressListData` struct contains all fields needed by the template (Phase 8)
- [ ] No N+1 query issues - addresses fetched in a single query
- [ ] Page numbers list is correctly computed for pagination controls
