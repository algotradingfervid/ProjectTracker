# Phase 8: Address List Template

## Overview & Objectives

Create the Templ template for the address list page, featuring a data table with configurable visible columns, sortable headers, search with HTMX debounce, pagination controls, bulk selection, and action buttons (Add, Import, Export, Download Template). The template follows the existing design system established in `boq_list.templ` -- using Space Grotesk headings, Inter body text, CSS custom properties for colors, and the `PageShell` layout wrapper.

### Key Goals

- Configurable column visibility using Alpine.js with localStorage persistence per address type
- Sortable column headers with click-to-sort and arrow indicators
- HTMX-driven search with debounce (300ms delay)
- Pagination controls with Previous/Next and page number buttons
- Action bar: Add Address, Import from Excel, Export to Excel, Download Template
- Bulk select checkboxes with bulk delete
- Empty state display
- Responsive to HTMX partial vs full page rendering

---

## Files to Create/Modify

| File | Action | Purpose |
|------|--------|---------|
| `templates/address_list.templ` | **Create** | Address list page template with data types and components |

---

## Detailed Implementation Steps

### Step 1: Define Data Types

Place these at the top of `templates/address_list.templ`. The handler (Phase 7) populates these structs.

```go
package templates

import "fmt"
import "strconv"

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
    AddressType  string // slug: "bill_from", "ship_from", etc.
    AddressLabel string // display: "Bill From", "Ship From", etc.
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

### Step 2: Define Column Configuration Type

A helper for the column selector feature. Each column has an ID, label, and default visibility flag.

```go
// AddressColumn represents a toggleable column in the address list table.
type AddressColumn struct {
    ID      string
    Label   string
    Default bool // visible by default
}

// AddressColumns defines all available columns and their default visibility.
var AddressColumns = []AddressColumn{
    {ID: "company_name", Label: "Company Name", Default: true},
    {ID: "contact_person", Label: "Contact Person", Default: true},
    {ID: "address_line_1", Label: "Address Line 1", Default: false},
    {ID: "address_line_2", Label: "Address Line 2", Default: false},
    {ID: "city", Label: "City", Default: true},
    {ID: "state", Label: "State", Default: true},
    {ID: "pin_code", Label: "PIN Code", Default: true},
    {ID: "country", Label: "Country", Default: false},
    {ID: "phone", Label: "Phone", Default: false},
    {ID: "email", Label: "Email", Default: false},
    {ID: "gstin", Label: "GSTIN", Default: true},
    {ID: "pan", Label: "PAN", Default: false},
    {ID: "cin", Label: "CIN", Default: false},
    {ID: "website", Label: "Website", Default: false},
    {ID: "fax", Label: "Fax", Default: false},
    {ID: "landmark", Label: "Landmark", Default: false},
    {ID: "district", Label: "District", Default: false},
}
```

### Step 3: Implement AddressListContent (Partial)

This is the HTMX-swappable partial that goes inside `#main-content`. It follows the same pattern as `BOQListContent`.

```templ
templ AddressListContent(data AddressListData) {
    // Alpine.js data for column toggling, bulk select, and search
    <div
        x-data={ fmt.Sprintf(`{
            // Column visibility - load from localStorage or use defaults
            columns: JSON.parse(localStorage.getItem('addr_cols_%s') || 'null') || {
                company_name: true, contact_person: true, address_line_1: false,
                address_line_2: false, city: true, state: true, pin_code: true,
                country: false, phone: false, email: false, gstin: true,
                pan: false, cin: false, website: false, fax: false,
                landmark: false, district: false
            },
            showColumnPicker: false,
            selectedIds: [],
            selectAll: false,

            toggleColumn(col) {
                this.columns[col] = !this.columns[col];
                localStorage.setItem('addr_cols_%s', JSON.stringify(this.columns));
            },
            toggleSelectAll() {
                if (this.selectAll) {
                    this.selectedIds = [];
                } else {
                    this.selectedIds = [%s];
                }
                this.selectAll = !this.selectAll;
            },
            isSelected(id) {
                return this.selectedIds.includes(id);
            },
            toggleSelect(id) {
                const idx = this.selectedIds.indexOf(id);
                if (idx > -1) {
                    this.selectedIds.splice(idx, 1);
                } else {
                    this.selectedIds.push(id);
                }
                this.selectAll = false;
            }
        }`, data.AddressType, data.AddressType, buildItemIDsJS(data.Items)) }
    >
```

**Note:** The `buildItemIDsJS` helper generates a JS array literal from the item IDs for the selectAll functionality. This would be a templ helper or inline string builder.

#### 3a: Breadcrumbs

```templ
        <!-- Breadcrumbs -->
        <div class="flex items-center" style="gap: 6px; margin-bottom: 16px;">
            <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 500; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px;">
                PROJECTS
            </span>
            <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; color: var(--text-muted);">/</span>
            <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 500; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px;">
                { data.ProjectName }
            </span>
            <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; color: var(--text-muted);">/</span>
            <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; color: var(--terracotta); text-transform: uppercase; letter-spacing: 0.5px;">
                { data.AddressLabel }
            </span>
        </div>
```

#### 3b: Page Header with Search and Action Buttons

```templ
        <!-- Page Header -->
        <div class="flex justify-between items-center">
            <div>
                <h1 style="font-family: 'Space Grotesk', sans-serif; font-size: 36px; font-weight: 700; color: var(--text-primary); margin: 0;">
                    { data.AddressLabel } Addresses
                </h1>
                <p style="font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-secondary); margin-top: 8px;">
                    { strconv.Itoa(data.TotalCount) } address(es) found
                </p>
            </div>
            <div class="flex items-center" style="gap: 12px;">
                <!-- Search Input with HTMX Debounce -->
                <input
                    type="text"
                    name="search"
                    value={ data.Search }
                    placeholder="Search addresses..."
                    hx-get={ fmt.Sprintf("/projects/%s/addresses/%s", data.ProjectID, data.AddressType) }
                    hx-target="#main-content"
                    hx-trigger="keyup changed delay:300ms"
                    hx-push-url="true"
                    hx-include="this"
                    style="padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-primary); background-color: var(--bg-card); border: 1px solid var(--border-light); outline: none; width: 250px;"
                />

                <!-- Column Selector Dropdown -->
                <div class="relative">
                    <button
                        type="button"
                        @click="showColumnPicker = !showColumnPicker"
                        class="flex items-center"
                        style="background-color: var(--bg-card); padding: 10px 14px; gap: 8px; border: 1px solid var(--border-light); cursor: pointer;"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--text-secondary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3v18"/><path d="M3 12h18"/><rect x="3" y="3" width="18" height="18" rx="2"/></svg>
                        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">COLUMNS</span>
                    </button>
                    <!-- Dropdown Panel -->
                    <div
                        x-show="showColumnPicker"
                        @click.outside="showColumnPicker = false"
                        x-cloak
                        style="position: absolute; right: 0; top: 100%; margin-top: 4px; background-color: var(--bg-card); border: 1px solid var(--border-light); padding: 12px; z-index: 50; width: 220px; max-height: 400px; overflow-y: auto;"
                    >
                        <div style="font-family: 'Space Grotesk', sans-serif; font-size: 10px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); margin-bottom: 8px;">
                            VISIBLE COLUMNS
                        </div>
                        <!-- One checkbox per column -->
                        for _, col := range AddressColumns {
                            <label class="flex items-center" style="gap: 8px; padding: 6px 0; cursor: pointer;">
                                <input
                                    type="checkbox"
                                    :checked={ fmt.Sprintf("columns.%s", col.ID) }
                                    @change={ fmt.Sprintf("toggleColumn('%s')", col.ID) }
                                    style="accent-color: var(--terracotta);"
                                />
                                <span style="font-family: 'Inter', sans-serif; font-size: 12px; color: var(--text-primary);">
                                    { col.Label }
                                </span>
                            </label>
                        }
                    </div>
                </div>

                <!-- Add Address Button -->
                <a
                    href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s/new", data.ProjectID, data.AddressType)) }
                    class="flex items-center hover:opacity-90"
                    style="background-color: var(--bg-sidebar); padding: 10px 16px; gap: 8px; text-decoration: none;"
                >
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--text-light)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>
                    <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-light);">ADD ADDRESS</span>
                </a>
            </div>
        </div>
```

#### 3c: Action Bar (Import/Export/Template/Bulk Delete)

```templ
        <!-- Action Bar -->
        <div class="flex items-center justify-between" style="margin-top: 20px; padding: 12px 0;">
            <div class="flex items-center" style="gap: 12px;">
                <!-- Bulk Delete (visible when items selected) -->
                <button
                    x-show="selectedIds.length > 0"
                    x-cloak
                    type="button"
                    hx-delete={ fmt.Sprintf("/projects/%s/addresses/%s/bulk", data.ProjectID, data.AddressType) }
                    hx-target="#main-content"
                    hx-confirm="Are you sure you want to delete the selected addresses?"
                    :hx-vals="JSON.stringify({ids: selectedIds})"
                    class="flex items-center"
                    style="background-color: #FEE2E2; padding: 8px 14px; gap: 6px; border: 1px solid #EF4444; cursor: pointer;"
                >
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#DC2626" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>
                    <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: #DC2626;"
                        x-text="'DELETE (' + selectedIds.length + ')'"></span>
                </button>
            </div>
            <div class="flex items-center" style="gap: 12px;">
                <!-- Download Template -->
                <a
                    href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s/template", data.ProjectID, data.AddressType)) }
                    class="flex items-center"
                    style="background-color: var(--bg-card); padding: 8px 14px; gap: 6px; border: 1px solid var(--border-light); text-decoration: none; cursor: pointer;"
                >
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--text-secondary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" x2="12" y1="15" y2="3"/></svg>
                    <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">TEMPLATE</span>
                </a>
                <!-- Import from Excel -->
                <a
                    href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s/import", data.ProjectID, data.AddressType)) }
                    class="flex items-center"
                    style="background-color: var(--bg-card); padding: 8px 14px; gap: 6px; border: 1px solid var(--border-light); text-decoration: none; cursor: pointer;"
                >
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--text-secondary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" x2="12" y1="3" y2="15"/></svg>
                    <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">IMPORT</span>
                </a>
                <!-- Export to Excel -->
                <a
                    href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s/export", data.ProjectID, data.AddressType)) }
                    class="flex items-center"
                    style="background-color: var(--bg-card); padding: 8px 14px; gap: 6px; border: 1px solid var(--border-light); text-decoration: none; cursor: pointer;"
                >
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--text-secondary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" x2="12" y1="15" y2="3"/></svg>
                    <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">EXPORT</span>
                </a>
            </div>
        </div>
```

#### 3d: Table with Sortable Headers

Each column header is conditionally shown based on `columns.{columnId}` and triggers a sort request via HTMX. The current sort state is indicated by arrow icons.

```templ
        <!-- Address Table -->
        <div style="background-color: var(--bg-card); margin-top: 12px;">
            <!-- Table Header Row -->
            <div class="flex items-center" style="padding: 12px 16px; border-bottom: 1px solid var(--bg-sidebar);">
                <!-- Bulk Select All Checkbox -->
                <div style="width: 36px; min-width: 36px;">
                    <input type="checkbox" :checked="selectAll" @change="toggleSelectAll()" style="accent-color: var(--terracotta);" />
                </div>
                <!-- S.No -->
                <div style="width: 50px; min-width: 50px; font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-primary);">
                    S.NO
                </div>
                <!-- Dynamic columns - each sortable -->
                for _, col := range AddressColumns {
                    <div
                        x-show={ fmt.Sprintf("columns.%s", col.ID) }
                        class="flex-1 flex items-center cursor-pointer"
                        style="gap: 4px; min-width: 100px;"
                        hx-get={ fmt.Sprintf("/projects/%s/addresses/%s?sort_by=%s&sort_order=%s",
                            data.ProjectID, data.AddressType, col.ID,
                            toggleSortOrder(data.SortBy, col.ID, data.SortOrder)) }
                        hx-target="#main-content"
                        hx-push-url="true"
                    >
                        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-primary);">
                            { col.Label }
                        </span>
                        <!-- Sort indicator arrows -->
                        if data.SortBy == col.ID {
                            if data.SortOrder == "asc" {
                                <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="var(--terracotta)" stroke-width="2"><path d="m18 15-6-6-6 6"/></svg>
                            } else {
                                <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="var(--terracotta)" stroke-width="2"><path d="m6 9 6 6 6-6"/></svg>
                            }
                        }
                    </div>
                }
                <!-- Actions column (always visible) -->
                <div style="width: 80px; min-width: 80px; font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-primary);">
                    ACTIONS
                </div>
            </div>
```

**Helper function `toggleSortOrder`** - used in templ to flip sort direction when clicking the same column:

```go
// toggleSortOrder returns the opposite sort order if clicking the same column,
// otherwise returns "asc" as default for a new column.
func toggleSortOrder(currentSortBy, clickedCol, currentOrder string) string {
    if currentSortBy == clickedCol {
        if currentOrder == "asc" {
            return "desc"
        }
        return "asc"
    }
    return "asc"
}
```

#### 3e: Table Body Rows

```templ
            <!-- Table Body -->
            if len(data.Items) == 0 {
                <div class="flex justify-center items-center" style="padding: 48px 0; color: var(--text-muted); font-family: 'Inter', sans-serif; font-size: 14px;">
                    <div class="flex flex-col items-center" style="gap: 12px;">
                        <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"><path d="M20 10c0 4.993-5.539 10.193-7.399 11.799a1 1 0 0 1-1.202 0C9.539 20.193 4 14.993 4 10a8 8 0 0 1 16 0"/><circle cx="12" cy="10" r="3"/></svg>
                        <span>No { data.AddressLabel } addresses found</span>
                        <a
                            href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s/new", data.ProjectID, data.AddressType)) }
                            style="color: var(--terracotta); font-weight: 600; text-decoration: none;"
                        >
                            Add your first address
                        </a>
                    </div>
                </div>
            } else {
                for _, item := range data.Items {
                    <div class="flex items-center" style="padding: 12px 16px; border-bottom: 1px solid var(--border-light);">
                        <!-- Checkbox -->
                        <div style="width: 36px; min-width: 36px;">
                            <input
                                type="checkbox"
                                :checked={ fmt.Sprintf("isSelected('%s')", item.ID) }
                                @change={ fmt.Sprintf("toggleSelect('%s')", item.ID) }
                                style="accent-color: var(--terracotta);"
                            />
                        </div>
                        <!-- S.No -->
                        <div style="width: 50px; min-width: 50px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; font-weight: 500; color: var(--text-secondary);">
                            { strconv.Itoa(item.Index) }
                        </div>
                        <!-- Dynamic data columns -->
                        <div x-show="columns.company_name" class="flex-1" style="min-width: 100px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; font-weight: 600; color: var(--text-primary);">
                            { item.CompanyName }
                        </div>
                        <div x-show="columns.contact_person" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.ContactPerson }
                        </div>
                        <div x-show="columns.address_line_1" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.AddressLine1 }
                        </div>
                        <div x-show="columns.address_line_2" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.AddressLine2 }
                        </div>
                        <div x-show="columns.city" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.City }
                        </div>
                        <div x-show="columns.state" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.State }
                        </div>
                        <div x-show="columns.pin_code" class="flex-1" style="min-width: 100px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.PinCode }
                        </div>
                        <div x-show="columns.country" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.Country }
                        </div>
                        <div x-show="columns.phone" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.Phone }
                        </div>
                        <div x-show="columns.email" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.Email }
                        </div>
                        <div x-show="columns.gstin" class="flex-1" style="min-width: 100px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.GSTIN }
                        </div>
                        <div x-show="columns.pan" class="flex-1" style="min-width: 100px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.PAN }
                        </div>
                        <div x-show="columns.cin" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.CIN }
                        </div>
                        <div x-show="columns.website" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.Website }
                        </div>
                        <div x-show="columns.fax" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.Fax }
                        </div>
                        <div x-show="columns.landmark" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.Landmark }
                        </div>
                        <div x-show="columns.district" class="flex-1" style="min-width: 100px; font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                            { item.District }
                        </div>
                        <!-- Action Buttons -->
                        <div class="flex items-center" style="width: 80px; min-width: 80px; gap: 8px;">
                            <!-- Edit -->
                            <a
                                href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s/%s/edit", data.ProjectID, data.AddressType, item.ID)) }
                                style="color: var(--text-secondary);"
                                title="Edit"
                            >
                                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z"/></svg>
                            </a>
                            <!-- Delete -->
                            <button
                                type="button"
                                hx-delete={ fmt.Sprintf("/projects/%s/addresses/%s/%s", data.ProjectID, data.AddressType, item.ID) }
                                hx-target="#main-content"
                                hx-confirm="Are you sure you want to delete this address?"
                                style="background: none; border: none; cursor: pointer; padding: 0; color: var(--text-secondary);"
                                title="Delete"
                            >
                                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--error)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/><line x1="10" x2="10" y1="11" y2="17"/><line x1="14" x2="14" y1="11" y2="17"/></svg>
                            </button>
                        </div>
                    </div>
                }
            }
        </div>
```

#### 3f: Pagination Controls

```templ
        <!-- Pagination -->
        if data.TotalCount > 0 {
            <div class="flex items-center justify-between" style="margin-top: 16px; padding: 12px 0;">
                <!-- Showing X of Y -->
                <div style="font-family: 'Inter', sans-serif; font-size: 13px; color: var(--text-secondary);">
                    Showing { strconv.Itoa(((data.Page - 1) * data.PageSize) + 1) } - { strconv.Itoa(min(data.Page * data.PageSize, data.TotalCount)) } of { strconv.Itoa(data.TotalCount) }
                </div>
                <!-- Page Navigation -->
                <div class="flex items-center" style="gap: 4px;">
                    <!-- Previous -->
                    if data.HasPrev {
                        <button
                            type="button"
                            hx-get={ fmt.Sprintf("/projects/%s/addresses/%s?page=%d&page_size=%d&search=%s&sort_by=%s&sort_order=%s",
                                data.ProjectID, data.AddressType, data.Page-1, data.PageSize, data.Search, data.SortBy, data.SortOrder) }
                            hx-target="#main-content"
                            hx-push-url="true"
                            style="padding: 8px 12px; font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 600; color: var(--text-primary); background-color: var(--bg-card); border: 1px solid var(--border-light); cursor: pointer;"
                        >
                            PREV
                        </button>
                    } else {
                        <span style="padding: 8px 12px; font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 600; color: var(--text-muted); background-color: var(--bg-card); border: 1px solid var(--border-light);">
                            PREV
                        </span>
                    }
                    <!-- Page Numbers -->
                    for _, pn := range data.PageNumbers {
                        if pn == data.Page {
                            <span style="padding: 8px 12px; font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 700; color: var(--text-light); background-color: var(--terracotta);">
                                { strconv.Itoa(pn) }
                            </span>
                        } else {
                            <button
                                type="button"
                                hx-get={ fmt.Sprintf("/projects/%s/addresses/%s?page=%d&page_size=%d&search=%s&sort_by=%s&sort_order=%s",
                                    data.ProjectID, data.AddressType, pn, data.PageSize, data.Search, data.SortBy, data.SortOrder) }
                                hx-target="#main-content"
                                hx-push-url="true"
                                style="padding: 8px 12px; font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 600; color: var(--text-primary); background-color: var(--bg-card); border: 1px solid var(--border-light); cursor: pointer;"
                            >
                                { strconv.Itoa(pn) }
                            </button>
                        }
                    }
                    <!-- Next -->
                    if data.HasNext {
                        <button
                            type="button"
                            hx-get={ fmt.Sprintf("/projects/%s/addresses/%s?page=%d&page_size=%d&search=%s&sort_by=%s&sort_order=%s",
                                data.ProjectID, data.AddressType, data.Page+1, data.PageSize, data.Search, data.SortBy, data.SortOrder) }
                            hx-target="#main-content"
                            hx-push-url="true"
                            style="padding: 8px 12px; font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 600; color: var(--text-primary); background-color: var(--bg-card); border: 1px solid var(--border-light); cursor: pointer;"
                        >
                            NEXT
                        </button>
                    } else {
                        <span style="padding: 8px 12px; font-family: 'Space Grotesk', sans-serif; font-size: 12px; font-weight: 600; color: var(--text-muted); background-color: var(--bg-card); border: 1px solid var(--border-light);">
                            NEXT
                        </span>
                    }
                </div>
            </div>
        }
    </div>
}
```

### Step 4: Implement AddressListPage (Full Page Wrapper)

Follows the `BOQListPage` pattern exactly:

```templ
templ AddressListPage(data AddressListData) {
    @PageShell(data.AddressLabel + " Addresses - Project Creation") {
        @AddressListContent(data)
    }
}
```

### Step 5: Helper Function for Building Item IDs in JavaScript

This is needed for the "select all" Alpine.js functionality:

```go
// buildItemIDsJS generates a JavaScript array literal string from address items.
// Example output: "'abc123','def456','ghi789'
func buildItemIDsJS(items []AddressListItem) string {
    if len(items) == 0 {
        return ""
    }
    var parts []string
    for _, item := range items {
        parts = append(parts, fmt.Sprintf("'%s'", item.ID))
    }
    return strings.Join(parts, ",")
}
```

---

## Column Visibility Behavior

### Default Visible Columns (8 columns)

| # | Column | Description |
|---|--------|-------------|
| 1 | S.No | Row index (always visible, not toggleable) |
| 2 | Company Name | Primary identifier |
| 3 | Contact Person | Main contact |
| 4 | City | Location |
| 5 | State | Location |
| 6 | PIN Code | Postal code |
| 7 | GSTIN | Tax registration |
| 8 | Actions | Edit/Delete buttons (always visible, not toggleable) |

### Hidden by Default (10 columns)

Address Line 1, Address Line 2, Country, Phone, Email, PAN, CIN, Website, Fax, Landmark, District

### localStorage Key Format

```
addr_cols_{address_type}
```

Example: `addr_cols_bill_from`, `addr_cols_ship_to`

Each key stores a JSON object like:
```json
{
    "company_name": true,
    "contact_person": true,
    "city": true,
    "state": true,
    "pin_code": true,
    "gstin": true,
    "address_line_1": false,
    "address_line_2": false,
    "country": false,
    "phone": false,
    "email": false,
    "pan": false,
    "cin": false,
    "website": false,
    "fax": false,
    "landmark": false,
    "district": false
}
```

---

## HTMX Interaction Patterns

| Interaction | Trigger | Target | URL |
|------------|---------|--------|-----|
| Search | `keyup changed delay:300ms` | `#main-content` | `GET /projects/{id}/addresses/{type}?search=...` |
| Sort column | Click header | `#main-content` | `GET /projects/{id}/addresses/{type}?sort_by=...&sort_order=...` |
| Pagination | Click page button | `#main-content` | `GET /projects/{id}/addresses/{type}?page=...` |
| Delete single | Click delete icon | `#main-content` | `DELETE /projects/{id}/addresses/{type}/{addressId}` |
| Bulk delete | Click bulk delete | `#main-content` | `DELETE /projects/{id}/addresses/{type}/bulk` |

All HTMX requests include `hx-push-url="true"` so the browser URL stays in sync.

---

## Dependencies on Other Phases

| Dependency | Phase | Required For |
|-----------|-------|--------------|
| `HandleAddressList` handler | Phase 7 | Populates `AddressListData` and serves the template |
| `PageShell` template | Existing | Full page wrapper with sidebar and header |
| `addresses` collection | Phase 5/6 | Data source |
| `projects` collection | Phase 1-3 | Project name in breadcrumbs |
| Alpine.js 3.x | Existing (layout.templ) | Column toggling, bulk selection |
| HTMX 2.x | Existing (layout.templ) | Search debounce, pagination, delete actions |

---

## Testing / Verification Steps

### Visual Testing

1. **Load address list page** - verify layout matches existing BOQ list style (same fonts, colors, spacing)
2. **Empty state** - with no addresses, verify the empty state message and "Add your first address" link appear
3. **Column visibility** - toggle columns on/off via dropdown, verify columns hide/show instantly
4. **localStorage persistence** - toggle columns, refresh page, verify selections persist
5. **Different address types** - verify each type has its own localStorage key (toggle columns for bill_from, check ship_to is unaffected)

### Search Testing

6. **Search debounce** - type in search box, verify request fires after 300ms pause (not on every keystroke)
7. **Search results** - search for a city name, verify only matching addresses appear
8. **Clear search** - clear the search box, verify all addresses return

### Sort Testing

9. **Click column header** - verify sort direction changes and arrow indicator updates
10. **Sort persistence** - sort by city desc, then paginate, verify sort is maintained across pages

### Pagination Testing

11. **Page navigation** - click page numbers, verify correct page loads
12. **Previous/Next** - verify disabled state on first/last page
13. **Showing X of Y** - verify counts are accurate

### Bulk Operations Testing

14. **Select individual** - check a row checkbox, verify selectedIds updates
15. **Select all** - click header checkbox, verify all visible rows selected
16. **Bulk delete button** - appears only when items selected, shows count
17. **Confirmation dialog** - bulk delete shows browser confirm dialog

### HTMX Testing

18. **Partial rendering** - send request with `HX-Request: true` header, verify only content partial returned (no full page shell)
19. **Full page rendering** - direct browser navigation, verify full page with sidebar renders

---

## Acceptance Criteria

- [ ] `templates/address_list.templ` compiles via `templ generate` without errors
- [ ] `AddressListPage` wraps content in `PageShell` for full-page loads
- [ ] `AddressListContent` renders as standalone partial for HTMX requests
- [ ] Column selector dropdown shows all 17 data columns with checkboxes
- [ ] Default visible columns: Company Name, Contact Person, City, State, PIN Code, GSTIN
- [ ] Column visibility toggles stored in localStorage keyed by address type
- [ ] S.No and Actions columns are always visible and not in the column picker
- [ ] Table headers are clickable and trigger server-side sort via HTMX
- [ ] Current sort column shows directional arrow indicator (up for asc, down for desc)
- [ ] Clicking same column toggles sort direction; clicking new column sorts asc
- [ ] Search input has `hx-trigger="keyup changed delay:300ms"` for debounced search
- [ ] Pagination shows Previous, page numbers, and Next buttons
- [ ] Current page number highlighted with terracotta background
- [ ] Previous disabled on page 1; Next disabled on last page
- [ ] "Showing X - Y of Z" text accurately reflects current view
- [ ] Each row has Edit link (navigates to edit form) and Delete button (HTMX DELETE with confirm)
- [ ] Bulk select checkbox in header selects/deselects all visible rows
- [ ] Bulk delete button appears only when items are selected, shows count
- [ ] Empty state shows icon, message, and link to add first address
- [ ] Breadcrumbs show: PROJECTS / {Project Name} / {Address Type Label}
- [ ] Action bar includes: Add Address, Import, Export, and Template buttons
- [ ] All URLs correctly include projectId and address type slug
