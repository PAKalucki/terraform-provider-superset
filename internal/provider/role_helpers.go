package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type roleResourceModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type roleDataSourceModel struct {
	ID            types.Int64  `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	UserIDs       types.Set    `tfsdk:"user_ids"`
	GroupIDs      types.Set    `tfsdk:"group_ids"`
	PermissionIDs types.Set    `tfsdk:"permission_ids"`
}

type permissionModel struct {
	ID             types.Int64  `tfsdk:"id"`
	PermissionName types.String `tfsdk:"permission_name"`
	ViewMenuName   types.String `tfsdk:"view_menu_name"`
}

type rolePermissionModel struct {
	ID            types.Int64  `tfsdk:"id"`
	RoleID        types.Int64  `tfsdk:"role_id"`
	RoleName      types.String `tfsdk:"role_name"`
	PermissionIDs types.Set    `tfsdk:"permission_ids"`
}

func expandRoleCreateRequest(data roleResourceModel) (supersetclient.RoleCreateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	name := strings.TrimSpace(stringValue(data.Name))
	if name == "" {
		diags.AddAttributeError(
			path.Root("name"),
			"Missing Role Name",
			"`name` must be configured.",
		)
	}

	return supersetclient.RoleCreateRequest{Name: name}, diags
}

func expandRoleUpdateRequest(data roleResourceModel) (supersetclient.RoleUpdateRequest, diag.Diagnostics) {
	request, diags := expandRoleCreateRequest(data)

	return supersetclient.RoleUpdateRequest(request), diags
}

func flattenRoleResourceModel(remote *supersetclient.Role) roleResourceModel {
	return roleResourceModel{
		ID:   types.Int64Value(remote.ID),
		Name: stringTypeValue(remote.Name),
	}
}

func flattenRoleDataSourceModel(ctx context.Context, remote *supersetclient.Role) (roleDataSourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	userIDs, userDiags := int64SetValueFrom(ctx, remote.UserIDs)
	diags.Append(userDiags...)

	groupIDs, groupDiags := int64SetValueFrom(ctx, remote.GroupIDs)
	diags.Append(groupDiags...)

	permissionIDs, permissionDiags := int64SetValueFrom(ctx, remote.PermissionIDs)
	diags.Append(permissionDiags...)

	return roleDataSourceModel{
		ID:            types.Int64Value(remote.ID),
		Name:          stringTypeValue(remote.Name),
		UserIDs:       userIDs,
		GroupIDs:      groupIDs,
		PermissionIDs: permissionIDs,
	}, diags
}

func expandRolePermissionIDs(ctx context.Context, value types.Set) ([]int64, diag.Diagnostics) {
	var permissionIDs []int64

	if value.IsNull() || value.IsUnknown() {
		return permissionIDs, nil
	}

	diags := value.ElementsAs(ctx, &permissionIDs, false)
	if diags.HasError() {
		return nil, diags
	}

	sort.Slice(permissionIDs, func(i, j int) bool { return permissionIDs[i] < permissionIDs[j] })

	for index, permissionID := range permissionIDs {
		if permissionID <= 0 {
			diags.AddAttributeError(
				path.Root("permission_ids").AtSetValue(types.Int64Value(permissionID)),
				"Invalid Permission Identifier",
				fmt.Sprintf("Permission id at index %d must be a positive integer.", index),
			)
		}
	}

	return permissionIDs, diags
}

func flattenRolePermissionModel(ctx context.Context, remote *supersetclient.Role) (rolePermissionModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	permissionIDs, permissionDiags := int64SetValueFrom(ctx, remote.PermissionIDs)
	diags.Append(permissionDiags...)

	return rolePermissionModel{
		ID:            types.Int64Value(remote.ID),
		RoleID:        types.Int64Value(remote.ID),
		RoleName:      stringTypeValue(remote.Name),
		PermissionIDs: permissionIDs,
	}, diags
}

func flattenPermissionModel(remote *supersetclient.Permission) permissionModel {
	return permissionModel{
		ID:             types.Int64Value(remote.ID),
		PermissionName: stringTypeValue(remote.PermissionName),
		ViewMenuName:   stringTypeValue(remote.ViewMenuName),
	}
}

func loadRoleWithAssignments(ctx context.Context, client *supersetclient.Client, id int64) (*supersetclient.Role, error) {
	role, err := client.GetRole(ctx, id)
	if err != nil {
		return nil, err
	}

	roles, err := client.ListRoles(ctx, 1000)
	if err != nil {
		return nil, err
	}

	for _, listedRole := range roles {
		if listedRole.ID != id {
			continue
		}

		role.UserIDs = listedRole.UserIDs
		role.GroupIDs = listedRole.GroupIDs
		role.PermissionIDs = listedRole.PermissionIDs

		if strings.TrimSpace(role.Name) == "" {
			role.Name = listedRole.Name
		}

		return role, nil
	}

	return role, nil
}

func findRoleByName(ctx context.Context, client *supersetclient.Client, name string) (*supersetclient.Role, error) {
	roles, err := client.ListRoles(ctx, 1000)
	if err != nil {
		return nil, err
	}

	normalizedName := strings.TrimSpace(name)
	var matches []supersetclient.Role

	for _, role := range roles {
		if strings.TrimSpace(role.Name) == normalizedName {
			matches = append(matches, role)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("role %q was not found", normalizedName)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("role %q matched %d roles", normalizedName, len(matches))
	}
}

func findPermissionByName(ctx context.Context, client *supersetclient.Client, permissionName string, viewMenuName string) (*supersetclient.Permission, error) {
	permissions, err := client.ListPermissions(ctx, 1000)
	if err != nil {
		return nil, err
	}

	normalizedPermissionName := strings.TrimSpace(permissionName)
	normalizedViewMenuName := strings.TrimSpace(viewMenuName)

	var matches []supersetclient.Permission

	for _, permission := range permissions {
		if strings.TrimSpace(permission.PermissionName) != normalizedPermissionName {
			continue
		}

		if strings.TrimSpace(permission.ViewMenuName) != normalizedViewMenuName {
			continue
		}

		matches = append(matches, permission)
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("permission %q on view menu %q was not found", normalizedPermissionName, normalizedViewMenuName)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("permission %q on view menu %q matched %d permissions", normalizedPermissionName, normalizedViewMenuName, len(matches))
	}
}

func int64SetValueFrom(ctx context.Context, values []int64) (types.Set, diag.Diagnostics) {
	if len(values) == 0 {
		return types.SetValueFrom(ctx, types.Int64Type, []int64{})
	}

	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	return types.SetValueFrom(ctx, types.Int64Type, sorted)
}
