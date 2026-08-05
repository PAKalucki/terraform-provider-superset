// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func TestSupersetAPIProviderRegistration(t *testing.T) {
	t.Parallel()

	providerInstance := testSupersetProvider(t)

	resourceRegistered := false
	for _, factory := range providerInstance.Resources(context.Background()) {
		var resp resource.MetadataResponse
		factory().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "superset"}, &resp)
		if resp.TypeName == "superset_api" {
			resourceRegistered = true
		}
	}

	if !resourceRegistered {
		t.Fatal("expected superset_api resource to be registered")
	}

	dataSourceRegistered := false
	for _, factory := range providerInstance.DataSources(context.Background()) {
		var resp datasource.MetadataResponse
		factory().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "superset"}, &resp)
		if resp.TypeName == "superset_api" {
			dataSourceRegistered = true
		}
	}

	if !dataSourceRegistered {
		t.Fatal("expected superset_api data source to be registered")
	}
}

func TestSupersetAPIResourceSchema(t *testing.T) {
	t.Parallel()

	resourceInstance := NewSupersetAPIResource()
	var resp resource.SchemaResponse
	resourceInstance.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	method, ok := resp.Schema.Attributes["method"].(resourceschema.StringAttribute)
	if !ok || !method.Required || len(method.Validators) == 0 || !hasStringPlanModifier(method.PlanModifiers, stringplanmodifier.RequiresReplace()) {
		t.Fatal("expected required validated replacement method attribute")
	}

	requestPath, ok := resp.Schema.Attributes["path"].(resourceschema.StringAttribute)
	if !ok || !requestPath.Required || len(requestPath.Validators) == 0 || !hasStringPlanModifier(requestPath.PlanModifiers, stringplanmodifier.RequiresReplace()) {
		t.Fatal("expected required validated replacement path attribute")
	}

	requestBody, ok := resp.Schema.Attributes["request_body"].(resourceschema.StringAttribute)
	if !ok || !requestBody.Optional || !requestBody.Sensitive || len(requestBody.Validators) == 0 {
		t.Fatal("expected optional sensitive validated request_body attribute")
	}

	responseBody, ok := resp.Schema.Attributes["response_body"].(resourceschema.StringAttribute)
	if !ok || !responseBody.Computed || !responseBody.Sensitive {
		t.Fatal("expected computed sensitive response_body attribute")
	}
}

func TestSupersetAPIDataSourceSchema(t *testing.T) {
	t.Parallel()

	dataSourceInstance := NewSupersetAPIDataSource()
	var resp datasource.SchemaResponse
	dataSourceInstance.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	requestPath, ok := resp.Schema.Attributes["path"].(datasourceschema.StringAttribute)
	if !ok || !requestPath.Required || len(requestPath.Validators) == 0 {
		t.Fatal("expected required validated path attribute")
	}

	responseBody, ok := resp.Schema.Attributes["response_body"].(datasourceschema.StringAttribute)
	if !ok || !responseBody.Computed || !responseBody.Sensitive {
		t.Fatal("expected computed sensitive response_body attribute")
	}
}
