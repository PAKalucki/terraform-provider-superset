package provider

import (
	"context"
	"fmt"
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RoleDataSource{}

func NewRoleDataSource() datasource.DataSource {
	return &RoleDataSource{}
}

type RoleDataSource struct {
	client *supersetclient.Client
}

func (d *RoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *RoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing Superset role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Superset role identifier used for lookup or returned from Superset.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Role name used for lookup or returned from Superset.",
			},
			"user_ids": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Superset user identifiers currently assigned to the role.",
			},
			"group_ids": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Superset group identifiers currently assigned to the role.",
			},
			"permission_ids": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Superset permission-view-menu identifiers currently assigned to the role.",
			},
		},
	}
}

func (d *RoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data roleDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the role data source.",
		)

		return
	}

	hasID := !data.ID.IsNull() && !data.ID.IsUnknown()
	hasName := !data.Name.IsNull() && !data.Name.IsUnknown() && strings.TrimSpace(data.Name.ValueString()) != ""

	switch {
	case hasID && hasName:
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Conflicting Role Lookup Arguments",
			"Configure only one of `id` or `name`.",
		)

		return
	case !hasID && !hasName:
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Missing Role Lookup Arguments",
			"Configure either `id` or `name`.",
		)

		return
	}

	var (
		role *supersetclient.Role
		err  error
	)

	if hasID {
		role, err = loadRoleWithAssignments(ctx, d.client, data.ID.ValueInt64())
	} else {
		role, err = findRoleByName(ctx, d.client, data.Name.ValueString())
	}

	if err != nil {
		if hasID && isSupersetNotFoundError(err) {
			resp.Diagnostics.AddAttributeError(
				path.Root("id"),
				"Superset Role Not Found",
				err.Error(),
			)
		} else {
			resp.Diagnostics.AddError(
				"Unable to Read Superset Role",
				err.Error(),
			)
		}

		return
	}

	state, diags := flattenRoleDataSourceModel(ctx, role)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
