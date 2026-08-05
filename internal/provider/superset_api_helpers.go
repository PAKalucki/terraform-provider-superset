// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	supersetAPIMethodGet  = "GET"
	supersetAPIMethodPost = "POST"
	supersetAPIMethodPut  = "PUT"
)

func supersetAPIMethodValidator() validator.String {
	return supersetAPIMethod{}
}

type supersetAPIMethod struct{}

func (v supersetAPIMethod) Description(context.Context) string {
	return "value must be POST or PUT"
}

func (v supersetAPIMethod) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v supersetAPIMethod) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	method := strings.ToUpper(strings.TrimSpace(req.ConfigValue.ValueString()))
	if method == supersetAPIMethodPost || method == supersetAPIMethodPut {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Superset API Method",
		"Method must be `POST` or `PUT`.",
	)
}

func supersetAPIPathValidator() validator.String {
	return supersetAPIPath{}
}

type supersetAPIPath struct{}

func (v supersetAPIPath) Description(context.Context) string {
	return "value must be a non-empty path relative to the configured Superset endpoint"
}

func (v supersetAPIPath) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v supersetAPIPath) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := strings.TrimSpace(req.ConfigValue.ValueString())
	parsed, err := url.Parse(value)
	if err == nil && strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") && parsed.Scheme == "" && parsed.Host == "" && parsed.User == nil && parsed.Fragment == "" {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Superset API Path",
		"Path must begin with `/`, must be relative to the configured Superset endpoint, and must not contain a URL fragment.",
	)
}

func jsonStringValidator() validator.String {
	return jsonString{}
}

type jsonString struct{}

func (v jsonString) Description(context.Context) string {
	return "value must contain valid JSON"
}

func (v jsonString) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonString) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if json.Valid([]byte(req.ConfigValue.ValueString())) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid JSON",
		"Value must contain valid JSON. Use `jsonencode` to build request bodies safely.",
	)
}

func executeSupersetAPIRequest(ctx context.Context, client *supersetclient.Client, method string, requestPath string, requestBody types.String) (types.String, error) {
	var body any
	if !requestBody.IsNull() {
		if requestBody.IsUnknown() {
			return types.StringNull(), fmt.Errorf("request body is unknown")
		}

		body = json.RawMessage(requestBody.ValueString())
	}

	var response json.RawMessage
	var err error

	switch strings.ToUpper(strings.TrimSpace(method)) {
	case supersetAPIMethodGet:
		err = client.Get(ctx, strings.TrimSpace(requestPath), &response)
	case supersetAPIMethodPost:
		err = client.Post(ctx, strings.TrimSpace(requestPath), body, &response)
	case supersetAPIMethodPut:
		err = client.Put(ctx, strings.TrimSpace(requestPath), body, &response)
	default:
		return types.StringNull(), fmt.Errorf("unsupported Superset API method %q", method)
	}
	if err != nil {
		return types.StringNull(), err
	}

	if len(response) == 0 {
		return types.StringNull(), nil
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, response); err != nil {
		return types.StringNull(), fmt.Errorf("compact Superset API response: %w", err)
	}

	return types.StringValue(compact.String()), nil
}

func supersetAPIOperationID(method string, requestPath string) types.String {
	return types.StringValue(fmt.Sprintf("%s %s", strings.ToUpper(strings.TrimSpace(method)), strings.TrimSpace(requestPath)))
}
