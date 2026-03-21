package provider

import (
	"context"
	"strings"
	"testing"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSupersetProviderMetadataAndSchema(t *testing.T) {
	t.Parallel()

	p := testSupersetProvider(t)

	var metadataResp frameworkprovider.MetadataResponse
	p.Metadata(context.Background(), frameworkprovider.MetadataRequest{}, &metadataResp)

	if metadataResp.TypeName != "superset" {
		t.Fatalf("expected provider type name superset, got %q", metadataResp.TypeName)
	}

	if metadataResp.Version != "test" {
		t.Fatalf("expected provider version test, got %q", metadataResp.Version)
	}

	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(context.Background(), frameworkprovider.SchemaRequest{}, &schemaResp)

	endpointAttr, ok := schemaResp.Schema.Attributes["endpoint"].(providerschema.StringAttribute)
	if !ok {
		t.Fatal("expected endpoint schema attribute")
	}

	if !endpointAttr.Optional {
		t.Fatal("expected endpoint to be optional")
	}

	usernameAttr, ok := schemaResp.Schema.Attributes["username"].(providerschema.StringAttribute)
	if !ok {
		t.Fatal("expected username schema attribute")
	}

	if !usernameAttr.Optional {
		t.Fatal("expected username to be optional")
	}

	passwordAttr, ok := schemaResp.Schema.Attributes["password"].(providerschema.StringAttribute)
	if !ok {
		t.Fatal("expected password schema attribute")
	}

	if !passwordAttr.Optional || !passwordAttr.Sensitive {
		t.Fatal("expected password to be optional and sensitive")
	}

	tokenAttr, ok := schemaResp.Schema.Attributes["access_token"].(providerschema.StringAttribute)
	if !ok {
		t.Fatal("expected access_token schema attribute")
	}

	if !tokenAttr.Optional || !tokenAttr.Sensitive {
		t.Fatal("expected access_token to be optional and sensitive")
	}
}

func TestSupersetProviderValidateConfig(t *testing.T) {
	ctx := context.Background()
	p := testSupersetProvider(t)
	clearProviderEnv(t)

	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResp)

	testCases := []struct {
		name      string
		config    SupersetProviderModel
		wantError string
	}{
		{
			name: "access token authentication",
			config: SupersetProviderModel{
				Endpoint:    types.StringValue("https://superset.example.com"),
				AccessToken: types.StringValue("token"),
				Username:    types.StringNull(),
				Password:    types.StringNull(),
			},
		},
		{
			name: "username and password authentication",
			config: SupersetProviderModel{
				Endpoint:    types.StringValue("https://superset.example.com"),
				Username:    types.StringValue("admin"),
				Password:    types.StringValue("secret"),
				AccessToken: types.StringNull(),
			},
		},
		{
			name: "missing endpoint",
			config: SupersetProviderModel{
				Endpoint:    types.StringNull(),
				AccessToken: types.StringValue("token"),
				Username:    types.StringNull(),
				Password:    types.StringNull(),
			},
			wantError: "endpoint must be configured",
		},
		{
			name: "missing authentication",
			config: SupersetProviderModel{
				Endpoint:    types.StringValue("https://superset.example.com"),
				AccessToken: types.StringNull(),
				Username:    types.StringNull(),
				Password:    types.StringNull(),
			},
			wantError: "Configure either `access_token` or `username` and `password`",
		},
		{
			name: "partial username and password",
			config: SupersetProviderModel{
				Endpoint:    types.StringValue("https://superset.example.com"),
				Username:    types.StringValue("admin"),
				Password:    types.StringNull(),
				AccessToken: types.StringNull(),
			},
			wantError: "`username` and `password` must both be configured",
		},
		{
			name: "conflicting authentication methods",
			config: SupersetProviderModel{
				Endpoint:    types.StringValue("https://superset.example.com"),
				Username:    types.StringValue("admin"),
				Password:    types.StringValue("secret"),
				AccessToken: types.StringValue("token"),
			},
			wantError: "`access_token` cannot be combined",
		},
		{
			name: "invalid endpoint",
			config: SupersetProviderModel{
				Endpoint:    types.StringValue("not-a-url"),
				AccessToken: types.StringValue("token"),
				Username:    types.StringNull(),
				Password:    types.StringNull(),
			},
			wantError: "endpoint must be a valid URL",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config := testProviderConfig(t, schemaResp.Schema, testCase.config)
			resp := &frameworkprovider.ValidateConfigResponse{}

			p.ValidateConfig(ctx, frameworkprovider.ValidateConfigRequest{Config: config}, resp)

			if testCase.wantError == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no validation errors, got %v", diagnosticsText(resp.Diagnostics))
				}

				return
			}

			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected validation error containing %q", testCase.wantError)
			}

			if !strings.Contains(diagnosticsText(resp.Diagnostics), testCase.wantError) {
				t.Fatalf("expected validation error containing %q, got %v", testCase.wantError, diagnosticsText(resp.Diagnostics))
			}
		})
	}
}

func TestSupersetProviderConfigureBuildsClient(t *testing.T) {
	ctx := context.Background()
	p := testSupersetProvider(t)
	clearProviderEnv(t)

	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResp)

	config := testProviderConfig(t, schemaResp.Schema, SupersetProviderModel{
		Endpoint:    types.StringValue("https://superset.example.com"),
		AccessToken: types.StringValue("token"),
		Username:    types.StringNull(),
		Password:    types.StringNull(),
	})

	resp := &frameworkprovider.ConfigureResponse{}
	p.Configure(ctx, frameworkprovider.ConfigureRequest{Config: config}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no configure errors, got %v", diagnosticsText(resp.Diagnostics))
	}

	dataSourceClient, ok := resp.DataSourceData.(*supersetclient.Client)
	if !ok {
		t.Fatalf("expected data source client, got %T", resp.DataSourceData)
	}

	if resourceClient, ok := resp.ResourceData.(*supersetclient.Client); !ok || resourceClient == nil {
		t.Fatalf("expected resource client, got %T", resp.ResourceData)
	}

	if actionClient, ok := resp.ActionData.(*supersetclient.Client); !ok || actionClient == nil {
		t.Fatalf("expected action client, got %T", resp.ActionData)
	}

	if dataSourceClient.Endpoint() != "https://superset.example.com" {
		t.Fatalf("expected configured endpoint, got %q", dataSourceClient.Endpoint())
	}

	if dataSourceClient.AccessToken() != "token" {
		t.Fatalf("expected configured access token, got %q", dataSourceClient.AccessToken())
	}
}

func TestSupersetProviderValidateConfigWithEnvFallback(t *testing.T) {
	ctx := context.Background()
	p := testSupersetProvider(t)
	clearProviderEnv(t)

	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResp)

	t.Setenv("SUPERSET_URL", "https://superset.example.com")
	t.Setenv("SUPERSET_ACCESS_TOKEN", "env-token")

	config := testProviderConfig(t, schemaResp.Schema, SupersetProviderModel{
		Endpoint:    types.StringNull(),
		AccessToken: types.StringNull(),
		Username:    types.StringNull(),
		Password:    types.StringNull(),
	})

	resp := &frameworkprovider.ValidateConfigResponse{}
	p.ValidateConfig(ctx, frameworkprovider.ValidateConfigRequest{Config: config}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no validation errors, got %v", diagnosticsText(resp.Diagnostics))
	}
}

func TestSupersetProviderValidateConfigConfigAuthTakesPrecedenceOverEnv(t *testing.T) {
	ctx := context.Background()
	p := testSupersetProvider(t)
	clearProviderEnv(t)

	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResp)

	t.Setenv("SUPERSET_ENDPOINT", "https://env.example.com")
	t.Setenv("SUPERSET_USERNAME", "env-admin")
	t.Setenv("SUPERSET_PASSWORD", "env-secret")

	config := testProviderConfig(t, schemaResp.Schema, SupersetProviderModel{
		Endpoint:    types.StringValue("https://config.example.com"),
		AccessToken: types.StringValue("config-token"),
		Username:    types.StringNull(),
		Password:    types.StringNull(),
	})

	resp := &frameworkprovider.ValidateConfigResponse{}
	p.ValidateConfig(ctx, frameworkprovider.ValidateConfigRequest{Config: config}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected config authentication to override env values, got %v", diagnosticsText(resp.Diagnostics))
	}
}

func TestSupersetProviderValidateConfigConfigUsernamePasswordTakePrecedenceOverEnvToken(t *testing.T) {
	ctx := context.Background()
	p := testSupersetProvider(t)
	clearProviderEnv(t)

	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResp)

	t.Setenv("SUPERSET_ENDPOINT", "https://env.example.com")
	t.Setenv("SUPERSET_ACCESS_TOKEN", "env-token")

	config := testProviderConfig(t, schemaResp.Schema, SupersetProviderModel{
		Endpoint:    types.StringValue("https://config.example.com"),
		AccessToken: types.StringNull(),
		Username:    types.StringValue("config-admin"),
		Password:    types.StringValue("config-secret"),
	})

	resp := &frameworkprovider.ValidateConfigResponse{}
	p.ValidateConfig(ctx, frameworkprovider.ValidateConfigRequest{Config: config}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected config username/password to override env token, got %v", diagnosticsText(resp.Diagnostics))
	}
}

func TestSupersetProviderValidateConfigMergesConfigAndEnvUsernamePassword(t *testing.T) {
	ctx := context.Background()
	p := testSupersetProvider(t)
	clearProviderEnv(t)

	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResp)

	t.Setenv("SUPERSET_ENDPOINT", "https://env.example.com")
	t.Setenv("SUPERSET_PASSWORD", "env-secret")

	config := testProviderConfig(t, schemaResp.Schema, SupersetProviderModel{
		Endpoint:    types.StringNull(),
		AccessToken: types.StringNull(),
		Username:    types.StringValue("config-admin"),
		Password:    types.StringNull(),
	})

	resp := &frameworkprovider.ValidateConfigResponse{}
	p.ValidateConfig(ctx, frameworkprovider.ValidateConfigRequest{Config: config}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected config username and env password to merge, got %v", diagnosticsText(resp.Diagnostics))
	}
}

func TestSupersetProviderConfigureBuildsClientFromEnvFallback(t *testing.T) {
	ctx := context.Background()
	p := testSupersetProvider(t)
	clearProviderEnv(t)

	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResp)

	t.Setenv("SUPERSET_URL", "https://superset.example.com")
	t.Setenv("SUPERSET_ACCESS_TOKEN", "env-token")

	config := testProviderConfig(t, schemaResp.Schema, SupersetProviderModel{
		Endpoint:    types.StringNull(),
		AccessToken: types.StringNull(),
		Username:    types.StringNull(),
		Password:    types.StringNull(),
	})

	resp := &frameworkprovider.ConfigureResponse{}
	p.Configure(ctx, frameworkprovider.ConfigureRequest{Config: config}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no configure errors, got %v", diagnosticsText(resp.Diagnostics))
	}

	dataSourceClient, ok := resp.DataSourceData.(*supersetclient.Client)
	if !ok {
		t.Fatalf("expected data source client, got %T", resp.DataSourceData)
	}

	if dataSourceClient.Endpoint() != "https://superset.example.com" {
		t.Fatalf("expected endpoint from env fallback, got %q", dataSourceClient.Endpoint())
	}

	if dataSourceClient.AccessToken() != "env-token" {
		t.Fatalf("expected access token from env fallback, got %q", dataSourceClient.AccessToken())
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	t.Run("returns configured value", func(t *testing.T) {
		t.Setenv("SUPERSET_ENDPOINT", "https://env.example.com")

		value := getEnvOrDefault(types.StringValue("https://config.example.com"), "SUPERSET_ENDPOINT")

		if value.ValueString() != "https://config.example.com" {
			t.Fatalf("expected configured value, got %q", value.ValueString())
		}
	})

	t.Run("returns first non-empty env value", func(t *testing.T) {
		t.Setenv("SUPERSET_ENDPOINT", "")
		t.Setenv("SUPERSET_URL", "https://env.example.com")

		value := getEnvOrDefault(types.StringNull(), "SUPERSET_ENDPOINT", "SUPERSET_URL")

		if value.ValueString() != "https://env.example.com" {
			t.Fatalf("expected env value, got %q", value.ValueString())
		}
	})

	t.Run("preserves unknown values", func(t *testing.T) {
		t.Setenv("SUPERSET_ENDPOINT", "https://env.example.com")

		value := getEnvOrDefault(types.StringUnknown(), "SUPERSET_ENDPOINT")

		if !value.IsUnknown() {
			t.Fatal("expected unknown value to be preserved")
		}
	})
}

func testProviderConfig(t *testing.T, schema providerschema.Schema, model SupersetProviderModel) tfsdk.Config {
	t.Helper()

	objectValue := types.ObjectValueMust(providerConfigAttributeTypes(), map[string]attr.Value{
		"endpoint":     model.Endpoint,
		"username":     model.Username,
		"password":     model.Password,
		"access_token": model.AccessToken,
	})

	rawValue, err := objectValue.ToTerraformValue(context.Background())
	if err != nil {
		t.Fatalf("expected terraform value, got error: %v", err)
	}

	return tfsdk.Config{
		Raw:    rawValue,
		Schema: schema,
	}
}

func clearProviderEnv(t *testing.T) {
	t.Helper()

	t.Setenv(providerEndpointEnv, "")
	t.Setenv(providerURLEnv, "")
	t.Setenv(providerUsernameEnv, "")
	t.Setenv(providerPasswordEnv, "")
	t.Setenv(providerAccessTokenEnv, "")
}

func testSupersetProvider(t *testing.T) *SupersetProvider {
	t.Helper()

	providerInstance, ok := New("test")().(*SupersetProvider)
	if !ok {
		t.Fatal("expected SupersetProvider")
	}

	return providerInstance
}

func providerConfigAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"endpoint":     types.StringType,
		"username":     types.StringType,
		"password":     types.StringType,
		"access_token": types.StringType,
	}
}

func diagnosticsText(diags diag.Diagnostics) string {
	parts := make([]string, 0, len(diags))

	for _, diagnostic := range diags {
		parts = append(parts, diagnostic.Summary()+": "+diagnostic.Detail())
	}

	return strings.Join(parts, "\n")
}
