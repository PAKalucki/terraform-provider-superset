package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type AnnotationLayer struct {
	ID          int64   `json:"id,omitempty"`
	Name        string  `json:"name,omitempty"`
	Description *string `json:"descr"`
}

type AnnotationLayerCreateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"descr,omitempty"`
}

type AnnotationLayerUpdateRequest struct {
	Name               string
	Description        *string
	IncludeDescription bool
}

func (r AnnotationLayerUpdateRequest) MarshalJSON() ([]byte, error) {
	body := map[string]any{
		"name": r.Name,
	}

	if r.IncludeDescription {
		if r.Description == nil {
			body["descr"] = ""
		} else {
			body["descr"] = r.Description
		}
	}

	return json.Marshal(body)
}

type annotationLayerResponse struct {
	ID     int64           `json:"id"`
	Result AnnotationLayer `json:"result"`
}

type annotationLayerCreateResponse struct {
	ID     int64           `json:"id"`
	Result AnnotationLayer `json:"result"`
}

func (c *Client) CreateAnnotationLayer(ctx context.Context, request AnnotationLayerCreateRequest) (*AnnotationLayer, error) {
	var response annotationLayerCreateResponse

	if err := c.Post(ctx, "/api/v1/annotation_layer/", request, &response); err != nil {
		return nil, err
	}

	layer := response.Result
	if layer.ID == 0 {
		layer.ID = response.ID
	}

	return &layer, nil
}

func (c *Client) GetAnnotationLayer(ctx context.Context, id int64) (*AnnotationLayer, error) {
	var response annotationLayerResponse

	if err := c.Get(ctx, annotationLayerPath(id), &response); err != nil {
		return nil, err
	}

	layer := response.Result
	if layer.ID == 0 {
		layer.ID = response.ID
	}

	return &layer, nil
}

func (c *Client) UpdateAnnotationLayer(ctx context.Context, id int64, request AnnotationLayerUpdateRequest) error {
	var response map[string]any

	return c.Put(ctx, annotationLayerPath(id), request, &response)
}

func (c *Client) DeleteAnnotationLayer(ctx context.Context, id int64) error {
	var response map[string]any

	return c.Delete(ctx, annotationLayerPath(id), &response)
}

func annotationLayerPath(id int64) string {
	return fmt.Sprintf("/api/v1/annotation_layer/%d", id)
}
