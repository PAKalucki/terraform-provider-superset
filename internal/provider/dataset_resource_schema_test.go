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
