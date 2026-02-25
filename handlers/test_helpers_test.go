package handlers

import (
	"net/http"
	"net/http/httptest"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// newTestRequestEvent creates a RequestEvent suitable for handler tests.
func newTestRequestEvent(app *pocketbase.PocketBase, req *http.Request, rec *httptest.ResponseRecorder) *core.RequestEvent {
	e := &core.RequestEvent{}
	e.App = app
	e.Request = req
	e.Response = rec
	return e
}
