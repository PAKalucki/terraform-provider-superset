package provider

import (
	"context"
	"testing"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNeedsDatasetUpdate(t *testing.T) {
	t.Parallel()

	if needsDatasetUpdate(datasetModel{}) {
		t.Fatal("expected empty dataset model to skip update")
	}

	if !needsDatasetUpdate(datasetModel{
		Description: types.StringValue("managed"),
	}) {
		t.Fatal("expected managed description to require update")
	}
}

func TestExpandDatasetUpdateRequestMatchesExistingColumnAndMetricIDs(t *testing.T) {
	t.Parallel()

	columns, columnDiags := types.ListValueFrom(context.Background(), datasetColumnObjectType, []datasetColumnModel{
		{
			ColumnName:  types.StringValue("id"),
			VerboseName: types.StringValue("Event ID"),
		},
	})
	if columnDiags.HasError() {
		t.Fatalf("expected columns list, got diagnostics: %v", columnDiags)
	}

	metrics, metricDiags := types.ListValueFrom(context.Background(), datasetMetricObjectType, []datasetMetricModel{
		{
			MetricName:  types.StringValue("event_count"),
			Expression:  types.StringValue("COUNT(*)"),
			MetricType:  types.StringValue("count"),
			VerboseName: types.StringValue("Event Count"),
		},
	})
	if metricDiags.HasError() {
		t.Fatalf("expected metrics list, got diagnostics: %v", metricDiags)
	}

	request, diags := expandDatasetUpdateRequest(context.Background(), datasetModel{
		DatabaseID: types.Int64Value(12),
		TableName:  types.StringValue("events"),
		Schema:     types.StringValue("analytics"),
		Columns:    columns,
		Metrics:    metrics,
	}, &supersetclient.Dataset{
		Columns: []supersetclient.DatasetColumn{
			{ID: 3, ColumnName: "id"},
		},
		Metrics: []supersetclient.DatasetMetric{
			{ID: 8, MetricName: "event_count"},
		},
	})
	if diags.HasError() {
		t.Fatalf("expected update request, got diagnostics: %v", diags)
	}

	if request.Columns == nil || len(*request.Columns) != 1 || (*request.Columns)[0].ID != 3 {
		t.Fatalf("expected existing column ID to be preserved, got %#v", request.Columns)
	}

	if request.Metrics == nil || len(*request.Metrics) != 1 || (*request.Metrics)[0].ID != 8 {
		t.Fatalf("expected existing metric ID to be preserved, got %#v", request.Metrics)
	}
}

func TestFlattenDatasetResourceModelRefreshesManagedFields(t *testing.T) {
	t.Parallel()

	columns, columnDiags := types.ListValueFrom(context.Background(), datasetColumnObjectType, []datasetColumnModel{
		{
			ColumnName: types.StringValue("id"),
			Filterable: types.BoolValue(true),
		},
	})
	if columnDiags.HasError() {
		t.Fatalf("expected columns list, got diagnostics: %v", columnDiags)
	}

	metrics, metricDiags := types.ListValueFrom(context.Background(), datasetMetricObjectType, []datasetMetricModel{
		{
			MetricName: types.StringValue("event_count"),
			Expression: types.StringValue("COUNT(*)"),
		},
	})
	if metricDiags.HasError() {
		t.Fatalf("expected metrics list, got diagnostics: %v", metricDiags)
	}

	state, diags := flattenDatasetResourceModel(context.Background(), datasetModel{
		DatabaseID:          types.Int64Value(12),
		TableName:           types.StringValue("events"),
		FilterSelectEnabled: types.BoolValue(true),
		Columns:             columns,
		Metrics:             metrics,
	}, &supersetclient.Dataset{
		ID:                  42,
		UUID:                "03b2c25a-86a0-42d8-82fe-8bf726c3bcff",
		TableName:           "events",
		Database:            supersetclient.DatasetDatabase{ID: 12, DatabaseName: "analytics"},
		Description:         "remote description",
		FilterSelectEnabled: boolPtr(true),
		Columns: []supersetclient.DatasetColumn{
			{
				ColumnName:  "id",
				VerboseName: "Remote Event ID",
				Filterable:  boolPtr(true),
			},
		},
		Metrics: []supersetclient.DatasetMetric{
			{
				MetricName:  "event_count",
				Expression:  "COUNT(*)",
				VerboseName: "Remote Event Count",
			},
		},
	})
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	if got := state.Description.ValueString(); got != "remote description" {
		t.Fatalf("expected description to refresh from Superset, got %q", got)
	}

	if !state.FilterSelectEnabled.ValueBool() {
		t.Fatal("expected filter_select_enabled to refresh from Superset")
	}

	var flattenedColumns []datasetColumnModel
	flattenDiags := state.Columns.ElementsAs(context.Background(), &flattenedColumns, false)
	if flattenDiags.HasError() {
		t.Fatalf("expected flattened columns, got diagnostics: %v", flattenDiags)
	}

	if len(flattenedColumns) != 1 {
		t.Fatalf("expected one flattened column, got %d", len(flattenedColumns))
	}

	if got := flattenedColumns[0].VerboseName.ValueString(); got != "Remote Event ID" {
		t.Fatalf("expected column verbose_name to refresh, got %q", got)
	}

	if !flattenedColumns[0].Filterable.ValueBool() {
		t.Fatal("expected column filterable to refresh from Superset")
	}

	var flattenedMetrics []datasetMetricModel
	metricFlattenDiags := state.Metrics.ElementsAs(context.Background(), &flattenedMetrics, false)
	if metricFlattenDiags.HasError() {
		t.Fatalf("expected flattened metrics, got diagnostics: %v", metricFlattenDiags)
	}

	if len(flattenedMetrics) != 1 {
		t.Fatalf("expected one flattened metric, got %d", len(flattenedMetrics))
	}

	if got := flattenedMetrics[0].VerboseName.ValueString(); got != "Remote Event Count" {
		t.Fatalf("expected metric verbose_name to refresh, got %q", got)
	}
}

func TestExpandDatasetUpdateRequestClearsOmittedBoolFieldsAgainstRemoteDefaults(t *testing.T) {
	t.Parallel()

	columns, columnDiags := types.ListValueFrom(context.Background(), datasetColumnObjectType, []datasetColumnModel{
		{
			ColumnName: types.StringValue("duration_s"),
			IsActive:   types.BoolValue(true),
			Type:       types.StringValue("DOUBLE"),
		},
	})
	if columnDiags.HasError() {
		t.Fatalf("expected columns list, got diagnostics: %v", columnDiags)
	}

	request, diags := expandDatasetUpdateRequest(context.Background(), datasetModel{
		DatabaseID: types.Int64Value(12),
		TableName:  types.StringValue("events"),
		Columns:    columns,
	}, &supersetclient.Dataset{
		FilterSelectEnabled: boolPtr(true),
		Columns: []supersetclient.DatasetColumn{
			{
				ID:         7,
				ColumnName: "duration_s",
				Filterable: boolPtr(true),
				Groupby:    boolPtr(true),
				IsActive:   boolPtr(true),
				Type:       "DOUBLE",
			},
		},
	})
	if diags.HasError() {
		t.Fatalf("expected update request, got diagnostics: %v", diags)
	}

	if !request.IncludeFilterSelectEnabled {
		t.Fatal("expected omitted filter_select_enabled to be included for clearing when Superset stores true")
	}

	if request.FilterSelectEnabled == nil || *request.FilterSelectEnabled {
		t.Fatalf("expected omitted filter_select_enabled to clear true default, got %#v", request.FilterSelectEnabled)
	}

	if request.Columns == nil || len(*request.Columns) != 1 {
		t.Fatalf("expected one column update request, got %#v", request.Columns)
	}

	column := (*request.Columns)[0]
	if column.ID != 7 {
		t.Fatalf("expected existing column ID to be preserved, got %d", column.ID)
	}

	if !column.IncludeFilterable || column.Filterable == nil || *column.Filterable {
		t.Fatalf("expected omitted filterable to clear true default, got include=%t value=%#v", column.IncludeFilterable, column.Filterable)
	}

	if !column.IncludeGroupby || column.Groupby == nil || *column.Groupby {
		t.Fatalf("expected omitted groupby to clear true default, got include=%t value=%#v", column.IncludeGroupby, column.Groupby)
	}

	if !column.IncludeIsActive || column.IsActive == nil || !*column.IsActive {
		t.Fatalf("expected managed is_active to remain true, got include=%t value=%#v", column.IncludeIsActive, column.IsActive)
	}
}

func TestFlattenDatasetResourceModelPreservesOmittedBoolFieldsWhenSupersetDefaultsTrue(t *testing.T) {
	t.Parallel()

	columns, columnDiags := types.ListValueFrom(context.Background(), datasetColumnObjectType, []datasetColumnModel{
		{
			ColumnName: types.StringValue("duration_s"),
			IsActive:   types.BoolValue(true),
			Type:       types.StringValue("DOUBLE"),
		},
	})
	if columnDiags.HasError() {
		t.Fatalf("expected columns list, got diagnostics: %v", columnDiags)
	}

	state, diags := flattenDatasetResourceModel(context.Background(), datasetModel{
		DatabaseID:           types.Int64Value(12),
		TableName:            types.StringValue("events"),
		FilterSelectEnabled:  types.BoolNull(),
		AlwaysFilterMainDttm: types.BoolNull(),
		Columns:              columns,
	}, &supersetclient.Dataset{
		ID:                  42,
		UUID:                "03b2c25a-86a0-42d8-82fe-8bf726c3bcff",
		TableName:           "events",
		Database:            supersetclient.DatasetDatabase{ID: 12, DatabaseName: "analytics"},
		FilterSelectEnabled: boolPtr(true),
		Columns: []supersetclient.DatasetColumn{
			{
				ColumnName: "duration_s",
				Filterable: boolPtr(true),
				Groupby:    boolPtr(true),
				IsActive:   boolPtr(true),
				Type:       "DOUBLE",
			},
		},
	})
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	if !state.FilterSelectEnabled.IsNull() {
		t.Fatalf("expected omitted filter_select_enabled to remain null, got %#v", state.FilterSelectEnabled)
	}

	var flattenedColumns []datasetColumnModel
	flattenDiags := state.Columns.ElementsAs(context.Background(), &flattenedColumns, false)
	if flattenDiags.HasError() {
		t.Fatalf("expected flattened columns, got diagnostics: %v", flattenDiags)
	}

	if len(flattenedColumns) != 1 {
		t.Fatalf("expected one flattened column, got %d", len(flattenedColumns))
	}

	if !flattenedColumns[0].Filterable.IsNull() {
		t.Fatalf("expected omitted filterable to remain null, got %#v", flattenedColumns[0].Filterable)
	}

	if !flattenedColumns[0].Groupby.IsNull() {
		t.Fatalf("expected omitted groupby to remain null, got %#v", flattenedColumns[0].Groupby)
	}

	if flattenedColumns[0].IsActive.IsNull() || !flattenedColumns[0].IsActive.ValueBool() {
		t.Fatalf("expected managed is_active to refresh from Superset, got %#v", flattenedColumns[0].IsActive)
	}
}

func TestFlattenDatasetResourceModelPreservesUnmanagedCollections(t *testing.T) {
	t.Parallel()

	state, diags := flattenDatasetResourceModel(context.Background(), datasetModel{
		DatabaseID: types.Int64Value(12),
		TableName:  types.StringValue("events"),
		Columns:    types.ListNull(datasetColumnObjectType),
		Metrics:    types.ListNull(datasetMetricObjectType),
	}, &supersetclient.Dataset{
		ID:        42,
		UUID:      "03b2c25a-86a0-42d8-82fe-8bf726c3bcff",
		TableName: "events",
		Database:  supersetclient.DatasetDatabase{ID: 12, DatabaseName: "analytics"},
		Columns: []supersetclient.DatasetColumn{
			{
				ColumnName: "id",
			},
		},
		Metrics: []supersetclient.DatasetMetric{
			{
				MetricName: "event_count",
				Expression: "COUNT(*)",
			},
		},
	})
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	if !state.Columns.IsNull() {
		t.Fatal("expected unmanaged columns collection to remain null")
	}

	if !state.Metrics.IsNull() {
		t.Fatal("expected unmanaged metrics collection to remain null")
	}
}

func boolPtr(value bool) *bool {
	return &value
}
