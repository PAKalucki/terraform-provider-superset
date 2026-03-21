package provider

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func nonEmptyTrimmedStringValidator() validator.String {
	return nonEmptyTrimmedString{}
}

type nonEmptyTrimmedString struct{}

func (v nonEmptyTrimmedString) Description(context.Context) string {
	return "value must not be empty or whitespace"
}

func (v nonEmptyTrimmedString) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v nonEmptyTrimmedString) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if strings.TrimSpace(req.ConfigValue.ValueString()) != "" {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid String Value",
		"Value must not be empty or whitespace.",
	)
}

func positiveInt64Validator() validator.Int64 {
	return positiveInt64{}
}

type positiveInt64 struct{}

func (v positiveInt64) Description(context.Context) string {
	return "value must be a positive integer"
}

func (v positiveInt64) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v positiveInt64) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if req.ConfigValue.ValueInt64() > 0 {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Integer Value",
		"Value must be a positive integer.",
	)
}

func emailAddressValidator() validator.String {
	return emailAddress{}
}

type emailAddress struct{}

func (v emailAddress) Description(context.Context) string {
	return "value must be a valid email address"
}

func (v emailAddress) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v emailAddress) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := strings.TrimSpace(req.ConfigValue.ValueString())
	if value == "" {
		return
	}

	address, err := mail.ParseAddress(value)
	if err == nil && address.Address == value {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Email Address",
		fmt.Sprintf("Value %q must be a valid email address.", req.ConfigValue.ValueString()),
	)
}
