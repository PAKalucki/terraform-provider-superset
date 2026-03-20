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

	if !endpointAttr.Required {
		t.Fatal("expected endpoint to be required")
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
	t.Parallel()

	ctx := context.Background()
	p := testSupersetProvider(t)

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
			t.Parallel()

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
	t.Parallel()

	ctx := context.Background()
	p := testSupersetProvider(t)

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
