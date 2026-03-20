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
		DatabaseID: types.Int64Value(12),
		TableName:  types.StringValue("events"),
		Columns:    columns,
		Metrics:    metrics,
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
