package provider

import (
	"context"
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
	}, dashboardModel{})
	if !requestDiags.HasError() {
		t.Fatalf("expected mismatched chart ids to fail validation, got request %#v", request)
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

	if !state.ChartIDs.IsNull() {
		t.Fatalf("expected unmanaged chart_ids to remain null, got %#v", state.ChartIDs)
	}
}
