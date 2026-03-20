package provider

import (
	"testing"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandSavedQueryCreateRequestRejectsInvalidTemplateParameters(t *testing.T) {
	t.Parallel()

	_, diags := expandSavedQueryCreateRequest(savedQueryModel{
		DatabaseID:         types.Int64Value(11),
		Label:              types.StringValue("Orders"),
		SQL:                types.StringValue("select 1"),
		TemplateParameters: types.StringValue("{"),
	})

	if !diags.HasError() {
		t.Fatal("expected invalid template_parameters JSON to fail validation")
	}
}

func TestFlattenSavedQueryModelPreservesConfiguredExtraJSON(t *testing.T) {
	t.Parallel()

	state, diags := flattenSavedQueryModel(savedQueryModel{
		ExtraJSON:          types.StringValue("{\"x\":1}"),
		TemplateParameters: types.StringValue("{\"region\":\"emea\"}"),
	}, &supersetclient.SavedQuery{
		ID:          7,
		Label:       "Orders",
		SQL:         "select 1",
		Database:    supersetclient.SavedQueryDatabase{ID: 11, DatabaseName: "analytics"},
		ExtraJSON:   nil,
		Catalog:     nil,
		Schema:      nil,
		Description: nil,
	})
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	if got := state.ExtraJSON.ValueString(); got != "{\"x\":1}" {
		t.Fatalf("expected configured extra_json to remain in state, got %q", got)
	}

	if got := state.TemplateParameters.ValueString(); got != "{\"region\":\"emea\"}" {
		t.Fatalf("expected configured template_parameters to remain in state, got %q", got)
	}
}
