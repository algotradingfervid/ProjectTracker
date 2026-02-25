package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"github.com/pocketbase/pocketbase/core"
)

// SetToast sets the HX-Trigger response header to show a toast notification
// on the client via HTMX. If an HX-Trigger header already exists, the toast
// payload is merged into the existing JSON object.
// It also sets a flash cookie so toasts survive regular (non-HTMX) redirects.
func SetToast(e *core.RequestEvent, toastType string, message string) {
	toast := map[string]any{
		"showToast": map[string]string{
			"message": message,
			"type":    toastType,
		},
	}

	existing := e.Response.Header().Get("HX-Trigger")
	if existing == "" {
		data, err := json.Marshal(toast)
		if err != nil {
			log.Printf("toast: failed to marshal HX-Trigger JSON: %v", err)
			return
		}
		e.Response.Header().Set("HX-Trigger", string(data))
	} else {
		// Merge with existing HX-Trigger value
		var merged map[string]any
		if err := json.Unmarshal([]byte(existing), &merged); err != nil {
			log.Printf("toast: existing HX-Trigger is not valid JSON, overwriting: %v", err)
			data, err := json.Marshal(toast)
			if err != nil {
				log.Printf("toast: failed to marshal HX-Trigger JSON: %v", err)
				return
			}
			e.Response.Header().Set("HX-Trigger", string(data))
		} else {
			merged["showToast"] = toast["showToast"]
			data, err := json.Marshal(merged)
			if err != nil {
				log.Printf("toast: failed to marshal merged HX-Trigger JSON: %v", err)
				return
			}
			e.Response.Header().Set("HX-Trigger", string(data))
		}
	}

	// Also set a flash cookie for non-HTMX redirects (302) where HX-Trigger is lost
	toastData := map[string]string{"message": message, "type": toastType}
	cookieVal, err := json.Marshal(toastData)
	if err == nil {
		http.SetCookie(e.Response, &http.Cookie{
			Name:     "flash_toast",
			Value:    url.QueryEscape(string(cookieVal)),
			Path:     "/",
			MaxAge:   10,
			HttpOnly: false, // JS needs to read it
			SameSite: http.SameSiteLaxMode,
		})
	}
}

// ErrorToast sets an error toast and prevents HTMX from swapping the error text into the DOM.
// It sets HX-Reswap: none so the response body is ignored by HTMX, while the HX-Trigger
// header still fires the toast event.
func ErrorToast(e *core.RequestEvent, statusCode int, message string) error {
	SetToast(e, "error", message)
	e.Response.Header().Set("HX-Reswap", "none")
	return e.String(statusCode, message)
}
