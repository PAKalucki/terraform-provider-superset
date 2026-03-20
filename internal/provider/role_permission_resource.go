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
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &RolePermissionResource{}

func NewRolePermissionResource() resource.Resource {
	return &RolePermissionResource{}
}

type RolePermissionResource struct {
	client *supersetclient.Client
}

func (r *RolePermissionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_permission"
}

func (r *RolePermissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the full Superset permission set for one role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Terraform resource identifier. This matches `role_id`.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"role_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Superset role identifier whose permissions are managed by this resource.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"role_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved Superset role name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"permission_ids": schema.SetAttribute{
				Required:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Authoritative set of permission-view-menu identifiers assigned to the role.",
			},
		},
	}
}

func (r *RolePermissionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RolePermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data rolePermissionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the role permission resource.",
		)

		return
	}

	permissionIDs, diags := expandRolePermissionIDs(ctx, data.PermissionIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := loadRoleWithAssignments(ctx, r.client, data.RoleID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Role Before Permission Assignment",
			err.Error(),
		)

		return
	}

	if err := r.client.SetRolePermissions(ctx, role.ID, permissionIDs); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Assign Superset Role Permissions",
			err.Error(),
		)

		return
	}

	role, err = loadRoleWithAssignments(ctx, r.client, role.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Role After Permission Assignment",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenRolePermissionModel(ctx, role)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RolePermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data rolePermissionModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the role permission resource.",
		)

		return
	}

	role, err := loadRoleWithAssignments(ctx, r.client, data.RoleID.ValueInt64())
	if err != nil {
		if isSupersetNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Unable to Read Superset Role Permissions",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenRolePermissionModel(ctx, role)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RolePermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data rolePermissionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the role permission resource.",
		)

		return
	}

	permissionIDs, diags := expandRolePermissionIDs(ctx, data.PermissionIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SetRolePermissions(ctx, data.RoleID.ValueInt64(), permissionIDs); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Superset Role Permissions",
			err.Error(),
		)

		return
	}

	role, err := loadRoleWithAssignments(ctx, r.client, data.RoleID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Role After Permission Update",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenRolePermissionModel(ctx, role)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RolePermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data rolePermissionModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the role permission resource.",
		)

		return
	}

	if err := r.client.SetRolePermissions(ctx, data.RoleID.ValueInt64(), nil); err != nil && !isSupersetNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Unable to Clear Superset Role Permissions",
			err.Error(),
		)
	}
}
