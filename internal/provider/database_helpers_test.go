package provider

import (
	"testing"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNormalizeJSONString(t *testing.T) {
	t.Parallel()

	got, err := normalizeJSONString("{\n  \"b\": 2,\n  \"a\": 1\n}")
	if err != nil {
		t.Fatalf("expected JSON to normalize, got error: %v", err)
	}

	want := "{\"a\":1,\"b\":2}"
	if got != want {
		t.Fatalf("expected normalized JSON %q, got %q", want, got)
	}
}

func TestExpandDatabaseRequestRejectsInvalidExtra(t *testing.T) {
	t.Parallel()

	_, diags := expandDatabaseRequest(databaseModel{
		DatabaseName:  types.StringValue("analytics"),
		SQLAlchemyURI: types.StringValue("postgresql+psycopg2://analytics:analytics@warehouse:5432/analytics"),
		Extra:         types.StringValue("{"),
	})

	if !diags.HasError() {
		t.Fatal("expected invalid extra JSON to fail validation")
	}
}

func TestFlattenDatabaseModelPreservesConfiguredSQLAlchemyURI(t *testing.T) {
	t.Parallel()

	current := databaseModel{
		SQLAlchemyURI: types.StringValue("postgresql+psycopg2://analytics:analytics@warehouse:5432/analytics"),
	}

	cacheTimeout := int64(600)
	exposeInSQLLab := true

	remote := &supersetclient.Database{
		ID:             42,
		UUID:           "6a1476d0-86e7-48a5-8e85-6f1e9fef73fb",
		DatabaseName:   "analytics",
		SQLAlchemyURI:  "postgresql+psycopg2://analytics:XXXXXXXXXX@warehouse:5432/analytics",
		Extra:          "{\n  \"metadata_cache_timeout\": {\n    \"schema_cache_timeout\": 600\n  }\n}",
		ExposeInSQLLab: &exposeInSQLLab,
		CacheTimeout:   &cacheTimeout,
		Backend:        "postgresql",
		Driver:         "psycopg2",
	}

	state, diags := flattenDatabaseModel(current, remote)
	if diags.HasError() {
		t.Fatalf("expected flatten to succeed, got diagnostics: %v", diags)
	}

	if got := state.SQLAlchemyURI.ValueString(); got != current.SQLAlchemyURI.ValueString() {
		t.Fatalf("expected configured SQLAlchemy URI to be preserved, got %q", got)
	}

	if got := state.Extra.ValueString(); got != "{\"metadata_cache_timeout\":{\"schema_cache_timeout\":600}}" {
		t.Fatalf("expected normalized extra JSON, got %q", got)
	}

	if got := state.Backend.ValueString(); got != "postgresql" {
		t.Fatalf("expected backend to be populated, got %q", got)
	}

	if got := state.CacheTimeout.ValueInt64(); got != 600 {
		t.Fatalf("expected cache timeout to be populated, got %d", got)
	}
}
