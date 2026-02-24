# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Dev Commands

```bash
make run          # generate templ + compile CSS + start server
make dev          # watch mode (templ + tailwind + server in parallel)
make templ        # generate templ files only
make css          # compile Tailwind CSS only
./restart.sh      # kill existing server, rebuild everything, restart
go run main.go serve  # start PocketBase server directly (port 8090)
```

Templ files (`.templ`) must be compiled before the Go build: `templ generate` produces `_templ.go` files. CSS is built with `npx @tailwindcss/cli -i input.css -o static/css/output.css`.

## Architecture

**Stack**: Go + PocketBase (embedded SQLite) + Templ + HTMX + Alpine.js + Tailwind CSS v4 + DaisyUI v5

**Module**: `projectcreation` (see `go.mod`)

### Request Flow
1. `main.go` — PocketBase app setup, route registration, collection bootstrap via `collections.Setup(app)`
2. `handlers/` — each handler is a closure over `*pocketbase.PocketBase`, returns `func(*core.RequestEvent) error`
3. `templates/` — Templ components; each page has a Content component (HTMX partial) and a Page component (full page via `PageShell`)
4. `services/` — business logic (pricing, formatting, validation, export)

### Navigation Model (SPA-like via HTMX)
- All page navigation uses `hx-get` + `hx-target="#main-content"` + `hx-push-url="true"`
- Layout hierarchy: `Layout` (HTML shell) → `PageShell` (header + sidebar + `<main id="main-content">`) → Content component
- Handlers detect HTMX via `e.Request.Header.Get("HX-Request") == "true"` — render partial for HTMX, full page otherwise
- Form POST success: set `HX-Redirect` header + return 200; non-HTMX: return 302 redirect
- HTMX extensions: idiomorph, response-targets, loading-states, json-enc

### Database Collections
- `projects` → `boqs` → `main_boq_items` → `sub_items` → `sub_sub_items` (cascade deletes)
- `addresses` — multi-type (bill_from, ship_from, bill_to, ship_to, install_at); `ship_to_parent` self-relation
- `project_address_settings` — per-project required field config (boolean `req_*` fields)
- Collections created idempotently in `collections/setup.go` via `ensureCollection` / `ensureField`

### PocketBase Access Patterns
```go
app.FindRecordById("collection", id)
app.FindRecordsByFilter("collection", "field = {:p}", "sort", limit, offset, params)
app.FindAllRecords(collection)
record := core.NewRecord(col); record.Set("field", val); app.Save(record)
app.Delete(record)
// Read: record.GetString(), GetBool(), GetFloat64(), GetInt(), GetDateTime()
```

## Handler Conventions

- File per operation: `handlers/project_create.go`, `handlers/project_list.go`, etc.
- Function signature: `func HandleXxx(app *pocketbase.PocketBase) func(*core.RequestEvent) error`
- Path params: `e.Request.PathValue("id")`
- Form parsing: `e.Request.ParseForm()` then `e.Request.FormValue("field")`
- Validation: build `map[string]string` errors, re-render form with errors if invalid

## Template Conventions

- Each page defines a typed data struct, a `XxxContent` component (partial), and a `XxxPage` component (full)
- Templates use inline `style=` attributes alongside Tailwind classes
- Alpine.js for client-side interactivity; re-initialized after HTMX swaps via `htmx:afterSettle`

## Styling

- **Fonts**: Inter (body), Space Grotesk (headings/labels)
- **Colors** (CSS custom properties in `input.css`): `--bg-page: #F5F3EF`, `--bg-card: #E8E4DC`, `--bg-sidebar: #1a1a1a`, `--terracotta: #C05A3C` (primary accent), `--success: #4A7C59`, `--error: #DC2626`
- **Design**: Flat, minimal, square corners, uppercase labels with letter-spacing
- **Layout**: Sidebar 260px fixed left, header 56px, main content padded 40px/48px
- Currency formatting: `services.FormatINR()` — Indian Rupee notation (₹X,XX,XXX.XX)

## Routes

`GET /` redirects to `/projects`. Route groups: `/projects/*` (CRUD + settings), `/boq/*` (CRUD + items + export). See `main.go` for the full route table.
