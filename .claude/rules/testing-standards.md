# Testing Standards

## File Organization
- Unit tests: `services/*_test.go` (same package)
- Integration tests: `handlers/*_test.go` (same package)
- E2E tests: `tests/e2e/` (Playwright)
- Test helpers: `testhelpers/testhelpers.go`

## Naming
- File: `{name}_test.go` in same package as code under test
- Function: `TestXxx_ScenarioDescription`
- Subtests: `t.Run("scenario name", func(t *testing.T) { ... })`

## Table-Driven Tests
```go
func TestFormatINR_Scenarios(t *testing.T) {
    tests := []struct {
        name   string
        input  float64
        expect string
    }{
        {"zero", 0, "₹0.00"},
        {"thousands", 1234.56, "₹1,234.56"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := FormatINR(tt.input)
            if got != tt.expect {
                t.Errorf("FormatINR(%v) = %q, want %q", tt.input, got, tt.expect)
            }
        })
    }
}
```

## PocketBase Test App
```go
app := testhelpers.NewTestApp(t)
// app is fully bootstrapped with collections, cleaned up via t.Cleanup
```

## Handler Integration Tests
```go
// Setup
app := testhelpers.NewTestApp(t)
handler := HandleProjectCreate(app)

// Create request
req := httptest.NewRequest("GET", "/projects/create", nil)
rec := httptest.NewRecorder()
event := &core.RequestEvent{...} // construct with request and recorder

// Assert
testhelpers.AssertHTMLContains(t, rec.Body.String(), "Create Project")
```

## Makefile Targets
- `make test` — all tests
- `make test-unit` — services only
- `make test-integration` — handlers only
- `make test-coverage` — with coverage report

## TDD Flow
1. Write failing test
2. Implement minimal code to pass
3. Refactor
4. Repeat
