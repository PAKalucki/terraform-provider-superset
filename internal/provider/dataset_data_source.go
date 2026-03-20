package provider

import (
	"context"
	"fmt"
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

var _ datasource.DataSource = &DatasetDataSource{}

func NewDatasetDataSource() datasource.DataSource {
	return &DatasetDataSource{}
}

type DatasetDataSource struct {
	client *supersetclient.Client
}

func (d *DatasetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (d *DatasetDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing Superset dataset.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Superset dataset identifier used for lookup or returned from Superset.",
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Superset dataset UUID.",
			},
			"database_id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Superset database identifier that owns the dataset.",
			},
			"database_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved Superset database name.",
			},
			"table_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Dataset table name.",
			},
			"schema": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Dataset schema name.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Dataset description.",
			},
			"main_dttm_col": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Main datetime column used by Superset.",
			},
			"filter_select_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether filter select is enabled for the dataset.",
			},
			"normalize_columns": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether Superset normalizes columns for the dataset.",
			},
			"always_filter_main_dttm": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the main datetime column is always filtered.",
			},
			"cache_timeout": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Dataset cache timeout in seconds.",
			},
			"columns": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Dataset columns returned by Superset.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"column_name":        schema.StringAttribute{Computed: true},
						"verbose_name":       schema.StringAttribute{Computed: true},
						"description":        schema.StringAttribute{Computed: true},
						"expression":         schema.StringAttribute{Computed: true},
						"filterable":         schema.BoolAttribute{Computed: true},
						"groupby":            schema.BoolAttribute{Computed: true},
						"is_active":          schema.BoolAttribute{Computed: true},
						"is_dttm":            schema.BoolAttribute{Computed: true},
						"type":               schema.StringAttribute{Computed: true},
						"python_date_format": schema.StringAttribute{Computed: true},
					},
				},
			},
			"metrics": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Dataset metrics returned by Superset.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"metric_name":  schema.StringAttribute{Computed: true},
						"expression":   schema.StringAttribute{Computed: true},
						"metric_type":  schema.StringAttribute{Computed: true},
						"verbose_name": schema.StringAttribute{Computed: true},
						"description":  schema.StringAttribute{Computed: true},
						"d3format":     schema.StringAttribute{Computed: true},
						"warning_text": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *DatasetDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*supersetclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *DatasetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasetModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the dataset data source.",
		)

		return
	}

	hasID := !data.ID.IsNull() && !data.ID.IsUnknown()
	hasDatabaseID := !data.DatabaseID.IsNull() && !data.DatabaseID.IsUnknown()
	hasTableName := !data.TableName.IsNull() && !data.TableName.IsUnknown() && strings.TrimSpace(data.TableName.ValueString()) != ""

	switch {
	case hasID && (hasDatabaseID || hasTableName):
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Conflicting Dataset Lookup Arguments",
			"Configure either `id` or `database_id` with `table_name` and optional `schema`.",
		)

		return
	case !hasID && (!hasDatabaseID || !hasTableName):
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Missing Dataset Lookup Arguments",
			"Configure either `id` or `database_id` with `table_name` and optional `schema`.",
		)

		return
	}

	var (
		dataset *supersetclient.Dataset
		err     error
	)

	if hasID {
		dataset, err = d.client.GetDataset(ctx, data.ID.ValueInt64())
	} else {
		dataset, err = findDataset(ctx, d.client, data.DatabaseID.ValueInt64(), data.TableName.ValueString(), data.Schema.ValueString())
	}

	if err != nil {
		if hasID && isSupersetNotFoundError(err) {
			resp.Diagnostics.AddAttributeError(
				path.Root("id"),
				"Superset Dataset Not Found",
				err.Error(),
			)
		} else {
			resp.Diagnostics.AddError(
				"Unable to Read Superset Dataset",
				err.Error(),
			)
		}

		return
	}

	state, diags := flattenDatasetDataSourceModel(ctx, dataset)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
