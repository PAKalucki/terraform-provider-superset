package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DashboardDataSource{}

func NewDashboardDataSource() datasource.DataSource {
	return &DashboardDataSource{}
}

type DashboardDataSource struct {
	client *supersetclient.Client
}

func (d *DashboardDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboard"
}

func (d *DashboardDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing Superset dashboard.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Superset dashboard identifier used for lookup or returned from Superset.",
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Superset dashboard UUID.",
			},
			"dashboard_title": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Human-readable dashboard title in Superset.",
			},
			"slug": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Dashboard slug used in the Superset URL.",
			},
			"css": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Custom dashboard CSS.",
			},
			"published": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the dashboard is published in Superset.",
			},
			"show_native_filters": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether native dashboard filters are shown in Superset.",
			},
			"chart_ids": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Superset chart identifiers associated with the dashboard.",
			},
			"position_json": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Superset dashboard layout JSON string.",
			},
			"native_filter_configuration": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Superset native filter configuration JSON array.",
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved Superset dashboard URL.",
			},
		},
	}
}

func (d *DashboardDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DashboardDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dashboardModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the dashboard data source.",
		)

		return
	}

	hasID := !data.ID.IsNull() && !data.ID.IsUnknown()
	hasSlug := !data.Slug.IsNull() && !data.Slug.IsUnknown() && strings.TrimSpace(data.Slug.ValueString()) != ""
	hasTitle := !data.DashboardTitle.IsNull() && !data.DashboardTitle.IsUnknown() && strings.TrimSpace(data.DashboardTitle.ValueString()) != ""

	lookupCount := 0
	if hasID {
		lookupCount++
	}
	if hasSlug {
		lookupCount++
	}
	if hasTitle {
		lookupCount++
	}

	switch {
	case lookupCount > 1:
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Conflicting Dashboard Lookup Arguments",
			"Configure only one of `id`, `slug`, or `dashboard_title`.",
		)

		return
	case lookupCount == 0:
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Missing Dashboard Lookup Arguments",
			"Configure exactly one of `id`, `slug`, or `dashboard_title`.",
		)

		return
	}

	var (
		dashboard       *supersetclient.Dashboard
		dashboardCharts []supersetclient.DashboardChart
		err             error
		idOrSlug        string
	)

	switch {
	case hasID:
		idOrSlug = strconv.FormatInt(data.ID.ValueInt64(), 10)
		dashboard, dashboardCharts, err = loadDashboardWithCharts(ctx, d.client, idOrSlug)
	case hasSlug:
		idOrSlug = strings.TrimSpace(data.Slug.ValueString())
		dashboard, dashboardCharts, err = loadDashboardWithCharts(ctx, d.client, idOrSlug)
	default:
		dashboard, err = findDashboardByTitle(ctx, d.client, strings.TrimSpace(data.DashboardTitle.ValueString()))
		if err == nil {
			dashboardCharts, err = d.client.GetDashboardCharts(ctx, strconv.FormatInt(dashboard.ID, 10))
		}
	}

	if err != nil {
		switch {
		case hasID && isSupersetNotFoundError(err):
			resp.Diagnostics.AddAttributeError(
				path.Root("id"),
				"Superset Dashboard Not Found",
				err.Error(),
			)
		case hasSlug && isSupersetNotFoundError(err):
			resp.Diagnostics.AddAttributeError(
				path.Root("slug"),
				"Superset Dashboard Not Found",
				err.Error(),
			)
		default:
			resp.Diagnostics.AddError(
				"Unable to Read Superset Dashboard",
				err.Error(),
			)
		}

		return
	}

	state, diags := flattenDashboardDataSourceModel(ctx, dashboard, dashboardCharts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
