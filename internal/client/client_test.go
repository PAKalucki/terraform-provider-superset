package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

	serverURL := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chart/" {
			if r.URL.Path == "/api/v1/security/csrf_token/" {
				http.SetCookie(w, &http.Cookie{
					Name:  "session",
					Value: "csrf-cookie",
					Path:  "/",
				})

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"result":"csrf-token"}`))

				return
			}

			t.Fatalf("unexpected request path %q", r.URL.Path)
		}

		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer static-token" {
			t.Fatalf("expected bearer token, got %q", got)
		}

		if got := r.Header.Get("X-CSRFToken"); got != "csrf-token" {
			t.Fatalf("expected CSRF token, got %q", got)
		}

		if got := r.Header.Get("Referer"); got != serverURL+"/" {
			t.Fatalf("expected referer header, got %q", got)
		}

		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"chart payload rejected"}`))
	}))
	defer server.Close()
	serverURL = server.URL

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

func TestClientPostFetchesCSRFTokenAndReusesCookies(t *testing.T) {
	t.Parallel()

	csrfCalls := 0
	createCalls := 0
	serverURL := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/security/csrf_token/":
			csrfCalls++

			if r.Method != http.MethodGet {
				t.Fatalf("expected GET CSRF request, got %s", r.Method)
			}

			if got := r.Header.Get("Authorization"); got != "Bearer static-token" {
				t.Fatalf("expected bearer token on CSRF request, got %q", got)
			}

			http.SetCookie(w, &http.Cookie{
				Name:  "session",
				Value: "csrf-cookie",
				Path:  "/",
			})

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":"csrf-token"}`))
		case "/api/v1/database/":
			createCalls++

			if r.Method != http.MethodPost {
				t.Fatalf("expected POST database request, got %s", r.Method)
			}

			if got := r.Header.Get("Authorization"); got != "Bearer static-token" {
				t.Fatalf("expected bearer token on create request, got %q", got)
			}

			if got := r.Header.Get("X-CSRFToken"); got != "csrf-token" {
				t.Fatalf("expected CSRF token header, got %q", got)
			}

			if got := r.Header.Get("Referer"); got != serverURL+"/" {
				t.Fatalf("expected referer header on create request, got %q", got)
			}

			if got := r.Header.Get("Cookie"); !strings.Contains(got, "session=csrf-cookie") {
				t.Fatalf("expected session cookie on create request, got %q", got)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":12,"result":{"database_name":"analytics"}}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	c, err := New(Config{
		Endpoint:    server.URL,
		AccessToken: "static-token",
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	for range 2 {
		if err := c.Post(context.Background(), "/api/v1/database/", map[string]string{"database_name": "analytics"}, nil); err != nil {
			t.Fatalf("expected successful POST request, got error: %v", err)
		}
	}

	if csrfCalls != 1 {
		t.Fatalf("expected one CSRF request, got %d", csrfCalls)
	}

	if createCalls != 2 {
		t.Fatalf("expected two create requests, got %d", createCalls)
	}
}

func TestClientPostUsesCSRFTokenSafelyUnderConcurrency(t *testing.T) {
	t.Parallel()

	csrfCalls := 0
	createCalls := 0
	var mu sync.Mutex
	serverURL := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch r.URL.Path {
		case "/api/v1/security/csrf_token/":
			csrfCalls++

			http.SetCookie(w, &http.Cookie{
				Name:  "session",
				Value: "csrf-cookie",
				Path:  "/",
			})

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":"csrf-token"}`))
		case "/api/v1/database/":
			createCalls++

			if got := r.Header.Get("X-CSRFToken"); got != "csrf-token" {
				t.Fatalf("expected CSRF token header, got %q", got)
			}

			if got := r.Header.Get("Referer"); got != serverURL+"/" {
				t.Fatalf("expected referer header on create request, got %q", got)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":12,"result":{"database_name":"analytics"}}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	c, err := New(Config{
		Endpoint:    server.URL,
		AccessToken: "static-token",
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, 2)

	for range 2 {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start
			errs <- c.Post(context.Background(), "/api/v1/database/", map[string]string{"database_name": "analytics"}, nil)
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("expected successful POST request, got error: %v", err)
		}
	}

	if csrfCalls != 1 {
		t.Fatalf("expected one CSRF request, got %d", csrfCalls)
	}

	if createCalls != 2 {
		t.Fatalf("expected two create requests, got %d", createCalls)
	}
}

func TestClientPostSendsRefererForBasePathEndpoint(t *testing.T) {
	t.Parallel()

	serverURL := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/superset/api/v1/security/csrf_token/":
			http.SetCookie(w, &http.Cookie{
				Name:  "session",
				Value: "csrf-cookie",
				Path:  "/",
			})

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":"csrf-token"}`))
		case "/superset/api/v1/database/":
			if got := r.Header.Get("Referer"); got != serverURL+"/superset/" {
				t.Fatalf("expected referer with endpoint base path, got %q", got)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":12,"result":{"database_name":"analytics"}}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	c, err := New(Config{
		Endpoint:    serverURL + "/superset",
		AccessToken: "static-token",
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	if err := c.Post(context.Background(), "/api/v1/database/", map[string]string{"database_name": "analytics"}, nil); err != nil {
		t.Fatalf("expected successful POST request, got error: %v", err)
	}
}

func TestResolveURLKeepsQueryString(t *testing.T) {
	t.Parallel()

	c, err := New(Config{
		Endpoint:    "https://superset.example.com/base",
		AccessToken: "static-token",
	})
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	got := c.resolveURL("/api/v1/database/?page_size=1000")
	want := "https://superset.example.com/base/api/v1/database/?page_size=1000"

	if got != want {
		t.Fatalf("expected resolved URL %q, got %q", want, got)
	}
}
