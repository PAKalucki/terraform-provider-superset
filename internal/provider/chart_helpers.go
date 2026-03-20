package provider

import (
	"context"
	"fmt"
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type chartModel struct {
	ID             types.Int64  `tfsdk:"id"`
	UUID           types.String `tfsdk:"uuid"`
	SliceName      types.String `tfsdk:"slice_name"`
	Description    types.String `tfsdk:"description"`
	DatasourceID   types.Int64  `tfsdk:"datasource_id"`
	DatasourceType types.String `tfsdk:"datasource_type"`
	DatasourceName types.String `tfsdk:"datasource_name"`
	VizType        types.String `tfsdk:"viz_type"`
	Params         types.String `tfsdk:"params"`
	QueryContext   types.String `tfsdk:"query_context"`
	CacheTimeout   types.Int64  `tfsdk:"cache_timeout"`
	URL            types.String `tfsdk:"url"`
}

func expandChartCreateRequest(data chartModel) (supersetclient.ChartCreateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	sliceName := strings.TrimSpace(stringValue(data.SliceName))
	if sliceName == "" {
		diags.AddAttributeError(
			path.Root("slice_name"),
			"Missing Chart Name",
			"`slice_name` must be configured.",
		)
	}

	datasourceID := int64Value(data.DatasourceID)
	if datasourceID <= 0 {
		diags.AddAttributeError(
			path.Root("datasource_id"),
			"Missing Chart Datasource",
			"`datasource_id` must be configured with a valid Superset datasource identifier.",
		)
	}

	datasourceType := strings.TrimSpace(stringValue(data.DatasourceType))
	if datasourceType == "" {
		diags.AddAttributeError(
			path.Root("datasource_type"),
			"Missing Chart Datasource Type",
			"`datasource_type` must be configured.",
		)
	}

	vizType := strings.TrimSpace(stringValue(data.VizType))
	if vizType == "" {
		diags.AddAttributeError(
			path.Root("viz_type"),
			"Missing Chart Visualization Type",
			"`viz_type` must be configured.",
		)
	}

	params, paramsDiags := normalizeRequiredJSONString(data.Params, path.Root("params"), "`params` must be configured with a valid JSON string.")
	diags.Append(paramsDiags...)

	queryContext, queryContextDiags := normalizeOptionalJSONString(data.QueryContext, path.Root("query_context"))
	diags.Append(queryContextDiags...)

	if diags.HasError() {
		return supersetclient.ChartCreateRequest{}, diags
	}

	request := supersetclient.ChartCreateRequest{
		SliceName:      sliceName,
		Description:    stringPointerValue(data.Description),
		VizType:        vizType,
		Params:         params.ValueString(),
		QueryContext:   stringPointerValue(queryContext),
		CacheTimeout:   int64PointerValue(data.CacheTimeout),
		DatasourceID:   datasourceID,
		DatasourceType: datasourceType,
	}

	if request.QueryContext == nil {
		value := true
		request.QueryContextGeneration = &value
	}

	return request, diags
}

func expandChartUpdateRequest(data chartModel, current *supersetclient.Chart) (supersetclient.ChartUpdateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	sliceName := strings.TrimSpace(stringValue(data.SliceName))
	if sliceName == "" {
		diags.AddAttributeError(
			path.Root("slice_name"),
			"Missing Chart Name",
			"`slice_name` must be configured.",
		)
	}

	datasourceID := int64Value(data.DatasourceID)
	if datasourceID <= 0 {
		diags.AddAttributeError(
			path.Root("datasource_id"),
			"Missing Chart Datasource",
			"`datasource_id` must be configured with a valid Superset datasource identifier.",
		)
	}

	datasourceType := strings.TrimSpace(stringValue(data.DatasourceType))
	if datasourceType == "" {
		diags.AddAttributeError(
			path.Root("datasource_type"),
			"Missing Chart Datasource Type",
			"`datasource_type` must be configured.",
		)
	}

	vizType := strings.TrimSpace(stringValue(data.VizType))
	if vizType == "" {
		diags.AddAttributeError(
			path.Root("viz_type"),
			"Missing Chart Visualization Type",
			"`viz_type` must be configured.",
		)
	}

	params, paramsDiags := normalizeRequiredJSONString(data.Params, path.Root("params"), "`params` must be configured with a valid JSON string.")
	diags.Append(paramsDiags...)

	queryContext, queryContextDiags := normalizeOptionalJSONString(data.QueryContext, path.Root("query_context"))
	diags.Append(queryContextDiags...)

	if diags.HasError() {
		return supersetclient.ChartUpdateRequest{}, diags
	}

	return supersetclient.ChartUpdateRequest{
		SliceName:           sliceName,
		Description:         stringPointerValue(data.Description),
		VizType:             vizType,
		Params:              params.ValueString(),
		QueryContext:        stringPointerValue(queryContext),
		CacheTimeout:        int64PointerValue(data.CacheTimeout),
		DatasourceID:        datasourceID,
		DatasourceType:      datasourceType,
		IncludeDescription:  includeManagedString(data.Description, current.Description),
		IncludeQueryContext: includeManagedString(data.QueryContext, stringPointerValueOrEmpty(current.QueryContext)),
		IncludeCacheTimeout: includeManagedInt64(data.CacheTimeout, current.CacheTimeout),
	}, diags
}

func flattenChartModel(current chartModel, remote *supersetclient.Chart) (chartModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	params, paramsDiags := normalizeOptionalJSONString(types.StringValue(remote.Params), path.Root("params"))
	diags.Append(paramsDiags...)

	var queryContext types.String
	if remote.QueryContext == nil {
		queryContext = types.StringNull()
	} else {
		normalizedQueryContext, queryContextDiags := normalizeOptionalJSONString(types.StringValue(*remote.QueryContext), path.Root("query_context"))
		diags.Append(queryContextDiags...)
		queryContext = normalizedQueryContext
	}

	state := current
	state.ID = types.Int64Value(remote.ID)
	state.UUID = stringTypeValue(remote.UUID)
	state.SliceName = stringTypeValue(remote.SliceName)
	state.Description = stringTypeValue(remote.Description)
	state.DatasourceID = types.Int64Value(remote.DatasourceID)
	state.DatasourceType = stringTypeValue(remote.DatasourceType)
	state.DatasourceName = stringTypeValue(remote.DatasourceNameText)
	state.VizType = stringTypeValue(remote.VizType)
	state.Params = params
	state.QueryContext = queryContext
	state.CacheTimeout = int64TypeValue(remote.CacheTimeout)
	state.URL = stringTypeValue(remote.URL)

	return state, diags
}

func findChart(ctx context.Context, client *supersetclient.Client, datasourceID int64, datasourceType string, sliceName string) (*supersetclient.Chart, error) {
	charts, err := client.ListCharts(ctx, 1000)
	if err != nil {
		return nil, err
	}

	normalizedDatasourceType := strings.TrimSpace(datasourceType)
	normalizedSliceName := strings.TrimSpace(sliceName)
	requireDatasourceType := normalizedDatasourceType != ""

	var matches []supersetclient.Chart

	for _, chart := range charts {
		if chart.DatasourceID != datasourceID {
			continue
		}

		if strings.TrimSpace(chart.SliceName) != normalizedSliceName {
			continue
		}

		if requireDatasourceType && strings.TrimSpace(chart.DatasourceType) != normalizedDatasourceType {
			continue
		}

		matches = append(matches, chart)
	}

	switch len(matches) {
	case 0:
		if !requireDatasourceType {
			return nil, fmt.Errorf("chart %q on datasource %d was not found", normalizedSliceName, datasourceID)
		}

		return nil, fmt.Errorf("chart %q on datasource %d with type %q was not found", normalizedSliceName, datasourceID, normalizedDatasourceType)
	case 1:
		return client.GetChart(ctx, matches[0].ID)
	default:
		if !requireDatasourceType {
			return nil, fmt.Errorf("chart %q on datasource %d matched %d charts; configure `datasource_type` to disambiguate the lookup", normalizedSliceName, datasourceID, len(matches))
		}

		return nil, fmt.Errorf("chart %q on datasource %d with type %q matched %d charts", normalizedSliceName, datasourceID, normalizedDatasourceType, len(matches))
	}
}

func normalizeRequiredJSONString(value types.String, attributePath path.Path, missingMessage string) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if value.IsNull() || value.IsUnknown() || strings.TrimSpace(value.ValueString()) == "" {
		diags.AddAttributeError(
			attributePath,
			"Missing JSON String",
			missingMessage,
		)

		return types.StringNull(), diags
	}

	return normalizeOptionalJSONString(value, attributePath)
}

func stringPointerValueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
