package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

type Database struct {
	ID              int64   `json:"id,omitempty"`
	UUID            string  `json:"uuid,omitempty"`
	DatabaseName    string  `json:"database_name,omitempty"`
	SQLAlchemyURI   string  `json:"sqlalchemy_uri,omitempty"`
	Extra           string  `json:"extra,omitempty"`
	ExposeInSQLLab  *bool   `json:"expose_in_sqllab,omitempty"`
	AllowCTAS       *bool   `json:"allow_ctas,omitempty"`
	AllowCVAS       *bool   `json:"allow_cvas,omitempty"`
	AllowDML        *bool   `json:"allow_dml,omitempty"`
	AllowFileUpload *bool   `json:"allow_file_upload,omitempty"`
	AllowRunAsync   *bool   `json:"allow_run_async,omitempty"`
	CacheTimeout    *int64  `json:"cache_timeout,omitempty"`
	ForceCTASSchema *string `json:"force_ctas_schema,omitempty"`
	ImpersonateUser *bool   `json:"impersonate_user,omitempty"`
	Backend         string  `json:"backend,omitempty"`
	Driver          string  `json:"driver,omitempty"`
}

type databaseResponse struct {
	ID     int64    `json:"id"`
	Result Database `json:"result"`
}

type databaseListResponse struct {
	Result []Database `json:"result"`
}

func (c *Client) CreateDatabase(ctx context.Context, database Database) (*Database, error) {
	var response databaseResponse

	if err := c.Post(ctx, "/api/v1/database/", database, &response); err != nil {
		return nil, err
	}

	return databaseFromResponse(response), nil
}

func (c *Client) GetDatabase(ctx context.Context, id int64) (*Database, error) {
	var response databaseResponse

	if err := c.Get(ctx, databasePath(id), &response); err != nil {
		return nil, err
	}

	return databaseFromResponse(response), nil
}

func (c *Client) GetDatabaseConnection(ctx context.Context, id int64) (*Database, error) {
	var response databaseResponse

	if err := c.Get(ctx, databaseConnectionPath(id), &response); err != nil {
		return nil, err
	}

	return databaseFromResponse(response), nil
}

func (c *Client) UpdateDatabase(ctx context.Context, id int64, database Database) (*Database, error) {
	var response databaseResponse

	if err := c.Put(ctx, databasePath(id), database, &response); err != nil {
		return nil, err
	}

	return databaseFromResponse(response), nil
}

func (c *Client) DeleteDatabase(ctx context.Context, id int64) error {
	var response map[string]any

	return c.Delete(ctx, databasePath(id), &response)
}

func (c *Client) ListDatabases(ctx context.Context, pageSize int) ([]Database, error) {
	if pageSize <= 0 {
		pageSize = 1000
	}

	values := url.Values{}
	values.Set("page_size", strconv.Itoa(pageSize))

	var response databaseListResponse

	if err := c.Get(ctx, fmt.Sprintf("/api/v1/database/?%s", values.Encode()), &response); err != nil {
		return nil, err
	}

	return response.Result, nil
}

func databaseFromResponse(response databaseResponse) *Database {
	database := response.Result

	if database.ID == 0 {
		database.ID = response.ID
	}

	return &database
}

func databaseConnectionPath(id int64) string {
	return fmt.Sprintf("%s/connection", databasePath(id))
}

func databasePath(id int64) string {
	return fmt.Sprintf("/api/v1/database/%d", id)
}
