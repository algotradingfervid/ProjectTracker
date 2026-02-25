package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestSetToast_Basic(t *testing.T) {
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{}
	e.Response = rec

	SetToast(e, "success", "Item saved")

	trigger := rec.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Fatal("expected HX-Trigger header to be set")
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trigger), &parsed); err != nil {
		t.Fatalf("HX-Trigger is not valid JSON: %v", err)
	}

	raw, ok := parsed["showToast"]
	if !ok {
		t.Fatal("expected showToast key in HX-Trigger JSON")
	}

	var toast map[string]string
	if err := json.Unmarshal(raw, &toast); err != nil {
		t.Fatalf("showToast value is not valid JSON: %v", err)
	}

	if toast["message"] != "Item saved" {
		t.Errorf("expected message %q, got %q", "Item saved", toast["message"])
	}
	if toast["type"] != "success" {
		t.Errorf("expected type %q, got %q", "success", toast["type"])
	}
}

func TestSetToast_Types(t *testing.T) {
	tests := []struct {
		name      string
		toastType string
		message   string
	}{
		{"success", "success", "Operation completed"},
		{"error", "error", "Something went wrong"},
		{"info", "info", "Please note this"},
		{"warning", "warning", "Proceed with caution"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e := &core.RequestEvent{}
			e.Response = rec

			SetToast(e, tt.toastType, tt.message)

			trigger := rec.Header().Get("HX-Trigger")
			if trigger == "" {
				t.Fatal("expected HX-Trigger header to be set")
			}

			var parsed map[string]json.RawMessage
			if err := json.Unmarshal([]byte(trigger), &parsed); err != nil {
				t.Fatalf("HX-Trigger is not valid JSON: %v", err)
			}

			var toast map[string]string
			if err := json.Unmarshal(parsed["showToast"], &toast); err != nil {
				t.Fatalf("showToast is not valid JSON: %v", err)
			}

			if toast["type"] != tt.toastType {
				t.Errorf("expected type %q, got %q", tt.toastType, toast["type"])
			}
			if toast["message"] != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, toast["message"])
			}
		})
	}
}

func TestSetToast_MergesWithExisting(t *testing.T) {
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{}
	e.Response = rec

	// Set an existing HX-Trigger header
	rec.Header().Set("HX-Trigger", `{"someEvent":{"key":"value"}}`)

	SetToast(e, "success", "Merged toast")

	trigger := rec.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Fatal("expected HX-Trigger header to be set")
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trigger), &parsed); err != nil {
		t.Fatalf("HX-Trigger is not valid JSON: %v", err)
	}

	// Verify the original event is preserved
	if _, ok := parsed["someEvent"]; !ok {
		t.Error("expected someEvent key to be preserved after merge")
	}

	// Verify the toast was added
	raw, ok := parsed["showToast"]
	if !ok {
		t.Fatal("expected showToast key in merged HX-Trigger JSON")
	}

	var toast map[string]string
	if err := json.Unmarshal(raw, &toast); err != nil {
		t.Fatalf("showToast is not valid JSON: %v", err)
	}

	if toast["message"] != "Merged toast" {
		t.Errorf("expected message %q, got %q", "Merged toast", toast["message"])
	}

	// Verify someEvent contents are intact
	var someEvent map[string]string
	if err := json.Unmarshal(parsed["someEvent"], &someEvent); err != nil {
		t.Fatalf("someEvent is not valid JSON: %v", err)
	}
	if someEvent["key"] != "value" {
		t.Errorf("expected someEvent.key %q, got %q", "value", someEvent["key"])
	}
}

func TestSetToast_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"quotes", `Item "Special" saved`},
		{"angle brackets", `<script>alert("xss")</script>`},
		{"backslash", `path\to\file`},
		{"newline", "line1\nline2"},
		{"unicode", "Saved \u2714 successfully"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e := &core.RequestEvent{}
			e.Response = rec

			SetToast(e, "info", tt.message)

			trigger := rec.Header().Get("HX-Trigger")
			if trigger == "" {
				t.Fatal("expected HX-Trigger header to be set")
			}

			// Verify it is valid JSON (special chars are properly escaped)
			var parsed map[string]json.RawMessage
			if err := json.Unmarshal([]byte(trigger), &parsed); err != nil {
				t.Fatalf("HX-Trigger is not valid JSON for message %q: %v", tt.message, err)
			}

			var toast map[string]string
			if err := json.Unmarshal(parsed["showToast"], &toast); err != nil {
				t.Fatalf("showToast is not valid JSON: %v", err)
			}

			// After round-tripping through JSON, the message should match the original
			if toast["message"] != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, toast["message"])
			}
		})
	}
}

func TestSetToast_OverwritesInvalidExisting(t *testing.T) {
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{}
	e.Response = rec

	// Set an invalid (non-JSON) HX-Trigger header
	rec.Header().Set("HX-Trigger", "notValidJSON")

	SetToast(e, "error", "Overwritten")

	trigger := rec.Header().Get("HX-Trigger")

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trigger), &parsed); err != nil {
		t.Fatalf("HX-Trigger should be valid JSON after overwrite: %v", err)
	}

	if _, ok := parsed["showToast"]; !ok {
		t.Error("expected showToast key after overwriting invalid header")
	}
}

func TestSetToast_HeaderAccessibleViaResponseWriter(t *testing.T) {
	// Verify the header works through the standard http.ResponseWriter interface,
	// matching how PocketBase handlers access e.Response.
	rec := httptest.NewRecorder()
	var rw http.ResponseWriter = rec

	e := &core.RequestEvent{}
	e.Response = rw

	SetToast(e, "warning", "Check this")

	got := rw.Header().Get("HX-Trigger")
	if got == "" {
		t.Fatal("expected HX-Trigger header to be accessible via http.ResponseWriter")
	}
}

func TestErrorToast_SetsHeaderAndReswap(t *testing.T) {
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{}
	e.Response = rec

	err := ErrorToast(e, http.StatusNotFound, "Project not found")
	if err != nil {
		t.Fatalf("ErrorToast returned error: %v", err)
	}

	// Check HX-Trigger header has error toast
	trigger := rec.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Fatal("Expected HX-Trigger header to be set")
	}
	var parsed map[string]map[string]string
	if err := json.Unmarshal([]byte(trigger), &parsed); err != nil {
		t.Fatalf("Failed to parse HX-Trigger JSON: %v", err)
	}
	toast, ok := parsed["showToast"]
	if !ok {
		t.Fatal("Expected showToast key in HX-Trigger")
	}
	if toast["type"] != "error" {
		t.Errorf("Expected type 'error', got %q", toast["type"])
	}
	if toast["message"] != "Project not found" {
		t.Errorf("Expected message 'Project not found', got %q", toast["message"])
	}

	// Check HX-Reswap header
	reswap := rec.Header().Get("HX-Reswap")
	if reswap != "none" {
		t.Errorf("Expected HX-Reswap 'none', got %q", reswap)
	}

	// Check response body
	if rec.Body.String() != "Project not found" {
		t.Errorf("Expected body 'Project not found', got %q", rec.Body.String())
	}

	// Check status code
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestErrorToast_StatusCodes(t *testing.T) {
	tests := []struct {
		name string
		code int
		msg  string
	}{
		{"bad request", http.StatusBadRequest, "Invalid input"},
		{"not found", http.StatusNotFound, "Not found"},
		{"conflict", http.StatusConflict, "Resource conflict"},
		{"server error", http.StatusInternalServerError, "Something went wrong"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e := &core.RequestEvent{}
			e.Response = rec

			ErrorToast(e, tt.code, tt.msg)

			if rec.Code != tt.code {
				t.Errorf("Expected status %d, got %d", tt.code, rec.Code)
			}
			if rec.Header().Get("HX-Reswap") != "none" {
				t.Error("Expected HX-Reswap: none")
			}
		})
	}
}
