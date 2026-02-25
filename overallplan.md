# Overall Implementation Plan — Project Lifecycle & P&L

## Current State (Complete)

| Module | What It Does |
|--------|-------------|
| Projects | CRUD, status (active/completed/on_hold), settings, project switching |
| BOQ | 3-level item hierarchy, quoted vs budgeted pricing, GST, Excel/PDF export |
| Addresses | 5 types, dynamic required fields, CSV import/export, ship-to→install-at linking |
| Vendors | Global directory, project linking, bank details, delete protection |
| Purchase Orders | Auto-numbering (FSS-PO-ref/FY/seq), line items from BOQ, status flow, PDF export |

---

## Phase 1: Financial Foundation

The backbone for all financial tracking. Every module after this writes to the central ledger.

### 1.1 Cost Categories
- Collection: `cost_categories`
- Fields: name, code, description, is_active
- Seed defaults: material, labor, logistics, subcontracting, overheads, permits, travel, tools_equipment
- UI: Settings page to manage categories (add/edit/deactivate)

### 1.2 Project Budget
- Collection: `project_budgets`
- Fields: project (relation), cost_category (relation), budgeted_amount, notes
- Auto-populate from BOQ: material budget = sum of budgeted prices for product-type items, labor budget = sum for service-type items
- UI: Budget tab under project details — table of categories with budgeted amounts, editable
- Allow manual override and addition of non-BOQ budget lines

### 1.3 Financial Transactions Ledger
- Collection: `financial_transactions`
- Fields: project (relation), transaction_type (cost/revenue), cost_category (relation, nullable for revenue), amount, gst_amount, total_amount, transaction_date, description, source_type (grn/vendor_invoice/vendor_payment/client_invoice/client_payment/expense), source_id, created_by, created, updated
- This is append-mostly — records are created by other modules, rarely edited directly
- No UI for direct entry (populated by downstream modules)

### 1.4 Chart of Accounts (Simple)
- Not a full double-entry system — keep it single-entry project cost/revenue tracking
- Categorization is handled by cost_categories + transaction_type
- If double-entry is needed later, can layer it on top of financial_transactions

---

## Phase 2: Goods Receipt Notes (GRN)

Track what was actually received against POs. This is where budgeted cost becomes actual cost.

### 2.1 GRN Collection
- Collection: `grn`
- Fields: grn_number (auto-generated), purchase_order (relation), project (relation), received_date, received_by, warehouse_location, transporter_name, lr_number, vehicle_number, remarks, status (draft/verified/accepted/rejected), created, updated
- GRN number format: GRN-{project_ref}/{FY}/{seq}

### 2.2 GRN Line Items
- Collection: `grn_line_items`
- Fields: grn (relation, cascade), po_line_item (relation), description, qty_ordered, qty_received, qty_rejected, rejection_reason, uom, rate (from PO), created, updated
- Qty validation: total received across all GRNs for a PO line item cannot exceed qty ordered

### 2.3 GRN Handlers
- List GRNs for a project: GET `/projects/{projectId}/grn`
- View GRN: GET `/projects/{projectId}/grn/{id}`
- Create GRN (select PO, auto-populate pending items): GET/POST `/projects/{projectId}/grn/create`
- Edit GRN (draft only): GET/POST `/projects/{projectId}/grn/{id}/edit`
- Accept GRN: POST `/projects/{projectId}/grn/{id}/accept`
- Delete GRN (draft only): DELETE `/projects/{projectId}/grn/{id}`

### 2.4 GRN → PO Status Updates
- When GRN accepted: check if all PO line items fully received
- If fully received → PO status can move to "completed"
- If partially received → PO stays in current status, show "partially received" indicator

### 2.5 GRN → Financial Transactions
- On GRN acceptance: create financial_transaction per line item
- transaction_type: cost, cost_category: material, amount: qty_received × rate
- source_type: grn, source_id: grn record ID

### 2.6 GRN Templates
- grn_list.templ — list with status badges, linked PO number
- grn_create.templ — select PO, show pending items, enter received quantities
- grn_view.templ — read-only with line items, totals, status

### 2.7 GRN PDF Export
- Similar to PO PDF — header, PO reference, line items received, signatures
- Used as goods inward note for warehouse records

### 2.8 Sidebar Integration
- Add "GRN" under project details in sidebar with count badge

---

## Phase 3: Vendor Invoices & Accounts Payable

Track vendor bills and what you owe them.

### 3.1 Vendor Invoice Collection
- Collection: `vendor_invoices`
- Fields: invoice_number (vendor's number), project (relation), vendor (relation), purchase_order (relation, optional), grn (relation, optional), invoice_date, due_date, subtotal, gst_amount, tds_amount, total_amount, status (pending/verified/approved/partially_paid/paid/disputed), remarks, created, updated
- Support invoices with or without PO (for non-PO expenses billed by vendors)

### 3.2 Vendor Invoice Line Items
- Collection: `vendor_invoice_items`
- Fields: vendor_invoice (relation, cascade), description, hsn_code, qty, uom, rate, gst_percent, gst_amount, total, po_line_item (relation, optional), grn_line_item (relation, optional), created, updated

### 3.3 Three-Way Matching
- Match: PO line item ↔ GRN line item ↔ Invoice line item
- Show match status per line: matched / quantity mismatch / rate mismatch / unmatched
- Flag discrepancies for review before approval
- Allow tolerance threshold (e.g., ±2% on rate)

### 3.4 TDS (Tax Deducted at Source)
- Fields: tds_section (194C/194J/etc), tds_percent, tds_amount
- Auto-calculate based on vendor type and section
- Deduct TDS from payment amount

### 3.5 Vendor Payments
- Collection: `vendor_payments`
- Fields: vendor_invoice (relation), project (relation), vendor (relation), payment_date, amount, payment_mode (neft/rtgs/cheque/cash/upi), reference_number (UTR/cheque no), bank_account, remarks, created, updated
- Support partial payments (multiple payments against one invoice)
- On payment: create financial_transaction (cost, relevant category)

### 3.6 Vendor Invoice Handlers
- List: GET `/projects/{projectId}/vendor-invoices`
- Create: GET/POST `/projects/{projectId}/vendor-invoices/create`
- View: GET `/projects/{projectId}/vendor-invoices/{id}`
- Approve: POST `/projects/{projectId}/vendor-invoices/{id}/approve`
- Record payment: POST `/projects/{projectId}/vendor-invoices/{id}/pay`
- Delete (pending only): DELETE `/projects/{projectId}/vendor-invoices/{id}`

### 3.7 Vendor Ledger
- Per-vendor statement: all invoices, payments, outstanding balance
- Ageing report: current, 30 days, 60 days, 90 days, 90+ days overdue

### 3.8 Templates
- vendor_invoice_list.templ — list with status, amount, due date, overdue highlighting
- vendor_invoice_create.templ — form with PO/GRN selection, line items
- vendor_invoice_view.templ — detail view with 3-way match status, payment history
- vendor_payment_form.templ — record payment modal/form
- vendor_ledger.templ — statement view per vendor

### 3.9 Sidebar Integration
- Add "Payables" or "Vendor Invoices" under project details

---

## Phase 4: Client Invoicing & Accounts Receivable

Track what the client owes you. This is the revenue side of P&L.

### 4.1 Billing Milestones
- Collection: `billing_milestones`
- Fields: project (relation), milestone_name, milestone_order (sort), percentage_of_contract, amount, trigger_description (what triggers this milestone), status (pending/eligible/invoiced/paid), created, updated
- Common milestones: advance, on dispatch, on delivery, on installation, on commissioning, retention release
- Total percentage must equal 100%

### 4.2 Client Invoice Collection
- Collection: `client_invoices`
- Fields: invoice_number (auto-generated), project (relation), billing_milestone (relation, optional), invoice_date, due_date, bill_to_address (relation to project's bill_to address), subtotal, gst_amount, tds_receivable, total_amount, status (draft/sent/acknowledged/partially_paid/paid/disputed), remarks, created, updated
- Invoice number format: FSS/INV/{FY}/{seq}

### 4.3 Client Invoice Line Items
- Collection: `client_invoice_items`
- Fields: client_invoice (relation, cascade), description, hsn_code, qty, uom, rate, gst_percent, gst_amount, total, boq_item_type (main_item/sub_item/manual), boq_item_id, created, updated
- Pull from BOQ quoted prices (this is what you charge the client)

### 4.4 Client Payments
- Collection: `client_payments`
- Fields: client_invoice (relation), project (relation), payment_date, amount, payment_mode, reference_number, tds_deducted, remarks, created, updated
- On payment: create financial_transaction (revenue type)
- Handle TDS deducted by client (client pays less, TDS is a receivable)

### 4.5 Client Invoice Handlers
- List: GET `/projects/{projectId}/invoices`
- Create from milestone: GET/POST `/projects/{projectId}/invoices/create`
- Create ad-hoc: GET/POST `/projects/{projectId}/invoices/create-adhoc`
- View: GET `/projects/{projectId}/invoices/{id}`
- Send (mark as sent): POST `/projects/{projectId}/invoices/{id}/send`
- Record payment: POST `/projects/{projectId}/invoices/{id}/receive-payment`
- Delete (draft only): DELETE `/projects/{projectId}/invoices/{id}`

### 4.6 Client Invoice PDF Export
- Professional invoice format with company letterhead details
- BOQ-based line items with HSN, GST breakdown
- Payment terms, bank details for remittance
- Similar structure to PO PDF but from seller perspective

### 4.7 Revenue Recognition
- On invoice creation: create financial_transaction (revenue, amount = subtotal)
- On payment: update invoice status, track outstanding

### 4.8 Templates
- billing_milestones.templ — milestone setup/edit for project
- client_invoice_list.templ — list with status, amounts, overdue flags
- client_invoice_create.templ — form with milestone selection or ad-hoc line items
- client_invoice_view.templ — detail with payment history
- client_payment_form.templ — record payment received

### 4.9 Sidebar Integration
- Add "Billing" or "Invoices" under project details with count/amount badge

---

## Phase 5: Dispatch & Delivery Tracking

Track physical movement of goods from receipt to site.

### 5.1 Dispatch Collection
- Collection: `dispatches`
- Fields: dispatch_number (auto), project (relation), purchase_order (relation, optional), dispatch_date, expected_delivery_date, from_location (text or address relation), to_address (relation to ship_to address), transporter_name, lr_number, vehicle_number, freight_cost, freight_gst, freight_paid_by (company/vendor/client), status (scheduled/in_transit/delivered/partially_delivered), remarks, created, updated

### 5.2 Dispatch Items
- Collection: `dispatch_items`
- Fields: dispatch (relation, cascade), description, grn_line_item (relation, optional), qty_dispatched, uom, weight, dimensions, created, updated
- Link to GRN line items to track what was received vs what was dispatched

### 5.3 Delivery Confirmation
- Collection: `delivery_confirmations`
- Fields: dispatch (relation), delivery_date, received_by_name, received_by_phone, condition_notes, photo_proof (file field), signed_challan (file field), status (confirmed/damaged/short), created, updated

### 5.4 Secondary Dispatch (Scenario 2)
- For ship-to ≠ install-at scenario: track client's redistribution
- Collection: `secondary_dispatches`
- Fields: project (relation), from_ship_to (relation to ship_to address), to_install_at (relation to install_at address), dispatch_date, qty_description, status (pending/dispatched/delivered), remarks, created, updated
- This may be partially tracked by client — allow manual status updates

### 5.5 Dispatch → Financial Transactions
- If company pays freight: create financial_transaction (cost, logistics category)
- amount = freight_cost + freight_gst

### 5.6 Handlers
- List dispatches: GET `/projects/{projectId}/dispatches`
- Create dispatch: GET/POST `/projects/{projectId}/dispatches/create`
- View dispatch: GET `/projects/{projectId}/dispatches/{id}`
- Record delivery: POST `/projects/{projectId}/dispatches/{id}/deliver`
- Track secondary dispatches: GET `/projects/{projectId}/dispatches/secondary`

### 5.7 Templates
- dispatch_list.templ — list with status, destination, transporter
- dispatch_create.templ — form with item selection from GRN, destination from ship-to addresses
- dispatch_view.templ — detail with delivery confirmation status
- dispatch_tracker.templ — map/table view of all dispatches and their status

### 5.8 Sidebar Integration
- Add "Dispatches" under project details

---

## Phase 6: Installation & Commissioning

Track field work at install-at locations. Major labor cost center.

### 6.1 Work Order Collection
- Collection: `work_orders`
- Fields: wo_number (auto), project (relation), install_at_address (relation), scope_description, assigned_team, team_lead, planned_start_date, planned_end_date, actual_start_date, actual_end_date, labor_cost_estimate, status (planned/in_progress/completed/on_hold/cancelled), remarks, created, updated

### 6.2 Work Order Items
- Collection: `work_order_items`
- Fields: work_order (relation, cascade), description, boq_item_type, boq_item_id, qty, uom, status (pending/installed/tested), created, updated
- Track which BOQ items are being installed at this location

### 6.3 Installation Log
- Collection: `installation_logs`
- Fields: work_order (relation, cascade), log_date, crew_count, hours_worked, tasks_completed, issues_encountered, materials_used, photos (file field, multiple), created, updated
- Daily progress recording per work order

### 6.4 Commissioning Records
- Collection: `commissioning_records`
- Fields: work_order (relation), project (relation), install_at_address (relation), commissioning_date, tested_by, test_results, client_representative, client_sign_off (boolean), sign_off_date, certificate_number, certificate_file (file), remarks, status (pending/testing/passed/failed/signed_off), created, updated

### 6.5 Work Order → Financial Transactions
- On work order completion: create financial_transaction (cost, labor category)
- amount = actual labor cost (hours × rate or lump sum)
- Materials consumed: additional transaction (cost, material category)

### 6.6 Commissioning → Billing Milestone
- When commissioning is signed off at a threshold of sites (e.g., all 1000), trigger billing milestone status to "eligible"
- Link commissioning completion % to milestone eligibility

### 6.7 Handlers
- List work orders: GET `/projects/{projectId}/work-orders`
- Create work order: GET/POST `/projects/{projectId}/work-orders/create`
- Bulk create (for all install-at addresses): POST `/projects/{projectId}/work-orders/bulk-create`
- View work order: GET `/projects/{projectId}/work-orders/{id}`
- Log daily progress: POST `/projects/{projectId}/work-orders/{id}/log`
- Complete work order: POST `/projects/{projectId}/work-orders/{id}/complete`
- Commissioning: GET/POST `/projects/{projectId}/work-orders/{id}/commission`

### 6.8 Site Tracker Dashboard
- Table/card view of all install-at addresses with status:
  - Not started / In progress / Installed / Commissioned / Signed off
- Filter by status, city, state
- Progress bar: X of N sites completed
- Export site status report to Excel

### 6.9 Templates
- work_order_list.templ — list with status, location, team, dates
- work_order_create.templ — form with address selection, scope, team assignment
- work_order_view.templ — detail with installation logs timeline, commissioning status
- installation_log_form.templ — daily log entry form
- commissioning_form.templ — test results and sign-off form
- site_tracker.templ — dashboard view of all sites

### 6.10 Sidebar Integration
- Add "Installation" or "Site Tracker" under project details

---

## Phase 7: Expense Tracking

Capture project costs that don't flow through POs — travel, rentals, permits, etc.

### 7.1 Expense Collection
- Collection: `expenses`
- Fields: expense_number (auto), project (relation), cost_category (relation), description, amount, gst_amount, total_amount, expense_date, incurred_by, payment_mode, reference_number, vendor_name (text, not from vendor master — for petty expenses), status (submitted/approved/rejected/reimbursed), remarks, created, updated

### 7.2 Expense Attachments
- Use PocketBase file field on expenses collection: receipt_files (file, multiple)
- Store scanned receipts, bills, invoices

### 7.3 Expense → Financial Transactions
- On approval: create financial_transaction (cost, linked category)

### 7.4 Handlers
- List expenses: GET `/projects/{projectId}/expenses`
- Create expense: GET/POST `/projects/{projectId}/expenses/create`
- View expense: GET `/projects/{projectId}/expenses/{id}`
- Approve/reject: POST `/projects/{projectId}/expenses/{id}/approve`
- Delete (submitted only): DELETE `/projects/{projectId}/expenses/{id}`

### 7.5 Templates
- expense_list.templ — list with category, amount, status, date
- expense_create.templ — form with category dropdown, amount, receipt upload
- expense_view.templ — detail with receipt images, approval status

### 7.6 Sidebar Integration
- Add "Expenses" under project details

---

## Phase 8: Project Dashboard & P&L Reports

Everything feeds into this. The `financial_transactions` table is now populated by GRNs, vendor invoices, vendor payments, client invoices, client payments, dispatches, work orders, and expenses.

### 8.1 Project P&L Statement
- Query financial_transactions grouped by transaction_type and cost_category
- Layout:
  ```
  CONTRACT VALUE (BOQ total quoted)

  REVENUE
    Invoiced to date          ₹XX,XX,XXX
    Collected to date         ₹XX,XX,XXX
    Outstanding receivables   ₹XX,XX,XXX

  COSTS
    Materials                 ₹XX,XX,XXX
    Labor                     ₹XX,XX,XXX
    Logistics                 ₹XX,XX,XXX
    Subcontracting            ₹XX,XX,XXX
    Overheads                 ₹XX,XX,XXX
    Other                     ₹XX,XX,XXX
    ─────────────────────────────────────
    TOTAL COST                ₹XX,XX,XXX

  GROSS PROFIT                ₹XX,XX,XXX
  MARGIN                      XX.X%

  COMMITTED (POs issued, not yet received/invoiced)
  PROJECTED TOTAL COST
  PROJECTED MARGIN
  ```

### 8.2 Budget vs Actual Report
- Per cost category: budgeted (from project_budgets) vs actual (from financial_transactions)
- Variance: amount and percentage
- Color coding: green (under budget), amber (within 10%), red (over budget)
- Drill-down: click a category to see individual transactions

### 8.3 Cash Flow Report
- Monthly view: inflows (client payments) vs outflows (vendor payments + expenses)
- Running balance per month
- Cumulative chart data (can render with simple bar chart or table)

### 8.4 Vendor-wise Spending Report
- Per vendor: total PO value, total invoiced, total paid, outstanding
- Top vendors by spend
- Filter by date range

### 8.5 Project Overview Dashboard (Enhanced)
- Replace current project_view with a richer dashboard:
  - Contract value, invoiced %, collected %
  - Cost summary by category (mini bar chart or progress bars)
  - PO count and total committed value
  - Sites: total, dispatched, installed, commissioned (progress bar)
  - Recent activity feed (last 10 transactions/events)
  - Key alerts: overdue invoices, over-budget categories, pending approvals

### 8.6 GST Reports
- Output GST: from client invoices (what you charge)
- Input GST: from vendor invoices (what you pay)
- Net GST liability: output - input
- HSN-wise summary for GST return filing
- Monthly/quarterly grouping

### 8.7 Multi-Project Summary
- Table of all projects with: contract value, revenue, cost, margin, status
- Sort by margin %, value, status
- Quick health indicator per project
- Accessible from the main dashboard (no project selected)

### 8.8 Report Services
- `services/report_pnl.go` — query and aggregate P&L data
- `services/report_budget.go` — budget vs actual calculations
- `services/report_cashflow.go` — cash flow aggregation
- `services/report_gst.go` — GST computations
- `services/report_vendor.go` — vendor spending aggregation

### 8.9 Report Export
- All reports exportable to Excel (using existing excelize patterns)
- P&L exportable to PDF
- Date range filters on all reports

### 8.10 Templates
- project_dashboard.templ — enhanced project overview with financial summary
- report_pnl.templ — P&L statement view
- report_budget.templ — budget vs actual table
- report_cashflow.templ — cash flow table
- report_vendor.templ — vendor spending report
- report_gst.templ — GST summary
- multi_project_summary.templ — cross-project dashboard

### 8.11 Sidebar Integration
- Add "Reports" section under project details with sub-items:
  - P&L, Budget vs Actual, Cash Flow, Vendor Spending, GST
- Add "Portfolio" or "All Projects Summary" in the global sidebar

---

## Phase 9: Document Management

Centralize all project documents.

### 9.1 Attachments Collection
- Collection: `attachments`
- Fields: project (relation), entity_type (po/grn/vendor_invoice/client_invoice/expense/work_order/commissioning), entity_id, file (file field), file_name, file_type, description, uploaded_by, created
- PocketBase handles file storage natively

### 9.2 Attachment UI
- Reusable attachment component (templ partial) — embed in PO view, invoice view, etc.
- Upload via drag-and-drop or file picker
- List attachments with download links
- Delete attachment

### 9.3 Project Document Library
- All attachments for a project in one view, filterable by entity type
- GET `/projects/{projectId}/documents`

### 9.4 Templates
- attachment_widget.templ — reusable upload/list component
- document_library.templ — project-wide document browser

---

## Phase 10: Approval Workflows

Add control gates for financial operations.

### 10.1 Approval Configuration
- Collection: `approval_rules`
- Fields: project (relation, optional — null for global), entity_type (po/vendor_invoice/client_invoice/expense), threshold_amount, requires_approval (boolean), approver_role, created, updated
- Example: POs above ₹1,00,000 require approval

### 10.2 Approval Records
- Collection: `approvals`
- Fields: entity_type, entity_id, project (relation), requested_by, requested_at, status (pending/approved/rejected), approved_by, approved_at, rejection_reason, remarks, created
- Each approvable entity gets an approval record when submitted

### 10.3 Approval Flow
- Entity created → status = draft (no approval needed)
- Entity submitted → check approval_rules → if threshold met, create approval record, status = pending_approval
- Approver approves → entity status advances (e.g., PO: draft → approved → sent)
- Approver rejects → entity status = rejected, can be edited and resubmitted

### 10.4 Approval UI
- Approval banner on entity view page (pending approval / approved / rejected)
- Approval action buttons for approvers
- Pending approvals list: GET `/approvals` (cross-project)
- Project approvals: GET `/projects/{projectId}/approvals`

### 10.5 Templates
- approval_banner.templ — reusable component showing approval status
- approval_list.templ — list of pending approvals

---

## Phase 11: User Management & Permissions

Currently single-user. For team use, add roles and access control.

### 11.1 PocketBase Auth
- Use PocketBase built-in auth collection (users)
- Fields: email, name, role (admin/manager/accountant/field_engineer/viewer)
- PocketBase handles login, sessions, password reset

### 11.2 Role-Based Access
- Admin: full access
- Manager: all operations + approvals
- Accountant: invoices, payments, expenses, reports (no BOQ/installation)
- Field Engineer: work orders, installation logs, commissioning (no financials)
- Viewer: read-only access to everything

### 11.3 Implementation
- Middleware to check auth and role on each request
- Template-level conditional rendering (hide buttons/tabs based on role)
- API-level enforcement in handlers

### 11.4 Audit Trail
- Collection: `audit_log`
- Fields: user (relation), action (create/update/delete/approve/reject), entity_type, entity_id, project (relation), changes_json (text — old/new values), timestamp
- PocketBase hooks: OnRecordAfterCreateRequest, OnRecordAfterUpdateRequest, OnRecordAfterDeleteRequest
- Viewer: GET `/projects/{projectId}/audit-log` — timeline of all changes

---

## Phase 12: Notifications & Alerts

Keep the team informed of important events.

### 12.1 Notification Collection
- Collection: `notifications`
- Fields: user (relation), project (relation), title, message, entity_type, entity_id, link (URL to navigate to), is_read (boolean), created

### 12.2 Trigger Events
- PO pending approval → notify approver
- PO approved → notify creator
- Vendor invoice due in 3 days → notify accountant
- Client payment received → notify manager
- Budget exceeded for a category → notify manager
- Work order completed → notify manager
- Commissioning signed off → notify manager

### 12.3 UI
- Notification bell in header with unread count
- Dropdown list of recent notifications
- Full notification list page
- Mark as read on click

### 12.4 Optional: Email Notifications
- Use PocketBase mailer or external SMTP
- Configurable per user: which events to email

---

## Phase 13: Import/Export Enhancements

Extend the existing import/export patterns to new modules.

### 13.1 Vendor Import
- CSV import for bulk vendor creation (similar to address import pattern)
- Template download, validation, error report, commit

### 13.2 PO Line Item Import
- CSV import for PO line items (for large POs not sourced from BOQ)

### 13.3 Work Order Bulk Import
- CSV import of site addresses → auto-create work orders
- Map install-at addresses to work orders in bulk

### 13.4 Expense Bulk Import
- CSV import of expenses (from accounting exports)

### 13.5 Financial Data Export
- Export financial_transactions to CSV/Excel for external accounting software
- Tally-compatible export format (if needed)
- Export format configurable per project

### 13.6 Report Scheduling (Optional)
- Weekly/monthly auto-export of P&L to Excel
- Email to configured recipients

---

## Phase 14: Advanced Features

Nice-to-haves that add significant value once the core is solid.

### 14.1 PO Amendments/Revisions
- Collection: `po_revisions`
- Track PO changes: original values, amended values, amendment reason, amendment date
- PO revision number: Rev 0 (original), Rev 1, Rev 2
- Amendment requires approval if amount increases beyond threshold

### 14.2 Credit/Debit Notes
- Collection: `credit_debit_notes`
- Against vendor or client invoices
- Adjust amounts for returns, short shipments, rate differences
- Update financial_transactions accordingly

### 14.3 Retention Tracking
- Track retention amounts held by client (common in project contracts)
- Retention release milestones
- Include in P&L as receivable

### 14.4 Subcontractor Management
- Treat subcontractors as vendors with work-order-like scope
- Track subcontractor progress and payments
- RA bills (Running Account bills) for subcontractors

### 14.5 Inventory/Stock Register
- Collection: `stock_register`
- Track material in/out per project: GRN in, dispatch out, returns
- Current stock per item per location
- Minimum stock alerts

### 14.6 Rate Comparison
- Compare vendor quotes for same BOQ items across POs
- Historical rate analysis per item
- Identify cost-saving opportunities

---

## Phase 15: Performance, Polish & Production Readiness

### 15.1 Database Indexing
- Add indexes on frequently queried fields: project relations, status fields, dates
- PocketBase supports indexes via collection configuration
- Benchmark query performance on realistic data volumes

### 15.2 Pagination
- All list views should paginate (current implementation may load all records)
- Consistent pagination component: page size selector, page navigation
- Server-side pagination using PocketBase limit/offset

### 15.3 Search & Filters
- Global search across entities (POs, invoices, vendors, addresses)
- Advanced filters on list pages: date range, status, amount range, vendor
- Save filter presets per user

### 15.4 Data Backup & Restore
- Automated SQLite backup (PocketBase pb_data directory)
- Backup scheduling (daily)
- Restore procedure documented and tested

### 15.5 Error Handling & Logging
- Consistent error pages (404, 500)
- Structured logging for all financial operations
- Request logging middleware

### 15.6 Loading States & UX
- HTMX loading indicators on all operations
- Optimistic UI where appropriate
- Confirmation dialogs for destructive actions (already using Alpine.js)
- Toast notifications for success/error feedback

### 15.7 Mobile Responsiveness
- Field engineers need mobile access for installation logs
- Responsive sidebar (collapsible on mobile)
- Touch-friendly form inputs
- Camera integration for photo upload on mobile

### 15.8 Deployment
- Docker containerization (Go binary + pb_data volume)
- Reverse proxy configuration (nginx/caddy)
- HTTPS setup
- Environment-based configuration (dev/staging/production)

---

## Build Order & Dependencies

```
Phase 1:  Financial Foundation          ← START HERE (no dependencies)
Phase 2:  GRN                           ← depends on Phase 1 (writes to ledger)
Phase 3:  Vendor Invoices & Payables    ← depends on Phase 2 (matches GRN)
Phase 4:  Client Invoicing & Receivables← depends on Phase 1 (writes to ledger)
Phase 5:  Dispatch & Delivery           ← depends on Phase 2 (uses GRN items)
Phase 6:  Installation & Commissioning  ← depends on Phase 5 (delivery triggers install)
Phase 7:  Expense Tracking              ← depends on Phase 1 (writes to ledger)
Phase 8:  P&L & Reports                 ← depends on Phases 1-7 (reads ledger)
Phase 9:  Document Management           ← independent (can build anytime)
Phase 10: Approval Workflows            ← depends on Phases 2-7 (gates on entities)
Phase 11: User Management               ← independent (can build anytime)
Phase 12: Notifications                 ← depends on Phase 11 (needs users)
Phase 13: Import/Export Enhancements    ← depends on respective entity phases
Phase 14: Advanced Features             ← depends on Phases 1-8
Phase 15: Performance & Production      ← ongoing, finalize after core phases
```

**Parallel tracks possible:**
- Phase 4 (Client Invoicing) can be built in parallel with Phases 2-3
- Phase 7 (Expenses) can be built in parallel with Phases 5-6
- Phase 9 (Documents) and Phase 11 (Users) can be built anytime
