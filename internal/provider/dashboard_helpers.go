package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type dashboardModel struct {
	ID                        types.Int64  `tfsdk:"id"`
	UUID                      types.String `tfsdk:"uuid"`
	DashboardTitle            types.String `tfsdk:"dashboard_title"`
	Slug                      types.String `tfsdk:"slug"`
	CSS                       types.String `tfsdk:"css"`
	Published                 types.Bool   `tfsdk:"published"`
	ShowNativeFilters         types.Bool   `tfsdk:"show_native_filters"`
	ChartIDs                  types.List   `tfsdk:"chart_ids"`
	PositionJSON              types.String `tfsdk:"position_json"`
	NativeFilterConfiguration types.String `tfsdk:"native_filter_configuration"`
	URL                       types.String `tfsdk:"url"`
}

type dashboardPositionChart struct {
	ID        int64
	SliceName string
	UUID      string
}

func expandDashboardCreateRequest(data dashboardModel) (supersetclient.DashboardCreateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	dashboardTitle := strings.TrimSpace(stringValue(data.DashboardTitle))
	if dashboardTitle == "" {
		diags.AddAttributeError(
			path.Root("dashboard_title"),
			"Missing Dashboard Title",
			"`dashboard_title` must be configured.",
		)
	}

	if diags.HasError() {
		return supersetclient.DashboardCreateRequest{}, diags
	}

	return supersetclient.DashboardCreateRequest{
		DashboardTitle: dashboardTitle,
		Slug:           stringPointerValue(data.Slug),
		CSS:            stringPointerValue(data.CSS),
		Published:      boolPointerValue(data.Published),
	}, diags
}

func expandDashboardUpdateRequest(ctx context.Context, client *supersetclient.Client, plan dashboardModel, current dashboardModel, remote *supersetclient.Dashboard) (supersetclient.DashboardUpdateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	dashboardTitle := strings.TrimSpace(stringValue(plan.DashboardTitle))
	if dashboardTitle == "" {
		diags.AddAttributeError(
			path.Root("dashboard_title"),
			"Missing Dashboard Title",
			"`dashboard_title` must be configured.",
		)
	}

	chartIDs, chartIDDiags := dashboardChartIDsFromList(ctx, plan.ChartIDs)
	diags.Append(chartIDDiags...)

	nativeFilterConfiguration, nativeFilterDiags := normalizeDashboardNativeFilterConfiguration(plan.NativeFilterConfiguration)
	diags.Append(nativeFilterDiags...)

	var (
		normalizedPositionJSON types.String
		positionJSON           types.String
		positionChartIDs       []int64
		includeDashboardLayout bool
	)

	if !plan.PositionJSON.IsNull() && !plan.PositionJSON.IsUnknown() {
		var positionDiags diag.Diagnostics
		positionJSON, positionDiags = normalizeOptionalJSONString(plan.PositionJSON, path.Root("position_json"))
		diags.Append(positionDiags...)
		positionChartIDs, chartIDDiags = extractDashboardChartIDs(positionJSON.ValueString())
		diags.Append(chartIDDiags...)
	}

	if diags.HasError() {
		return supersetclient.DashboardUpdateRequest{}, diags
	}

	planChartsManaged := !plan.ChartIDs.IsNull() && !plan.ChartIDs.IsUnknown()
	currentChartsManaged := !current.ChartIDs.IsNull() && !current.ChartIDs.IsUnknown()
	planPositionManaged := !plan.PositionJSON.IsNull() && !plan.PositionJSON.IsUnknown()
	currentPositionManaged := !current.PositionJSON.IsNull() && !current.PositionJSON.IsUnknown()
	planNativeFiltersManaged := !plan.NativeFilterConfiguration.IsNull() && !plan.NativeFilterConfiguration.IsUnknown()
	currentNativeFiltersManaged := !current.NativeFilterConfiguration.IsNull() && !current.NativeFilterConfiguration.IsUnknown()
	planShowNativeFiltersManaged := !plan.ShowNativeFilters.IsNull() && !plan.ShowNativeFilters.IsUnknown()
	currentShowNativeFiltersManaged := !current.ShowNativeFilters.IsNull() && !current.ShowNativeFilters.IsUnknown()

	if planPositionManaged && planChartsManaged && !equalInt64Slices(chartIDs, positionChartIDs) {
		diags.AddAttributeError(
			path.Root("chart_ids"),
			"Mismatched Dashboard Chart Associations",
			"`chart_ids` must match the chart identifiers referenced in `position_json`.",
		)

		return supersetclient.DashboardUpdateRequest{}, diags
	}

	includeDashboardLayout = planChartsManaged || currentChartsManaged || planPositionManaged || currentPositionManaged
	includeDashboardMetadata := includeDashboardLayout || planNativeFiltersManaged || currentNativeFiltersManaged || planShowNativeFiltersManaged || currentShowNativeFiltersManaged

	request := supersetclient.DashboardUpdateRequest{
		DashboardTitle: dashboardTitle,
		Slug:           stringPointerValue(plan.Slug),
		CSS:            stringPointerValue(plan.CSS),
		Published:      boolPointerValue(plan.Published),
		IncludeSlug:    !plan.Slug.IsNull() && !plan.Slug.IsUnknown() || !current.Slug.IsNull() && !current.Slug.IsUnknown(),
		IncludeCSS:     !plan.CSS.IsNull() && !plan.CSS.IsUnknown() || !current.CSS.IsNull() && !current.CSS.IsUnknown(),
		IncludePublished: !plan.Published.IsNull() && !plan.Published.IsUnknown() ||
			!current.Published.IsNull() && !current.Published.IsUnknown(),
	}

	if !includeDashboardMetadata {
		return request, diags
	}

	if includeDashboardLayout {
		layoutPositionJSON, layoutDiags := resolveDashboardPositionJSON(ctx, client, dashboardTitle, chartIDs, positionJSON, planChartsManaged, planPositionManaged, currentChartsManaged || currentPositionManaged)
		diags.Append(layoutDiags...)
		if diags.HasError() {
			return supersetclient.DashboardUpdateRequest{}, diags
		}

		normalizedPositionJSON = layoutPositionJSON
		request.PositionJSON = stringPointerValue(normalizedPositionJSON)
		request.IncludePositionJSON = true
	}

	remoteMetadata, remoteMetadataDiags := dashboardMetadataMapFromRemote(remote)
	diags.Append(remoteMetadataDiags...)

	jsonMetadata, metadataDiags := buildDashboardJSONMetadata(
		remoteMetadata,
		normalizedPositionJSON,
		nativeFilterConfiguration,
		plan,
		current,
		includeDashboardLayout,
	)
	diags.Append(metadataDiags...)

	request.JSONMetadata = stringPointerValue(jsonMetadata)
	request.IncludeJSONMetadata = true

	return request, diags
}

func flattenDashboardResourceModel(ctx context.Context, current dashboardModel, remote *supersetclient.Dashboard, remoteCharts []supersetclient.DashboardChart) (dashboardModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	metadata, metadataDiags := dashboardMetadataMapFromRemote(remote)
	diags.Append(metadataDiags...)

	showNativeFilters, showNativeFiltersDiags := dashboardShowNativeFiltersResourceValue(current.ShowNativeFilters, metadata)
	diags.Append(showNativeFiltersDiags...)

	nativeFilterConfiguration, nativeFilterConfigurationDiags := dashboardNativeFilterConfigurationResourceValue(current.NativeFilterConfiguration, metadata)
	diags.Append(nativeFilterConfigurationDiags...)

	state := dashboardModel{
		ID:                        types.Int64Value(remote.ID),
		UUID:                      stringTypeValue(remote.UUID),
		DashboardTitle:            stringTypeValue(remote.DashboardTitle),
		Slug:                      managedStringValue(current.Slug, remote.Slug),
		CSS:                       managedStringValue(current.CSS, remote.CSS),
		Published:                 managedDashboardBoolValue(current.Published, remote.Published),
		ShowNativeFilters:         showNativeFilters,
		PositionJSON:              managedDashboardPositionValue(current.PositionJSON, remote.PositionJSON),
		NativeFilterConfiguration: nativeFilterConfiguration,
		URL:                       stringTypeValue(remote.URL),
	}

	chartIDs, chartIDDiags := flattenDashboardChartIDs(ctx, current.ChartIDs, remoteCharts)
	diags.Append(chartIDDiags...)
	state.ChartIDs = chartIDs

	return state, diags
}

func flattenDashboardDataSourceModel(ctx context.Context, remote *supersetclient.Dashboard, remoteCharts []supersetclient.DashboardChart) (dashboardModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	positionJSON, positionDiags := normalizeOptionalJSONString(stringTypeValue(remote.PositionJSON), path.Root("position_json"))
	diags.Append(positionDiags...)

	chartIDs, chartIDDiags := int64ListValueFromCharts(ctx, remoteCharts)
	diags.Append(chartIDDiags...)

	metadata, metadataDiags := dashboardMetadataMapFromRemote(remote)
	diags.Append(metadataDiags...)

	showNativeFilters, showNativeFiltersDiags := dashboardShowNativeFiltersDataSourceValue(metadata)
	diags.Append(showNativeFiltersDiags...)

	nativeFilterConfiguration, nativeFilterConfigurationDiags := dashboardNativeFilterConfigurationDataSourceValue(metadata)
	diags.Append(nativeFilterConfigurationDiags...)

	return dashboardModel{
		ID:                        types.Int64Value(remote.ID),
		UUID:                      stringTypeValue(remote.UUID),
		DashboardTitle:            stringTypeValue(remote.DashboardTitle),
		Slug:                      stringTypeValue(remote.Slug),
		CSS:                       stringTypeValue(remote.CSS),
		Published:                 boolTypeValue(remote.Published),
		ShowNativeFilters:         showNativeFilters,
		ChartIDs:                  chartIDs,
		PositionJSON:              positionJSON,
		NativeFilterConfiguration: nativeFilterConfiguration,
		URL:                       stringTypeValue(remote.URL),
	}, diags
}

func findDashboardByTitle(ctx context.Context, client *supersetclient.Client, dashboardTitle string) (*supersetclient.Dashboard, error) {
	dashboards, err := client.ListDashboards(ctx, 1000)
	if err != nil {
		return nil, err
	}

	normalizedTitle := strings.TrimSpace(dashboardTitle)
	var matches []supersetclient.Dashboard

	for _, dashboard := range dashboards {
		if strings.TrimSpace(dashboard.DashboardTitle) == normalizedTitle {
			matches = append(matches, dashboard)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("dashboard %q was not found", normalizedTitle)
	case 1:
		return client.GetDashboard(ctx, strconv.FormatInt(matches[0].ID, 10))
	default:
		return nil, fmt.Errorf("dashboard %q matched %d dashboards; configure `id` or `slug` to disambiguate the lookup", normalizedTitle, len(matches))
	}
}

func dashboardChartIDsFromList(ctx context.Context, value types.List) ([]int64, diag.Diagnostics) {
	var chartIDs []int64

	if value.IsNull() || value.IsUnknown() {
		return chartIDs, nil
	}

	diags := value.ElementsAs(ctx, &chartIDs, false)
	if diags.HasError() {
		return nil, diags
	}

	seen := make(map[int64]struct{}, len(chartIDs))
	for index, chartID := range chartIDs {
		if chartID <= 0 {
			diags.AddAttributeError(
				path.Root("chart_ids").AtListIndex(index),
				"Invalid Dashboard Chart Identifier",
				"Each `chart_ids` value must be a valid Superset chart identifier.",
			)
			continue
		}

		if _, ok := seen[chartID]; ok {
			diags.AddAttributeError(
				path.Root("chart_ids").AtListIndex(index),
				"Duplicate Dashboard Chart Identifier",
				fmt.Sprintf("Chart %d appears more than once in `chart_ids`.", chartID),
			)
			continue
		}

		seen[chartID] = struct{}{}
	}

	return chartIDs, diags
}

func resolveDashboardPositionJSON(ctx context.Context, client *supersetclient.Client, dashboardTitle string, chartIDs []int64, positionJSON types.String, planChartsManaged bool, planPositionManaged bool, clearLayout bool) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch {
	case planPositionManaged:
		return positionJSON, diags
	case planChartsManaged:
		charts, err := loadDashboardPositionCharts(ctx, client, chartIDs)
		if err != nil {
			diags.AddError(
				"Unable to Read Superset Charts For Dashboard Layout",
				err.Error(),
			)

			return types.StringNull(), diags
		}

		normalized, err := buildDashboardPositionJSON(dashboardTitle, charts)
		if err != nil {
			diags.AddError(
				"Unable to Build Dashboard Layout",
				err.Error(),
			)

			return types.StringNull(), diags
		}

		return types.StringValue(normalized), diags
	case clearLayout:
		normalized, err := buildDashboardPositionJSON(dashboardTitle, nil)
		if err != nil {
			diags.AddError(
				"Unable to Build Dashboard Layout",
				err.Error(),
			)

			return types.StringNull(), diags
		}

		return types.StringValue(normalized), diags
	default:
		return types.StringNull(), diags
	}
}

func loadDashboardPositionCharts(ctx context.Context, client *supersetclient.Client, chartIDs []int64) ([]dashboardPositionChart, error) {
	charts := make([]dashboardPositionChart, 0, len(chartIDs))

	for _, chartID := range chartIDs {
		chart, err := client.GetChart(ctx, chartID)
		if err != nil {
			return nil, fmt.Errorf("read chart %d: %w", chartID, err)
		}

		charts = append(charts, dashboardPositionChart{
			ID:        chart.ID,
			SliceName: chart.SliceName,
			UUID:      chart.UUID,
		})
	}

	return charts, nil
}

func buildDashboardPositionJSON(dashboardTitle string, charts []dashboardPositionChart) (string, error) {
	position := map[string]any{
		"DASHBOARD_VERSION_KEY": "v2",
		"ROOT_ID": map[string]any{
			"children": []string{"GRID_ID"},
			"id":       "ROOT_ID",
			"type":     "ROOT",
		},
		"GRID_ID": map[string]any{
			"children": []string{},
			"id":       "GRID_ID",
			"parents":  []string{"ROOT_ID"},
			"type":     "GRID",
		},
		"HEADER_ID": map[string]any{
			"id":   "HEADER_ID",
			"meta": map[string]any{"text": dashboardTitle},
			"type": "HEADER",
		},
	}

	if len(charts) > 0 {
		rowID := "ROW-1"
		rowChildren := make([]string, 0, len(charts))

		grid, ok := position["GRID_ID"].(map[string]any)
		if !ok {
			return "", fmt.Errorf("dashboard grid layout is invalid")
		}

		grid["children"] = []string{rowID}
		position[rowID] = map[string]any{
			"children": rowChildren,
			"id":       rowID,
			"meta": map[string]any{
				"0":          "ROOT_ID",
				"background": "BACKGROUND_TRANSPARENT",
			},
			"parents": []string{"ROOT_ID", "GRID_ID"},
			"type":    "ROW",
		}

		for index, chart := range charts {
			chartNodeID := fmt.Sprintf("CHART-%d", index+1)
			rowChildren = append(rowChildren, chartNodeID)
			position[chartNodeID] = map[string]any{
				"children": []string{},
				"id":       chartNodeID,
				"meta": map[string]any{
					"chartId":   chart.ID,
					"height":    50,
					"sliceName": chart.SliceName,
					"uuid":      chart.UUID,
					"width":     4,
				},
				"parents": []string{"ROOT_ID", "GRID_ID", rowID},
				"type":    "CHART",
			}
		}

		row, ok := position[rowID].(map[string]any)
		if !ok {
			return "", fmt.Errorf("dashboard row layout is invalid")
		}

		row["children"] = rowChildren
	}

	normalized, err := json.Marshal(position)
	if err != nil {
		return "", fmt.Errorf("normalize dashboard position JSON: %w", err)
	}

	return string(normalized), nil
}

func buildDashboardJSONMetadata(remoteMetadata map[string]any, positionJSON types.String, nativeFilterConfiguration types.String, plan dashboardModel, current dashboardModel, includeDashboardLayout bool) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	metadata := cloneDashboardMetadata(remoteMetadata)

	if includeDashboardLayout {
		positions, positionsDiags := decodeDashboardPositionMetadata(positionJSON)
		diags.Append(positionsDiags...)
		if diags.HasError() {
			return types.StringNull(), diags
		}

		if positions == nil {
			delete(metadata, "positions")
		} else {
			metadata["positions"] = positions
		}
	}

	manageNativeFilters := !plan.NativeFilterConfiguration.IsNull() && !plan.NativeFilterConfiguration.IsUnknown() ||
		!current.NativeFilterConfiguration.IsNull() && !current.NativeFilterConfiguration.IsUnknown()
	if manageNativeFilters {
		nativeFilters, nativeFiltersDiags := decodeDashboardNativeFilterConfiguration(nativeFilterConfiguration)
		diags.Append(nativeFiltersDiags...)
		if diags.HasError() {
			return types.StringNull(), diags
		}

		if nativeFilters == nil {
			delete(metadata, "native_filter_configuration")
		} else {
			metadata["native_filter_configuration"] = nativeFilters
		}
	}

	if showNativeFilters, includeShowNativeFilters := resolveDashboardShowNativeFilters(plan.ShowNativeFilters, current.ShowNativeFilters, nativeFilterConfiguration, plan.NativeFilterConfiguration, current.NativeFilterConfiguration); includeShowNativeFilters {
		if showNativeFilters == nil {
			delete(metadata, "show_native_filters")
		} else {
			metadata["show_native_filters"] = *showNativeFilters
		}
	}

	normalized, err := json.Marshal(metadata)
	if err != nil {
		diags.AddError("Invalid Dashboard Metadata", fmt.Sprintf("Unable to build dashboard metadata payload: %v", err))

		return types.StringNull(), diags
	}

	return types.StringValue(string(normalized)), diags
}

func extractDashboardChartIDs(positionJSON string) ([]int64, diag.Diagnostics) {
	var diags diag.Diagnostics

	if strings.TrimSpace(positionJSON) == "" {
		return nil, diags
	}

	var position map[string]any
	if err := json.Unmarshal([]byte(positionJSON), &position); err != nil {
		diags.AddAttributeError(
			path.Root("position_json"),
			"Invalid Dashboard Layout JSON",
			fmt.Sprintf("`position_json` must be valid JSON: %v", err),
		)

		return nil, diags
	}

	chartIDs := make([]int64, 0)
	seen := make(map[int64]struct{})

	for nodeID, rawNode := range position {
		node, ok := rawNode.(map[string]any)
		if !ok {
			continue
		}

		if strings.TrimSpace(stringFromAny(node["type"])) != "CHART" {
			continue
		}

		meta, ok := node["meta"].(map[string]any)
		if !ok {
			diags.AddAttributeError(
				path.Root("position_json"),
				"Invalid Dashboard Chart Layout Node",
				fmt.Sprintf("Chart node %q is missing a `meta` object.", nodeID),
			)
			continue
		}

		chartID, ok := int64FromAny(meta["chartId"])
		if !ok || chartID <= 0 {
			diags.AddAttributeError(
				path.Root("position_json"),
				"Invalid Dashboard Chart Layout Node",
				fmt.Sprintf("Chart node %q is missing a valid `meta.chartId` value.", nodeID),
			)
			continue
		}

		if _, exists := seen[chartID]; exists {
			diags.AddAttributeError(
				path.Root("position_json"),
				"Duplicate Dashboard Chart Layout Node",
				fmt.Sprintf("Chart %d appears more than once in `position_json`.", chartID),
			)
			continue
		}

		seen[chartID] = struct{}{}
		chartIDs = append(chartIDs, chartID)
	}

	sort.Slice(chartIDs, func(i, j int) bool { return chartIDs[i] < chartIDs[j] })

	return chartIDs, diags
}

func flattenDashboardChartIDs(ctx context.Context, current types.List, remoteCharts []supersetclient.DashboardChart) (types.List, diag.Diagnostics) {
	if current.IsNull() || current.IsUnknown() {
		return types.ListNull(types.Int64Type), nil
	}

	currentChartIDs, diags := dashboardChartIDsFromList(ctx, current)
	if diags.HasError() {
		return types.ListNull(types.Int64Type), diags
	}

	return types.ListValueFrom(ctx, types.Int64Type, orderedDashboardChartIDs(remoteCharts, currentChartIDs))
}

func int64ListValueFromCharts(ctx context.Context, remoteCharts []supersetclient.DashboardChart) (types.List, diag.Diagnostics) {
	return types.ListValueFrom(ctx, types.Int64Type, orderedDashboardChartIDs(remoteCharts, nil))
}

func orderedDashboardChartIDs(remoteCharts []supersetclient.DashboardChart, preferredOrder []int64) []int64 {
	remoteChartIDs := make([]int64, 0, len(remoteCharts))
	seen := make(map[int64]struct{}, len(remoteCharts))

	for _, chart := range remoteCharts {
		if _, ok := seen[chart.ID]; ok {
			continue
		}

		seen[chart.ID] = struct{}{}
		remoteChartIDs = append(remoteChartIDs, chart.ID)
	}

	if len(preferredOrder) == 0 {
		sort.Slice(remoteChartIDs, func(i, j int) bool { return remoteChartIDs[i] < remoteChartIDs[j] })
		return remoteChartIDs
	}

	ordered := make([]int64, 0, len(remoteChartIDs))
	used := make(map[int64]struct{}, len(remoteChartIDs))

	for _, chartID := range preferredOrder {
		if _, ok := seen[chartID]; !ok {
			continue
		}

		if _, ok := used[chartID]; ok {
			continue
		}

		ordered = append(ordered, chartID)
		used[chartID] = struct{}{}
	}

	extraChartIDs := make([]int64, 0, len(remoteChartIDs)-len(ordered))
	for _, chartID := range remoteChartIDs {
		if _, ok := used[chartID]; ok {
			continue
		}

		extraChartIDs = append(extraChartIDs, chartID)
	}

	sort.Slice(extraChartIDs, func(i, j int) bool { return extraChartIDs[i] < extraChartIDs[j] })

	return append(ordered, extraChartIDs...)
}

func managedStringValue(current types.String, remote string) types.String {
	if current.IsNull() || current.IsUnknown() {
		return types.StringNull()
	}

	return stringTypeValue(remote)
}

func managedDashboardBoolValue(current types.Bool, remote *bool) types.Bool {
	if current.IsNull() || current.IsUnknown() {
		return types.BoolNull()
	}

	return boolTypeValue(remote)
}

func managedDashboardPositionValue(current types.String, remote string) types.String {
	if current.IsNull() || current.IsUnknown() {
		return types.StringNull()
	}

	normalized, diags := normalizeOptionalJSONString(types.StringValue(remote), path.Root("position_json"))
	if diags.HasError() {
		return types.StringNull()
	}

	return normalized
}

func normalizeDashboardNativeFilterConfiguration(value types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if value.IsNull() || value.IsUnknown() {
		return value, diags
	}

	normalized, normalizeDiags := normalizeOptionalJSONString(value, path.Root("native_filter_configuration"))
	diags.Append(normalizeDiags...)
	if diags.HasError() || normalized.IsNull() || normalized.IsUnknown() {
		return normalized, diags
	}

	var filters []any
	if err := json.Unmarshal([]byte(normalized.ValueString()), &filters); err != nil {
		diags.AddAttributeError(
			path.Root("native_filter_configuration"),
			"Invalid Native Filter Configuration JSON",
			fmt.Sprintf("`native_filter_configuration` must be a valid JSON array: %v", err),
		)

		return types.StringNull(), diags
	}

	return normalized, diags
}

func dashboardMetadataMapFromRemote(remote *supersetclient.Dashboard) (map[string]any, diag.Diagnostics) {
	if remote == nil || strings.TrimSpace(remote.JSONMetadata) == "" {
		return map[string]any{}, nil
	}

	var metadata map[string]any
	if err := json.Unmarshal([]byte(remote.JSONMetadata), &metadata); err != nil {
		var diags diag.Diagnostics
		diags.AddError("Invalid Dashboard Metadata", fmt.Sprintf("Unable to decode Superset dashboard metadata JSON: %v", err))

		return map[string]any{}, diags
	}

	if metadata == nil {
		return map[string]any{}, nil
	}

	return metadata, nil
}

func decodeDashboardPositionMetadata(positionJSON types.String) (any, diag.Diagnostics) {
	var diags diag.Diagnostics

	if positionJSON.IsNull() || positionJSON.IsUnknown() {
		return nil, diags
	}

	var positions any
	if err := json.Unmarshal([]byte(positionJSON.ValueString()), &positions); err != nil {
		diags.AddAttributeError(
			path.Root("position_json"),
			"Invalid Dashboard Layout JSON",
			fmt.Sprintf("Unable to decode normalized dashboard layout JSON: %v", err),
		)

		return nil, diags
	}

	return positions, diags
}

func decodeDashboardNativeFilterConfiguration(value types.String) (any, diag.Diagnostics) {
	var diags diag.Diagnostics

	if value.IsNull() || value.IsUnknown() {
		return nil, diags
	}

	var filters any
	if err := json.Unmarshal([]byte(value.ValueString()), &filters); err != nil {
		diags.AddAttributeError(
			path.Root("native_filter_configuration"),
			"Invalid Native Filter Configuration JSON",
			fmt.Sprintf("Unable to decode normalized native filter configuration JSON: %v", err),
		)

		return nil, diags
	}

	return filters, diags
}

func resolveDashboardShowNativeFilters(planShowNativeFilters types.Bool, currentShowNativeFilters types.Bool, normalizedNativeFilterConfiguration types.String, planNativeFilterConfiguration types.String, currentNativeFilterConfiguration types.String) (*bool, bool) {
	if !planShowNativeFilters.IsNull() && !planShowNativeFilters.IsUnknown() {
		value := planShowNativeFilters.ValueBool()

		return &value, true
	}

	if !normalizedNativeFilterConfiguration.IsNull() && !normalizedNativeFilterConfiguration.IsUnknown() {
		value := true

		return &value, true
	}

	if (!planNativeFilterConfiguration.IsNull() && !planNativeFilterConfiguration.IsUnknown()) ||
		(!currentNativeFilterConfiguration.IsNull() && !currentNativeFilterConfiguration.IsUnknown()) ||
		(!currentShowNativeFilters.IsNull() && !currentShowNativeFilters.IsUnknown()) {
		return nil, true
	}

	return nil, false
}

func cloneDashboardMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}

	clone := make(map[string]any, len(metadata))
	for key, value := range metadata {
		clone[key] = value
	}

	return clone
}

func dashboardShowNativeFiltersResourceValue(current types.Bool, metadata map[string]any) (types.Bool, diag.Diagnostics) {
	if current.IsNull() || current.IsUnknown() {
		return types.BoolNull(), nil
	}

	return dashboardShowNativeFiltersDataSourceValue(metadata)
}

func dashboardShowNativeFiltersDataSourceValue(metadata map[string]any) (types.Bool, diag.Diagnostics) {
	value, ok := metadata["show_native_filters"]
	if !ok || value == nil {
		return types.BoolNull(), nil
	}

	showNativeFilters, ok := value.(bool)
	if !ok {
		var diags diag.Diagnostics
		diags.AddError("Invalid Dashboard Metadata", fmt.Sprintf("Expected `show_native_filters` to be a boolean, got %T.", value))

		return types.BoolNull(), diags
	}

	return types.BoolValue(showNativeFilters), nil
}

func dashboardNativeFilterConfigurationResourceValue(current types.String, metadata map[string]any) (types.String, diag.Diagnostics) {
	if current.IsNull() || current.IsUnknown() {
		return types.StringNull(), nil
	}

	return dashboardNativeFilterConfigurationDataSourceValue(metadata)
}

func dashboardNativeFilterConfigurationDataSourceValue(metadata map[string]any) (types.String, diag.Diagnostics) {
	value, ok := metadata["native_filter_configuration"]
	if !ok || value == nil {
		return types.StringNull(), nil
	}

	normalized, err := json.Marshal(value)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Invalid Dashboard Metadata", fmt.Sprintf("Unable to normalize `native_filter_configuration`: %v", err))

		return types.StringNull(), diags
	}

	return types.StringValue(string(normalized)), nil
}

func equalInt64Slices(left []int64, right []int64) bool {
	if len(left) != len(right) {
		return false
	}

	leftCopy := append([]int64(nil), left...)
	rightCopy := append([]int64(nil), right...)

	sort.Slice(leftCopy, func(i, j int) bool { return leftCopy[i] < leftCopy[j] })
	sort.Slice(rightCopy, func(i, j int) bool { return rightCopy[i] < rightCopy[j] })

	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
	}

	return true
}

func stringFromAny(value any) string {
	text, _ := value.(string)
	return text
}

func int64FromAny(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), true
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case json.Number:
		number, err := typed.Int64()
		if err != nil {
			return 0, false
		}

		return number, true
	default:
		return 0, false
	}
}
