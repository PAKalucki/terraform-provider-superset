package provider

import (
	"context"
	"fmt"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var _ resource.Resource = &DatasetResource{}
var _ resource.ResourceWithImportState = &DatasetResource{}

func NewDatasetResource() resource.Resource {
	return &DatasetResource{}
}

type DatasetResource struct {
	client *supersetclient.Client
}

func (r *DatasetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (r *DatasetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Superset physical dataset.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Superset dataset identifier.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Superset dataset UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Superset database identifier that owns the dataset.",
			},
			"database_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved Superset database name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"table_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Dataset table name.",
			},
			"schema": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Dataset schema name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Dataset description.",
			},
			"main_dttm_col": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Main datetime column used by Superset.",
			},
			"filter_select_enabled": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether filter select is enabled for the dataset.",
			},
			"normalize_columns": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether Superset should normalize columns on create or update.",
			},
			"always_filter_main_dttm": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether the main datetime column is always filtered.",
			},
			"cache_timeout": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Dataset cache timeout in seconds.",
			},
			"columns": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Authoritative list of dataset columns when configured. Omit this attribute to leave Superset-managed columns unmanaged.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"column_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Dataset column name.",
						},
						"verbose_name": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Human-friendly column label.",
						},
						"description": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Column description.",
						},
						"expression": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Optional SQL expression for a virtual column.",
						},
						"filterable": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "Whether the column is filterable.",
						},
						"groupby": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "Whether the column can be grouped by.",
						},
						"is_active": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "Whether the column is active.",
						},
						"is_dttm": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "Whether the column is a datetime column.",
						},
						"type": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Superset column type.",
						},
						"python_date_format": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Optional Python date format.",
						},
					},
				},
			},
			"metrics": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Authoritative list of dataset metrics when configured. Omit this attribute to leave Superset-managed metrics unmanaged.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"metric_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Metric name.",
						},
						"expression": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Metric SQL expression.",
						},
						"metric_type": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Metric type, for example `count`.",
						},
						"verbose_name": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Human-friendly metric label.",
						},
						"description": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Metric description.",
						},
						"d3format": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Optional D3 format string.",
						},
						"warning_text": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Optional metric warning text.",
						},
					},
				},
			},
		},
	}
}

func (r *DatasetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DatasetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importInt64Attributes(ctx, req, resp, "id")
}

func (r *DatasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data datasetModel
	var datasetID int64
	persistedState := false

	defer func() {
		if datasetID == 0 || persistedState {
			return
		}

		if err := r.client.DeleteDataset(ctx, datasetID); err != nil && !isSupersetNotFoundError(err) {
			resp.Diagnostics.AddWarning(
				"Unable to Roll Back Superset Dataset After Create Failure",
				fmt.Sprintf("The provider created Superset dataset %d but could not delete it after the Terraform create operation failed: %v", datasetID, err),
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
			"The provider client was not configured for the dataset resource.",
		)

		return
	}

	createRequest, diags := expandDatasetCreateRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	datasetID, err := r.client.CreateDataset(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Superset Dataset",
			err.Error(),
		)

		return
	}

	dataset, err := r.client.GetDataset(ctx, datasetID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Dataset After Create",
			err.Error(),
		)

		return
	}

	if needsDatasetUpdate(data) {
		updateRequest, updateDiags := expandDatasetUpdateRequest(ctx, data, dataset)
		resp.Diagnostics.Append(updateDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if err := r.client.UpdateDataset(ctx, datasetID, updateRequest); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Configure Superset Dataset After Create",
				err.Error(),
			)

			return
		}

		dataset, err = r.client.GetDataset(ctx, datasetID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read Superset Dataset After Update",
				err.Error(),
			)

			return
		}
	}

	state, stateDiags := flattenDatasetResourceModel(ctx, data, dataset)
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

func (r *DatasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data datasetModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the dataset resource.",
		)

		return
	}

	dataset, err := r.client.GetDataset(ctx, data.ID.ValueInt64())
	if err != nil {
		if isSupersetNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Unable to Read Superset Dataset",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenDatasetResourceModel(ctx, data, dataset)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DatasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data datasetModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the dataset resource.",
		)

		return
	}

	var current datasetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dataset, err := r.client.GetDataset(ctx, current.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Dataset Before Update",
			err.Error(),
		)

		return
	}

	updateRequest, diags := expandDatasetUpdateRequest(ctx, data, dataset)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateDataset(ctx, current.ID.ValueInt64(), updateRequest); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Superset Dataset",
			err.Error(),
		)

		return
	}

	dataset, err = r.client.GetDataset(ctx, current.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Dataset After Update",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenDatasetResourceModel(ctx, data, dataset)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DatasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data datasetModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the dataset resource.",
		)

		return
	}

	if err := r.client.DeleteDataset(ctx, data.ID.ValueInt64()); err != nil && !isSupersetNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Unable to Delete Superset Dataset",
			err.Error(),
		)
	}
}
