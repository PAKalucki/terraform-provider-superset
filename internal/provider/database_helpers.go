package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type databaseModel struct {
	ID              types.Int64  `tfsdk:"id"`
	UUID            types.String `tfsdk:"uuid"`
	DatabaseName    types.String `tfsdk:"database_name"`
	SQLAlchemyURI   types.String `tfsdk:"sqlalchemy_uri"`
	Extra           types.String `tfsdk:"extra"`
	ExposeInSQLLab  types.Bool   `tfsdk:"expose_in_sqllab"`
	AllowCTAS       types.Bool   `tfsdk:"allow_ctas"`
	AllowCVAS       types.Bool   `tfsdk:"allow_cvas"`
	AllowDML        types.Bool   `tfsdk:"allow_dml"`
	AllowFileUpload types.Bool   `tfsdk:"allow_file_upload"`
	AllowRunAsync   types.Bool   `tfsdk:"allow_run_async"`
	CacheTimeout    types.Int64  `tfsdk:"cache_timeout"`
	ForceCTASSchema types.String `tfsdk:"force_ctas_schema"`
	ImpersonateUser types.Bool   `tfsdk:"impersonate_user"`
	Backend         types.String `tfsdk:"backend"`
	Driver          types.String `tfsdk:"driver"`
}

func expandDatabaseRequest(data databaseModel) (supersetclient.Database, diag.Diagnostics) {
	var diags diag.Diagnostics

	databaseName := strings.TrimSpace(stringValue(data.DatabaseName))
	if databaseName == "" {
		diags.AddAttributeError(
			path.Root("database_name"),
			"Missing Database Name",
			"`database_name` must be configured.",
		)
	}

	sqlalchemyURI := strings.TrimSpace(stringValue(data.SQLAlchemyURI))
	if sqlalchemyURI == "" {
		diags.AddAttributeError(
			path.Root("sqlalchemy_uri"),
			"Missing SQLAlchemy URI",
			"`sqlalchemy_uri` must be configured.",
		)
	}

	normalizedExtra, extraDiags := normalizeOptionalJSONString(data.Extra, path.Root("extra"))
	diags.Append(extraDiags...)

	if diags.HasError() {
		return supersetclient.Database{}, diags
	}

	request := supersetclient.Database{
		DatabaseName:    databaseName,
		SQLAlchemyURI:   sqlalchemyURI,
		ExposeInSQLLab:  boolPointerValue(data.ExposeInSQLLab),
		AllowCTAS:       boolPointerValue(data.AllowCTAS),
		AllowCVAS:       boolPointerValue(data.AllowCVAS),
		AllowDML:        boolPointerValue(data.AllowDML),
		AllowFileUpload: boolPointerValue(data.AllowFileUpload),
		AllowRunAsync:   boolPointerValue(data.AllowRunAsync),
		CacheTimeout:    int64PointerValue(data.CacheTimeout),
		ForceCTASSchema: stringPointerValue(data.ForceCTASSchema),
		ImpersonateUser: boolPointerValue(data.ImpersonateUser),
	}

	if !normalizedExtra.IsNull() && !normalizedExtra.IsUnknown() {
		request.Extra = normalizedExtra.ValueString()
	}

	return request, diags
}

func applyDatabaseRequestToModel(data databaseModel, request supersetclient.Database) databaseModel {
	data.DatabaseName = types.StringValue(request.DatabaseName)
	data.SQLAlchemyURI = types.StringValue(request.SQLAlchemyURI)

	if request.Extra == "" {
		data.Extra = types.StringNull()
	} else {
		data.Extra = types.StringValue(request.Extra)
	}

	return data
}

func flattenDatabaseModel(current databaseModel, remote *supersetclient.Database) (databaseModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	state := current
	state.ID = types.Int64Value(remote.ID)
	state.UUID = stringTypeValue(remote.UUID)
	state.DatabaseName = stringTypeValue(remote.DatabaseName)
	state.Backend = stringTypeValue(remote.Backend)
	state.Driver = stringTypeValue(remote.Driver)
	state.ExposeInSQLLab = boolTypeValue(remote.ExposeInSQLLab)
	state.AllowCTAS = boolTypeValue(remote.AllowCTAS)
	state.AllowCVAS = boolTypeValue(remote.AllowCVAS)
	state.AllowDML = boolTypeValue(remote.AllowDML)
	state.AllowFileUpload = boolTypeValue(remote.AllowFileUpload)
	state.AllowRunAsync = boolTypeValue(remote.AllowRunAsync)
	state.CacheTimeout = int64TypeValue(remote.CacheTimeout)
	state.ForceCTASSchema = stringPointerTypeValue(remote.ForceCTASSchema)
	state.ImpersonateUser = boolTypeValue(remote.ImpersonateUser)

	if current.SQLAlchemyURI.IsNull() || current.SQLAlchemyURI.IsUnknown() {
		state.SQLAlchemyURI = stringTypeValue(remote.SQLAlchemyURI)
	}

	if strings.TrimSpace(remote.Extra) == "" {
		if current.Extra.IsNull() || current.Extra.IsUnknown() {
			state.Extra = types.StringNull()
		}
	} else {
		normalizedExtra, extraDiags := normalizeOptionalJSONString(types.StringValue(remote.Extra), path.Root("extra"))
		diags.Append(extraDiags...)

		if !extraDiags.HasError() {
			state.Extra = normalizedExtra
		}
	}

	return state, diags
}

func loadDatabase(ctx context.Context, client *supersetclient.Client, id int64) (*supersetclient.Database, error) {
	if _, err := client.GetDatabase(ctx, id); err != nil {
		return nil, err
	}

	return client.GetDatabaseConnection(ctx, id)
}

func findDatabaseByName(ctx context.Context, client *supersetclient.Client, databaseName string) (*supersetclient.Database, error) {
	databases, err := client.ListDatabases(ctx, 1000)
	if err != nil {
		return nil, err
	}

	var matches []supersetclient.Database

	for _, database := range databases {
		if database.DatabaseName == databaseName {
			matches = append(matches, database)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("database %q was not found", databaseName)
	case 1:
		return loadDatabase(ctx, client, matches[0].ID)
	default:
		return nil, fmt.Errorf("database %q matched %d Superset connections", databaseName, len(matches))
	}
}

func isSupersetNotFoundError(err error) bool {
	var apiErr *supersetclient.APIError

	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

func normalizeOptionalJSONString(value types.String, attributePath path.Path) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if value.IsNull() || value.IsUnknown() {
		return value, diags
	}

	normalized, err := normalizeJSONString(value.ValueString())
	if err != nil {
		diags.AddAttributeError(
			attributePath,
			"Invalid JSON String",
			err.Error(),
		)

		return types.StringNull(), diags
	}

	return types.StringValue(normalized), diags
}

func normalizeJSONString(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("value must be valid JSON")
	}

	var decoded any

	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return "", fmt.Errorf("value must be valid JSON: %w", err)
	}

	normalized, err := json.Marshal(decoded)
	if err != nil {
		return "", fmt.Errorf("normalize JSON value: %w", err)
	}

	return string(normalized), nil
}

func boolPointerValue(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	v := value.ValueBool()

	return &v
}

func int64PointerValue(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	v := value.ValueInt64()

	return &v
}

func stringPointerValue(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	v := value.ValueString()

	return &v
}

func boolTypeValue(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}

	return types.BoolValue(*value)
}

func int64TypeValue(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}

	return types.Int64Value(*value)
}

func stringPointerTypeValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}

	return stringTypeValue(*value)
}

func stringTypeValue(value string) types.String {
	if strings.TrimSpace(value) == "" {
		return types.StringNull()
	}

	return types.StringValue(value)
}
