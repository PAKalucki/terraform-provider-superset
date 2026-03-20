package provider

import (
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type savedQueryModel struct {
	ID                 types.Int64  `tfsdk:"id"`
	DatabaseID         types.Int64  `tfsdk:"database_id"`
	DatabaseName       types.String `tfsdk:"database_name"`
	Label              types.String `tfsdk:"label"`
	Description        types.String `tfsdk:"description"`
	Catalog            types.String `tfsdk:"catalog"`
	Schema             types.String `tfsdk:"schema"`
	SQL                types.String `tfsdk:"sql"`
	TemplateParameters types.String `tfsdk:"template_parameters"`
	ExtraJSON          types.String `tfsdk:"extra_json"`
}

func expandSavedQueryCreateRequest(data savedQueryModel) (supersetclient.SavedQueryCreateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	databaseID := int64Value(data.DatabaseID)
	if databaseID <= 0 {
		diags.AddAttributeError(
			path.Root("database_id"),
			"Missing Saved Query Database",
			"`database_id` must be configured with a valid Superset database identifier.",
		)
	}

	label := strings.TrimSpace(stringValue(data.Label))
	if label == "" {
		diags.AddAttributeError(
			path.Root("label"),
			"Missing Saved Query Label",
			"`label` must be configured.",
		)
	}

	sql := strings.TrimSpace(stringValue(data.SQL))
	if sql == "" {
		diags.AddAttributeError(
			path.Root("sql"),
			"Missing Saved Query SQL",
			"`sql` must be configured.",
		)
	}

	templateParameters, templateDiags := normalizeOptionalJSONString(data.TemplateParameters, path.Root("template_parameters"))
	diags.Append(templateDiags...)

	extraJSON, extraDiags := normalizeOptionalJSONString(data.ExtraJSON, path.Root("extra_json"))
	diags.Append(extraDiags...)

	if diags.HasError() {
		return supersetclient.SavedQueryCreateRequest{}, diags
	}

	return supersetclient.SavedQueryCreateRequest{
		DatabaseID:         databaseID,
		Label:              label,
		Description:        stringPointerValue(data.Description),
		Catalog:            stringPointerValue(data.Catalog),
		Schema:             stringPointerValue(data.Schema),
		SQL:                sql,
		TemplateParameters: stringPointerValue(templateParameters),
		ExtraJSON:          stringPointerValue(extraJSON),
	}, diags
}

func expandSavedQueryUpdateRequest(data savedQueryModel, current *supersetclient.SavedQuery) (supersetclient.SavedQueryUpdateRequest, diag.Diagnostics) {
	createRequest, diags := expandSavedQueryCreateRequest(data)
	if diags.HasError() {
		return supersetclient.SavedQueryUpdateRequest{}, diags
	}

	request := supersetclient.SavedQueryUpdateRequest{
		DatabaseID:         createRequest.DatabaseID,
		Label:              createRequest.Label,
		Description:        createRequest.Description,
		Catalog:            createRequest.Catalog,
		Schema:             createRequest.Schema,
		SQL:                createRequest.SQL,
		TemplateParameters: createRequest.TemplateParameters,
		ExtraJSON:          createRequest.ExtraJSON,
		IncludeDescription: includeManagedString(data.Description, stringPointerValueOrEmpty(current.Description)),
		IncludeCatalog:     includeManagedString(data.Catalog, stringPointerValueOrEmpty(current.Catalog)),
		IncludeSchema:      includeManagedString(data.Schema, stringPointerValueOrEmpty(current.Schema)),
	}

	if !data.TemplateParameters.IsNull() && !data.TemplateParameters.IsUnknown() {
		request.IncludeTemplateParameters = true
	}

	if !data.ExtraJSON.IsNull() && !data.ExtraJSON.IsUnknown() {
		request.IncludeExtraJSON = true
	}

	return request, diags
}

func flattenSavedQueryModel(current savedQueryModel, remote *supersetclient.SavedQuery) (savedQueryModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	state := current
	state.ID = types.Int64Value(remote.ID)
	state.DatabaseID = types.Int64Value(remote.Database.ID)
	state.DatabaseName = stringTypeValue(remote.Database.DatabaseName)
	state.Label = stringTypeValue(remote.Label)
	state.Description = stringTypeValue(stringPointerValueOrEmpty(remote.Description))
	state.Catalog = stringTypeValue(stringPointerValueOrEmpty(remote.Catalog))
	state.Schema = stringTypeValue(stringPointerValueOrEmpty(remote.Schema))
	state.SQL = stringTypeValue(remote.SQL)

	if current.TemplateParameters.IsNull() || current.TemplateParameters.IsUnknown() {
		state.TemplateParameters = stringTypeValue(stringPointerValueOrEmpty(remote.TemplateParameters))
	}

	if current.ExtraJSON.IsNull() || current.ExtraJSON.IsUnknown() {
		state.ExtraJSON = stringTypeValue(stringPointerValueOrEmpty(remote.ExtraJSON))
	}

	if !current.TemplateParameters.IsNull() && !current.TemplateParameters.IsUnknown() {
		templateParameters, templateDiags := normalizeOptionalJSONString(current.TemplateParameters, path.Root("template_parameters"))
		diags.Append(templateDiags...)
		state.TemplateParameters = templateParameters
	}

	if !current.ExtraJSON.IsNull() && !current.ExtraJSON.IsUnknown() {
		extraJSON, extraDiags := normalizeOptionalJSONString(current.ExtraJSON, path.Root("extra_json"))
		diags.Append(extraDiags...)
		state.ExtraJSON = extraJSON
	}

	return state, diags
}
