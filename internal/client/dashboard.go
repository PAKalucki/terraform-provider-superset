package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Dashboard struct {
	ID             int64  `json:"id,omitempty"`
	UUID           string `json:"uuid,omitempty"`
	DashboardTitle string `json:"dashboard_title,omitempty"`
	Slug           string `json:"slug,omitempty"`
	URL            string `json:"url,omitempty"`
	Published      *bool  `json:"published,omitempty"`
	CSS            string `json:"css,omitempty"`
	PositionJSON   string `json:"position_json,omitempty"`
}

type DashboardChart struct {
	ID        int64  `json:"id,omitempty"`
	SliceName string `json:"slice_name,omitempty"`
}

type DashboardCreateRequest struct {
	DashboardTitle string  `json:"dashboard_title"`
	Slug           *string `json:"slug,omitempty"`
	CSS            *string `json:"css,omitempty"`
	Published      *bool   `json:"published,omitempty"`
}

type DashboardUpdateRequest struct {
	DashboardTitle string  `json:"dashboard_title"`
	Slug           *string `json:"slug,omitempty"`
	CSS            *string `json:"css,omitempty"`
	Published      *bool   `json:"published,omitempty"`
	PositionJSON   *string `json:"position_json,omitempty"`
	JSONMetadata   *string `json:"json_metadata,omitempty"`

	IncludeSlug         bool
	IncludeCSS          bool
	IncludePublished    bool
	IncludePositionJSON bool
	IncludeJSONMetadata bool
}

type dashboardResponse struct {
	ID     int64     `json:"id"`
	Result Dashboard `json:"result"`
}

type dashboardListResponse struct {
	Count  int         `json:"count"`
	Result []Dashboard `json:"result"`
}

type dashboardChartsResponse struct {
	Result []DashboardChart `json:"result"`
}

func (c *Client) CreateDashboard(ctx context.Context, request DashboardCreateRequest) (*Dashboard, error) {
	var response dashboardResponse

	if err := c.Post(ctx, "/api/v1/dashboard/", request, &response); err != nil {
		return nil, err
	}

	return dashboardFromResponse(response), nil
}

func (c *Client) GetDashboard(ctx context.Context, idOrSlug string) (*Dashboard, error) {
	var response dashboardResponse

	requestPath, err := dashboardPath(idOrSlug)
	if err != nil {
		return nil, err
	}

	if err := c.Get(ctx, requestPath, &response); err != nil {
		return nil, err
	}

	return dashboardFromResponse(response), nil
}

func (c *Client) UpdateDashboard(ctx context.Context, id int64, request DashboardUpdateRequest) error {
	var response dashboardResponse

	requestPath, err := dashboardPath(strconv.FormatInt(id, 10))
	if err != nil {
		return err
	}

	return c.Put(ctx, requestPath, request, &response)
}

func (c *Client) DeleteDashboard(ctx context.Context, id int64) error {
	var response map[string]any

	requestPath, err := dashboardPath(strconv.FormatInt(id, 10))
	if err != nil {
		return err
	}

	return c.Delete(ctx, requestPath, &response)
}

func (c *Client) ListDashboards(ctx context.Context, pageSize int) ([]Dashboard, error) {
	if pageSize <= 0 {
		pageSize = 1000
	}

	dashboards := make([]Dashboard, 0, pageSize)

	for page := 0; ; page++ {
		if err := validatePagination(ctx, page, c.paginationLimit()); err != nil {
			return nil, err
		}

		values := url.Values{}
		values.Set("page", strconv.Itoa(page))
		values.Set("page_size", strconv.Itoa(pageSize))

		var response dashboardListResponse

		if err := c.Get(ctx, fmt.Sprintf("/api/v1/dashboard/?%s", values.Encode()), &response); err != nil {
			return nil, err
		}

		dashboards = append(dashboards, response.Result...)

		if len(response.Result) == 0 || len(response.Result) < pageSize {
			return dashboards, nil
		}

		if response.Count > 0 && len(dashboards) >= response.Count {
			return dashboards, nil
		}
	}
}

func (c *Client) GetDashboardCharts(ctx context.Context, idOrSlug string) ([]DashboardChart, error) {
	var response dashboardChartsResponse

	requestPath, err := dashboardPath(idOrSlug)
	if err != nil {
		return nil, err
	}

	if err := c.Get(ctx, fmt.Sprintf("%s/charts", requestPath), &response); err != nil {
		return nil, err
	}

	return response.Result, nil
}

func (r DashboardUpdateRequest) MarshalJSON() ([]byte, error) {
	body := map[string]any{
		"dashboard_title": r.DashboardTitle,
	}

	if r.IncludeSlug {
		body["slug"] = nullableStringValue(r.Slug)
	}

	if r.IncludeCSS {
		body["css"] = nullableStringValue(r.CSS)
	}

	if r.IncludePublished {
		body["published"] = nullableBoolValue(r.Published)
	}

	if r.IncludePositionJSON {
		body["position_json"] = nullableStringValue(r.PositionJSON)
	}

	if r.IncludeJSONMetadata {
		body["json_metadata"] = nullableStringValue(r.JSONMetadata)
	}

	return json.Marshal(body)
}

func dashboardFromResponse(response dashboardResponse) *Dashboard {
	dashboard := response.Result

	if dashboard.ID == 0 {
		dashboard.ID = response.ID
	}

	return &dashboard
}

func dashboardPath(idOrSlug string) (string, error) {
	normalized := strings.TrimSpace(idOrSlug)
	if normalized == "" {
		return "", fmt.Errorf("dashboard identifier must not be empty")
	}

	return fmt.Sprintf("/api/v1/dashboard/%s", url.PathEscape(normalized)), nil
}
