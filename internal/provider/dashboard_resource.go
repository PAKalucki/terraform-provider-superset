package provider

import (
	"context"
	"fmt"
	"strconv"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DashboardResource{}

func NewDashboardResource() resource.Resource {
	return &DashboardResource{}
}

type DashboardResource struct {
	client *supersetclient.Client
}

func (r *DashboardResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboard"
}

func (r *DashboardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Superset dashboard.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Superset dashboard identifier.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Superset dashboard UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dashboard_title": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable dashboard title in Superset.",
			},
			"slug": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional dashboard slug used in the Superset URL.",
			},
			"css": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional custom CSS for the dashboard.",
			},
			"published": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether the dashboard is published in Superset.",
			},
			"chart_ids": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Optional list of Superset chart identifiers associated with the dashboard. When configured without `position_json`, the provider generates a simple default layout for those charts.",
			},
			"position_json": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional Superset dashboard layout JSON string. When configured, the chart identifiers referenced in the layout become the authoritative dashboard-chart associations.",
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved Superset dashboard URL.",
			},
		},
	}
}

func (r *DashboardResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data dashboardModel
	var dashboardID int64
	persistedState := false

	defer func() {
		if dashboardID == 0 || persistedState {
			return
		}

		if err := r.client.DeleteDashboard(ctx, dashboardID); err != nil && !isSupersetNotFoundError(err) {
			resp.Diagnostics.AddWarning(
				"Unable to Roll Back Superset Dashboard After Create Failure",
				fmt.Sprintf("The provider created Superset dashboard %d but could not delete it after the Terraform create operation failed: %v", dashboardID, err),
			)
		}
	}()

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the dashboard resource.",
		)

		return
	}

	createRequest, diags := expandDashboardCreateRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateDashboard(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Superset Dashboard",
			err.Error(),
		)

		return
	}

	dashboardID = created.ID

	updateRequest, diags := expandDashboardUpdateRequest(ctx, r.client, data, dashboardModel{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if updateRequest.IncludePositionJSON || updateRequest.IncludeJSONMetadata {
		if err := r.client.UpdateDashboard(ctx, created.ID, updateRequest); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Configure Superset Dashboard After Create",
				err.Error(),
			)

			return
		}
	}

	dashboard, dashboardCharts, err := loadDashboardWithCharts(ctx, r.client, strconv.FormatInt(created.ID, 10))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Dashboard After Create",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenDashboardResourceModel(ctx, data, dashboard, dashboardCharts)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	persistedState = true
}

func (r *DashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dashboardModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the dashboard resource.",
		)

		return
	}

	dashboard, dashboardCharts, err := loadDashboardWithCharts(ctx, r.client, strconv.FormatInt(data.ID.ValueInt64(), 10))
	if err != nil {
		if isSupersetNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Unable to Read Superset Dashboard",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenDashboardResourceModel(ctx, data, dashboard, dashboardCharts)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data dashboardModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the dashboard resource.",
		)

		return
	}

	var current dashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest, diags := expandDashboardUpdateRequest(ctx, r.client, data, current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateDashboard(ctx, current.ID.ValueInt64(), updateRequest); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Superset Dashboard",
			err.Error(),
		)

		return
	}

	dashboard, dashboardCharts, err := loadDashboardWithCharts(ctx, r.client, strconv.FormatInt(current.ID.ValueInt64(), 10))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Dashboard After Update",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenDashboardResourceModel(ctx, data, dashboard, dashboardCharts)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data dashboardModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the dashboard resource.",
		)

		return
	}

	if err := r.client.DeleteDashboard(ctx, data.ID.ValueInt64()); err != nil && !isSupersetNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Unable to Delete Superset Dashboard",
			err.Error(),
		)
	}
}

func loadDashboardWithCharts(ctx context.Context, client *supersetclient.Client, idOrSlug string) (*supersetclient.Dashboard, []supersetclient.DashboardChart, error) {
	dashboard, err := client.GetDashboard(ctx, idOrSlug)
	if err != nil {
		return nil, nil, err
	}

	dashboardCharts, err := client.GetDashboardCharts(ctx, idOrSlug)
	if err != nil {
		return nil, nil, err
	}

	return dashboard, dashboardCharts, nil
}
