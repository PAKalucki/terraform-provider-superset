package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

type Dataset struct {
	ID                   int64           `json:"id,omitempty"`
	UUID                 string          `json:"uuid,omitempty"`
	Database             DatasetDatabase `json:"database"`
	TableName            string          `json:"table_name,omitempty"`
	Schema               string          `json:"schema,omitempty"`
	Description          string          `json:"description,omitempty"`
	MainDttmCol          string          `json:"main_dttm_col,omitempty"`
	FilterSelectEnabled  *bool           `json:"filter_select_enabled,omitempty"`
	NormalizeColumns     *bool           `json:"normalize_columns,omitempty"`
	AlwaysFilterMainDttm *bool           `json:"always_filter_main_dttm,omitempty"`
	CacheTimeout         *int64          `json:"cache_timeout,omitempty"`
	Columns              []DatasetColumn `json:"columns,omitempty"`
	Metrics              []DatasetMetric `json:"metrics,omitempty"`
}

type DatasetDatabase struct {
	ID           int64  `json:"id,omitempty"`
	DatabaseName string `json:"database_name,omitempty"`
	Backend      string `json:"backend,omitempty"`
	UUID         string `json:"uuid,omitempty"`
}

type DatasetColumn struct {
	ID               int64  `json:"id,omitempty"`
	ColumnName       string `json:"column_name,omitempty"`
	VerboseName      string `json:"verbose_name,omitempty"`
	Description      string `json:"description,omitempty"`
	Expression       string `json:"expression,omitempty"`
	Filterable       *bool  `json:"filterable,omitempty"`
	Groupby          *bool  `json:"groupby,omitempty"`
	IsActive         *bool  `json:"is_active,omitempty"`
	IsDttm           *bool  `json:"is_dttm,omitempty"`
	Type             string `json:"type,omitempty"`
	PythonDateFormat string `json:"python_date_format,omitempty"`
}

type DatasetMetric struct {
	ID          int64  `json:"id,omitempty"`
	MetricName  string `json:"metric_name,omitempty"`
	Expression  string `json:"expression,omitempty"`
	MetricType  string `json:"metric_type,omitempty"`
	VerboseName string `json:"verbose_name,omitempty"`
	Description string `json:"description,omitempty"`
	D3Format    string `json:"d3format,omitempty"`
	WarningText string `json:"warning_text,omitempty"`
}

type DatasetCreateRequest struct {
	Database  int64   `json:"database"`
	TableName string  `json:"table_name"`
	Schema    *string `json:"schema,omitempty"`
}

type DatasetUpdateRequest struct {
	DatabaseID           int64            `json:"database_id"`
	TableName            string           `json:"table_name"`
	Schema               *string          `json:"schema,omitempty"`
	Description          *string          `json:"description,omitempty"`
	MainDttmCol          *string          `json:"main_dttm_col,omitempty"`
	FilterSelectEnabled  *bool            `json:"filter_select_enabled,omitempty"`
	NormalizeColumns     *bool            `json:"normalize_columns,omitempty"`
	AlwaysFilterMainDttm *bool            `json:"always_filter_main_dttm,omitempty"`
	CacheTimeout         *int64           `json:"cache_timeout,omitempty"`
	Columns              *[]DatasetColumn `json:"columns,omitempty"`
	Metrics              *[]DatasetMetric `json:"metrics,omitempty"`
}

type datasetResponse struct {
	ID     int64   `json:"id"`
	Result Dataset `json:"result"`
}

type datasetCreateResponse struct {
	ID int64 `json:"id"`
}

type datasetListResponse struct {
	Result []Dataset `json:"result"`
}

func (c *Client) CreateDataset(ctx context.Context, request DatasetCreateRequest) (int64, error) {
	var response datasetCreateResponse

	if err := c.Post(ctx, "/api/v1/dataset/", request, &response); err != nil {
		return 0, err
	}

	return response.ID, nil
}

func (c *Client) GetDataset(ctx context.Context, id int64) (*Dataset, error) {
	var response datasetResponse

	if err := c.Get(ctx, datasetPath(id), &response); err != nil {
		return nil, err
	}

	dataset := response.Result

	if dataset.ID == 0 {
		dataset.ID = response.ID
	}

	return &dataset, nil
}

func (c *Client) UpdateDataset(ctx context.Context, id int64, request DatasetUpdateRequest) error {
	var response datasetResponse

	return c.Put(ctx, datasetPath(id), request, &response)
}

func (c *Client) DeleteDataset(ctx context.Context, id int64) error {
	var response map[string]any

	return c.Delete(ctx, datasetPath(id), &response)
}

func (c *Client) ListDatasets(ctx context.Context, pageSize int) ([]Dataset, error) {
	if pageSize <= 0 {
		pageSize = 1000
	}

	values := url.Values{}
	values.Set("page_size", strconv.Itoa(pageSize))

	var response datasetListResponse

	if err := c.Get(ctx, fmt.Sprintf("/api/v1/dataset/?%s", values.Encode()), &response); err != nil {
		return nil, err
	}

	return response.Result, nil
}

func datasetPath(id int64) string {
	return fmt.Sprintf("/api/v1/dataset/%d", id)
}
