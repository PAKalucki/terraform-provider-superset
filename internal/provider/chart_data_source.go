package provider

import (
	"context"
	"fmt"
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

var _ datasource.DataSource = &ChartDataSource{}

func NewChartDataSource() datasource.DataSource {
	return &ChartDataSource{}
}

type ChartDataSource struct {
	client *supersetclient.Client
}

func (d *ChartDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_chart"
}

func (d *ChartDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing Superset chart.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Superset chart identifier used for lookup or returned from Superset.",
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Superset chart UUID.",
			},
			"slice_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Human-readable chart name in Superset.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Chart description.",
			},
			"datasource_id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Superset datasource identifier for the chart.",
			},
			"datasource_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Superset datasource type, for example `table`.",
			},
			"datasource_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved Superset datasource name.",
			},
			"viz_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Superset visualization type.",
			},
			"params": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Chart form data JSON string returned by Superset.",
			},
			"query_context": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Chart query context JSON string returned by Superset.",
			},
			"cache_timeout": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Chart cache timeout in seconds.",
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved Superset chart URL.",
			},
		},
	}
}

func (d *ChartDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ChartDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data chartModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the chart data source.",
		)

		return
	}

	hasID := !data.ID.IsNull() && !data.ID.IsUnknown()
	hasDatasourceID := !data.DatasourceID.IsNull() && !data.DatasourceID.IsUnknown()
	hasSliceName := !data.SliceName.IsNull() && !data.SliceName.IsUnknown() && strings.TrimSpace(data.SliceName.ValueString()) != ""

	switch {
	case hasID && (hasDatasourceID || hasSliceName):
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Conflicting Chart Lookup Arguments",
			"Configure either `id` or `datasource_id` with `slice_name` and optional `datasource_type`.",
		)

		return
	case !hasID && (!hasDatasourceID || !hasSliceName):
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Missing Chart Lookup Arguments",
			"Configure either `id` or `datasource_id` with `slice_name` and optional `datasource_type`.",
		)

		return
	}

	var (
		chart *supersetclient.Chart
		err   error
	)

	if hasID {
		chart, err = d.client.GetChart(ctx, data.ID.ValueInt64())
	} else {
		chart, err = findChart(ctx, d.client, data.DatasourceID.ValueInt64(), data.DatasourceType.ValueString(), data.SliceName.ValueString())
	}

	if err != nil {
		if hasID && isSupersetNotFoundError(err) {
			resp.Diagnostics.AddAttributeError(
				path.Root("id"),
				"Superset Chart Not Found",
				err.Error(),
			)
		} else {
			resp.Diagnostics.AddError(
				"Unable to Read Superset Chart",
				err.Error(),
			)
		}

		return
	}

	state, diags := flattenChartModel(chartModel{}, chart)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
