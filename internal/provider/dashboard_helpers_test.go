package provider

import (
	"context"
	"encoding/json"
	"testing"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildDashboardPositionJSONRoundTrip(t *testing.T) {
	t.Parallel()

	positionJSON, err := buildDashboardPositionJSON("Operations", []dashboardPositionChart{
		{ID: 11, SliceName: "Orders", UUID: "11111111-1111-1111-1111-111111111111"},
		{ID: 22, SliceName: "Revenue", UUID: "22222222-2222-2222-2222-222222222222"},
	})
	if err != nil {
		t.Fatalf("expected dashboard position JSON to build, got error: %v", err)
	}

	chartIDs, diags := extractDashboardChartIDs(positionJSON)
	if diags.HasError() {
		t.Fatalf("expected dashboard position JSON to decode, got diagnostics: %v", diags)
	}

	if len(chartIDs) != 2 || chartIDs[0] != 11 || chartIDs[1] != 22 {
		t.Fatalf("expected chart ids [11 22], got %#v", chartIDs)
	}
}

func TestExpandDashboardUpdateRequestRejectsMismatchedChartIDs(t *testing.T) {
	t.Parallel()

	chartIDs, diags := types.ListValueFrom(context.Background(), types.Int64Type, []int64{11})
	if diags.HasError() {
		t.Fatalf("expected chart id list to build, got diagnostics: %v", diags)
	}

	request, requestDiags := expandDashboardUpdateRequest(context.Background(), nil, dashboardModel{
		DashboardTitle: types.StringValue("Operations"),
		ChartIDs:       chartIDs,
		PositionJSON: types.StringValue(`{
			"DASHBOARD_VERSION_KEY": "v2",
			"ROOT_ID": {"children":["GRID_ID"],"id":"ROOT_ID","type":"ROOT"},
			"GRID_ID": {"children":["ROW-1"],"id":"GRID_ID","parents":["ROOT_ID"],"type":"GRID"},
			"ROW-1": {"children":["CHART-1"],"id":"ROW-1","meta":{"0":"ROOT_ID","background":"BACKGROUND_TRANSPARENT"},"parents":["ROOT_ID","GRID_ID"],"type":"ROW"},
			"CHART-1": {"children":[],"id":"CHART-1","meta":{"chartId":22},"parents":["ROOT_ID","GRID_ID","ROW-1"],"type":"CHART"}
		}`),
	}, dashboardModel{}, nil)
	if !requestDiags.HasError() {
		t.Fatalf("expected mismatched chart ids to fail validation, got request %#v", request)
	}
}

func TestExpandDashboardUpdateRequestPreservesUnmanagedNativeFiltersWhenUpdatingLayout(t *testing.T) {
	t.Parallel()

	request, requestDiags := expandDashboardUpdateRequest(context.Background(), nil, dashboardModel{
		DashboardTitle: types.StringValue("Operations"),
		PositionJSON: types.StringValue(`{
			"DASHBOARD_VERSION_KEY": "v2",
			"ROOT_ID": {"children":["GRID_ID"],"id":"ROOT_ID","type":"ROOT"},
			"GRID_ID": {"children":["ROW-1"],"id":"GRID_ID","parents":["ROOT_ID"],"type":"GRID"},
			"ROW-1": {"children":["CHART-1"],"id":"ROW-1","meta":{"0":"ROOT_ID","background":"BACKGROUND_TRANSPARENT"},"parents":["ROOT_ID","GRID_ID"],"type":"ROW"},
			"CHART-1": {"children":[],"id":"CHART-1","meta":{"chartId":11},"parents":["ROOT_ID","GRID_ID","ROW-1"],"type":"CHART"}
		}`),
	}, dashboardModel{}, &supersetclient.Dashboard{
		JSONMetadata: `{"positions":{"existing":"layout"},"show_native_filters":true,"native_filter_configuration":[{"id":"NATIVE_FILTER-1","filterType":"filter_select"}]}`,
	})
	if requestDiags.HasError() {
		t.Fatalf("expected dashboard update request, got diagnostics: %v", requestDiags)
	}

	if request.JSONMetadata == nil {
		t.Fatal("expected dashboard update request to include json_metadata")
	}

	var metadata map[string]any
	if err := json.Unmarshal([]byte(*request.JSONMetadata), &metadata); err != nil {
		t.Fatalf("expected metadata JSON to decode, got error: %v", err)
	}

	if _, ok := metadata["positions"]; !ok {
		t.Fatalf("expected metadata to include positions, got %#v", metadata)
	}

	if showNativeFilters, ok := metadata["show_native_filters"].(bool); !ok || !showNativeFilters {
		t.Fatalf("expected unmanaged native filter visibility to be preserved, got %#v", metadata["show_native_filters"])
	}

	filters, ok := metadata["native_filter_configuration"].([]any)
	if !ok || len(filters) != 1 {
		t.Fatalf("expected unmanaged native filters to be preserved, got %#v", metadata["native_filter_configuration"])
	}
}

func TestExpandDashboardUpdateRequestIncludesNativeFiltersAndVisibility(t *testing.T) {
	t.Parallel()

	request, requestDiags := expandDashboardUpdateRequest(context.Background(), nil, dashboardModel{
		DashboardTitle:            types.StringValue("Operations"),
		NativeFilterConfiguration: types.StringValue(`[{"id":"NATIVE_FILTER-1","filterType":"filter_select","targets":[{"datasetId":11,"column":{"name":"country_code"}}]}]`),
	}, dashboardModel{}, nil)
	if requestDiags.HasError() {
		t.Fatalf("expected dashboard update request, got diagnostics: %v", requestDiags)
	}

	if request.JSONMetadata == nil {
		t.Fatal("expected dashboard update request to include json_metadata")
	}

	var metadata map[string]any
	if err := json.Unmarshal([]byte(*request.JSONMetadata), &metadata); err != nil {
		t.Fatalf("expected metadata JSON to decode, got error: %v", err)
	}

	if showNativeFilters, ok := metadata["show_native_filters"].(bool); !ok || !showNativeFilters {
		t.Fatalf("expected native filters to be enabled automatically, got %#v", metadata["show_native_filters"])
	}

	filters, ok := metadata["native_filter_configuration"].([]any)
	if !ok || len(filters) != 1 {
		t.Fatalf("expected native filter configuration to be encoded, got %#v", metadata["native_filter_configuration"])
	}
}

func TestFlattenDashboardResourceModelKeepsUnmanagedFieldsNull(t *testing.T) {
	t.Parallel()

	state, diags := flattenDashboardResourceModel(context.Background(), dashboardModel{}, &supersetclient.Dashboard{
		ID:             7,
		UUID:           "fef350f4-d046-404c-b6c8-2713af6334c8",
		DashboardTitle: "Operations",
		Slug:           "operations",
		CSS:            ".dashboard { color: red; }",
		URL:            "/superset/dashboard/operations/",
		PositionJSON:   `{"DASHBOARD_VERSION_KEY":"v2"}`,
		JSONMetadata:   `{"positions":{"DASHBOARD_VERSION_KEY":"v2"},"show_native_filters":true,"native_filter_configuration":[{"id":"NATIVE_FILTER-1"}]}`,
	}, []supersetclient.DashboardChart{
		{ID: 11, SliceName: "Orders"},
	})
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	if !state.Slug.IsNull() {
		t.Fatalf("expected unmanaged slug to remain null, got %q", state.Slug.ValueString())
	}

	if !state.CSS.IsNull() {
		t.Fatalf("expected unmanaged css to remain null, got %q", state.CSS.ValueString())
	}

	if !state.PositionJSON.IsNull() {
		t.Fatalf("expected unmanaged position_json to remain null, got %q", state.PositionJSON.ValueString())
	}

	if !state.ShowNativeFilters.IsNull() {
		t.Fatalf("expected unmanaged show_native_filters to remain null, got %#v", state.ShowNativeFilters)
	}

	if !state.NativeFilterConfiguration.IsNull() {
		t.Fatalf("expected unmanaged native_filter_configuration to remain null, got %q", state.NativeFilterConfiguration.ValueString())
	}

	if !state.ChartIDs.IsNull() {
		t.Fatalf("expected unmanaged chart_ids to remain null, got %#v", state.ChartIDs)
	}
}

func TestFlattenDashboardResourceModelRefreshesManagedNativeFilters(t *testing.T) {
	t.Parallel()

	state, diags := flattenDashboardResourceModel(context.Background(), dashboardModel{
		ShowNativeFilters:         types.BoolValue(true),
		NativeFilterConfiguration: types.StringValue(`[]`),
	}, &supersetclient.Dashboard{
		ID:             7,
		UUID:           "fef350f4-d046-404c-b6c8-2713af6334c8",
		DashboardTitle: "Operations",
		URL:            "/superset/dashboard/operations/",
		JSONMetadata:   `{"show_native_filters":true,"native_filter_configuration":[{"id":"NATIVE_FILTER-1","filterType":"filter_time"}]}`,
	}, nil)
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	if state.ShowNativeFilters.IsNull() || !state.ShowNativeFilters.ValueBool() {
		t.Fatalf("expected managed show_native_filters to refresh, got %#v", state.ShowNativeFilters)
	}

	if got := state.NativeFilterConfiguration.ValueString(); got != `[{"filterType":"filter_time","id":"NATIVE_FILTER-1"}]` {
		t.Fatalf("expected managed native_filter_configuration to refresh, got %q", got)
	}
}

func TestFlattenDashboardChartIDsPreservesManagedOrder(t *testing.T) {
	t.Parallel()

	current, diags := types.ListValueFrom(context.Background(), types.Int64Type, []int64{24, 19, 22, 21, 23, 20})
	if diags.HasError() {
		t.Fatalf("expected managed chart_ids list to build, got diagnostics: %v", diags)
	}

	chartIDs, diags := flattenDashboardChartIDs(context.Background(), current, []supersetclient.DashboardChart{
		{ID: 19, SliceName: "Runs By Status"},
		{ID: 20, SliceName: "Recent Runs"},
		{ID: 21, SliceName: "Marker Failures"},
		{ID: 22, SliceName: "Pipeline Duration"},
		{ID: 23, SliceName: "Scraper Volume"},
		{ID: 24, SliceName: "LLM Failures"},
	})
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	var got []int64
	diags = chartIDs.ElementsAs(context.Background(), &got, false)
	if diags.HasError() {
		t.Fatalf("expected flattened chart_ids to decode, got diagnostics: %v", diags)
	}

	want := []int64{24, 19, 22, 21, 23, 20}
	if len(got) != len(want) {
		t.Fatalf("expected %d chart ids, got %#v", len(want), got)
	}

	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("expected chart_ids %v, got %v", want, got)
		}
	}
}

func TestInt64ListValueFromChartsSortsIDs(t *testing.T) {
	t.Parallel()

	chartIDs, diags := int64ListValueFromCharts(context.Background(), []supersetclient.DashboardChart{
		{ID: 22, SliceName: "Revenue"},
		{ID: 11, SliceName: "Orders"},
	})
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	var got []int64
	diags = chartIDs.ElementsAs(context.Background(), &got, false)
	if diags.HasError() {
		t.Fatalf("expected flattened chart_ids to decode, got diagnostics: %v", diags)
	}

	want := []int64{11, 22}
	if len(got) != len(want) {
		t.Fatalf("expected %d chart ids, got %#v", len(want), got)
	}

	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("expected chart_ids %v, got %v", want, got)
		}
	}
}
