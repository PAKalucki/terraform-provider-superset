// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ExampleDataSource{}

func NewExampleDataSource() datasource.DataSource {
	return &ExampleDataSource{}
}

// ExampleDataSource defines the data source implementation.
type ExampleDataSource struct {
	client *supersetclient.Client
}

// ExampleDataSourceModel describes the data source data model.
type ExampleDataSourceModel struct {
	ConfigurableAttribute types.String `tfsdk:"configurable_attribute"`
	Id                    types.String `tfsdk:"id"`
}

type availableDatabasesResponse struct {
	Databases []availableDatabaseEngine `json:"databases"`
}

type availableDatabaseEngine struct {
	Engine string `json:"engine"`
}

func (d *ExampleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_example"
}

func (d *ExampleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example data source",

		Attributes: map[string]schema.Attribute{
			"configurable_attribute": schema.StringAttribute{
				MarkdownDescription: "Example configurable attribute",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Example identifier",
				Computed:            true,
			},
		},
	}
}

func (d *ExampleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (d *ExampleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ExampleDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the example data source.",
		)

		return
	}

	var available availableDatabasesResponse

	if err := d.client.Get(ctx, "/api/v1/database/available/", &available); err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to read available Superset databases, got error: %s", err),
		)

		return
	}

	for _, database := range available.Databases {
		if database.Engine == "postgresql" {
			data.Id = types.StringValue(database.Engine)

			// Write logs using the tflog package
			// Documentation: https://terraform.io/plugin/log
			tflog.Trace(ctx, "read a data source")

			// Save data into Terraform state
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

			return
		}
	}

	resp.Diagnostics.AddError(
		"Unexpected Superset API Response",
		"The `/api/v1/database/available/` response did not include the PostgreSQL engine.",
	)
}
