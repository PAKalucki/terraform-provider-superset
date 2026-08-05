// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SupersetAPIResource{}

func NewSupersetAPIResource() resource.Resource {
	return &SupersetAPIResource{}
}

type SupersetAPIResource struct {
	client *supersetclient.Client
}

type supersetAPIResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Method       types.String `tfsdk:"method"`
	Path         types.String `tfsdk:"path"`
	RequestBody  types.String `tfsdk:"request_body"`
	ResponseBody types.String `tfsdk:"response_body"`
}

func (r *SupersetAPIResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api"
}

func (r *SupersetAPIResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Executes an authenticated POST or PUT request against a Superset API path that does not yet have a dedicated resource. The operation runs during create and whenever an input changes. Refresh does not replay the operation, and destroy only removes it from Terraform state; it does not send a DELETE request.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Stable identifier formed from the HTTP method and API path.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"method": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "HTTP method used for the operation. Supported values are `POST` and `PUT` (case-insensitive).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					supersetAPIMethodValidator(),
				},
			},
			"path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Superset API path, including optional query parameters. The path must begin with `/` and is resolved against the configured provider endpoint.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					supersetAPIPathValidator(),
				},
			},
			"request_body": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Optional JSON request body. Use `jsonencode` to construct it. This value is stored in Terraform state and marked sensitive.",
				Validators: []validator.String{
					jsonStringValidator(),
				},
			},
			"response_body": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Compact JSON response body returned by the last successful operation. This value is stored in Terraform state and marked sensitive because a generic API response can contain secrets.",
			},
		},
	}
}

func (r *SupersetAPIResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*supersetclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *SupersetAPIResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data supersetAPIResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SupersetAPIResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Intentionally preserve the existing state. Replaying POST or PUT during
	// refresh would make a normal Terraform plan mutate Superset.
}

func (r *SupersetAPIResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data supersetAPIResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SupersetAPIResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// An arbitrary POST or PUT path does not provide enough information to
	// infer a safe DELETE operation. Terraform removes the state automatically.
}

func (r *SupersetAPIResource) apply(ctx context.Context, data *supersetAPIResourceModel, diagnostics *diag.Diagnostics) {
	if r.client == nil {
		diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the Superset API resource.",
		)

		return
	}

	responseBody, err := executeSupersetAPIRequest(ctx, r.client, data.Method.ValueString(), data.Path.ValueString(), data.RequestBody)
	if err != nil {
		diagnostics.AddError(
			"Unable to Execute Superset API Operation",
			err.Error(),
		)

		return
	}

	data.ID = supersetAPIOperationID(data.Method.ValueString(), data.Path.ValueString())
	data.ResponseBody = responseBody
}
