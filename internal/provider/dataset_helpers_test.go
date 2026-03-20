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

func TestFlattenDatasetResourceModelPreservesUnmanagedNestedFields(t *testing.T) {
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

	state, diags := flattenDatasetResourceModel(context.Background(), datasetModel{
		DatabaseID:          types.Int64Value(12),
		TableName:           types.StringValue("events"),
		Description:         types.StringNull(),
		FilterSelectEnabled: types.BoolNull(),
		Columns:             columns,
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
	})
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	if !state.Description.IsNull() {
		t.Fatalf("expected unmanaged description to remain null, got %q", state.Description.ValueString())
	}

	if !state.FilterSelectEnabled.IsNull() {
		t.Fatalf("expected unmanaged filter_select_enabled to remain null, got %t", state.FilterSelectEnabled.ValueBool())
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
		t.Fatalf("expected managed verbose name to refresh, got %q", got)
	}

	if !flattenedColumns[0].Filterable.IsNull() {
		t.Fatalf("expected unmanaged filterable field to remain null, got %t", flattenedColumns[0].Filterable.ValueBool())
	}
}

func boolPtr(value bool) *bool {
	return &value
}
