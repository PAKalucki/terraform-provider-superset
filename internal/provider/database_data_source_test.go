package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatabaseDataSource(t *testing.T) {
	databaseName := fmt.Sprintf("tfacc-database-ds-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabaseDataSourceConfig(databaseName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.superset_database.lookup", "id", "superset_database.test", "id"),
					resource.TestCheckResourceAttrPair("data.superset_database.lookup", "database_name", "superset_database.test", "database_name"),
					resource.TestCheckResourceAttrPair("data.superset_database.lookup", "extra", "superset_database.test", "extra"),
					resource.TestCheckResourceAttr("data.superset_database.lookup", "backend", "postgresql"),
					resource.TestCheckResourceAttr("data.superset_database.lookup", "driver", "psycopg2"),
					resource.TestCheckResourceAttr("data.superset_database.lookup", "sqlalchemy_uri", "postgresql+psycopg2://analytics:XXXXXXXXXX@warehouse:5432/analytics"),
				),
			},
		},
	})
}

func testAccDatabaseDataSourceConfig(databaseName string) string {
	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name    = %q
  sqlalchemy_uri   = %q
  expose_in_sqllab = true
  extra = jsonencode({
    metadata_cache_timeout = {
      schema_cache_timeout = 900
    }
  })
}

data "superset_database" "lookup" {
  database_name = superset_database.test.database_name
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI())
}
