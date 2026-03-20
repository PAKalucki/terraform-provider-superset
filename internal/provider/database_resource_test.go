package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDatabaseResource(t *testing.T) {
	databaseNameOne := fmt.Sprintf("tfacc-database-%d", time.Now().UnixNano())
	databaseNameTwo := fmt.Sprintf("tfacc-database-%d-updated", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabaseResourceConfig(databaseNameOne, true, 600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_database.test", "id"),
					resource.TestCheckResourceAttrSet("superset_database.test", "uuid"),
					resource.TestCheckResourceAttr("superset_database.test", "database_name", databaseNameOne),
					resource.TestCheckResourceAttr("superset_database.test", "sqlalchemy_uri", testAccWarehouseSQLAlchemyURI()),
					resource.TestCheckResourceAttr("superset_database.test", "backend", "postgresql"),
					resource.TestCheckResourceAttr("superset_database.test", "driver", "psycopg2"),
					resource.TestCheckResourceAttr("superset_database.test", "expose_in_sqllab", "true"),
					resource.TestCheckResourceAttr("superset_database.test", "extra", `{"metadata_cache_timeout":{"schema_cache_timeout":600}}`),
				),
			},
			{
				Config: testAccDatabaseResourceConfig(databaseNameTwo, false, 1200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_database.test", "database_name", databaseNameTwo),
					resource.TestCheckResourceAttr("superset_database.test", "sqlalchemy_uri", testAccWarehouseSQLAlchemyURI()),
					resource.TestCheckResourceAttr("superset_database.test", "backend", "postgresql"),
					resource.TestCheckResourceAttr("superset_database.test", "driver", "psycopg2"),
					resource.TestCheckResourceAttr("superset_database.test", "expose_in_sqllab", "false"),
					resource.TestCheckResourceAttr("superset_database.test", "extra", `{"metadata_cache_timeout":{"schema_cache_timeout":1200}}`),
				),
			},
		},
	})
}

func testAccCheckDatabaseDestroy(state *terraform.State) error {
	client, err := testAccSupersetClient()
	if err != nil {
		return err
	}

	for _, resourceState := range state.RootModule().Resources {
		if resourceState.Type != "superset_database" {
			continue
		}

		id, err := strconv.ParseInt(resourceState.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("parse Superset database id %q: %w", resourceState.Primary.ID, err)
		}

		_, err = client.GetDatabase(context.Background(), id)
		if err == nil {
			return fmt.Errorf("Superset database %d still exists", id)
		}

		if !isSupersetNotFoundError(err) {
			return err
		}
	}

	return nil
}

func testAccDatabaseResourceConfig(databaseName string, exposeInSQLLab bool, schemaCacheTimeout int) string {
	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name    = %q
  sqlalchemy_uri   = %q
  expose_in_sqllab = %t
  extra = jsonencode({
    metadata_cache_timeout = {
      schema_cache_timeout = %d
    }
  })
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI(), exposeInSQLLab, schemaCacheTimeout)
}

func testAccWarehouseSQLAlchemyURI() string {
	return "postgresql+psycopg2://analytics:analytics@warehouse:5432/analytics"
}
