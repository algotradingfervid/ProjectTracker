# Delivery Challans (DC) Feature Design

**Date**: 2026-03-10
**Status**: Approved
**Approach**: Type-Driven Configuration Model

## Overview

Delivery Challans are internal process documents for tracking goods movement. They mirror e-way bill data models (parties, items, transport) but are not statutory. DCs can be linked to a project (pulling data from BOQ items and project address masters) or standalone (manual entry with a separate address book).

## Requirements Summary

| Aspect | Decision |
|--------|----------|
| Scope | Project-linked DCs + non-project DCs |
| DC Types | Configurable via `dc_type_configs` collection |
| Numbering | Auto-generated: `{prefix}-{type_code}-{FY}-{seq}`, configurable prefix & type codes, ability to start new series |
| Chain | Transfer DC -> 1 Transit DC + multiple Official DCs |
| Line Items | From BOQ items (project DCs) or manual (non-project); soft over-dispatch warning with reason capture |
| Financials | Priced or unpriced per DC type config |
| Serial Numbers | Mandatory for certain item categories (configurable per DC type) |
| Statuses | Draft -> Finalized -> Dispatched -> Delivered |
| E-way Bill | Manual entry field only, no API integration |
| Navigation | Both project-level sidebar AND global top-level access |
| PDF Export | Replicate existing DC sample layout exactly |
| Addresses | Project DCs use project address masters; non-project DCs use standalone address book |

## Data Model

### Collection: `dc_type_configs`

Defines each DC type and its behavior.

| Field | Type | Notes |
|-------|------|-------|
| `name` | Text (required) | Display name, e.g., "Transfer Delivery Challan" |
| `code` | Text (required, unique) | Short code, e.g., "TDC", "ODC", "RDC" |
| `is_priced` | Bool | Whether rate/taxable/GST fields appear |
| `is_multi_destination` | Bool | Whether destination breakdown annexure is enabled |
| `requires_serial_numbers` | Bool | Whether serial numbers are mandatory for items |
| `allows_parent_dc` | Bool | Whether this type can reference a parent DC |
| `description` | Text | Optional description of when to use this type |
| `active` | Bool | Soft disable without deleting |

### Collection: `dc_number_series`

Manages auto-numbering per prefix + type combination.

| Field | Type | Notes |
|-------|------|-------|
| `prefix` | Text (required) | Company prefix, e.g., "OAVS" |
| `dc_type` | Relation -> dc_type_configs | Which DC type this series is for |
| `financial_year` | Text (required) | e.g., "2526" (2025-26) |
| `last_number` | Number | Current counter, incremented on each DC |
| `active` | Bool | Only one active series per prefix+type+FY |

Generated format: `{prefix}-{type_code}-{financial_year}-{seq padded to 3}`
Example: `OAVS-TDC-2526-013`

### Collection: `delivery_challans`

The main DC record.

| Field | Type | Notes |
|-------|------|-------|
| `dc_number` | Text (required, unique) | Auto-generated from series |
| `dc_type` | Relation -> dc_type_configs (required) | Determines behavior |
| `dc_date` | Text (required) | Date of challan |
| `status` | Select (required) | draft, finalized, dispatched, delivered |
| `project` | Relation -> projects | Null for non-project DCs |
| `parent_dc` | Relation -> delivery_challans | Self-relation for chain |
| `po_number` | Text | Reference PO number |
| `po_date` | Text | Reference PO date |
| `bill_from_name` | Text | |
| `bill_from_address1` | Text | |
| `bill_from_address2` | Text | |
| `bill_from_city` | Text | |
| `bill_from_state` | Text | |
| `bill_from_pincode` | Text | |
| `bill_from_gstin` | Text | |
| `bill_from_contact` | Text | |
| `bill_from_phone` | Text | |
| `bill_to_name` | Text | Same pattern as bill_from |
| `bill_to_address1` | Text | |
| `bill_to_address2` | Text | |
| `bill_to_city` | Text | |
| `bill_to_state` | Text | |
| `bill_to_pincode` | Text | |
| `bill_to_gstin` | Text | |
| `bill_to_contact` | Text | |
| `bill_to_phone` | Text | |
| `dispatch_from_name` | Text | Same pattern |
| `dispatch_from_address1` | Text | |
| `dispatch_from_address2` | Text | |
| `dispatch_from_city` | Text | |
| `dispatch_from_state` | Text | |
| `dispatch_from_pincode` | Text | |
| `dispatch_from_gstin` | Text | |
| `dispatch_from_contact` | Text | |
| `dispatch_from_phone` | Text | |
| `ship_to_name` | Text | Same pattern |
| `ship_to_address1` | Text | |
| `ship_to_address2` | Text | |
| `ship_to_city` | Text | |
| `ship_to_state` | Text | |
| `ship_to_pincode` | Text | |
| `ship_to_gstin` | Text | |
| `ship_to_contact` | Text | |
| `ship_to_phone` | Text | |
| `transporter_name` | Text | |
| `vehicle_number` | Text | |
| `eway_bill_number` | Text | Manual entry |
| `docket_number` | Text | |
| `reverse_charge` | Bool | |
| `transport_mode` | Select | road, rail, air, ship |
| `total_taxable_value` | Number | Priced DCs only |
| `total_cgst` | Number | |
| `total_sgst` | Number | |
| `total_igst` | Number | |
| `round_off` | Number | |
| `invoice_value` | Number | Grand total |
| `over_dispatch_reason` | Text | Captured when qty exceeds BOQ |
| `notes` | Text | General remarks |

Design decision: Address fields are flattened (denormalized) on the DC record rather than relations to address masters. A DC is a point-in-time document -- if someone updates an address master later, the DC should still show the address as it was when created.

### Collection: `dc_line_items`

| Field | Type | Notes |
|-------|------|-------|
| `dc` | Relation -> delivery_challans (cascade delete) | Parent DC |
| `sort_order` | Number | Display sequence |
| `description` | Text (required) | Product description |
| `make_model` | Text | Make/model info |
| `uom` | Text | Unit of measurement |
| `hsn_code` | Text | HSN code |
| `qty` | Number (required) | Quantity |
| `rate` | Number | Unit rate (priced DCs only) |
| `taxable_value` | Number | qty x rate |
| `gst_percent` | Number | GST percentage |
| `gst_amount` | Number | Calculated GST |
| `total` | Number | taxable + GST |
| `serial_numbers` | Text | Comma-separated or JSON array |
| `remarks` | Text | Per-item remarks |
| `boq_item_id` | Text | Reference to source BOQ item |
| `boq_item_level` | Text | "main", "sub", or "sub_sub" |

### Collection: `dc_destinations` (for multi-destination DCs)

| Field | Type | Notes |
|-------|------|-------|
| `dc` | Relation -> delivery_challans (cascade delete) | Parent DC |
| `destination_name` | Text | Location name/code |
| `destination_address` | Text | Full address |
| `destination_city` | Text | |
| `destination_state` | Text | |
| `destination_pincode` | Text | |
| `contact_person` | Text | |
| `contact_phone` | Text | |

### Collection: `dc_destination_items`

Qty allocation per destination per item.

| Field | Type | Notes |
|-------|------|-------|
| `dc_destination` | Relation -> dc_destinations (cascade delete) | |
| `dc_line_item` | Relation -> dc_line_items (cascade delete) | |
| `qty` | Number | Quantity allocated to this destination |

### Collection: `standalone_addresses`

Address book for non-project DCs.

| Field | Type | Notes |
|-------|------|-------|
| `address_type` | Select | bill_from, bill_to, dispatch_from, ship_to |
| `name` | Text (required) | |
| `address1` | Text | |
| `address2` | Text | |
| `city` | Text | |
| `state` | Text | |
| `pincode` | Text | |
| `gstin` | Text | |
| `contact_person` | Text | |
| `phone` | Text | |
| `email` | Text | |

## Routes

```
# Global DC routes
GET    /dcs                              -> DC list (all DCs, filterable)
GET    /dcs/create                       -> DC create form
POST   /dcs                              -> DC save
GET    /dcs/{id}                         -> DC view
GET    /dcs/{id}/edit                    -> DC edit form
POST   /dcs/{id}/save                   -> DC update
DELETE /dcs/{id}                         -> DC delete
GET    /dcs/{id}/export/pdf              -> DC PDF export

# Project-scoped DC routes
GET    /projects/{projectId}/dcs         -> DC list filtered by project
GET    /projects/{projectId}/dcs/create  -> DC create with project pre-selected
POST   /projects/{projectId}/dcs         -> DC save with project
GET    /projects/{projectId}/dcs/{id}    -> DC view within project context

# DC line items (HTMX partials)
POST   /dcs/{id}/line-items             -> Add line item
PUT    /dcs/{id}/line-items/{itemId}    -> Update line item
DELETE /dcs/{id}/line-items/{itemId}    -> Remove line item

# BOQ item lookup (HTMX endpoint)
GET    /projects/{projectId}/boq-items-lookup -> Search BOQ items with available qty

# DC type config (settings)
GET    /settings/dc-types                -> List/manage DC types
POST   /settings/dc-types               -> Create DC type
GET    /settings/dc-types/{id}/edit      -> Edit DC type
POST   /settings/dc-types/{id}/save     -> Update DC type

# DC number series (settings)
GET    /settings/dc-series               -> List/manage number series
POST   /settings/dc-series              -> Create series
POST   /settings/dc-series/{id}/save    -> Update series

# Standalone address book
GET    /settings/dc-addresses            -> List standalone addresses
POST   /settings/dc-addresses           -> Create
GET    /settings/dc-addresses/{id}/edit  -> Edit
POST   /settings/dc-addresses/{id}/save -> Update
DELETE /settings/dc-addresses/{id}      -> Delete
```

## Handler Files

```
handlers/
  dc_list.go              # HandleDCList (global + project-scoped)
  dc_create.go            # HandleDCCreate, HandleDCSave
  dc_view.go              # HandleDCView
  dc_edit.go              # HandleDCEdit, HandleDCUpdate
  dc_delete.go            # HandleDCDelete
  dc_export_pdf.go        # HandleDCExportPDF
  dc_line_items.go        # HandleDCAddLineItem, UpdateLineItem, RemoveLineItem
  dc_boq_lookup.go        # HandleBOQItemsLookup
  dc_type_config.go       # CRUD for DC type configurations
  dc_number_series.go     # CRUD for number series
  dc_standalone_addr.go   # CRUD for standalone address book
```

## Template Files

```
templates/
  dc_list.templ           # List page (global + project-scoped)
  dc_create.templ         # Create/edit form
  dc_view.templ           # View page
  dc_line_item_row.templ  # HTMX partial for dynamic line items
  dc_boq_lookup.templ     # HTMX partial for BOQ item search results
  dc_destination.templ    # HTMX partial for destination rows
  dc_export.templ         # PDF export template
  dc_type_config.templ    # Settings: DC type management
  dc_number_series.templ  # Settings: Number series management
  dc_standalone_addr.templ # Settings: Standalone address book
```

## Services

```
services/
  dc_number.go           # DC number generation logic
  dc_validation.go       # Validation rules, over-dispatch checks
  dc_calculations.go     # Line item totals, GST, invoice value
  dc_export_data.go      # Data assembly for PDF export
  dc_export_pdf.go       # PDF rendering (HTML template -> PDF)
  dc_boq_tracker.go      # BOQ dispatched qty tracking
```

### Service Details

**`dc_number.go`** -- Number generation:
- `GenerateDCNumber(app, dcTypeId) (string, error)`
- Finds active series for the DC type
- Atomically increments `last_number`
- Returns formatted string: `{prefix}-{code}-{FY}-{seq:03d}`

**`dc_validation.go`** -- Validation:
- `ValidateDC(app, dc, lineItems, destinations) map[string]string`
- Required fields based on DC type config
- Destination qty totals match line item qty (multi-destination)
- Serial number count matches qty (if required)

**`dc_calculations.go`** -- Financials:
- `CalcDCLineItem(qty, rate, gstPercent) (taxable, gstAmt, total)`
- `CalcDCTotals(lineItems) (taxableTotal, cgst, sgst, igst, roundOff, invoiceValue)`
- Reuses existing `AmountToWords()` from services

**`dc_boq_tracker.go`** -- BOQ dispatch tracking:
- `GetBOQItemDispatchedQty(app, boqItemId, level) float64`
- `GetBOQItemAvailableQty(app, boqItemId, level) float64`
- `CheckOverDispatch(app, boqItemId, level, requestedQty) (isOver bool, available float64)`

**`dc_export_data.go`** -- PDF data assembly:
- `BuildDCExportData(app, dcId) (*DCExportData, error)`
- Fetches DC, line items, destinations, type config
- Formats values (INR, dates)
- Determines sections based on type config

**`dc_export_pdf.go`** -- PDF rendering:
- Same approach as existing `po_export_pdf.go`
- HTML template via Templ, converted to PDF
- Conditionally includes sections based on priced/unpriced, multi-destination, serial numbers

## UI Design

### Sidebar Integration

Within project context (after PURCHASE ORDERS):
```
PROJECT DETAILS
  OVERVIEW
  BOQ (3)
  PURCHASE ORDERS (2)
  DELIVERY CHALLANS (5)    <-- new
  VENDORS (4)
  ADDRESSES
```

Global top-level (before SETTINGS):
```
DASHBOARD
DELIVERY CHALLANS              <-- new (all DCs across projects + non-project)
ALL VENDORS
SETTINGS
  DC Types                     <-- new
  DC Number Series             <-- new
  DC Address Book              <-- new
```

### DC List Page

- Filter bar: DC type dropdown, status dropdown, project dropdown (with "Non-project" option), date range
- Table columns: DC Number, Type (badge), Date, Project (or "--"), Ship To, Status (badge), Actions
- Sort by date descending
- Project-scoped view: pre-filtered, no project column

### DC Create/Edit Form

**Top section**: DC type dropdown (drives form behavior), project (optional, searchable), parent DC (if type allows), PO ref

**Address section**: 4-box layout matching PDF (Bill From, Bill To, Dispatch From, Ship To). Project DCs pick from project address masters. Non-project DCs pick from standalone address book. Fields populated but editable (point-in-time snapshot).

**Transport section**: Transporter, vehicle, transport mode, e-way bill number, docket number, reverse charge toggle.

**Line items**: Dynamic table via HTMX. Rate/GST columns hidden for unpriced types. "Pull from BOQ" button (project DCs only) shows BOQ items with available qty. Over-dispatch shows yellow warning + reason field. Serial numbers expandable per item.

**Multi-destination section** (if type config enables): Destination breakdown grid with qty allocation per item per destination. Validates totals match line item qty.

**Totals section** (priced DCs only): Taxable value, CGST, SGST, round off, invoice value, amount in words.

### DC View Page

- Breadcrumbs (project context or global)
- Status badge (Draft=gray, Finalized=blue, Dispatched=amber, Delivered=green)
- Action buttons: Edit, Export PDF, status transitions
- Read-only render of all sections
- DC Chain section: parent DC link + child DCs list

### Settings Pages

- DC Types: table with toggles, create/edit form
- Number Series: table, create form, "Start New Series" action
- DC Address Book: similar to project address management, filtered by type

## PDF Export

Replicates sample DC layout exactly. Adapts based on DC type config:
- Priced: full item table with financials + totals
- Unpriced: simplified table with serial numbers + remarks, no totals
- Multi-destination: Annexure 1 (destination breakdown)
- Serial numbers present: Annexure 2 (serial number list)

## DC Chain Model

```
Transfer DC (parent)
  +-- 1 Transit DC (references parent via parent_dc)
  +-- Multiple Official DCs (each references parent via parent_dc)
```

Parent DC reference is a simple self-relation field. Not enforced as a strict hierarchy -- any DC type with `allows_parent_dc=true` can reference another DC.

## Over-Dispatch Handling

When a user enters qty exceeding available BOQ qty:
1. Yellow inline warning shows: "Available qty: X, you are dispatching Y"
2. Reason text field appears (required to proceed)
3. Reason stored on DC record in `over_dispatch_reason`
4. DC creation is NOT blocked
