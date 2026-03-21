package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
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

	IncludeVerboseName      bool
	IncludeDescription      bool
	IncludeExpression       bool
	IncludeFilterable       bool
	IncludeGroupby          bool
	IncludeIsActive         bool
	IncludeIsDttm           bool
	IncludeType             bool
	IncludePythonDateFormat bool
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

	IncludeMetricType  bool
	IncludeVerboseName bool
	IncludeDescription bool
	IncludeD3Format    bool
	IncludeWarningText bool
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

	IncludeSchema               bool
	IncludeDescription          bool
	IncludeMainDttmCol          bool
	IncludeFilterSelectEnabled  bool
	IncludeNormalizeColumns     bool
	IncludeAlwaysFilterMainDttm bool
	IncludeCacheTimeout         bool
}

type datasetResponse struct {
	ID     int64   `json:"id"`
	Result Dataset `json:"result"`
}

type datasetCreateResponse struct {
	ID int64 `json:"id"`
}

type datasetListResponse struct {
	Count  int       `json:"count"`
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

	datasets := make([]Dataset, 0, pageSize)

	for page := 0; ; page++ {
		if err := validatePagination(ctx, page, c.paginationLimit()); err != nil {
			return nil, err
		}

		values := url.Values{}
		values.Set("page", strconv.Itoa(page))
		values.Set("page_size", strconv.Itoa(pageSize))

		var response datasetListResponse

		if err := c.Get(ctx, fmt.Sprintf("/api/v1/dataset/?%s", values.Encode()), &response); err != nil {
			return nil, err
		}

		datasets = append(datasets, response.Result...)

		if len(response.Result) == 0 || len(response.Result) < pageSize {
			return datasets, nil
		}

		if response.Count > 0 && len(datasets) >= response.Count {
			return datasets, nil
		}
	}
}

func datasetPath(id int64) string {
	return fmt.Sprintf("/api/v1/dataset/%d", id)
}

func (r DatasetUpdateRequest) MarshalJSON() ([]byte, error) {
	body := map[string]any{
		"database_id": r.DatabaseID,
		"table_name":  r.TableName,
	}

	if r.IncludeSchema {
		body["schema"] = nullableStringValue(r.Schema)
	}

	if r.IncludeDescription {
		body["description"] = nullableStringValue(r.Description)
	}

	if r.IncludeMainDttmCol {
		body["main_dttm_col"] = nullableStringValue(r.MainDttmCol)
	}

	if r.IncludeFilterSelectEnabled {
		body["filter_select_enabled"] = nullableBoolValue(r.FilterSelectEnabled)
	}

	if r.IncludeNormalizeColumns {
		body["normalize_columns"] = nullableBoolValue(r.NormalizeColumns)
	}

	if r.IncludeAlwaysFilterMainDttm {
		body["always_filter_main_dttm"] = nullableBoolValue(r.AlwaysFilterMainDttm)
	}

	if r.IncludeCacheTimeout {
		body["cache_timeout"] = nullableInt64Value(r.CacheTimeout)
	}

	if r.Columns != nil {
		body["columns"] = *r.Columns
	}

	if r.Metrics != nil {
		body["metrics"] = *r.Metrics
	}

	return json.Marshal(body)
}

func (c DatasetColumn) MarshalJSON() ([]byte, error) {
	body := map[string]any{
		"column_name": c.ColumnName,
	}

	if c.ID > 0 {
		body["id"] = c.ID
	}

	if c.IncludeVerboseName {
		body["verbose_name"] = emptyStringToNil(c.VerboseName)
	}

	if c.IncludeDescription {
		body["description"] = emptyStringToNil(c.Description)
	}

	if c.IncludeExpression {
		body["expression"] = emptyStringToNil(c.Expression)
	}

	if c.IncludeFilterable {
		body["filterable"] = nullableBoolValue(c.Filterable)
	}

	if c.IncludeGroupby {
		body["groupby"] = nullableBoolValue(c.Groupby)
	}

	if c.IncludeIsActive {
		body["is_active"] = nullableBoolValue(c.IsActive)
	}

	if c.IncludeIsDttm {
		body["is_dttm"] = nullableBoolValue(c.IsDttm)
	}

	if c.IncludeType {
		body["type"] = emptyStringToNil(c.Type)
	}

	if c.IncludePythonDateFormat {
		body["python_date_format"] = emptyStringToNil(c.PythonDateFormat)
	}

	return json.Marshal(body)
}

func (m DatasetMetric) MarshalJSON() ([]byte, error) {
	body := map[string]any{
		"metric_name": m.MetricName,
		"expression":  m.Expression,
	}

	if m.ID > 0 {
		body["id"] = m.ID
	}

	if m.IncludeMetricType {
		body["metric_type"] = emptyStringToNil(m.MetricType)
	}

	if m.IncludeVerboseName {
		body["verbose_name"] = emptyStringToNil(m.VerboseName)
	}

	if m.IncludeDescription {
		body["description"] = emptyStringToNil(m.Description)
	}

	if m.IncludeD3Format {
		body["d3format"] = emptyStringToNil(m.D3Format)
	}

	if m.IncludeWarningText {
		body["warning_text"] = emptyStringToNil(m.WarningText)
	}

	return json.Marshal(body)
}

func nullableStringValue(value *string) any {
	if value == nil {
		return nil
	}

	return emptyStringToNil(*value)
}

func nullableBoolValue(value *bool) any {
	if value == nil {
		return nil
	}

	return *value
}

func nullableInt64Value(value *int64) any {
	if value == nil {
		return nil
	}

	return *value
}

func emptyStringToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return value
}
