// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSupersetAPIResourceProtocolLifecycle(t *testing.T) {
	t.Parallel()

	var operationCalls atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/security/csrf_token/":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":"csrf-token"}`))
		case "/api/v1/example/":
			if r.Method != http.MethodPost {
				t.Errorf("expected POST request, got %s", r.Method)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			if r.Header.Get("X-CSRFToken") != "csrf-token" {
				t.Error("expected CSRF token")
				w.WriteHeader(http.StatusForbidden)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read request body: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			operationCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"result":%s}`, body)
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testSupersetAPIResourceProtocolConfig(server.URL, "created"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_api.test", "id", "POST /api/v1/example/"),
					resource.TestCheckResourceAttr("superset_api.test", "response_body", `{"result":{"value":"created"}}`),
				),
			},
			{
				Config: testSupersetAPIResourceProtocolConfig(server.URL, "updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_api.test", "id", "POST /api/v1/example/"),
					resource.TestCheckResourceAttr("superset_api.test", "response_body", `{"result":{"value":"updated"}}`),
				),
			},
		},
	})

	if got := operationCalls.Load(); got != 2 {
		t.Fatalf("expected exactly two POST operations, got %d", got)
	}
}

func TestSupersetAPIDataSourceProtocolRead(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/example/" || r.URL.Query().Get("q") != "terraform" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"value":"read"}}`))
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "superset" {
  endpoint     = %q
  access_token = "static-token"
}

data "superset_api" "test" {
  path = "/api/v1/example/?q=terraform"
}
`, server.URL),
				Check: resource.TestCheckResourceAttr(
					"data.superset_api.test",
					"response_body",
					`{"result":{"value":"read"}}`,
				),
			},
		},
	})
}

func testSupersetAPIResourceProtocolConfig(endpoint string, value string) string {
	return fmt.Sprintf(`
provider "superset" {
  endpoint     = %q
  access_token = "static-token"
}

resource "superset_api" "test" {
  method = "POST"
  path   = "/api/v1/example/"

  request_body = jsonencode({
    value = %q
  })
}
`, endpoint, value)
}
