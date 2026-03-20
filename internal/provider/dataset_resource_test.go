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

func TestAccDatasetResource(t *testing.T) {
	databaseName := fmt.Sprintf("tfacc-dataset-db-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDatasetAndDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasetResourceConfig(databaseName, "Event dataset", "Event ID", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_dataset.test", "id"),
					resource.TestCheckResourceAttrSet("superset_dataset.test", "uuid"),
					resource.TestCheckResourceAttrPair("superset_dataset.test", "database_id", "superset_database.test", "id"),
					resource.TestCheckResourceAttrPair("superset_dataset.test", "database_name", "superset_database.test", "database_name"),
					resource.TestCheckResourceAttr("superset_dataset.test", "table_name", "events"),
					resource.TestCheckResourceAttr("superset_dataset.test", "schema", "analytics"),
					resource.TestCheckResourceAttr("superset_dataset.test", "description", "Event dataset"),
					resource.TestCheckResourceAttr("superset_dataset.test", "main_dttm_col", "created_at"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.#", "2"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.0.column_name", "id"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.0.verbose_name", "Event ID"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.1.column_name", "created_at"),
					resource.TestCheckResourceAttr("superset_dataset.test", "metrics.#", "1"),
					resource.TestCheckResourceAttr("superset_dataset.test", "metrics.0.metric_name", "event_count"),
				),
			},
			{
				Config: testAccDatasetResourceConfig(databaseName, "Event dataset updated", "Dataset Event ID", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_dataset.test", "description", "Event dataset updated"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.#", "3"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.0.column_name", "id"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.0.verbose_name", "Dataset Event ID"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.1.column_name", "event_name"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.2.column_name", "created_at"),
					resource.TestCheckResourceAttr("superset_dataset.test", "metrics.0.metric_name", "event_rows"),
					resource.TestCheckResourceAttr("superset_dataset.test", "metrics.0.verbose_name", "Event Rows"),
				),
			},
			{
				Config: testAccDatasetResourceClearingConfig(databaseName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("superset_dataset.test", "description"),
					resource.TestCheckNoResourceAttr("superset_dataset.test", "main_dttm_col"),
					resource.TestCheckNoResourceAttr("superset_dataset.test", "filter_select_enabled"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.#", "3"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.0.column_name", "id"),
					resource.TestCheckNoResourceAttr("superset_dataset.test", "columns.0.verbose_name"),
					resource.TestCheckNoResourceAttr("superset_dataset.test", "columns.0.filterable"),
					resource.TestCheckResourceAttr("superset_dataset.test", "metrics.#", "1"),
					resource.TestCheckResourceAttr("superset_dataset.test", "metrics.0.metric_name", "event_rows"),
					resource.TestCheckNoResourceAttr("superset_dataset.test", "metrics.0.verbose_name"),
				),
			},
			testAccImportStateStep("superset_dataset.test", "columns", "metrics"),
		},
	})
}

func testAccCheckDatasetAndDatabaseDestroy(state *terraform.State) error {
	client, err := testAccSupersetClient()
	if err != nil {
		return err
	}

	for _, resourceState := range state.RootModule().Resources {
		id, err := strconv.ParseInt(resourceState.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("parse resource id %q: %w", resourceState.Primary.ID, err)
		}

		switch resourceState.Type {
		case "superset_dataset":
			_, err = client.GetDataset(context.Background(), id)
			if err == nil {
				return fmt.Errorf("Superset dataset %d still exists", id)
			}

			if !isSupersetNotFoundError(err) {
				return err
			}
		case "superset_database":
			_, err = client.GetDatabase(context.Background(), id)
			if err == nil {
				return fmt.Errorf("Superset database %d still exists", id)
			}

			if !isSupersetNotFoundError(err) {
				return err
			}
		}
	}

	return nil
}

func testAccDatasetResourceConfig(databaseName string, description string, idVerboseName string, includeEventName bool) string {
	eventNameColumn := ""
	if includeEventName {
		eventNameColumn = `
    {
      column_name = "event_name"
      filterable  = true
      groupby     = true
      is_active   = true
      type        = "TEXT"
    },`
	}

	metricName := "event_count"
	metricVerboseName := "Event Count"
	if includeEventName {
		metricName = "event_rows"
		metricVerboseName = "Event Rows"
	}

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
  description          = %q
  main_dttm_col        = "created_at"
  filter_select_enabled = true

  columns = [
    {
      column_name = "id"
      verbose_name = %q
      filterable  = true
      groupby     = true
      is_active   = true
      type        = "INTEGER"
    },%s
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
      metric_name = %q
      expression  = "COUNT(*)"
      metric_type = "count"
      verbose_name = %q
    }
  ]
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI(), description, idVerboseName, eventNameColumn, metricName, metricVerboseName)
}

func testAccDatasetResourceClearingConfig(databaseName string) string {
	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name  = %q
  sqlalchemy_uri = %q
}

resource "superset_dataset" "test" {
  database_id = superset_database.test.id
  schema      = "analytics"
  table_name  = "events"

  columns = [
    {
      column_name = "id"
      groupby     = true
      is_active   = true
      type        = "INTEGER"
    },
    {
      column_name = "event_name"
      groupby     = true
      is_active   = true
      type        = "TEXT"
    },
    {
      column_name = "created_at"
      groupby     = true
      is_active   = true
      is_dttm     = true
      type        = "TIMESTAMP"
    }
  ]

  metrics = [
    {
      metric_name = "event_rows"
      expression  = "COUNT(*)"
      metric_type = "count"
    }
  ]
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI())
}
