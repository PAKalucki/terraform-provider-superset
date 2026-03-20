package provider

import (
	"context"
	"fmt"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var _ resource.Resource = &DatabaseResource{}

func NewDatabaseResource() resource.Resource {
	return &DatabaseResource{}
}

type DatabaseResource struct {
	client *supersetclient.Client
}

func (r *DatabaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (r *DatabaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Superset database connection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Superset database identifier.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Superset database UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable name for the Superset database connection.",
			},
			"sqlalchemy_uri": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "SQLAlchemy connection URI used by Superset to reach the database.",
			},
			"extra": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Optional database `extra` JSON string. Prefer `jsonencode(...)` so Terraform and Superset use the same normalized JSON representation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expose_in_sqllab": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether to expose the database in SQL Lab.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_ctas": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether `CREATE TABLE AS` statements are allowed in SQL Lab.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_cvas": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether `CREATE VIEW AS` statements are allowed in SQL Lab.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_dml": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether non-SELECT statements are allowed in SQL Lab.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_file_upload": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether CSV uploads are allowed for this database.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_run_async": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether queries on this database run asynchronously.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"cache_timeout": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Database-level chart cache timeout in seconds.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"force_ctas_schema": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Schema enforced for `CREATE TABLE AS` statements when enabled.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"impersonate_user": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether Superset impersonates the current user when querying this database.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"backend": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved Superset database backend.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"driver": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved SQLAlchemy driver.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *DatabaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data databaseModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the database resource.",
		)

		return
	}

	request, diags := expandDatabaseRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data = applyDatabaseRequestToModel(data, request)

	created, err := r.client.CreateDatabase(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Superset Database",
			err.Error(),
		)

		return
	}

	database, err := loadDatabase(ctx, r.client, created.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Database After Create",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenDatabaseModel(data, database)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data databaseModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the database resource.",
		)

		return
	}

	database, err := loadDatabase(ctx, r.client, data.ID.ValueInt64())
	if err != nil {
		if isSupersetNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Unable to Read Superset Database",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenDatabaseModel(data, database)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data databaseModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the database resource.",
		)

		return
	}

	request, diags := expandDatabaseRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data = applyDatabaseRequestToModel(data, request)

	var current databaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.UpdateDatabase(ctx, current.ID.ValueInt64(), request); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Superset Database",
			err.Error(),
		)

		return
	}

	database, err := loadDatabase(ctx, r.client, current.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Database After Update",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenDatabaseModel(data, database)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data databaseModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the database resource.",
		)

		return
	}

	if err := r.client.DeleteDatabase(ctx, data.ID.ValueInt64()); err != nil && !isSupersetNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Unable to Delete Superset Database",
			err.Error(),
		)
	}
}
