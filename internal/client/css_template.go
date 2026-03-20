package client

import (
	"context"
	"fmt"
)

type CSSTemplate struct {
	ID           int64  `json:"id,omitempty"`
	TemplateName string `json:"template_name,omitempty"`
	CSS          string `json:"css,omitempty"`
}

type CSSTemplateCreateRequest struct {
	TemplateName string `json:"template_name"`
	CSS          string `json:"css"`
}

type CSSTemplateUpdateRequest = CSSTemplateCreateRequest

type cssTemplateResponse struct {
	ID     int64       `json:"id"`
	Result CSSTemplate `json:"result"`
}

type cssTemplateCreateResponse struct {
	ID     int64       `json:"id"`
	Result CSSTemplate `json:"result"`
}

func (c *Client) CreateCSSTemplate(ctx context.Context, request CSSTemplateCreateRequest) (*CSSTemplate, error) {
	var response cssTemplateCreateResponse

	if err := c.Post(ctx, "/api/v1/css_template/", request, &response); err != nil {
		return nil, err
	}

	cssTemplate := response.Result
	if cssTemplate.ID == 0 {
		cssTemplate.ID = response.ID
	}

	return &cssTemplate, nil
}

func (c *Client) GetCSSTemplate(ctx context.Context, id int64) (*CSSTemplate, error) {
	var response cssTemplateResponse

	if err := c.Get(ctx, cssTemplatePath(id), &response); err != nil {
		return nil, err
	}

	cssTemplate := response.Result
	if cssTemplate.ID == 0 {
		cssTemplate.ID = response.ID
	}

	return &cssTemplate, nil
}

func (c *Client) UpdateCSSTemplate(ctx context.Context, id int64, request CSSTemplateUpdateRequest) error {
	var response map[string]any

	return c.Put(ctx, cssTemplatePath(id), request, &response)
}

func (c *Client) DeleteCSSTemplate(ctx context.Context, id int64) error {
	var response map[string]any

	return c.Delete(ctx, cssTemplatePath(id), &response)
}

func cssTemplatePath(id int64) string {
	return fmt.Sprintf("/api/v1/css_template/%d", id)
}
