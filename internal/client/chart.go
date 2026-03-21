package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Chart struct {
	ID                 int64   `json:"id,omitempty"`
	UUID               string  `json:"uuid,omitempty"`
	SliceName          string  `json:"slice_name,omitempty"`
	Description        string  `json:"description,omitempty"`
	VizType            string  `json:"viz_type,omitempty"`
	Params             string  `json:"params,omitempty"`
	QueryContext       *string `json:"query_context,omitempty"`
	CacheTimeout       *int64  `json:"cache_timeout,omitempty"`
	DatasourceID       int64   `json:"datasource_id,omitempty"`
	DatasourceType     string  `json:"datasource_type,omitempty"`
	DatasourceNameText string  `json:"datasource_name_text,omitempty"`
	URL                string  `json:"url,omitempty"`
}

func (c *Chart) UnmarshalJSON(data []byte) error {
	type chartAlias struct {
		ID                 int64           `json:"id,omitempty"`
		UUID               string          `json:"uuid,omitempty"`
		SliceName          string          `json:"slice_name,omitempty"`
		Description        string          `json:"description,omitempty"`
		VizType            string          `json:"viz_type,omitempty"`
		Params             string          `json:"params,omitempty"`
		QueryContext       *string         `json:"query_context,omitempty"`
		CacheTimeout       json.RawMessage `json:"cache_timeout,omitempty"`
		DatasourceID       int64           `json:"datasource_id,omitempty"`
		DatasourceType     string          `json:"datasource_type,omitempty"`
		DatasourceNameText string          `json:"datasource_name_text,omitempty"`
		URL                string          `json:"url,omitempty"`
	}

	var aux chartAlias

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	cacheTimeout, err := parseNullableInt64(aux.CacheTimeout)
	if err != nil {
		return err
	}

	c.ID = aux.ID
	c.UUID = aux.UUID
	c.SliceName = aux.SliceName
	c.Description = aux.Description
	c.VizType = aux.VizType
	c.Params = aux.Params
	c.QueryContext = aux.QueryContext
	c.CacheTimeout = cacheTimeout
	c.DatasourceID = aux.DatasourceID
	c.DatasourceType = aux.DatasourceType
	c.DatasourceNameText = aux.DatasourceNameText
	c.URL = aux.URL

	return nil
}

type ChartCreateRequest struct {
	SliceName              string  `json:"slice_name"`
	Description            *string `json:"description,omitempty"`
	VizType                string  `json:"viz_type"`
	Params                 string  `json:"params"`
	QueryContext           *string `json:"query_context,omitempty"`
	QueryContextGeneration *bool   `json:"query_context_generation,omitempty"`
	CacheTimeout           *int64  `json:"cache_timeout,omitempty"`
	DatasourceID           int64   `json:"datasource_id"`
	DatasourceType         string  `json:"datasource_type"`
}

type ChartUpdateRequest struct {
	SliceName      string  `json:"slice_name"`
	Description    *string `json:"description,omitempty"`
	VizType        string  `json:"viz_type"`
	Params         string  `json:"params"`
	QueryContext   *string `json:"query_context,omitempty"`
	CacheTimeout   *int64  `json:"cache_timeout,omitempty"`
	DatasourceID   int64   `json:"datasource_id"`
	DatasourceType string  `json:"datasource_type"`

	IncludeDescription  bool
	IncludeQueryContext bool
	IncludeCacheTimeout bool
}

type chartResponse struct {
	ID     int64 `json:"id"`
	Result Chart `json:"result"`
}

type chartListResponse struct {
	Count  int     `json:"count"`
	Result []Chart `json:"result"`
}

func (c *Client) CreateChart(ctx context.Context, request ChartCreateRequest) (*Chart, error) {
	var response chartResponse

	if err := c.Post(ctx, "/api/v1/chart/", request, &response); err != nil {
		return nil, err
	}

	return chartFromResponse(response), nil
}

func (c *Client) GetChart(ctx context.Context, id int64) (*Chart, error) {
	var response chartResponse

	if err := c.Get(ctx, chartPath(id), &response); err != nil {
		return nil, err
	}

	return chartFromResponse(response), nil
}

func (c *Client) UpdateChart(ctx context.Context, id int64, request ChartUpdateRequest) (*Chart, error) {
	var response chartResponse

	if err := c.Put(ctx, chartPath(id), request, &response); err != nil {
		return nil, err
	}

	return chartFromResponse(response), nil
}

func (c *Client) DeleteChart(ctx context.Context, id int64) error {
	var response map[string]any

	return c.Delete(ctx, chartPath(id), &response)
}

func (c *Client) ListCharts(ctx context.Context, pageSize int) ([]Chart, error) {
	if pageSize <= 0 {
		pageSize = 1000
	}

	charts := make([]Chart, 0, pageSize)

	for page := 0; ; page++ {
		if err := validatePagination(ctx, page, c.paginationLimit()); err != nil {
			return nil, err
		}

		values := url.Values{}
		values.Set("page", strconv.Itoa(page))
		values.Set("page_size", strconv.Itoa(pageSize))

		var response chartListResponse

		if err := c.Get(ctx, fmt.Sprintf("/api/v1/chart/?%s", values.Encode()), &response); err != nil {
			return nil, err
		}

		charts = append(charts, response.Result...)

		if len(response.Result) == 0 || len(response.Result) < pageSize {
			return charts, nil
		}

		if response.Count > 0 && len(charts) >= response.Count {
			return charts, nil
		}
	}
}

func (r ChartUpdateRequest) MarshalJSON() ([]byte, error) {
	body := map[string]any{
		"slice_name":      r.SliceName,
		"viz_type":        r.VizType,
		"params":          r.Params,
		"datasource_id":   r.DatasourceID,
		"datasource_type": r.DatasourceType,
	}

	if r.IncludeDescription {
		body["description"] = nullableStringValue(r.Description)
	}

	if r.IncludeQueryContext {
		body["query_context"] = nullableStringValue(r.QueryContext)
	}

	if r.IncludeCacheTimeout {
		body["cache_timeout"] = nullableInt64Value(r.CacheTimeout)
	}

	return json.Marshal(body)
}

func chartFromResponse(response chartResponse) *Chart {
	chart := response.Result

	if chart.ID == 0 {
		chart.ID = response.ID
	}

	return &chart
}

func chartPath(id int64) string {
	return fmt.Sprintf("/api/v1/chart/%d", id)
}

func parseNullableInt64(value json.RawMessage) (*int64, error) {
	if len(value) == 0 || string(value) == "null" {
		return nil, nil
	}

	var number int64
	if err := json.Unmarshal(value, &number); err == nil {
		return &number, nil
	}

	var text string
	if err := json.Unmarshal(value, &text); err != nil {
		return nil, err
	}

	if strings.TrimSpace(text) == "" {
		return nil, nil
	}

	number, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return nil, err
	}

	return &number, nil
}
