package provider

import (
	"context"
	"fmt"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var _ resource.Resource = &ChartResource{}

func NewChartResource() resource.Resource {
	return &ChartResource{}
}

type ChartResource struct {
	client *supersetclient.Client
}

func (r *ChartResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_chart"
}

func (r *ChartResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Superset chart.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Superset chart identifier.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Superset chart UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slice_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable chart name in Superset.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional chart description.",
			},
			"datasource_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Superset datasource identifier for the chart.",
			},
			"datasource_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("table"),
				MarkdownDescription: "Superset datasource type, for example `table`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"datasource_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved Superset datasource name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"viz_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Superset visualization type, for example `table` or `bar`.",
			},
			"params": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Chart form data JSON string. Prefer `jsonencode(...)` so Terraform and Superset use the same normalized JSON representation.",
			},
			"query_context": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional chart query context JSON string. When omitted, Superset stores no explicit query context until one is generated.",
			},
			"cache_timeout": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Optional chart cache timeout in seconds.",
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved Superset chart URL.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ChartResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ChartResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data chartModel
	var chartID int64
	persistedState := false

	defer func() {
		if chartID == 0 || persistedState {
			return
		}

		if err := r.client.DeleteChart(ctx, chartID); err != nil && !isSupersetNotFoundError(err) {
			resp.Diagnostics.AddWarning(
				"Unable to Roll Back Superset Chart After Create Failure",
				fmt.Sprintf("The provider created Superset chart %d but could not delete it after the Terraform create operation failed: %v", chartID, err),
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
			"The provider client was not configured for the chart resource.",
		)

		return
	}

	createRequest, diags := expandChartCreateRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateChartDatasourceExists(ctx, r.client, createRequest.DatasourceID, createRequest.DatasourceType)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateChart(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Superset Chart",
			err.Error(),
		)

		return
	}

	chartID = created.ID

	chart, err := r.client.GetChart(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Chart After Create",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenChartModel(data, chart)
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

func (r *ChartResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data chartModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the chart resource.",
		)

		return
	}

	chart, err := r.client.GetChart(ctx, data.ID.ValueInt64())
	if err != nil {
		if isSupersetNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Unable to Read Superset Chart",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenChartModel(data, chart)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChartResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data chartModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the chart resource.",
		)

		return
	}

	var current chartModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	chart, err := r.client.GetChart(ctx, current.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Chart Before Update",
			err.Error(),
		)

		return
	}

	updateRequest, diags := expandChartUpdateRequest(data, chart)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateChartDatasourceExists(ctx, r.client, updateRequest.DatasourceID, updateRequest.DatasourceType)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.UpdateChart(ctx, current.ID.ValueInt64(), updateRequest); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Superset Chart",
			err.Error(),
		)

		return
	}

	chart, err = r.client.GetChart(ctx, current.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Chart After Update",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenChartModel(data, chart)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChartResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data chartModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the chart resource.",
		)

		return
	}

	if err := r.client.DeleteChart(ctx, data.ID.ValueInt64()); err != nil && !isSupersetNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Unable to Delete Superset Chart",
			err.Error(),
		)
	}
}

func validateChartDatasourceExists(ctx context.Context, client *supersetclient.Client, datasourceID int64, datasourceType string) diag.Diagnostics {
	var diags diag.Diagnostics

	if datasourceType != "table" {
		return diags
	}

	_, err := client.GetDataset(ctx, datasourceID)
	if err == nil {
		return diags
	}

	if isSupersetNotFoundError(err) {
		diags.AddAttributeError(
			path.Root("datasource_id"),
			"Superset Datasource Not Found",
			fmt.Sprintf("Superset dataset %d was not found for datasource type %q.", datasourceID, datasourceType),
		)

		return diags
	}

	diags.AddError(
		"Unable to Validate Superset Datasource",
		err.Error(),
	)

	return diags
}
