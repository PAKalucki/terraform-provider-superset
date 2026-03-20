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

var _ datasource.DataSource = &DatabaseDataSource{}

func NewDatabaseDataSource() datasource.DataSource {
	return &DatabaseDataSource{}
}

type DatabaseDataSource struct {
	client *supersetclient.Client
}

func (d *DatabaseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (d *DatabaseDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing Superset database connection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Superset database identifier used for lookup or returned from Superset.",
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Superset database UUID.",
			},
			"database_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Human-readable name for the Superset database connection.",
			},
			"sqlalchemy_uri": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "SQLAlchemy connection URI returned by Superset. Stored credentials are masked by the API.",
			},
			"extra": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Database `extra` JSON string returned by Superset.",
			},
			"expose_in_sqllab": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the database is exposed in SQL Lab.",
			},
			"allow_ctas": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether `CREATE TABLE AS` statements are allowed in SQL Lab.",
			},
			"allow_cvas": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether `CREATE VIEW AS` statements are allowed in SQL Lab.",
			},
			"allow_dml": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether non-SELECT statements are allowed in SQL Lab.",
			},
			"allow_file_upload": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether CSV uploads are allowed for this database.",
			},
			"allow_run_async": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether queries on this database run asynchronously.",
			},
			"cache_timeout": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Database-level chart cache timeout in seconds.",
			},
			"force_ctas_schema": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Schema enforced for `CREATE TABLE AS` statements when enabled.",
			},
			"impersonate_user": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether Superset impersonates the current user when querying this database.",
			},
			"backend": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved Superset database backend.",
			},
			"driver": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resolved SQLAlchemy driver.",
			},
		},
	}
}

func (d *DatabaseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DatabaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data databaseModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the database data source.",
		)

		return
	}

	hasID := !data.ID.IsNull() && !data.ID.IsUnknown()
	hasName := !data.DatabaseName.IsNull() && !data.DatabaseName.IsUnknown() && strings.TrimSpace(data.DatabaseName.ValueString()) != ""

	switch {
	case hasID && hasName:
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Conflicting Database Lookup Arguments",
			"Configure only one of `id` or `database_name`.",
		)

		return
	case !hasID && !hasName:
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Missing Database Lookup Arguments",
			"Configure either `id` or `database_name`.",
		)

		return
	}

	var (
		database *supersetclient.Database
		err      error
	)

	if hasID {
		database, err = loadDatabase(ctx, d.client, data.ID.ValueInt64())
	} else {
		databaseName := strings.TrimSpace(data.DatabaseName.ValueString())
		database, err = findDatabaseByName(ctx, d.client, databaseName)
	}

	if err != nil {
		if hasID && isSupersetNotFoundError(err) {
			resp.Diagnostics.AddAttributeError(
				path.Root("id"),
				"Superset Database Not Found",
				err.Error(),
			)
		} else {
			resp.Diagnostics.AddError(
				"Unable to Read Superset Database",
				err.Error(),
			)
		}

		return
	}

	state, diags := flattenDatabaseModel(databaseModel{}, database)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
