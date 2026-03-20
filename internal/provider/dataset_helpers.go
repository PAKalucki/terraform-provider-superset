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
		DatabaseID:                  databaseID,
		TableName:                   tableName,
		Schema:                      stringPointerValue(data.Schema),
		Description:                 stringPointerValue(data.Description),
		MainDttmCol:                 stringPointerValue(data.MainDttmCol),
		FilterSelectEnabled:         datasetBoolUpdateValue(data.FilterSelectEnabled, current.FilterSelectEnabled),
		NormalizeColumns:            datasetBoolUpdateValue(data.NormalizeColumns, current.NormalizeColumns),
		AlwaysFilterMainDttm:        datasetBoolUpdateValue(data.AlwaysFilterMainDttm, current.AlwaysFilterMainDttm),
		CacheTimeout:                int64PointerValue(data.CacheTimeout),
		IncludeSchema:               includeManagedString(data.Schema, current.Schema),
		IncludeDescription:          includeManagedString(data.Description, current.Description),
		IncludeMainDttmCol:          includeManagedString(data.MainDttmCol, current.MainDttmCol),
		IncludeFilterSelectEnabled:  includeManagedBool(data.FilterSelectEnabled, current.FilterSelectEnabled),
		IncludeNormalizeColumns:     includeManagedBool(data.NormalizeColumns, current.NormalizeColumns),
		IncludeAlwaysFilterMainDttm: includeManagedBool(data.AlwaysFilterMainDttm, current.AlwaysFilterMainDttm),
		IncludeCacheTimeout:         includeManagedInt64(data.CacheTimeout, current.CacheTimeout),
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

	state.Schema = stringTypeValue(remote.Schema)
	state.Description = stringTypeValue(remote.Description)
	state.MainDttmCol = stringTypeValue(remote.MainDttmCol)
	state.FilterSelectEnabled = managedDatasetBoolValue(current.FilterSelectEnabled, remote.FilterSelectEnabled)
	state.NormalizeColumns = managedDatasetBoolValue(current.NormalizeColumns, remote.NormalizeColumns)
	state.AlwaysFilterMainDttm = managedDatasetBoolValue(current.AlwaysFilterMainDttm, remote.AlwaysFilterMainDttm)
	state.CacheTimeout = int64TypeValue(remote.CacheTimeout)

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

		currentColumn, ok := currentByName[columnName]

		request := supersetclient.DatasetColumn{
			ColumnName:       columnName,
			VerboseName:      stringValue(column.VerboseName),
			Description:      stringValue(column.Description),
			Expression:       stringValue(column.Expression),
			Filterable:       datasetBoolUpdateValue(column.Filterable, currentColumn.Filterable),
			Groupby:          datasetBoolUpdateValue(column.Groupby, currentColumn.Groupby),
			IsActive:         datasetBoolUpdateValue(column.IsActive, currentColumn.IsActive),
			IsDttm:           datasetBoolUpdateValue(column.IsDttm, currentColumn.IsDttm),
			Type:             stringValue(column.Type),
			PythonDateFormat: stringValue(column.PythonDateFormat),
		}

		if ok {
			request.ID = currentColumn.ID
		}

		request.IncludeVerboseName = includeManagedString(column.VerboseName, currentColumn.VerboseName)
		request.IncludeDescription = includeManagedString(column.Description, currentColumn.Description)
		request.IncludeExpression = includeManagedString(column.Expression, currentColumn.Expression)
		request.IncludeFilterable = includeManagedBool(column.Filterable, currentColumn.Filterable)
		request.IncludeGroupby = includeManagedBool(column.Groupby, currentColumn.Groupby)
		request.IncludeIsActive = includeManagedBool(column.IsActive, currentColumn.IsActive)
		request.IncludeIsDttm = includeManagedBool(column.IsDttm, currentColumn.IsDttm)
		request.IncludeType = includeManagedString(column.Type, currentColumn.Type)
		request.IncludePythonDateFormat = includeManagedString(column.PythonDateFormat, currentColumn.PythonDateFormat)

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

		currentMetric, ok := currentByName[metricName]
		if ok {
			request.ID = currentMetric.ID
		}

		request.IncludeMetricType = includeManagedString(metric.MetricType, currentMetric.MetricType)
		request.IncludeVerboseName = includeManagedString(metric.VerboseName, currentMetric.VerboseName)
		request.IncludeDescription = includeManagedString(metric.Description, currentMetric.Description)
		request.IncludeD3Format = includeManagedString(metric.D3Format, currentMetric.D3Format)
		request.IncludeWarningText = includeManagedString(metric.WarningText, currentMetric.WarningText)

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
			VerboseName:      stringTypeValue(remoteColumn.VerboseName),
			Description:      stringTypeValue(remoteColumn.Description),
			Expression:       stringTypeValue(remoteColumn.Expression),
			Filterable:       managedDatasetBoolValue(currentColumn.Filterable, remoteColumn.Filterable),
			Groupby:          managedDatasetBoolValue(currentColumn.Groupby, remoteColumn.Groupby),
			IsActive:         managedDatasetBoolValue(currentColumn.IsActive, remoteColumn.IsActive),
			IsDttm:           managedDatasetBoolValue(currentColumn.IsDttm, remoteColumn.IsDttm),
			Type:             stringTypeValue(remoteColumn.Type),
			PythonDateFormat: stringTypeValue(remoteColumn.PythonDateFormat),
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
			MetricType:  stringTypeValue(remoteMetric.MetricType),
			VerboseName: stringTypeValue(remoteMetric.VerboseName),
			Description: stringTypeValue(remoteMetric.Description),
			D3Format:    stringTypeValue(remoteMetric.D3Format),
			WarningText: stringTypeValue(remoteMetric.WarningText),
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

func findDataset(ctx context.Context, client *supersetclient.Client, databaseID int64, tableName string, schemaName string) (*supersetclient.Dataset, error) {
	datasets, err := client.ListDatasets(ctx, 1000)
	if err != nil {
		return nil, err
	}

	normalizedTableName := strings.TrimSpace(tableName)
	normalizedSchemaName := strings.TrimSpace(schemaName)
	requireSchemaMatch := normalizedSchemaName != ""

	var matches []supersetclient.Dataset

	for _, dataset := range datasets {
		if dataset.Database.ID != databaseID {
			continue
		}

		if strings.TrimSpace(dataset.TableName) != normalizedTableName {
			continue
		}

		if requireSchemaMatch && strings.TrimSpace(dataset.Schema) != normalizedSchemaName {
			continue
		}

		matches = append(matches, dataset)
	}

	switch len(matches) {
	case 0:
		if !requireSchemaMatch {
			return nil, fmt.Errorf("dataset %q in database %d was not found", normalizedTableName, databaseID)
		}

		return nil, fmt.Errorf("dataset %q in database %d with schema %q was not found", normalizedTableName, databaseID, normalizedSchemaName)
	case 1:
		return client.GetDataset(ctx, matches[0].ID)
	default:
		if !requireSchemaMatch {
			return nil, fmt.Errorf("dataset %q in database %d matched %d datasets; configure `schema` to disambiguate the lookup", normalizedTableName, databaseID, len(matches))
		}

		return nil, fmt.Errorf("dataset %q in database %d with schema %q matched %d datasets", normalizedTableName, databaseID, normalizedSchemaName, len(matches))
	}
}

func includeManagedString(plan types.String, remote string) bool {
	if !plan.IsNull() && !plan.IsUnknown() {
		return true
	}

	return strings.TrimSpace(remote) != ""
}

func includeManagedBool(plan types.Bool, remote *bool) bool {
	if !plan.IsNull() && !plan.IsUnknown() {
		return true
	}

	return remote != nil && *remote
}

func includeManagedInt64(plan types.Int64, remote *int64) bool {
	if !plan.IsNull() && !plan.IsUnknown() {
		return true
	}

	return remote != nil
}

func datasetBoolUpdateValue(plan types.Bool, remote *bool) *bool {
	if !plan.IsNull() && !plan.IsUnknown() {
		return boolPointerValue(plan)
	}

	if remote != nil && *remote {
		value := false

		return &value
	}

	return nil
}

func managedDatasetBoolValue(current types.Bool, remote *bool) types.Bool {
	if current.IsNull() || current.IsUnknown() {
		if remote != nil && *remote {
			return types.BoolValue(true)
		}

		return types.BoolNull()
	}

	return boolTypeValue(remote)
}
