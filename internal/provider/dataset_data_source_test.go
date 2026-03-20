package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatasetDataSource(t *testing.T) {
	databaseName := fmt.Sprintf("tfacc-dataset-ds-db-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDatasetAndDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasetDataSourceConfig(databaseName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.superset_dataset.lookup", "id", "superset_dataset.test", "id"),
					resource.TestCheckResourceAttrPair("data.superset_dataset.lookup", "database_id", "superset_dataset.test", "database_id"),
					resource.TestCheckResourceAttrPair("data.superset_dataset.lookup", "database_name", "superset_dataset.test", "database_name"),
					resource.TestCheckResourceAttr("data.superset_dataset.lookup", "table_name", "events"),
					resource.TestCheckResourceAttr("data.superset_dataset.lookup", "schema", "analytics"),
					resource.TestCheckResourceAttr("data.superset_dataset.lookup", "description", "Event dataset"),
					resource.TestCheckResourceAttr("data.superset_dataset.lookup", "columns.#", "2"),
					resource.TestCheckResourceAttr("data.superset_dataset.lookup", "columns.0.column_name", "created_at"),
					resource.TestCheckResourceAttr("data.superset_dataset.lookup", "columns.1.column_name", "id"),
					resource.TestCheckResourceAttr("data.superset_dataset.lookup", "metrics.#", "1"),
					resource.TestCheckResourceAttr("data.superset_dataset.lookup", "metrics.0.metric_name", "event_count"),
				),
			},
		},
	})
}

func testAccDatasetDataSourceConfig(databaseName string) string {
	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name  = %q
  sqlalchemy_uri = %q
}

resource "superset_dataset" "test" {
  database_id          = superset_database.test.id
  schema               = "analytics"
  table_name           = "events"
  description          = "Event dataset"
  main_dttm_col        = "created_at"
  filter_select_enabled = true

  columns = [
    {
      column_name = "id"
      verbose_name = "Event ID"
      filterable  = true
      groupby     = true
      is_active   = true
      type        = "INTEGER"
    },
    {
      column_name = "created_at"
      filterable  = true
      groupby     = true
      is_active   = true
      is_dttm     = true
      type        = "TIMESTAMP"
    }
  ]

  metrics = [
    {
      metric_name = "event_count"
      expression  = "COUNT(*)"
      metric_type = "count"
      verbose_name = "Event Count"
    }
  ]
}

data "superset_dataset" "lookup" {
  database_id = superset_dataset.test.database_id
  table_name  = superset_dataset.test.table_name
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI())
}
