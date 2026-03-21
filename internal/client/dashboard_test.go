package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashboardUpdateRequestMarshalJSONIncludesNullsForClears(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(DashboardUpdateRequest{
		DashboardTitle:      "Operations",
		IncludeSlug:         true,
		IncludeCSS:          true,
		IncludePublished:    true,
		IncludePositionJSON: true,
		IncludeJSONMetadata: true,
	})
	if err != nil {
		t.Fatalf("expected dashboard update request to marshal, got error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("expected dashboard update JSON to decode, got error: %v", err)
	}

	for _, key := range []string{"slug", "css", "published", "position_json", "json_metadata"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("expected %q to be present in dashboard update payload", key)
		}

		if body[key] != nil {
			t.Fatalf("expected %q to encode as null, got %#v", key, body[key])
		}
	}
}

func TestListDashboardsPaginates(t *testing.T) {
	t.Parallel()

	requestedPages := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/dashboard/" {
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}

		requestedPages = append(requestedPages, r.URL.Query().Get("page"))

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Query().Get("page") {
		case "0":
			_, _ = w.Write([]byte(`{"count":2,"result":[{"id":1,"dashboard_title":"Operations"}]}`))
		case "1":
			_, _ = w.Write([]byte(`{"count":2,"result":[{"id":2,"dashboard_title":"Finance"}]}`))
		default:
			t.Fatalf("unexpected page %q", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	c, err := New(Config{
		Endpoint:    server.URL,
		AccessToken: "static-token",
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	dashboards, err := c.ListDashboards(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected paginated list request to succeed, got error: %v", err)
	}

	if len(dashboards) != 2 {
		t.Fatalf("expected two dashboards across pages, got %d", len(dashboards))
	}

	if len(requestedPages) != 2 || requestedPages[0] != "0" || requestedPages[1] != "1" {
		t.Fatalf("expected pagination to request pages 0 and 1, got %#v", requestedPages)
	}
}

func TestGetDashboardRejectsEmptyIdentifier(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("did not expect request for empty dashboard identifier")
	}))
	defer server.Close()

	c, err := New(Config{
		Endpoint:    server.URL,
		AccessToken: "static-token",
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	_, err = c.GetDashboard(context.Background(), "   ")
	if err == nil {
		t.Fatal("expected empty dashboard identifier to fail")
	}

	if !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("expected empty identifier error, got %v", err)
	}
}
