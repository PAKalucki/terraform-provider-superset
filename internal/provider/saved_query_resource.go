package provider

import (
	"context"
	"fmt"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var _ resource.Resource = &SavedQueryResource{}

func NewSavedQueryResource() resource.Resource {
	return &SavedQueryResource{}
}

type SavedQueryResource struct {
	client *supersetclient.Client
}

func (r *SavedQueryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saved_query"
}

func (r *SavedQueryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Superset saved query.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Superset saved query identifier.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"database_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Superset database identifier used by the saved query.",
			},
			"database_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved Superset database name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"label": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Saved query label shown in SQL Lab.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional saved query description.",
			},
			"catalog": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional catalog for the saved query.",
			},
			"schema": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional database schema for the saved query.",
			},
			"sql": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "SQL text stored in the saved query.",
			},
			"template_parameters": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Optional JSON string for template parameters. When omitted, the provider leaves the current value unchanged.",
			},
			"extra_json": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Optional JSON string for saved-query metadata. When omitted, the provider leaves the current value unchanged.",
			},
		},
	}
}

func (r *SavedQueryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SavedQueryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data savedQueryModel
	var createdID int64
	persistedState := false

	defer func() {
		if createdID == 0 || persistedState {
			return
		}

		if err := r.client.DeleteSavedQuery(ctx, createdID); err != nil && !isSupersetNotFoundError(err) {
			resp.Diagnostics.AddWarning(
				"Unable to Roll Back Superset Saved Query After Create Failure",
				fmt.Sprintf("The provider created Superset saved query %d but could not delete it after the Terraform create operation failed: %v", createdID, err),
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
			"The provider client was not configured for the saved query resource.",
		)

		return
	}

	request, diags := expandSavedQueryCreateRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	savedQuery, err := r.client.CreateSavedQuery(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Superset Saved Query",
			err.Error(),
		)

		return
	}

	createdID = savedQuery.ID

	savedQuery, err = r.client.GetSavedQuery(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Saved Query After Create",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenSavedQueryModel(data, savedQuery)
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

func (r *SavedQueryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data savedQueryModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the saved query resource.",
		)

		return
	}

	savedQuery, err := r.client.GetSavedQuery(ctx, data.ID.ValueInt64())
	if err != nil {
		if isSupersetNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Unable to Read Superset Saved Query",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenSavedQueryModel(data, savedQuery)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SavedQueryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data savedQueryModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the saved query resource.",
		)

		return
	}

	var current savedQueryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	savedQuery, err := r.client.GetSavedQuery(ctx, current.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Saved Query Before Update",
			err.Error(),
		)

		return
	}

	request, diags := expandSavedQueryUpdateRequest(data, savedQuery)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateSavedQuery(ctx, current.ID.ValueInt64(), request); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Superset Saved Query",
			err.Error(),
		)

		return
	}

	savedQuery, err = r.client.GetSavedQuery(ctx, current.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Saved Query After Update",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenSavedQueryModel(data, savedQuery)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SavedQueryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data savedQueryModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the saved query resource.",
		)

		return
	}

	if err := r.client.DeleteSavedQuery(ctx, data.ID.ValueInt64()); err != nil && !isSupersetNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Unable to Delete Superset Saved Query",
			err.Error(),
		)
	}
}
