package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDatasetUpdateRequestMarshalJSONIncludesNullsForManagedClears(t *testing.T) {
	t.Parallel()

	columns := []DatasetColumn{
		{
			ID:                 9,
			ColumnName:         "id",
			IncludeVerboseName: true,
			IncludeFilterable:  true,
		},
	}
	metrics := []DatasetMetric{
		{
			ID:                 4,
			MetricName:         "event_count",
			Expression:         "COUNT(*)",
			IncludeVerboseName: true,
		},
	}

	payload, err := json.Marshal(DatasetUpdateRequest{
		DatabaseID:                  7,
		TableName:                   "events",
		Columns:                     &columns,
		Metrics:                     &metrics,
		IncludeSchema:               true,
		IncludeDescription:          true,
		IncludeMainDttmCol:          true,
		IncludeFilterSelectEnabled:  true,
		IncludeNormalizeColumns:     true,
		IncludeAlwaysFilterMainDttm: true,
		IncludeCacheTimeout:         true,
	})
	if err != nil {
		t.Fatalf("expected update request to marshal, got error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("expected update request JSON to decode, got error: %v", err)
	}

	for _, key := range []string{"schema", "description", "main_dttm_col", "filter_select_enabled", "normalize_columns", "always_filter_main_dttm", "cache_timeout"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("expected %q to be present in update payload", key)
		}

		if body[key] != nil {
			t.Fatalf("expected %q to encode as null when Terraform clears the field, got %#v", key, body[key])
		}
	}

	columnValues, ok := body["columns"].([]any)
	if !ok || len(columnValues) != 1 {
		t.Fatalf("expected one encoded column payload, got %#v", body["columns"])
	}

	columnBody, ok := columnValues[0].(map[string]any)
	if !ok {
		t.Fatalf("expected encoded column object, got %#v", columnValues[0])
	}

	if _, ok := columnBody["verbose_name"]; !ok {
		t.Fatal("expected column verbose_name to be present in update payload")
	}

	if columnBody["verbose_name"] != nil {
		t.Fatalf("expected column verbose_name to encode as null, got %#v", columnBody["verbose_name"])
	}

	if _, ok := columnBody["filterable"]; !ok {
		t.Fatal("expected column filterable to be present in update payload")
	}

	if columnBody["filterable"] != nil {
		t.Fatalf("expected column filterable to encode as null, got %#v", columnBody["filterable"])
	}

	metricValues, ok := body["metrics"].([]any)
	if !ok || len(metricValues) != 1 {
		t.Fatalf("expected one encoded metric payload, got %#v", body["metrics"])
	}

	metricBody, ok := metricValues[0].(map[string]any)
	if !ok {
		t.Fatalf("expected encoded metric object, got %#v", metricValues[0])
	}

	if _, ok := metricBody["verbose_name"]; !ok {
		t.Fatal("expected metric verbose_name to be present in update payload")
	}

	if metricBody["verbose_name"] != nil {
		t.Fatalf("expected metric verbose_name to encode as null, got %#v", metricBody["verbose_name"])
	}
}

func TestListDatabasesPaginates(t *testing.T) {
	t.Parallel()

	requestedPages := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/database/" {
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}

		requestedPages = append(requestedPages, r.URL.Query().Get("page"))

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Query().Get("page") {
		case "0":
			_, _ = w.Write([]byte(`{"count":2,"result":[{"id":1,"database_name":"analytics"}]}`))
		case "1":
			_, _ = w.Write([]byte(`{"count":2,"result":[{"id":2,"database_name":"warehouse"}]}`))
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

	databases, err := c.ListDatabases(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected paginated list request to succeed, got error: %v", err)
	}

	if len(databases) != 2 {
		t.Fatalf("expected two databases across pages, got %d", len(databases))
	}

	if len(requestedPages) != 2 || requestedPages[0] != "0" || requestedPages[1] != "1" {
		t.Fatalf("expected pagination to request pages 0 and 1, got %#v", requestedPages)
	}
}

func TestListDatasetsPaginates(t *testing.T) {
	t.Parallel()

	requestedPages := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/dataset/" {
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}

		requestedPages = append(requestedPages, r.URL.Query().Get("page"))

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Query().Get("page") {
		case "0":
			_, _ = w.Write([]byte(`{"count":2,"result":[{"id":11,"table_name":"events","database":{"id":5,"database_name":"analytics"}}]}`))
		case "1":
			_, _ = w.Write([]byte(`{"count":2,"result":[{"id":12,"table_name":"orders","database":{"id":5,"database_name":"analytics"}}]}`))
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

	datasets, err := c.ListDatasets(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected paginated list request to succeed, got error: %v", err)
	}

	if len(datasets) != 2 {
		t.Fatalf("expected two datasets across pages, got %d", len(datasets))
	}

	if len(requestedPages) != 2 || requestedPages[0] != "0" || requestedPages[1] != "1" {
		t.Fatalf("expected pagination to request pages 0 and 1, got %#v", requestedPages)
	}
}
