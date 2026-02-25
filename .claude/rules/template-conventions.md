# Template Conventions

## Three Components Per Page
Each page template defines:
1. **Data struct**: Typed data for the template
2. **XxxContent(data)**: Partial content component (for HTMX swaps)
3. **XxxPage(data, headerData, sidebarData)**: Full page wrapping Content in `PageShellWithProject`

```go
type ProjectCreateData struct {
    Name    string
    Errors  map[string]string
    // ...
}

templ ProjectCreateContent(data ProjectCreateData) {
    // page content
}

templ ProjectCreatePage(data ProjectCreateData, header HeaderData, sidebar SidebarData) {
    @PageShellWithProject(header, sidebar) {
        @ProjectCreateContent(data)
    }
}
```

## Navigation
All navigation uses HTMX with these attributes:
```html
hx-get="/target-url"
hx-target="#main-content"
hx-push-url="true"
```

## Styling
- CSS custom properties: `--bg-page: #F5F3EF`, `--bg-card: #E8E4DC`, `--terracotta: #C05A3C`
- Fonts: Space Grotesk for headings/labels, Inter for body text
- Design: flat, minimal, square corners (`rounded-none` or no border-radius)
- Uppercase labels with `letter-spacing: 0.05em`
- Inline `style=` attributes alongside Tailwind classes where needed

## Alpine.js
- Use Alpine.js for client-side interactivity (toggles, modals, dropdowns)
- Re-initialize after HTMX swaps via `htmx:afterSettle` event
- Keep Alpine state minimal — server is source of truth

## Layout Constants
- Sidebar: 260px fixed left
- Header: 56px height
- Main content padding: 40px horizontal, 48px vertical
