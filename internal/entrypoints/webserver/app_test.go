package webserver

import (
	"net/http/httptest"
	"testing"
)

func TestUIRoutes(t *testing.T) {
	app, err := NewApplication(":0", nil)
	if err != nil {
		t.Fatalf("new application: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	app.server.Handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected / status 200, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/ui", nil)
	app.server.Handler.ServeHTTP(rec, req)
	if rec.Code != 301 {
		t.Fatalf("expected /ui status 301, got %d", rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/ui/" {
		t.Fatalf("expected /ui redirect to /ui/, got %q", location)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/ui/", nil)
	app.server.Handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected /ui/ status 200, got %d", rec.Code)
	}
}
