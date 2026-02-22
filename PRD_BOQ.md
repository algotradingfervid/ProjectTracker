# PRD: Bill of Quantities (BOQ) Management Page

## 1. Overview

### 1.1 Purpose
The BOQ Management Page is the first feature of a Project Creation Tool. It allows Project Managers to create, view, and edit Bills of Quantities for projects. A BOQ captures all line items from a purchase order along with their cost breakdowns via sub-items.

### 1.2 Tech Stack

| Layer | Technology | Version | Notes |
|-------|-----------|---------|-------|
| **Backend** | PocketBase (embedded Go binary) | **v0.36.5** | Released Feb 21, 2025. Uses embedded SQLite. |
| **Go** | Go toolchain | **1.23+** | Required by PocketBase v0.36.x |
| **Templating** | templ | **v0.3.977** | Type-safe compiled HTML templates. First-class HTMX support. |
| **Frontend** | HTMX | **v2.0.7** | HTML-over-the-wire. CDN: `https://cdn.jsdelivr.net/npm/htmx.org@2.0.7/dist/htmx.min.js` |
| **Client-side UI** | Alpine.js | **v3.15.8** | Accordion toggles, client-side filtering, edit-mode state. CDN: `https://cdn.jsdelivr.net/npm/alpinejs@3.15.8/dist/cdn.min.js` |
| **Styling** | Tailwind CSS | **v4.1.18** | CSS-first config via `@import "tailwindcss"` |
| **Components** | DaisyUI | **v5.5.19** | Pre-built table, accordion, modal, badge, stat, toast components |
| **Excel Export** | Excelize | **v2.9.1** | Pure Go .xlsx generation. `github.com/xuri/excelize/v2` |
| **CSV Export** | encoding/csv | stdlib | Go standard library |
| **PDF Export** | Maroto | **v2.3.3** | Bootstrap-style grid layout for invoice/BOQ PDFs. `github.com/johnfercher/maroto/v2` |
| **Number Formatting** | golang.org/x/text | latest | Indian locale (en-IN) for ₹1,23,456.78 formatting |
| **Deployment** | Local development for now; single Go binary deployment later |

### 1.3 HTMX Extensions

| Extension | Purpose | CDN |
|-----------|---------|-----|
| **idiomorph** | Morph-based DOM merging (preserves focus/scroll during swaps) | `https://unpkg.com/idiomorph@0.7.4/dist/idiomorph-ext.min.js` |
| **response-targets** | Swap different targets based on HTTP status (validation errors) | `https://cdn.jsdelivr.net/npm/htmx-ext-response-targets` |
| **loading-states** | Loading spinners, disable buttons during requests | `https://cdn.jsdelivr.net/npm/htmx-ext-loading-states` |
| **json-enc** | Encode form data as JSON for batch save | `https://cdn.jsdelivr.net/npm/htmx-ext-json-enc` |

### 1.4 Why These Choices

- **templ over html/template**: Compile-time type safety, IDE autocompletion, cleaner syntax for HTMX partial responses. Community starter projects exist for PocketBase + templ + HTMX + Tailwind.
- **Alpine.js alongside HTMX**: HTMX handles server communication; Alpine.js handles client-side state (accordion open/close, edit-mode toggle, unsaved-changes tracking, search filtering). No build step needed for either.
- **idiomorph extension**: Critical for edit mode — when the server returns updated price cells after recalculation, idiomorph preserves input focus and cursor position.
- **Excelize over alternatives**: 20k+ GitHub stars, pure Go, supports styling/formatting needed for professional BOQ exports.
- **Maroto for PDF**: Grid-based layout is ideal for tabular BOQ/invoice documents with automatic page breaks.

### 1.5 Scope
- Standalone application for now, designed to integrate into a larger project management system later
- No authentication or role-based access control in this phase
- No approval workflows in this phase

---

## 2. Users

| Role | Description |
|------|-------------|
| Project Manager | Primary user. Creates, edits, and manages BOQs for projects. |

---

## 3. Data Model

### 3.1 BOQ (Header)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string (auto) | Yes | PocketBase auto-generated ID |
| title | string | Yes | BOQ name/title |
| reference_number | string | No | PO or reference number |
| created_date | datetime | Yes | Auto-set on creation |
| updated_date | datetime | Yes | Auto-updated on modification |

**Constraints**: One BOQ per project (project entity to be introduced later).

### 3.2 Main BOQ Line Item

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string (auto) | Yes | PocketBase auto-generated ID |
| boq_id | relation | Yes | FK to BOQ |
| sort_order | int | Yes | Display order |
| description | string | Yes | Line item description |
| qty | number | Yes | Quantity |
| uom | string | Yes | Unit of Measure (dropdown + custom) |
| quoted_price | number | Yes | Client-facing price (from PO) |
| budgeted_price | number | Computed | Our internal cost. Auto-calculated as sum of sub-item budgeted prices. Manually set if no sub-items. |
| hsn_code | string | No | HSN/SAC code (optional) |
| gst_percent | number | Yes | GST percentage |

**Pricing Logic**:
- If sub-items exist: `budgeted_price = Σ(sub-item qty_per_unit × sub-item unit_price)`
- This is the **per-unit** budgeted cost
- Total for line item = `budgeted_price × qty`
- If no sub-items: `budgeted_price` is manually entered

### 3.3 Sub-Item (Level 1)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string (auto) | Yes | PocketBase auto-generated ID |
| main_item_id | relation | Yes | FK to Main BOQ Line Item |
| sort_order | int | Yes | Display order |
| type | string | Yes | "product" or "service" (label only, no functional difference) |
| description | string | Yes | Sub-item description |
| qty_per_unit | number | Yes | Quantity per 1 unit of the parent main item |
| uom | string | Yes | Unit of Measure |
| unit_price | number | Yes | Price per unit of sub-item |
| budgeted_price | number | Computed | `qty_per_unit × unit_price`. Auto-calculated from sub-sub-items if they exist. |
| hsn_code | string | No | HSN/SAC code (optional) |
| gst_percent | number | Yes | GST percentage |

### 3.4 Sub-Sub-Item (Level 2)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string (auto) | Yes | PocketBase auto-generated ID |
| sub_item_id | relation | Yes | FK to Sub-Item |
| sort_order | int | Yes | Display order |
| type | string | Yes | "product" or "service" |
| description | string | Yes | Description |
| qty_per_unit | number | Yes | Quantity per 1 unit of the parent sub-item |
| uom | string | Yes | Unit of Measure |
| unit_price | number | Yes | Price per unit |
| budgeted_price | number | Computed | `qty_per_unit × unit_price` |
| hsn_code | string | No | HSN/SAC code (optional) |
| gst_percent | number | Yes | GST percentage |

### 3.5 Hierarchy Summary

```
BOQ (header: title, ref number, date)
 └── Main BOQ Line Item (qty, description, uom, quoted_price, budgeted_price, hsn, gst%)
      └── Sub-Item L1 (type, qty_per_unit, description, uom, unit_price, hsn, gst%)
           └── Sub-Sub-Item L2 (type, qty_per_unit, description, uom, unit_price, hsn, gst%)
```

**Max depth**: 3 levels (Main → Sub-Item → Sub-Sub-Item)

---

## 4. Price Calculation Rules

### 4.1 Bottom-Up Auto-Calculation

All budgeted prices are auto-recalculated in real-time when any child item changes.

```
Sub-Sub-Item budgeted_price = qty_per_unit × unit_price

Sub-Item budgeted_price:
  - If has sub-sub-items: Σ(sub-sub-item budgeted_price)
  - If no sub-sub-items: qty_per_unit × unit_price

Main Item budgeted_price (per unit):
  - If has sub-items: Σ(sub-item budgeted_price)
  - If no sub-items: manually entered

Main Item total = budgeted_price × qty

BOQ Total Quoted = Σ(main_item quoted_price × main_item qty)
BOQ Total Budgeted = Σ(main_item budgeted_price × main_item qty)
```

### 4.2 Example

```
Main BOQ Line Item: Qty=10, Description="Wall Work"
  Sub-item 1: qty/unit=2, unit_price=100 → budgeted_price = 200
  Sub-item 2: qty/unit=1, unit_price=50  → budgeted_price =  50

Main item budgeted_price = 250 (per unit)
Main item total = 250 × 10 = 2,500
```

---

## 5. UOM & GST Dropdowns

### 5.1 UOM Options (Dropdown + Custom)
Predefined list with option to add custom values:
- Nos, Sqm, Sqft, Rmt, Cum, Kg, MT, Lot, Set, Lumpsum, Ltr, Pair, Bag, Box, Roll, Bundle, Trip, Day, Month, Hour

### 5.2 GST Options (Dropdown + Custom)
Standard Indian GST slabs with option for custom entry:
- 0%, 5%, 12%, 18%, 28%
- Custom: free-text numeric input

---

## 6. UI/UX Design

### 6.1 Page Structure

```
┌──────────────────────────────────────────────────────┐
│  BOQ Header                                          │
│  [Title]  [Ref #]  [Date]         [Edit BOQ] button  │
├──────────────────────────────────────────────────────┤
│  Search/Filter bar                                    │
├──────────────────────────────────────────────────────┤
│  Accordion Table                                      │
│  ┌─ # ─┬─ Description ─┬─ Qty ─┬─ UOM ─┬─ ... ──┐  │
│  │  1  │ Wall Work      │  10  │ Sqm   │  ...   │  │
│  │  ▼  │                │      │       │        │  │
│  │     │ ┌ Sub-items ─────────────────────────┐  │  │
│  │     │ │ S1: Cement    │ 2/u  │ Bag  │ ...  │  │  │
│  │     │ │ S2: Labour    │ 1/u  │ Day  │ ...  │  │  │
│  │     │ └───────────────────────────────────┘  │  │
│  │  2  │ Plumbing       │  5   │ Lot   │  ...   │  │
│  └─────┴────────────────┴──────┴───────┴────────┘  │
├──────────────────────────────────────────────────────┤
│  Summary: Total Quoted: ₹X,XX,XXX                    │
│           Total Budgeted: ₹X,XX,XXX                  │
└──────────────────────────────────────────────────────┘
```

### 6.2 View Mode (Default)
- Read-only accordion table showing all fields
- All cells display plain text (non-editable)
- Expand/collapse arrows on main items to show/hide sub-items
- Sub-items show nested under their parent with visual indentation
- Sub-sub-items show nested under their parent sub-item with further indentation
- **"Edit BOQ"** button in the header to switch to edit mode
- Summary totals (Total Quoted Price, Total Budgeted Price) at the bottom

### 6.3 Edit Mode
- Activated by clicking **"Edit BOQ"** button → button changes to **"Save BOQ"** / **"Cancel"**
- All fields become editable (input fields, dropdowns replace plain text)
- **"Add Main Item"** button appears at the bottom of the main table
- **"Add Sub-Item"** button appears within each expanded main item
- **"Add Sub-Sub-Item"** button appears within each expanded sub-item
- Delete buttons (✕) appear on each row
- Budgeted prices update in real-time as sub-item values change
- Reordering via drag-and-drop or up/down arrows (nice-to-have)

### 6.4 Search & Filter
- Text search across descriptions
- Filter by item type (product/service) for sub-items
- Filter/search should work in both view and edit modes

---

## 7. HTMX Interaction Patterns

### 7.1 Page Load
- Full page render with BOQ header + accordion table in view mode
- All main items loaded; sub-items loaded on expand (lazy) or all at once (eager, for small BOQs)

### 7.2 Toggle Edit Mode
- `hx-get="/boq/{id}/edit"` → returns the entire BOQ table in edit mode (all fields as inputs)
- `hx-get="/boq/{id}/view"` → returns the BOQ table in view mode (read-only)
- Target: the BOQ table container
- Swap: `innerHTML`

### 7.3 Expand/Collapse Sub-Items
- `hx-get="/boq/{id}/main-item/{item_id}/subitems"` → returns sub-item rows
- Uses `hx-target` to insert into the sub-items container under the main item

### 7.4 Add Line Item
- `hx-post="/boq/{id}/main-items"` → creates a new empty main item, returns the new row HTML
- `hx-post="/boq/{id}/main-item/{item_id}/subitems"` → creates new sub-item
- `hx-post="/boq/{id}/subitem/{subitem_id}/subsubitems"` → creates new sub-sub-item
- Append new row to the table using `hx-swap="beforeend"`

### 7.5 Update Field
- On change/blur of any input field:
  - `hx-patch="/boq/{id}/main-item/{item_id}"` with changed field value
  - Server recalculates budgeted prices and returns updated price cells
  - Use `hx-swap="outerHTML"` on relevant price display elements
- Alternatively: batch save all changes on "Save BOQ" click

### 7.6 Delete Item
- `hx-delete="/boq/{id}/main-item/{item_id}"` → cascade deletes all sub-items
- Confirmation via browser `confirm()` or DaisyUI modal
- Removes the row from DOM via `hx-swap="delete"`

---

## 8. API Endpoints (PocketBase Custom Routes)

### 8.1 BOQ
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/boq` | List all BOQs |
| POST | `/boq` | Create new BOQ |
| GET | `/boq/{id}` | Get BOQ with all line items (view mode HTML) |
| GET | `/boq/{id}/edit` | Get BOQ in edit mode (editable HTML) |
| PATCH | `/boq/{id}` | Update BOQ header fields |
| DELETE | `/boq/{id}` | Delete BOQ and all children |

### 8.2 Main BOQ Line Items
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/boq/{id}/main-items` | Add new main line item |
| PATCH | `/boq/{id}/main-item/{item_id}` | Update main line item |
| DELETE | `/boq/{id}/main-item/{item_id}` | Delete main item + cascade sub-items |

### 8.3 Sub-Items (Level 1)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/boq/{id}/main-item/{item_id}/subitems` | Get sub-items for a main item |
| POST | `/boq/{id}/main-item/{item_id}/subitems` | Add sub-item |
| PATCH | `/boq/{id}/subitem/{subitem_id}` | Update sub-item |
| DELETE | `/boq/{id}/subitem/{subitem_id}` | Delete sub-item + cascade |

### 8.4 Sub-Sub-Items (Level 2)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/boq/{id}/subitem/{subitem_id}/subsubitems` | Get sub-sub-items |
| POST | `/boq/{id}/subitem/{subitem_id}/subsubitems` | Add sub-sub-item |
| PATCH | `/boq/{id}/subsubitem/{id}` | Update sub-sub-item |
| DELETE | `/boq/{id}/subsubitem/{id}` | Delete sub-sub-item |

### 8.5 Export
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/boq/{id}/export/excel` | Export BOQ as Excel file |
| GET | `/boq/{id}/export/pdf` | Export BOQ as PDF file |

---

## 9. PocketBase Collections

### 9.1 `boqs`
```json
{
  "name": "boqs",
  "type": "base",
  "fields": [
    { "name": "title", "type": "text", "required": true },
    { "name": "reference_number", "type": "text", "required": false },
  ]
}
```

### 9.2 `main_boq_items`
```json
{
  "name": "main_boq_items",
  "type": "base",
  "fields": [
    { "name": "boq", "type": "relation", "required": true, "options": { "collectionId": "boqs", "cascadeDelete": true } },
    { "name": "sort_order", "type": "number", "required": true },
    { "name": "description", "type": "text", "required": true },
    { "name": "qty", "type": "number", "required": true },
    { "name": "uom", "type": "text", "required": true },
    { "name": "quoted_price", "type": "number", "required": true },
    { "name": "budgeted_price", "type": "number", "required": false },
    { "name": "hsn_code", "type": "text", "required": false },
    { "name": "gst_percent", "type": "number", "required": true }
  ]
}
```

### 9.3 `sub_items`
```json
{
  "name": "sub_items",
  "type": "base",
  "fields": [
    { "name": "main_item", "type": "relation", "required": true, "options": { "collectionId": "main_boq_items", "cascadeDelete": true } },
    { "name": "sort_order", "type": "number", "required": true },
    { "name": "type", "type": "select", "required": true, "options": { "values": ["product", "service"] } },
    { "name": "description", "type": "text", "required": true },
    { "name": "qty_per_unit", "type": "number", "required": true },
    { "name": "uom", "type": "text", "required": true },
    { "name": "unit_price", "type": "number", "required": true },
    { "name": "budgeted_price", "type": "number", "required": false },
    { "name": "hsn_code", "type": "text", "required": false },
    { "name": "gst_percent", "type": "number", "required": true }
  ]
}
```

### 9.4 `sub_sub_items`
```json
{
  "name": "sub_sub_items",
  "type": "base",
  "fields": [
    { "name": "sub_item", "type": "relation", "required": true, "options": { "collectionId": "sub_items", "cascadeDelete": true } },
    { "name": "sort_order", "type": "number", "required": true },
    { "name": "type", "type": "select", "required": true, "options": { "values": ["product", "service"] } },
    { "name": "description", "type": "text", "required": true },
    { "name": "qty_per_unit", "type": "number", "required": true },
    { "name": "uom", "type": "text", "required": true },
    { "name": "unit_price", "type": "number", "required": true },
    { "name": "budgeted_price", "type": "number", "required": false },
    { "name": "hsn_code", "type": "text", "required": false },
    { "name": "gst_percent", "type": "number", "required": true }
  ]
}
```

---

## 10. Features Summary

### 10.1 MVP (This Phase)
- [x] Create, view, edit BOQ with header fields (title, ref #, date)
- [x] Add/edit/delete Main BOQ Line Items
- [x] Add/edit/delete Sub-Items (Level 1) under main items
- [x] Add/edit/delete Sub-Sub-Items (Level 2) under sub-items
- [x] Toggle between view mode (read-only) and edit mode (all editable)
- [x] Accordion table UI with expand/collapse for nested items
- [x] Auto-calculation of budgeted prices (bottom-up)
- [x] UOM dropdown with custom entry
- [x] GST% dropdown with custom entry
- [x] Product/Service type label on sub-items
- [x] Search and filter line items
- [x] Summary totals (Total Quoted, Total Budgeted)
- [x] Export to Excel
- [x] Export to PDF
- [x] Cascade delete (deleting parent removes all children)

### 10.2 Future Considerations (Not in Scope)
- User authentication and role-based access
- Approval workflows (draft → approved)
- Import BOQ from purchase order documents
- BOQ templates
- Version history / audit trail
- Duplicate/clone BOQ
- Project entity linking (one BOQ per project)
- Drag-and-drop reordering of line items
- Multi-currency support

---

## 11. Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| Scale | Should handle BOQs with 500+ main line items smoothly |
| Performance | Page load < 2s, field updates < 500ms |
| Browser Support | Modern browsers (Chrome, Firefox, Safari, Edge) |
| Data Persistence | PocketBase embedded SQLite |
| Offline Support | Not required |
| Mobile Responsive | Nice-to-have, not required for MVP |

---

## 12. Resolved Design Decisions

1. **Save Behavior**: Batch save — all changes held in memory until user clicks "Save BOQ", then sent to server at once. Unsaved changes should show a visual indicator (e.g., "unsaved changes" badge). Navigating away with unsaved changes should trigger a browser warning.
2. **Export Format**: Both .xlsx (formatted with headers, borders, hierarchy indentation) and .csv available. User chooses format via dropdown/buttons on the export action.
3. **Search Scope**: Search/filter works across all levels (main items, sub-items, sub-sub-items). If a child matches, its parent chain is shown expanded.
4. **Currency**: ₹ INR with Indian number formatting (lakhs/crores) — e.g., ₹1,23,456.78
