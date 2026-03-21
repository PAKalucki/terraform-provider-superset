// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/url"
	"os"
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure SupersetProvider satisfies various provider interfaces.
var _ provider.Provider = &SupersetProvider{}
var _ provider.ProviderWithValidateConfig = &SupersetProvider{}

const (
	providerEndpointEnv    = "SUPERSET_ENDPOINT"
	providerURLEnv         = "SUPERSET_URL"
	providerUsernameEnv    = "SUPERSET_USERNAME"
	providerPasswordEnv    = "SUPERSET_PASSWORD"
	providerAccessTokenEnv = "SUPERSET_ACCESS_TOKEN"
)

// SupersetProvider defines the provider implementation.
type SupersetProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// SupersetProviderModel describes the provider data model.
type SupersetProviderModel struct {
	Endpoint    types.String `tfsdk:"endpoint"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	AccessToken types.String `tfsdk:"access_token"`
}

func (p *SupersetProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "superset"
	resp.Version = p.version
}

func (p *SupersetProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for Apache Superset.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Superset base URL, for example `https://superset.example.com`. When omitted, the provider uses `SUPERSET_ENDPOINT` or `SUPERSET_URL`.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Superset username used for API login. Configure with `password` when `access_token` is not provided. When omitted, the provider uses `SUPERSET_USERNAME`.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Superset password used for API login. Configure with `username` when `access_token` is not provided. When omitted, the provider uses `SUPERSET_PASSWORD`.",
				Optional:            true,
				Sensitive:           true,
			},
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Superset API bearer token. Configure this instead of `username` and `password` when a token is already available. When omitted, the provider uses `SUPERSET_ACCESS_TOKEN`.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *SupersetProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var data SupersetProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateProviderModel(resolveProviderModel(data))...)
}

func (p *SupersetProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data SupersetProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data = resolveProviderModel(data)

	resp.Diagnostics.Append(validateProviderModel(data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := supersetclient.New(supersetclient.Config{
		Endpoint:    stringValue(data.Endpoint),
		Username:    stringValue(data.Username),
		Password:    stringValue(data.Password),
		AccessToken: stringValue(data.AccessToken),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Configure Superset Client",
			err.Error(),
		)

		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
	resp.EphemeralResourceData = client
	resp.ActionData = client
}

func (p *SupersetProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDatabaseResource,
		NewDatasetResource,
		NewChartResource,
		NewDashboardResource,
		NewRoleResource,
		NewRolePermissionResource,
		NewUserResource,
		NewSavedQueryResource,
		NewCSSTemplateResource,
		NewAnnotationLayerResource,
	}
}

func (p *SupersetProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDatabaseDataSource,
		NewDatasetDataSource,
		NewChartDataSource,
		NewDashboardDataSource,
		NewRoleDataSource,
		NewPermissionDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SupersetProvider{
			version: version,
		}
	}
}

func validateProviderModel(data SupersetProviderModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if data.Endpoint.IsNull() || isBlankString(data.Endpoint) {
		diags.AddAttributeError(
			path.Root("endpoint"),
			"Missing Superset Endpoint",
			"The provider endpoint must be configured.",
		)
	} else if !data.Endpoint.IsUnknown() {
		parsed, err := url.Parse(strings.TrimSpace(data.Endpoint.ValueString()))
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			diags.AddAttributeError(
				path.Root("endpoint"),
				"Invalid Superset Endpoint",
				"The provider endpoint must be a valid URL, for example https://superset.example.com.",
			)
		}
	}

	if data.AccessToken.IsUnknown() || data.Username.IsUnknown() || data.Password.IsUnknown() {
		return diags
	}

	accessTokenSet := hasStringValue(data.AccessToken)
	usernameSet := hasStringValue(data.Username)
	passwordSet := hasStringValue(data.Password)

	if accessTokenSet && (usernameSet || passwordSet) {
		diags.AddAttributeError(
			path.Root("access_token"),
			"Conflicting Superset Authentication Settings",
			"`access_token` cannot be combined with `username` or `password`.",
		)
	}

	if accessTokenSet {
		return diags
	}

	if usernameSet != passwordSet {
		diags.AddAttributeError(
			path.Root("password"),
			"Incomplete Superset Authentication Settings",
			"`username` and `password` must both be configured.",
		)

		return diags
	}

	if !usernameSet && !passwordSet {
		diags.AddAttributeError(
			path.Root("access_token"),
			"Missing Superset Authentication Settings",
			"Configure either `access_token` or `username` and `password`.",
		)
	}

	return diags
}

func resolveProviderModel(data SupersetProviderModel) SupersetProviderModel {
	resolved := SupersetProviderModel{
		Endpoint: getEnvOrDefault(data.Endpoint, providerEndpointEnv, providerURLEnv),
	}

	switch {
	case data.AccessToken.IsUnknown() || hasStringValue(data.AccessToken):
		resolved.AccessToken = getEnvOrDefault(data.AccessToken, providerAccessTokenEnv)
		resolved.Username = data.Username
		resolved.Password = data.Password
	case data.Username.IsUnknown() || data.Password.IsUnknown() || hasStringValue(data.Username) || hasStringValue(data.Password):
		resolved.AccessToken = data.AccessToken
		resolved.Username = getEnvOrDefault(data.Username, providerUsernameEnv)
		resolved.Password = getEnvOrDefault(data.Password, providerPasswordEnv)
	default:
		resolved.AccessToken = getEnvOrDefault(data.AccessToken, providerAccessTokenEnv)
		resolved.Username = getEnvOrDefault(data.Username, providerUsernameEnv)
		resolved.Password = getEnvOrDefault(data.Password, providerPasswordEnv)
	}

	return resolved
}

func getEnvOrDefault(value types.String, envNames ...string) types.String {
	if value.IsUnknown() || hasStringValue(value) {
		return value
	}

	for _, envName := range envNames {
		envValue := strings.TrimSpace(os.Getenv(envName))
		if envValue != "" {
			return types.StringValue(envValue)
		}
	}

	return value
}

func hasStringValue(value types.String) bool {
	return !value.IsNull() && !value.IsUnknown() && strings.TrimSpace(value.ValueString()) != ""
}

func isBlankString(value types.String) bool {
	return !value.IsNull() && !value.IsUnknown() && strings.TrimSpace(value.ValueString()) == ""
}

func stringValue(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}

	return value.ValueString()
}

func int64Value(value types.Int64) int64 {
	if value.IsNull() || value.IsUnknown() {
		return 0
	}

	return value.ValueInt64()
}

func boolValue(value types.Bool) bool {
	if value.IsNull() || value.IsUnknown() {
		return false
	}

	return value.ValueBool()
}
