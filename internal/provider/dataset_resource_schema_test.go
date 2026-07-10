package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDatasetResourceSchemaRequiresReplaceForIdentityAttributes(t *testing.T) {
	t.Parallel()

	resourceInstance, ok := NewDatasetResource().(*DatasetResource)
	if !ok {
		t.Fatal("expected dataset resource instance")
	}

	var resp resource.SchemaResponse
	resourceInstance.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	databaseIDAttr, ok := resp.Schema.Attributes["database_id"].(resourceschema.Int64Attribute)
	if !ok {
		t.Fatal("expected database_id int64 attribute")
	}

	if !hasInt64PlanModifier(databaseIDAttr.PlanModifiers, int64planmodifier.RequiresReplace()) {
		t.Fatal("expected database_id to require replacement")
	}

	tableNameAttr, ok := resp.Schema.Attributes["table_name"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("expected table_name string attribute")
	}

	if !hasStringPlanModifier(tableNameAttr.PlanModifiers, stringplanmodifier.RequiresReplace()) {
		t.Fatal("expected table_name to require replacement")
	}

	sqlAttr, ok := resp.Schema.Attributes["sql"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("expected sql string attribute")
	}

	if len(sqlAttr.PlanModifiers) == 0 {
		t.Fatal("expected sql to have a conditional replacement plan modifier")
	}
}

func TestDatasetSQLRequiresReplaceOnlyForVirtualPhysicalConversion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		state types.String
		plan  types.String
		want  bool
	}{
		{
			name:  "changing the query updates in place",
			state: types.StringValue("SELECT col1 FROM my_table"),
			plan:  types.StringValue("SELECT col1, col2 FROM my_table"),
			want:  false,
		},
		{
			name:  "adding sql converts physical to virtual and forces replacement",
			state: types.StringNull(),
			plan:  types.StringValue("SELECT col1 FROM my_table"),
			want:  true,
		},
		{
			name:  "removing sql converts virtual to physical and forces replacement",
			state: types.StringValue("SELECT col1 FROM my_table"),
			plan:  types.StringNull(),
			want:  true,
		},
		{
			name:  "unknown planned sql does not force replacement",
			state: types.StringValue("SELECT col1 FROM my_table"),
			plan:  types.StringUnknown(),
			want:  false,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var resp stringplanmodifier.RequiresReplaceIfFuncResponse

			datasetSQLRequiresReplace(context.Background(), planmodifier.StringRequest{
				StateValue: tc.state,
				PlanValue:  tc.plan,
			}, &resp)

			if resp.RequiresReplace != tc.want {
				t.Fatalf("expected RequiresReplace %v, got %v", tc.want, resp.RequiresReplace)
			}
		})
	}
}

func hasInt64PlanModifier(modifiers []planmodifier.Int64, target planmodifier.Int64) bool {
	targetType := reflect.TypeOf(target)

	for _, modifier := range modifiers {
		if reflect.TypeOf(modifier) == targetType {
			return true
		}
	}

	return false
}

func hasStringPlanModifier(modifiers []planmodifier.String, target planmodifier.String) bool {
	targetType := reflect.TypeOf(target)

	for _, modifier := range modifiers {
		if reflect.TypeOf(modifier) == targetType {
			return true
		}
	}

	return false
}
