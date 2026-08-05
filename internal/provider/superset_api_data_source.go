// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SupersetAPIDataSource{}

func NewSupersetAPIDataSource() datasource.DataSource {
	return &SupersetAPIDataSource{}
}

type SupersetAPIDataSource struct {
	client *supersetclient.Client
}

type supersetAPIDataSourceModel struct {
	Path         types.String `tfsdk:"path"`
	ResponseBody types.String `tfsdk:"response_body"`
}

func (d *SupersetAPIDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api"
}

func (d *SupersetAPIDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Executes an authenticated GET request against a Superset API path that does not yet have a dedicated data source.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Superset API path, including optional query parameters. The path must begin with `/` and is resolved against the configured provider endpoint.",
				Validators: []validator.String{
					supersetAPIPathValidator(),
				},
			},
			"response_body": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Compact JSON response body returned by Superset. This value is stored in Terraform state and marked sensitive because a generic API response can contain secrets.",
			},
		},
	}
}

func (d *SupersetAPIDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*supersetclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *SupersetAPIDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data supersetAPIDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the Superset API data source.",
		)

		return
	}

	responseBody, err := executeSupersetAPIRequest(ctx, d.client, supersetAPIMethodGet, data.Path.ValueString(), types.StringNull())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset API Path",
			err.Error(),
		)

		return
	}

	data.ResponseBody = responseBody
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
