# Handler Conventions

## File Organization
- One file per operation: `handlers/{entity}_{operation}.go`
- Examples: `project_create.go`, `project_list.go`, `boq_edit.go`

## Function Signature
Every handler is a closure over `*pocketbase.PocketBase`:
```go
func HandleXxx(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        // handler logic
    }
}
```

## HTMX Detection
All GET handlers must support both HTMX partial and full-page rendering:
```go
if e.Request.Header.Get("HX-Request") == "true" {
    // render Content component only (partial)
} else {
    // render Page component (full page with layout)
}
```

## Form Handling
```go
if err := e.Request.ParseForm(); err != nil {
    return e.String(http.StatusBadRequest, "Invalid form data")
}
name := strings.TrimSpace(e.Request.FormValue("name"))
```
Always `strings.TrimSpace()` all text inputs.

## Validation Pattern
```go
errors := make(map[string]string)
if name == "" {
    errors["name"] = "Field is required"
}
if len(errors) > 0 {
    // re-render form with data + errors
    return component.Render(e.Request.Context(), e.Response)
}
```

## POST Success Response
```go
// HTMX request: set redirect header + return 200
if e.Request.Header.Get("HX-Request") == "true" {
    e.Response.Header().Set("HX-Redirect", "/target-url")
    return e.String(http.StatusOK, "")
}
// Non-HTMX: standard redirect
return e.Redirect(http.StatusFound, "/target-url")
```

## Context Helpers
Always pass header and sidebar data to page components:
```go
headerData := GetHeaderData(e.Request)
sidebarData := GetSidebarData(e.Request)
```

## Path Parameters
```go
id := e.Request.PathValue("id")
projectId := e.Request.PathValue("projectId")
```
