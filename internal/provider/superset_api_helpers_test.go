// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExecuteSupersetAPIRequest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		method      string
		requestBody types.String
	}{
		{
			name:        "get",
			method:      supersetAPIMethodGet,
			requestBody: types.StringNull(),
		},
		{
			name:        "post",
			method:      supersetAPIMethodPost,
			requestBody: types.StringValue(`{"name":"created"}`),
		},
		{
			name:        "put case insensitive",
			method:      "put",
			requestBody: types.StringValue(`{"name":"updated"}`),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/security/csrf_token/":
					if testCase.method == supersetAPIMethodGet {
						t.Fatal("GET must not request a CSRF token")
					}

					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"result":"csrf-token"}`))
				case "/api/v1/example/":
					expectedMethod := strings.ToUpper(testCase.method)
					if r.Method != expectedMethod {
						t.Fatalf("expected method %s, got %s", expectedMethod, r.Method)
					}

					if got := r.Header.Get("Authorization"); got != "Bearer static-token" {
						t.Fatalf("expected bearer token, got %q", got)
					}

					if got := r.URL.Query().Get("q"); got != "terraform" {
						t.Fatalf("expected query parameter, got %q", got)
					}

					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Fatalf("read request body: %v", err)
					}

					if testCase.requestBody.IsNull() {
						if len(body) != 0 {
							t.Fatalf("expected empty request body, got %s", body)
						}
					} else if string(body) != testCase.requestBody.ValueString() {
						t.Fatalf("expected request body %s, got %s", testCase.requestBody.ValueString(), body)
					}

					if expectedMethod != supersetAPIMethodGet && r.Header.Get("X-CSRFToken") != "csrf-token" {
						t.Fatalf("expected CSRF token on %s request", expectedMethod)
					}

					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte("{\n  \"result\": {\"ok\": true}\n}"))
				default:
					t.Fatalf("unexpected request path %q", r.URL.Path)
				}
			}))
			defer server.Close()

			client, err := supersetclient.New(supersetclient.Config{
				Endpoint:    server.URL,
				AccessToken: "static-token",
				HTTPClient:  server.Client(),
			})
			if err != nil {
				t.Fatalf("create test client: %v", err)
			}

			response, err := executeSupersetAPIRequest(
				context.Background(),
				client,
				testCase.method,
				"/api/v1/example/?q=terraform",
				testCase.requestBody,
			)
			if err != nil {
				t.Fatalf("execute API request: %v", err)
			}

			if response.IsNull() || response.ValueString() != `{"result":{"ok":true}}` {
				t.Fatalf("expected compact JSON response, got %#v", response)
			}
		})
	}
}

func TestExecuteSupersetAPIRequestWithEmptyResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := supersetclient.New(supersetclient.Config{
		Endpoint:    server.URL,
		AccessToken: "static-token",
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("create test client: %v", err)
	}

	response, err := executeSupersetAPIRequest(context.Background(), client, supersetAPIMethodGet, "/empty", types.StringNull())
	if err != nil {
		t.Fatalf("execute API request: %v", err)
	}

	if !response.IsNull() {
		t.Fatalf("expected null response for HTTP 204, got %#v", response)
	}
}

func TestSupersetAPIMethodValidator(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name      string
		value     string
		wantError bool
	}{
		{name: "post", value: "POST"},
		{name: "put case insensitive", value: "put"},
		{name: "get", value: "GET", wantError: true},
		{name: "blank", value: " ", wantError: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			resp := &validator.StringResponse{}
			supersetAPIMethodValidator().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("method"),
				ConfigValue: types.StringValue(testCase.value),
			}, resp)

			if got := resp.Diagnostics.HasError(); got != testCase.wantError {
				t.Fatalf("expected error=%t, got diagnostics %v", testCase.wantError, resp.Diagnostics)
			}
		})
	}
}

func TestSupersetAPIPathValidator(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name      string
		value     string
		wantError bool
	}{
		{name: "path", value: "/api/v1/chart/"},
		{name: "path with query", value: "/api/v1/chart/?q=(page:0)"},
		{name: "absolute URL", value: "https://example.com/api/v1/chart/", wantError: true},
		{name: "scheme relative URL", value: "//example.com/api/v1/chart/", wantError: true},
		{name: "missing leading slash", value: "api/v1/chart/", wantError: true},
		{name: "fragment", value: "/api/v1/chart/#result", wantError: true},
		{name: "blank", value: " ", wantError: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			resp := &validator.StringResponse{}
			supersetAPIPathValidator().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("path"),
				ConfigValue: types.StringValue(testCase.value),
			}, resp)

			if got := resp.Diagnostics.HasError(); got != testCase.wantError {
				t.Fatalf("expected error=%t, got diagnostics %v", testCase.wantError, resp.Diagnostics)
			}
		})
	}
}

func TestJSONStringValidator(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name      string
		value     string
		wantError bool
	}{
		{name: "object", value: `{"name":"dashboard"}`},
		{name: "null", value: "null"},
		{name: "blank", value: " ", wantError: true},
		{name: "invalid", value: "not-json", wantError: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			resp := &validator.StringResponse{}
			jsonStringValidator().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("request_body"),
				ConfigValue: types.StringValue(testCase.value),
			}, resp)

			if got := resp.Diagnostics.HasError(); got != testCase.wantError {
				t.Fatalf("expected error=%t, got diagnostics %v", testCase.wantError, resp.Diagnostics)
			}
		})
	}
}

func TestSupersetAPIOperationID(t *testing.T) {
	t.Parallel()

	got := supersetAPIOperationID(" post ", " /api/v1/example/ ")
	if got.IsNull() || got.ValueString() != "POST /api/v1/example/" {
		t.Fatalf("unexpected operation ID %#v", got)
	}
}
