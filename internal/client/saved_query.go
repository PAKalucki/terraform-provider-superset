package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type SavedQuery struct {
	ID                 int64              `json:"id,omitempty"`
	DatabaseID         int64              `json:"db_id,omitempty"`
	Database           SavedQueryDatabase `json:"database"`
	Label              string             `json:"label,omitempty"`
	Description        *string            `json:"description"`
	Catalog            *string            `json:"catalog"`
	Schema             *string            `json:"schema"`
	SQL                string             `json:"sql,omitempty"`
	TemplateParameters *string            `json:"template_parameters"`
	ExtraJSON          *string            `json:"extra_json"`
}

type SavedQueryDatabase struct {
	ID           int64  `json:"id,omitempty"`
	DatabaseName string `json:"database_name,omitempty"`
}

type SavedQueryCreateRequest struct {
	DatabaseID         int64   `json:"db_id"`
	Label              string  `json:"label"`
	Description        *string `json:"description,omitempty"`
	Catalog            *string `json:"catalog,omitempty"`
	Schema             *string `json:"schema,omitempty"`
	SQL                string  `json:"sql"`
	TemplateParameters *string `json:"template_parameters,omitempty"`
	ExtraJSON          *string `json:"extra_json,omitempty"`
}

type SavedQueryUpdateRequest struct {
	DatabaseID                int64
	Label                     string
	Description               *string
	Catalog                   *string
	Schema                    *string
	SQL                       string
	TemplateParameters        *string
	ExtraJSON                 *string
	IncludeDescription        bool
	IncludeCatalog            bool
	IncludeSchema             bool
	IncludeTemplateParameters bool
	IncludeExtraJSON          bool
}

func (r SavedQueryUpdateRequest) MarshalJSON() ([]byte, error) {
	body := map[string]any{
		"db_id": r.DatabaseID,
		"label": r.Label,
		"sql":   r.SQL,
	}

	if r.IncludeDescription {
		body["description"] = r.Description
	}

	if r.IncludeCatalog {
		body["catalog"] = r.Catalog
	}

	if r.IncludeSchema {
		body["schema"] = r.Schema
	}

	if r.IncludeTemplateParameters {
		body["template_parameters"] = r.TemplateParameters
	}

	if r.IncludeExtraJSON {
		body["extra_json"] = r.ExtraJSON
	}

	return json.Marshal(body)
}

type savedQueryResponse struct {
	ID     int64      `json:"id"`
	Result SavedQuery `json:"result"`
}

type savedQueryCreateResponse struct {
	ID     int64      `json:"id"`
	Result SavedQuery `json:"result"`
}

func (c *Client) CreateSavedQuery(ctx context.Context, request SavedQueryCreateRequest) (*SavedQuery, error) {
	var response savedQueryCreateResponse

	if err := c.Post(ctx, "/api/v1/saved_query/", request, &response); err != nil {
		return nil, err
	}

	savedQuery := response.Result
	if savedQuery.ID == 0 {
		savedQuery.ID = response.ID
	}

	return &savedQuery, nil
}

func (c *Client) GetSavedQuery(ctx context.Context, id int64) (*SavedQuery, error) {
	var response savedQueryResponse

	if err := c.Get(ctx, savedQueryPath(id), &response); err != nil {
		return nil, err
	}

	savedQuery := response.Result
	if savedQuery.ID == 0 {
		savedQuery.ID = response.ID
	}

	return &savedQuery, nil
}

func (c *Client) UpdateSavedQuery(ctx context.Context, id int64, request SavedQueryUpdateRequest) error {
	var response map[string]any

	return c.Put(ctx, savedQueryPath(id), request, &response)
}

func (c *Client) DeleteSavedQuery(ctx context.Context, id int64) error {
	var response map[string]any

	return c.Delete(ctx, savedQueryPath(id), &response)
}

func savedQueryPath(id int64) string {
	return fmt.Sprintf("/api/v1/saved_query/%d", id)
}
