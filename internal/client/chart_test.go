package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChartUpdateRequestMarshalJSONIncludesNullsForClears(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(ChartUpdateRequest{
		SliceName:           "Orders",
		VizType:             "table",
		Params:              `{"datasource":"11__table","viz_type":"table"}`,
		DatasourceID:        11,
		DatasourceType:      "table",
		IncludeDescription:  true,
		IncludeQueryContext: true,
		IncludeCacheTimeout: true,
	})
	if err != nil {
		t.Fatalf("expected chart update request to marshal, got error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("expected chart update JSON to decode, got error: %v", err)
	}

	for _, key := range []string{"description", "query_context", "cache_timeout"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("expected %q to be present in chart update payload", key)
		}

		if body[key] != nil {
			t.Fatalf("expected %q to encode as null, got %#v", key, body[key])
		}
	}
}

func TestListChartsPaginates(t *testing.T) {
	t.Parallel()

	requestedPages := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chart/" {
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}

		requestedPages = append(requestedPages, r.URL.Query().Get("page"))

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Query().Get("page") {
		case "0":
			_, _ = w.Write([]byte(`{"count":2,"result":[{"id":1,"slice_name":"Orders","datasource_id":7,"datasource_type":"table"}]}`))
		case "1":
			_, _ = w.Write([]byte(`{"count":2,"result":[{"id":2,"slice_name":"Revenue","datasource_id":7,"datasource_type":"table"}]}`))
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

	charts, err := c.ListCharts(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected paginated list request to succeed, got error: %v", err)
	}

	if len(charts) != 2 {
		t.Fatalf("expected two charts across pages, got %d", len(charts))
	}

	if len(requestedPages) != 2 || requestedPages[0] != "0" || requestedPages[1] != "1" {
		t.Fatalf("expected pagination to request pages 0 and 1, got %#v", requestedPages)
	}
}
