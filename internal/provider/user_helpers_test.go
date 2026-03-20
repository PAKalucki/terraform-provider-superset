package provider

import (
	"context"
	"testing"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandUserCreateRequestRequiresPassword(t *testing.T) {
	t.Parallel()

	roleIDs, diags := types.SetValueFrom(context.Background(), types.Int64Type, []int64{3})
	if diags.HasError() {
		t.Fatalf("expected role set, got diagnostics: %v", diags)
	}

	_, expandDiags := expandUserCreateRequest(context.Background(), userModel{
		Username:  types.StringValue("analyst"),
		FirstName: types.StringValue("Analytics"),
		LastName:  types.StringValue("User"),
		Email:     types.StringValue("analyst@example.com"),
		Active:    types.BoolValue(true),
		RoleIDs:   roleIDs,
	})

	if !expandDiags.HasError() {
		t.Fatal("expected missing password to fail create validation")
	}
}

func TestExpandUserUpdateRequestOmitsPasswordWhenUnset(t *testing.T) {
	t.Parallel()

	roleIDs, diags := types.SetValueFrom(context.Background(), types.Int64Type, []int64{3})
	if diags.HasError() {
		t.Fatalf("expected role set, got diagnostics: %v", diags)
	}

	request, expandDiags := expandUserUpdateRequest(context.Background(), userModel{
		Username:  types.StringValue("analyst"),
		FirstName: types.StringValue("Analytics"),
		LastName:  types.StringValue("User"),
		Email:     types.StringValue("analyst@example.com"),
		Active:    types.BoolValue(true),
		RoleIDs:   roleIDs,
		Password:  types.StringNull(),
	})
	if expandDiags.HasError() {
		t.Fatalf("expected update request, got diagnostics: %v", expandDiags)
	}

	if request.Password != nil {
		t.Fatal("expected password to be omitted when unset")
	}
}

func TestFlattenUserModelPreservesConfiguredPassword(t *testing.T) {
	t.Parallel()

	roleIDs, diags := types.SetValueFrom(context.Background(), types.Int64Type, []int64{3})
	if diags.HasError() {
		t.Fatalf("expected role set, got diagnostics: %v", diags)
	}

	state, flattenDiags := flattenUserModel(context.Background(), userModel{
		Password: types.StringValue("Terraform123!"),
		RoleIDs:  roleIDs,
	}, &supersetclient.User{
		ID:        7,
		Username:  "analyst",
		FirstName: "Analytics",
		LastName:  "User",
		Email:     "analyst@example.com",
		Active:    true,
		Roles: []supersetclient.RoleRef{
			{ID: 4, Name: "Gamma"},
			{ID: 3, Name: "Alpha"},
		},
	})
	if flattenDiags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", flattenDiags)
	}

	if got := state.Password.ValueString(); got != "Terraform123!" {
		t.Fatalf("expected configured password to remain in state, got %q", got)
	}

	var flattenedRoleIDs []int64
	roleDiags := state.RoleIDs.ElementsAs(context.Background(), &flattenedRoleIDs, false)
	if roleDiags.HasError() {
		t.Fatalf("expected flattened role ids, got diagnostics: %v", roleDiags)
	}

	if len(flattenedRoleIDs) != 2 || flattenedRoleIDs[0] != 3 || flattenedRoleIDs[1] != 4 {
		t.Fatalf("expected sorted role ids [3 4], got %#v", flattenedRoleIDs)
	}
}
