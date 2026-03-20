package provider

import (
	"context"
	"testing"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandRoleCreateRequestRejectsBlankName(t *testing.T) {
	t.Parallel()

	_, diags := expandRoleCreateRequest(roleResourceModel{
		Name: types.StringValue("   "),
	})

	if !diags.HasError() {
		t.Fatal("expected blank role name to fail validation")
	}
}

func TestExpandRolePermissionIDsSortsValues(t *testing.T) {
	t.Parallel()

	value, diags := types.SetValueFrom(context.Background(), types.Int64Type, []int64{15, 13, 14})
	if diags.HasError() {
		t.Fatalf("expected permission set, got diagnostics: %v", diags)
	}

	permissionIDs, expandDiags := expandRolePermissionIDs(context.Background(), value)
	if expandDiags.HasError() {
		t.Fatalf("expected permission ids to expand, got diagnostics: %v", expandDiags)
	}

	expected := []int64{13, 14, 15}
	for index, permissionID := range expected {
		if permissionIDs[index] != permissionID {
			t.Fatalf("expected sorted permission ids %#v, got %#v", expected, permissionIDs)
		}
	}
}

func TestExpandRolePermissionIDsRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	value, diags := types.SetValueFrom(context.Background(), types.Int64Type, []int64{13, 0})
	if diags.HasError() {
		t.Fatalf("expected permission set, got diagnostics: %v", diags)
	}

	_, expandDiags := expandRolePermissionIDs(context.Background(), value)
	if !expandDiags.HasError() {
		t.Fatal("expected non-positive permission ids to fail validation")
	}
}

func TestFlattenRolePermissionModelSortsPermissionIDs(t *testing.T) {
	t.Parallel()

	state, diags := flattenRolePermissionModel(context.Background(), &supersetclient.Role{
		ID:            9,
		Name:          "Analyst",
		PermissionIDs: []int64{15, 13, 14},
	})
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	var permissionIDs []int64
	flattenDiags := state.PermissionIDs.ElementsAs(context.Background(), &permissionIDs, false)
	if flattenDiags.HasError() {
		t.Fatalf("expected flattened permission ids, got diagnostics: %v", flattenDiags)
	}

	expected := []int64{13, 14, 15}
	for index, permissionID := range expected {
		if permissionIDs[index] != permissionID {
			t.Fatalf("expected sorted permission ids %#v, got %#v", expected, permissionIDs)
		}
	}

	if got := state.RoleName.ValueString(); got != "Analyst" {
		t.Fatalf("expected role name to be flattened, got %q", got)
	}
}

func TestFlattenRolePermissionModelUsesEmptySetForNoPermissions(t *testing.T) {
	t.Parallel()

	state, diags := flattenRolePermissionModel(context.Background(), &supersetclient.Role{
		ID:   9,
		Name: "Analyst",
	})
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	if state.PermissionIDs.IsNull() {
		t.Fatal("expected permission_ids to flatten to an empty set, not null")
	}

	if got := state.PermissionIDs.Elements(); len(got) != 0 {
		t.Fatalf("expected no permission ids, got %#v", got)
	}
}
