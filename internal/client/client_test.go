package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("rejects invalid endpoint", func(t *testing.T) {
		t.Parallel()

		_, err := New(Config{
			Endpoint:    "://bad",
			AccessToken: "token",
		})

		if err == nil {
			t.Fatal("expected invalid endpoint error")
		}

		if !strings.Contains(err.Error(), "endpoint") {
			t.Fatalf("expected endpoint error, got %v", err)
		}
	})

	t.Run("requires one authentication method", func(t *testing.T) {
		t.Parallel()

		_, err := New(Config{
			Endpoint: "https://superset.example.com",
		})

		if err == nil {
			t.Fatal("expected authentication error")
		}

		if !strings.Contains(err.Error(), "authentication") {
			t.Fatalf("expected authentication error, got %v", err)
		}
	})
}

func TestClientGetUsesProvidedAccessToken(t *testing.T) {
	t.Parallel()

	loginCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/security/login":
			loginCalls++
			t.Fatal("did not expect login request when access_token is configured")
		case "/api/v1/me/":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET request, got %s", r.Method)
			}

			if got := r.Header.Get("Authorization"); got != "Bearer static-token" {
				t.Fatalf("expected bearer token, got %q", got)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"username":"admin"}`))
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

	var me map[string]string

	if err := c.Get(context.Background(), "/api/v1/me/", &me); err != nil {
		t.Fatalf("expected successful GET request, got error: %v", err)
	}

	if loginCalls != 0 {
		t.Fatalf("expected no login calls, got %d", loginCalls)
	}

	if got := me["username"]; got != "admin" {
		t.Fatalf("expected response to decode, got %q", got)
	}
}

func TestClientGetAuthenticatesOnceAndReusesToken(t *testing.T) {
	t.Parallel()

	loginCalls := 0
	meCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/security/login":
			loginCalls++

			if r.Method != http.MethodPost {
				t.Fatalf("expected POST login request, got %s", r.Method)
			}

			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected JSON login request, got %q", got)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("expected to read login body, got %v", err)
			}

			bodyString := string(body)
			for _, expected := range []string{`"username":"admin"`, `"password":"secret"`, `"provider":"db"`} {
				if !strings.Contains(bodyString, expected) {
					t.Fatalf("expected login body to contain %s, got %s", expected, bodyString)
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"login-token"}`))
		case "/api/v1/me/":
			meCalls++

			if got := r.Header.Get("Authorization"); got != "Bearer login-token" {
				t.Fatalf("expected bearer token from login response, got %q", got)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"username":"admin"}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	c, err := New(Config{
		Endpoint:   server.URL,
		Username:   "admin",
		Password:   "secret",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	for range 2 {
		var me map[string]string

		if err := c.Get(context.Background(), "/api/v1/me/", &me); err != nil {
			t.Fatalf("expected successful GET request, got error: %v", err)
		}
	}

	if loginCalls != 1 {
		t.Fatalf("expected one login request, got %d", loginCalls)
	}

	if meCalls != 2 {
		t.Fatalf("expected two API requests, got %d", meCalls)
	}

	if got := c.AccessToken(); got != "login-token" {
		t.Fatalf("expected cached access token, got %q", got)
	}
}

func TestClientPostReturnsAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chart/" {
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}

		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer static-token" {
			t.Fatalf("expected bearer token, got %q", got)
		}

		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"chart payload rejected"}`))
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

	err = c.Post(context.Background(), "/api/v1/chart/", map[string]string{"slice_name": "broken"}, nil)
	if err == nil {
		t.Fatal("expected API error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400, got %d", apiErr.StatusCode)
	}

	if !strings.Contains(apiErr.Body, "chart payload rejected") {
		t.Fatalf("expected error body to be preserved, got %q", apiErr.Body)
	}
}
