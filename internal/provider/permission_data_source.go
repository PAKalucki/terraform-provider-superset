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

var _ datasource.DataSource = &PermissionDataSource{}

func NewPermissionDataSource() datasource.DataSource {
	return &PermissionDataSource{}
}

type PermissionDataSource struct {
	client *supersetclient.Client
}

func (d *PermissionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

func (d *PermissionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing Superset permission-view-menu resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Superset permission-view-menu identifier used for lookup or returned from Superset.",
			},
			"permission_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Permission name used for lookup or returned from Superset, for example `can_read`.",
			},
			"view_menu_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "View menu name used for lookup or returned from Superset, for example `SavedQuery`.",
			},
		},
	}
}

func (d *PermissionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PermissionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data permissionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the permission data source.",
		)

		return
	}

	hasID := !data.ID.IsNull() && !data.ID.IsUnknown()
	hasPermissionName := !data.PermissionName.IsNull() && !data.PermissionName.IsUnknown() && strings.TrimSpace(data.PermissionName.ValueString()) != ""
	hasViewMenuName := !data.ViewMenuName.IsNull() && !data.ViewMenuName.IsUnknown() && strings.TrimSpace(data.ViewMenuName.ValueString()) != ""

	switch {
	case hasID && (hasPermissionName || hasViewMenuName):
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Conflicting Permission Lookup Arguments",
			"Configure either `id` or `permission_name` with `view_menu_name`.",
		)

		return
	case !hasID && (!hasPermissionName || !hasViewMenuName):
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Missing Permission Lookup Arguments",
			"Configure either `id` or `permission_name` with `view_menu_name`.",
		)

		return
	}

	var (
		permission *supersetclient.Permission
		err        error
	)

	if hasID {
		permission, err = d.client.GetPermission(ctx, data.ID.ValueInt64())
	} else {
		permission, err = findPermissionByName(ctx, d.client, data.PermissionName.ValueString(), data.ViewMenuName.ValueString())
	}

	if err != nil {
		if hasID && isSupersetNotFoundError(err) {
			resp.Diagnostics.AddAttributeError(
				path.Root("id"),
				"Superset Permission Not Found",
				err.Error(),
			)
		} else {
			resp.Diagnostics.AddError(
				"Unable to Read Superset Permission",
				err.Error(),
			)
		}

		return
	}

	state := flattenPermissionModel(permission)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
