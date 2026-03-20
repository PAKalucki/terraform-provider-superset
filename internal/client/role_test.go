package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListRolesPaginates(t *testing.T) {
	t.Parallel()

	requestedQueries := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/security/roles/search/" {
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}

		requestedQueries = append(requestedQueries, r.URL.Query().Get("q"))

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Query().Get("q") {
		case "(page:0,page_size:1)":
			_, _ = w.Write([]byte(`{"count":2,"result":[{"id":1,"name":"Admin","permission_ids":[1,2]}]}`))
		case "(page:1,page_size:1)":
			_, _ = w.Write([]byte(`{"count":2,"result":[{"id":2,"name":"Gamma","permission_ids":[3]}]}`))
		default:
			t.Fatalf("unexpected query %q", r.URL.Query().Get("q"))
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

	roles, err := c.ListRoles(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected paginated list request to succeed, got error: %v", err)
	}

	if len(roles) != 2 {
		t.Fatalf("expected two roles across pages, got %d", len(roles))
	}

	if len(requestedQueries) != 2 || requestedQueries[0] != "(page:0,page_size:1)" || requestedQueries[1] != "(page:1,page_size:1)" {
		t.Fatalf("expected pagination queries for pages 0 and 1, got %#v", requestedQueries)
	}
}

func TestListPermissionsPaginates(t *testing.T) {
	t.Parallel()

	requestedQueries := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/security/permissions-resources/" {
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}

		requestedQueries = append(requestedQueries, r.URL.Query().Get("q"))

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Query().Get("q") {
		case "(page:0,page_size:1)":
			_, _ = w.Write([]byte(`{"count":2,"result":[{"id":13,"permission":{"name":"can_read"},"view_menu":{"name":"Log"}}]}`))
		case "(page:1,page_size:1)":
			_, _ = w.Write([]byte(`{"count":2,"result":[{"id":15,"permission":{"name":"can_read"},"view_menu":{"name":"Dashboard"}}]}`))
		default:
			t.Fatalf("unexpected query %q", r.URL.Query().Get("q"))
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

	permissions, err := c.ListPermissions(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected paginated list request to succeed, got error: %v", err)
	}

	if len(permissions) != 2 {
		t.Fatalf("expected two permissions across pages, got %d", len(permissions))
	}

	if got := permissions[0].PermissionName; got != "can_read" {
		t.Fatalf("expected flattened permission name, got %q", got)
	}

	if got := permissions[1].ViewMenuName; got != "Dashboard" {
		t.Fatalf("expected flattened view menu name, got %q", got)
	}

	if len(requestedQueries) != 2 || requestedQueries[0] != "(page:0,page_size:1)" || requestedQueries[1] != "(page:1,page_size:1)" {
		t.Fatalf("expected pagination queries for pages 0 and 1, got %#v", requestedQueries)
	}
}

func TestSetRolePermissionsSortsPermissionIDs(t *testing.T) {
	t.Parallel()

	csrfCalls := 0
	assignCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/security/csrf_token/":
			csrfCalls++

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":"csrf-token"}`))
		case "/api/v1/security/roles/7/permissions":
			assignCalls++

			if r.Method != http.MethodPost {
				t.Fatalf("expected POST request, got %s", r.Method)
			}

			if got := r.Header.Get("Authorization"); got != "Bearer static-token" {
				t.Fatalf("expected bearer token, got %q", got)
			}

			if got := r.Header.Get("X-CSRFToken"); got != "csrf-token" {
				t.Fatalf("expected CSRF token, got %q", got)
			}

			var payload map[string][]int64
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("expected JSON request body, got error: %v", err)
			}

			if got := payload["permission_view_menu_ids"]; len(got) != 3 || got[0] != 13 || got[1] != 14 || got[2] != 15 {
				t.Fatalf("expected sorted permission ids, got %#v", got)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":{"permission_view_menu_ids":[13,14,15]}}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
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

	if err := c.SetRolePermissions(context.Background(), 7, []int64{15, 13, 14}); err != nil {
		t.Fatalf("expected permission assignment to succeed, got error: %v", err)
	}

	if csrfCalls != 1 {
		t.Fatalf("expected one CSRF request, got %d", csrfCalls)
	}

	if assignCalls != 1 {
		t.Fatalf("expected one permission assignment request, got %d", assignCalls)
	}
}

func TestSetRolePermissionsEncodesEmptyList(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/security/csrf_token/":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":"csrf-token"}`))
		case "/api/v1/security/roles/7/permissions":
			var payload map[string][]int64
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("expected JSON request body, got error: %v", err)
			}

			if got := payload["permission_view_menu_ids"]; got == nil || len(got) != 0 {
				t.Fatalf("expected empty permission list, got %#v", got)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":{"permission_view_menu_ids":[]}}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
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

	if err := c.SetRolePermissions(context.Background(), 7, nil); err != nil {
		t.Fatalf("expected empty permission assignment to succeed, got error: %v", err)
	}
}
