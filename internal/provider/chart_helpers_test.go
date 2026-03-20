package provider

import (
	"testing"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandChartCreateRequestRejectsInvalidParams(t *testing.T) {
	t.Parallel()

	_, diags := expandChartCreateRequest(chartModel{
		SliceName:      types.StringValue("Orders"),
		DatasourceID:   types.Int64Value(11),
		DatasourceType: types.StringValue("table"),
		VizType:        types.StringValue("table"),
		Params:         types.StringValue("{"),
	})

	if !diags.HasError() {
		t.Fatal("expected invalid params JSON to fail validation")
	}
}

func TestFlattenChartModelNormalizesJSON(t *testing.T) {
	t.Parallel()

	queryContext := "{\n  \"b\": 2,\n  \"a\": 1\n}"
	cacheTimeout := int64(300)

	state, diags := flattenChartModel(chartModel{}, &supersetclient.Chart{
		ID:                 7,
		UUID:               "fef350f4-d046-404c-b6c8-2713af6334c8",
		SliceName:          "Orders",
		Description:        "Warehouse chart",
		VizType:            "table",
		Params:             "{\n  \"viz_type\": \"table\",\n  \"datasource\": \"11__table\"\n}",
		QueryContext:       &queryContext,
		CacheTimeout:       &cacheTimeout,
		DatasourceID:       11,
		DatasourceType:     "table",
		DatasourceNameText: "analytics.orders",
		URL:                "/explore/?slice_id=7",
	})
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	if got := state.Params.ValueString(); got != "{\"datasource\":\"11__table\",\"viz_type\":\"table\"}" {
		t.Fatalf("expected normalized params JSON, got %q", got)
	}

	if got := state.QueryContext.ValueString(); got != "{\"a\":1,\"b\":2}" {
		t.Fatalf("expected normalized query_context JSON, got %q", got)
	}

	if got := state.DatasourceName.ValueString(); got != "analytics.orders" {
		t.Fatalf("expected datasource name to be populated, got %q", got)
	}

	if got := state.CacheTimeout.ValueInt64(); got != 300 {
		t.Fatalf("expected cache timeout to be populated, got %d", got)
	}
}
