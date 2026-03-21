package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestValidatePaginationHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := validatePagination(ctx, 0, 2)
	if err != context.Canceled {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestValidatePaginationRejectsExcessivePages(t *testing.T) {
	t.Parallel()

	err := validatePagination(context.Background(), 2, 2)
	if err == nil {
		t.Fatal("expected excessive pagination to fail")
	}

	if !strings.Contains(err.Error(), "exceeded 2 pages") {
		t.Fatalf("expected excessive pagination error, got %v", err)
	}
}

func TestListDashboardsRejectsExcessivePagination(t *testing.T) {
	t.Parallel()

	requestedPages := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/dashboard/" {
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}

		requestedPages = append(requestedPages, r.URL.Query().Get("page"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":0,"result":[{"id":1,"dashboard_title":"Operations"}]}`))
	}))
	defer server.Close()

	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("expected valid test server URL, got %v", err)
	}

	c := &Client{
		baseURL:            endpoint,
		httpClient:         server.Client(),
		accessToken:        "static-token",
		maxPaginationPages: 2,
	}

	_, err = c.ListDashboards(context.Background(), 1)
	if err == nil {
		t.Fatal("expected excessive pagination to fail")
	}

	if !strings.Contains(err.Error(), "exceeded 2 pages") {
		t.Fatalf("expected excessive pagination error, got %v", err)
	}

	if len(requestedPages) != 2 || requestedPages[0] != "0" || requestedPages[1] != "1" {
		t.Fatalf("expected requests for pages 0 and 1 before stopping, got %#v", requestedPages)
	}
}
