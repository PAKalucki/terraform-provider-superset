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

type userModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Username  types.String `tfsdk:"username"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	Email     types.String `tfsdk:"email"`
	Active    types.Bool   `tfsdk:"active"`
	Password  types.String `tfsdk:"password"`
	RoleIDs   types.Set    `tfsdk:"role_ids"`
}

func expandUserCreateRequest(ctx context.Context, data userModel) (supersetclient.UserCreateRequest, diag.Diagnostics) {
	request, diags := expandUserUpdateRequest(ctx, data)

	password := strings.TrimSpace(stringValue(data.Password))
	if password == "" {
		diags.AddAttributeError(
			path.Root("password"),
			"Missing User Password",
			"`password` must be configured when creating a Superset user.",
		)
	}

	return supersetclient.UserCreateRequest{
		Username:  request.Username,
		FirstName: request.FirstName,
		LastName:  request.LastName,
		Email:     request.Email,
		Active:    request.Active,
		Password:  password,
		Roles:     request.Roles,
		Groups:    request.Groups,
	}, diags
}

func expandUserUpdateRequest(ctx context.Context, data userModel) (supersetclient.UserUpdateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	username := strings.TrimSpace(stringValue(data.Username))
	if username == "" {
		diags.AddAttributeError(
			path.Root("username"),
			"Missing Username",
			"`username` must be configured.",
		)
	}

	firstName := strings.TrimSpace(stringValue(data.FirstName))
	if firstName == "" {
		diags.AddAttributeError(
			path.Root("first_name"),
			"Missing First Name",
			"`first_name` must be configured.",
		)
	}

	lastName := strings.TrimSpace(stringValue(data.LastName))
	if lastName == "" {
		diags.AddAttributeError(
			path.Root("last_name"),
			"Missing Last Name",
			"`last_name` must be configured.",
		)
	}

	email := strings.TrimSpace(stringValue(data.Email))
	if email == "" {
		diags.AddAttributeError(
			path.Root("email"),
			"Missing Email Address",
			"`email` must be configured.",
		)
	}

	roleIDs, roleDiags := expandUserRoleIDs(ctx, data.RoleIDs)
	diags.Append(roleDiags...)
	if diags.HasError() {
		return supersetclient.UserUpdateRequest{}, diags
	}

	request := supersetclient.UserUpdateRequest{
		Username:  username,
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Active:    boolValue(data.Active),
		Roles:     roleIDs,
		Groups:    []int64{},
	}

	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		password := strings.TrimSpace(data.Password.ValueString())
		if password != "" {
			request.Password = &password
		}
	}

	return request, diags
}

func expandUserRoleIDs(ctx context.Context, value types.Set) ([]int64, diag.Diagnostics) {
	var roleIDs []int64

	if value.IsNull() || value.IsUnknown() {
		return roleIDs, nil
	}

	diags := value.ElementsAs(ctx, &roleIDs, false)
	if diags.HasError() {
		return nil, diags
	}

	sort.Slice(roleIDs, func(i, j int) bool { return roleIDs[i] < roleIDs[j] })

	for index, roleID := range roleIDs {
		if roleID <= 0 {
			diags.AddAttributeError(
				path.Root("role_ids").AtSetValue(types.Int64Value(roleID)),
				"Invalid Role Identifier",
				fmt.Sprintf("Role id at index %d must be a positive integer.", index),
			)
		}
	}

	return roleIDs, diags
}

func flattenUserModel(ctx context.Context, current userModel, remote *supersetclient.User) (userModel, diag.Diagnostics) {
	roleIDs, diags := int64SetValueFrom(ctx, userRoleIDs(remote.Roles))

	state := current
	state.ID = types.Int64Value(remote.ID)
	state.Username = stringTypeValue(remote.Username)
	state.FirstName = stringTypeValue(remote.FirstName)
	state.LastName = stringTypeValue(remote.LastName)
	state.Email = stringTypeValue(remote.Email)
	state.Active = types.BoolValue(remote.Active)
	state.RoleIDs = roleIDs

	return state, diags
}

func userRoleIDs(roles []supersetclient.RoleRef) []int64 {
	ids := make([]int64, 0, len(roles))

	for _, role := range roles {
		if role.ID > 0 {
			ids = append(ids, role.ID)
		}
	}

	return ids
}
