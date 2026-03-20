package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var datasetColumnObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"column_name":        types.StringType,
		"verbose_name":       types.StringType,
		"description":        types.StringType,
		"expression":         types.StringType,
		"filterable":         types.BoolType,
		"groupby":            types.BoolType,
		"is_active":          types.BoolType,
		"is_dttm":            types.BoolType,
		"type":               types.StringType,
		"python_date_format": types.StringType,
	},
}

var datasetMetricObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"metric_name":  types.StringType,
		"expression":   types.StringType,
		"metric_type":  types.StringType,
		"verbose_name": types.StringType,
		"description":  types.StringType,
		"d3format":     types.StringType,
		"warning_text": types.StringType,
	},
}

type datasetModel struct {
	ID                   types.Int64  `tfsdk:"id"`
	UUID                 types.String `tfsdk:"uuid"`
	DatabaseID           types.Int64  `tfsdk:"database_id"`
	DatabaseName         types.String `tfsdk:"database_name"`
	TableName            types.String `tfsdk:"table_name"`
	Schema               types.String `tfsdk:"schema"`
	Description          types.String `tfsdk:"description"`
	MainDttmCol          types.String `tfsdk:"main_dttm_col"`
	FilterSelectEnabled  types.Bool   `tfsdk:"filter_select_enabled"`
	NormalizeColumns     types.Bool   `tfsdk:"normalize_columns"`
	AlwaysFilterMainDttm types.Bool   `tfsdk:"always_filter_main_dttm"`
	CacheTimeout         types.Int64  `tfsdk:"cache_timeout"`
	Columns              types.List   `tfsdk:"columns"`
	Metrics              types.List   `tfsdk:"metrics"`
}

type datasetColumnModel struct {
	ColumnName       types.String `tfsdk:"column_name"`
	VerboseName      types.String `tfsdk:"verbose_name"`
	Description      types.String `tfsdk:"description"`
	Expression       types.String `tfsdk:"expression"`
	Filterable       types.Bool   `tfsdk:"filterable"`
	Groupby          types.Bool   `tfsdk:"groupby"`
	IsActive         types.Bool   `tfsdk:"is_active"`
	IsDttm           types.Bool   `tfsdk:"is_dttm"`
	Type             types.String `tfsdk:"type"`
	PythonDateFormat types.String `tfsdk:"python_date_format"`
}

type datasetMetricModel struct {
	MetricName  types.String `tfsdk:"metric_name"`
	Expression  types.String `tfsdk:"expression"`
	MetricType  types.String `tfsdk:"metric_type"`
	VerboseName types.String `tfsdk:"verbose_name"`
	Description types.String `tfsdk:"description"`
	D3Format    types.String `tfsdk:"d3format"`
	WarningText types.String `tfsdk:"warning_text"`
}

func expandDatasetCreateRequest(data datasetModel) (supersetclient.DatasetCreateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	databaseID := int64Value(data.DatabaseID)
	if databaseID <= 0 {
		diags.AddAttributeError(
			path.Root("database_id"),
			"Missing Dataset Database",
			"`database_id` must be configured with a valid Superset database identifier.",
		)
	}

	tableName := strings.TrimSpace(stringValue(data.TableName))
	if tableName == "" {
		diags.AddAttributeError(
			path.Root("table_name"),
			"Missing Dataset Table Name",
			"`table_name` must be configured.",
		)
	}

	if diags.HasError() {
		return supersetclient.DatasetCreateRequest{}, diags
	}

	return supersetclient.DatasetCreateRequest{
		Database:  databaseID,
		TableName: tableName,
		Schema:    stringPointerValue(data.Schema),
	}, diags
}

func expandDatasetUpdateRequest(ctx context.Context, data datasetModel, current *supersetclient.Dataset) (supersetclient.DatasetUpdateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	databaseID := int64Value(data.DatabaseID)
	if databaseID <= 0 {
		diags.AddAttributeError(
			path.Root("database_id"),
			"Missing Dataset Database",
			"`database_id` must be configured with a valid Superset database identifier.",
		)
	}

	tableName := strings.TrimSpace(stringValue(data.TableName))
	if tableName == "" {
		diags.AddAttributeError(
			path.Root("table_name"),
			"Missing Dataset Table Name",
			"`table_name` must be configured.",
		)
	}

	columnModels, columnDiags := datasetColumnsFromList(ctx, data.Columns)
	diags.Append(columnDiags...)

	metricModels, metricDiags := datasetMetricsFromList(ctx, data.Metrics)
	diags.Append(metricDiags...)

	if diags.HasError() {
		return supersetclient.DatasetUpdateRequest{}, diags
	}

	request := supersetclient.DatasetUpdateRequest{
		DatabaseID:           databaseID,
		TableName:            tableName,
		Schema:               stringPointerValue(data.Schema),
		Description:          stringPointerValue(data.Description),
		MainDttmCol:          stringPointerValue(data.MainDttmCol),
		FilterSelectEnabled:  boolPointerValue(data.FilterSelectEnabled),
		NormalizeColumns:     boolPointerValue(data.NormalizeColumns),
		AlwaysFilterMainDttm: boolPointerValue(data.AlwaysFilterMainDttm),
		CacheTimeout:         int64PointerValue(data.CacheTimeout),
	}

	if !data.Columns.IsNull() && !data.Columns.IsUnknown() {
		columnRequests, expandDiags := expandDatasetColumns(columnModels, current.Columns)
		diags.Append(expandDiags...)
		request.Columns = &columnRequests
	}

	if !data.Metrics.IsNull() && !data.Metrics.IsUnknown() {
		metricRequests, expandDiags := expandDatasetMetrics(metricModels, current.Metrics)
		diags.Append(expandDiags...)
		request.Metrics = &metricRequests
	}

	return request, diags
}

func needsDatasetUpdate(data datasetModel) bool {
	return hasStringValue(data.Description) ||
		hasStringValue(data.MainDttmCol) ||
		(!data.FilterSelectEnabled.IsNull() && !data.FilterSelectEnabled.IsUnknown()) ||
		(!data.NormalizeColumns.IsNull() && !data.NormalizeColumns.IsUnknown()) ||
		(!data.AlwaysFilterMainDttm.IsNull() && !data.AlwaysFilterMainDttm.IsUnknown()) ||
		(!data.CacheTimeout.IsNull() && !data.CacheTimeout.IsUnknown()) ||
		(!data.Columns.IsNull() && !data.Columns.IsUnknown()) ||
		(!data.Metrics.IsNull() && !data.Metrics.IsUnknown())
}

func flattenDatasetResourceModel(ctx context.Context, current datasetModel, remote *supersetclient.Dataset) (datasetModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	state := datasetModel{
		ID:           types.Int64Value(remote.ID),
		UUID:         stringTypeValue(remote.UUID),
		DatabaseID:   types.Int64Value(remote.Database.ID),
		DatabaseName: stringTypeValue(remote.Database.DatabaseName),
		TableName:    stringTypeValue(remote.TableName),
	}

	state.Schema = managedStringValue(current.Schema, remote.Schema)
	state.Description = managedStringValue(current.Description, remote.Description)
	state.MainDttmCol = managedStringValue(current.MainDttmCol, remote.MainDttmCol)
	state.FilterSelectEnabled = managedBoolValue(current.FilterSelectEnabled, remote.FilterSelectEnabled)
	state.NormalizeColumns = managedBoolValue(current.NormalizeColumns, remote.NormalizeColumns)
	state.AlwaysFilterMainDttm = managedBoolValue(current.AlwaysFilterMainDttm, remote.AlwaysFilterMainDttm)
	state.CacheTimeout = managedInt64Value(current.CacheTimeout, remote.CacheTimeout)

	columns, columnDiags := flattenManagedDatasetColumns(ctx, current.Columns, remote.Columns)
	diags.Append(columnDiags...)
	state.Columns = columns

	metrics, metricDiags := flattenManagedDatasetMetrics(ctx, current.Metrics, remote.Metrics)
	diags.Append(metricDiags...)
	state.Metrics = metrics

	return state, diags
}

func flattenDatasetDataSourceModel(ctx context.Context, remote *supersetclient.Dataset) (datasetModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	columns, columnDiags := flattenDatasetColumns(ctx, remote.Columns)
	diags.Append(columnDiags...)

	metrics, metricDiags := flattenDatasetMetrics(ctx, remote.Metrics)
	diags.Append(metricDiags...)

	return datasetModel{
		ID:                   types.Int64Value(remote.ID),
		UUID:                 stringTypeValue(remote.UUID),
		DatabaseID:           types.Int64Value(remote.Database.ID),
		DatabaseName:         stringTypeValue(remote.Database.DatabaseName),
		TableName:            stringTypeValue(remote.TableName),
		Schema:               stringTypeValue(remote.Schema),
		Description:          stringTypeValue(remote.Description),
		MainDttmCol:          stringTypeValue(remote.MainDttmCol),
		FilterSelectEnabled:  boolTypeValue(remote.FilterSelectEnabled),
		NormalizeColumns:     boolTypeValue(remote.NormalizeColumns),
		AlwaysFilterMainDttm: boolTypeValue(remote.AlwaysFilterMainDttm),
		CacheTimeout:         int64TypeValue(remote.CacheTimeout),
		Columns:              columns,
		Metrics:              metrics,
	}, diags
}

func datasetColumnsFromList(ctx context.Context, value types.List) ([]datasetColumnModel, diag.Diagnostics) {
	var columns []datasetColumnModel

	if value.IsNull() || value.IsUnknown() {
		return columns, nil
	}

	diags := value.ElementsAs(ctx, &columns, false)

	return columns, diags
}

func datasetMetricsFromList(ctx context.Context, value types.List) ([]datasetMetricModel, diag.Diagnostics) {
	var metrics []datasetMetricModel

	if value.IsNull() || value.IsUnknown() {
		return metrics, nil
	}

	diags := value.ElementsAs(ctx, &metrics, false)

	return metrics, diags
}

func expandDatasetColumns(columns []datasetColumnModel, current []supersetclient.DatasetColumn) ([]supersetclient.DatasetColumn, diag.Diagnostics) {
	var diags diag.Diagnostics

	currentByName := make(map[string]supersetclient.DatasetColumn, len(current))
	for _, column := range current {
		currentByName[column.ColumnName] = column
	}

	requests := make([]supersetclient.DatasetColumn, 0, len(columns))

	for index, column := range columns {
		columnName := strings.TrimSpace(stringValue(column.ColumnName))
		if columnName == "" {
			diags.AddAttributeError(
				path.Root("columns").AtListIndex(index).AtName("column_name"),
				"Missing Dataset Column Name",
				"`column_name` must be configured for every dataset column.",
			)

			continue
		}

		request := supersetclient.DatasetColumn{
			ColumnName:       columnName,
			VerboseName:      stringValue(column.VerboseName),
			Description:      stringValue(column.Description),
			Expression:       stringValue(column.Expression),
			Filterable:       boolPointerValue(column.Filterable),
			Groupby:          boolPointerValue(column.Groupby),
			IsActive:         boolPointerValue(column.IsActive),
			IsDttm:           boolPointerValue(column.IsDttm),
			Type:             stringValue(column.Type),
			PythonDateFormat: stringValue(column.PythonDateFormat),
		}

		if currentColumn, ok := currentByName[columnName]; ok {
			request.ID = currentColumn.ID
		}

		requests = append(requests, request)
	}

	return requests, diags
}

func expandDatasetMetrics(metrics []datasetMetricModel, current []supersetclient.DatasetMetric) ([]supersetclient.DatasetMetric, diag.Diagnostics) {
	var diags diag.Diagnostics

	currentByName := make(map[string]supersetclient.DatasetMetric, len(current))
	for _, metric := range current {
		currentByName[metric.MetricName] = metric
	}

	requests := make([]supersetclient.DatasetMetric, 0, len(metrics))

	for index, metric := range metrics {
		metricName := strings.TrimSpace(stringValue(metric.MetricName))
		if metricName == "" {
			diags.AddAttributeError(
				path.Root("metrics").AtListIndex(index).AtName("metric_name"),
				"Missing Dataset Metric Name",
				"`metric_name` must be configured for every dataset metric.",
			)

			continue
		}

		expression := strings.TrimSpace(stringValue(metric.Expression))
		if expression == "" {
			diags.AddAttributeError(
				path.Root("metrics").AtListIndex(index).AtName("expression"),
				"Missing Dataset Metric Expression",
				"`expression` must be configured for every dataset metric.",
			)

			continue
		}

		request := supersetclient.DatasetMetric{
			MetricName:  metricName,
			Expression:  expression,
			MetricType:  stringValue(metric.MetricType),
			VerboseName: stringValue(metric.VerboseName),
			Description: stringValue(metric.Description),
			D3Format:    stringValue(metric.D3Format),
			WarningText: stringValue(metric.WarningText),
		}

		if currentMetric, ok := currentByName[metricName]; ok {
			request.ID = currentMetric.ID
		}

		requests = append(requests, request)
	}

	return requests, diags
}

func flattenManagedDatasetColumns(ctx context.Context, current types.List, remote []supersetclient.DatasetColumn) (types.List, diag.Diagnostics) {
	if current.IsNull() || current.IsUnknown() {
		return current, nil
	}

	currentColumns, diags := datasetColumnsFromList(ctx, current)
	if diags.HasError() {
		return types.ListNull(datasetColumnObjectType), diags
	}

	remoteByName := make(map[string]supersetclient.DatasetColumn, len(remote))
	for _, column := range remote {
		remoteByName[column.ColumnName] = column
	}

	columns := make([]datasetColumnModel, 0, len(currentColumns))
	for _, currentColumn := range currentColumns {
		columnName := stringValue(currentColumn.ColumnName)
		remoteColumn, ok := remoteByName[columnName]
		if !ok {
			continue
		}

		columns = append(columns, datasetColumnModel{
			ColumnName:       stringTypeValue(remoteColumn.ColumnName),
			VerboseName:      managedStringValue(currentColumn.VerboseName, remoteColumn.VerboseName),
			Description:      managedStringValue(currentColumn.Description, remoteColumn.Description),
			Expression:       managedStringValue(currentColumn.Expression, remoteColumn.Expression),
			Filterable:       managedBoolValue(currentColumn.Filterable, remoteColumn.Filterable),
			Groupby:          managedBoolValue(currentColumn.Groupby, remoteColumn.Groupby),
			IsActive:         managedBoolValue(currentColumn.IsActive, remoteColumn.IsActive),
			IsDttm:           managedBoolValue(currentColumn.IsDttm, remoteColumn.IsDttm),
			Type:             managedStringValue(currentColumn.Type, remoteColumn.Type),
			PythonDateFormat: managedStringValue(currentColumn.PythonDateFormat, remoteColumn.PythonDateFormat),
		})
	}

	return types.ListValueFrom(ctx, datasetColumnObjectType, columns)
}

func flattenManagedDatasetMetrics(ctx context.Context, current types.List, remote []supersetclient.DatasetMetric) (types.List, diag.Diagnostics) {
	if current.IsNull() || current.IsUnknown() {
		return current, nil
	}

	currentMetrics, diags := datasetMetricsFromList(ctx, current)
	if diags.HasError() {
		return types.ListNull(datasetMetricObjectType), diags
	}

	remoteByName := make(map[string]supersetclient.DatasetMetric, len(remote))
	for _, metric := range remote {
		remoteByName[metric.MetricName] = metric
	}

	metrics := make([]datasetMetricModel, 0, len(currentMetrics))
	for _, currentMetric := range currentMetrics {
		metricName := stringValue(currentMetric.MetricName)
		remoteMetric, ok := remoteByName[metricName]
		if !ok {
			continue
		}

		metrics = append(metrics, datasetMetricModel{
			MetricName:  stringTypeValue(remoteMetric.MetricName),
			Expression:  stringTypeValue(remoteMetric.Expression),
			MetricType:  managedStringValue(currentMetric.MetricType, remoteMetric.MetricType),
			VerboseName: managedStringValue(currentMetric.VerboseName, remoteMetric.VerboseName),
			Description: managedStringValue(currentMetric.Description, remoteMetric.Description),
			D3Format:    managedStringValue(currentMetric.D3Format, remoteMetric.D3Format),
			WarningText: managedStringValue(currentMetric.WarningText, remoteMetric.WarningText),
		})
	}

	return types.ListValueFrom(ctx, datasetMetricObjectType, metrics)
}

func flattenDatasetColumns(ctx context.Context, remote []supersetclient.DatasetColumn) (types.List, diag.Diagnostics) {
	columns := make([]datasetColumnModel, 0, len(remote))

	sort.Slice(remote, func(i, j int) bool {
		return remote[i].ColumnName < remote[j].ColumnName
	})

	for _, column := range remote {
		columns = append(columns, datasetColumnModel{
			ColumnName:       stringTypeValue(column.ColumnName),
			VerboseName:      stringTypeValue(column.VerboseName),
			Description:      stringTypeValue(column.Description),
			Expression:       stringTypeValue(column.Expression),
			Filterable:       boolTypeValue(column.Filterable),
			Groupby:          boolTypeValue(column.Groupby),
			IsActive:         boolTypeValue(column.IsActive),
			IsDttm:           boolTypeValue(column.IsDttm),
			Type:             stringTypeValue(column.Type),
			PythonDateFormat: stringTypeValue(column.PythonDateFormat),
		})
	}

	return types.ListValueFrom(ctx, datasetColumnObjectType, columns)
}

func flattenDatasetMetrics(ctx context.Context, remote []supersetclient.DatasetMetric) (types.List, diag.Diagnostics) {
	metrics := make([]datasetMetricModel, 0, len(remote))

	sort.Slice(remote, func(i, j int) bool {
		return remote[i].MetricName < remote[j].MetricName
	})

	for _, metric := range remote {
		metrics = append(metrics, datasetMetricModel{
			MetricName:  stringTypeValue(metric.MetricName),
			Expression:  stringTypeValue(metric.Expression),
			MetricType:  stringTypeValue(metric.MetricType),
			VerboseName: stringTypeValue(metric.VerboseName),
			Description: stringTypeValue(metric.Description),
			D3Format:    stringTypeValue(metric.D3Format),
			WarningText: stringTypeValue(metric.WarningText),
		})
	}

	return types.ListValueFrom(ctx, datasetMetricObjectType, metrics)
}

func managedStringValue(current types.String, remote string) types.String {
	if current.IsNull() || current.IsUnknown() {
		return current
	}

	return stringTypeValue(remote)
}

func managedBoolValue(current types.Bool, remote *bool) types.Bool {
	if current.IsNull() || current.IsUnknown() {
		return current
	}

	return boolTypeValue(remote)
}

func managedInt64Value(current types.Int64, remote *int64) types.Int64 {
	if current.IsNull() || current.IsUnknown() {
		return current
	}

	return int64TypeValue(remote)
}

func findDataset(ctx context.Context, client *supersetclient.Client, databaseID int64, tableName string, schemaName string) (*supersetclient.Dataset, error) {
	datasets, err := client.ListDatasets(ctx, 1000)
	if err != nil {
		return nil, err
	}

	normalizedTableName := strings.TrimSpace(tableName)
	normalizedSchemaName := strings.TrimSpace(schemaName)

	var matches []supersetclient.Dataset

	for _, dataset := range datasets {
		if dataset.Database.ID != databaseID {
			continue
		}

		if dataset.TableName != normalizedTableName {
			continue
		}

		if strings.TrimSpace(dataset.Schema) != normalizedSchemaName {
			continue
		}

		matches = append(matches, dataset)
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("dataset %q in database %d with schema %q was not found", normalizedTableName, databaseID, normalizedSchemaName)
	case 1:
		return client.GetDataset(ctx, matches[0].ID)
	default:
		return nil, fmt.Errorf("dataset %q in database %d with schema %q matched %d datasets", normalizedTableName, databaseID, normalizedSchemaName, len(matches))
	}
}
