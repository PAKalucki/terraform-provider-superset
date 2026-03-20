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

func TestAccChartResource(t *testing.T) {
	databaseName := fmt.Sprintf("tfacc-chart-db-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckChartDatasetAndDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccChartResourceConfig(databaseName, "Event table chart", "Warehouse chart", true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_chart.test", "id"),
					resource.TestCheckResourceAttrSet("superset_chart.test", "uuid"),
					resource.TestCheckResourceAttr("superset_chart.test", "slice_name", "Event table chart"),
					resource.TestCheckResourceAttr("superset_chart.test", "description", "Warehouse chart"),
					resource.TestCheckResourceAttrPair("superset_chart.test", "datasource_id", "superset_dataset.test", "id"),
					resource.TestCheckResourceAttr("superset_chart.test", "datasource_type", "table"),
					resource.TestCheckResourceAttr("superset_chart.test", "viz_type", "table"),
					resource.TestCheckResourceAttr("superset_chart.test", "cache_timeout", "300"),
					resource.TestCheckResourceAttrSet("superset_chart.test", "params"),
					resource.TestCheckResourceAttrSet("superset_chart.test", "query_context"),
					resource.TestCheckResourceAttrSet("superset_chart.test", "url"),
				),
			},
			{
				Config: testAccChartResourceConfig(databaseName, "Event table chart updated", "", false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_chart.test", "slice_name", "Event table chart updated"),
					resource.TestCheckNoResourceAttr("superset_chart.test", "description"),
					resource.TestCheckNoResourceAttr("superset_chart.test", "query_context"),
					resource.TestCheckNoResourceAttr("superset_chart.test", "cache_timeout"),
					resource.TestCheckResourceAttr("superset_chart.test", "datasource_type", "table"),
					resource.TestCheckResourceAttr("superset_chart.test", "viz_type", "table"),
				),
			},
			testAccImportStateStep("superset_chart.test"),
		},
	})
}

func TestAccChartDataSource(t *testing.T) {
	databaseName := fmt.Sprintf("tfacc-chart-ds-db-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckChartDatasetAndDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccChartDataSourceConfig(databaseName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.superset_chart.lookup", "id", "superset_chart.test", "id"),
					resource.TestCheckResourceAttrPair("data.superset_chart.lookup", "datasource_id", "superset_chart.test", "datasource_id"),
					resource.TestCheckResourceAttrPair("data.superset_chart.lookup", "datasource_name", "superset_chart.test", "datasource_name"),
					resource.TestCheckResourceAttr("data.superset_chart.lookup", "slice_name", "Lookup chart"),
					resource.TestCheckResourceAttr("data.superset_chart.lookup", "datasource_type", "table"),
					resource.TestCheckResourceAttr("data.superset_chart.lookup", "viz_type", "table"),
					resource.TestCheckResourceAttrSet("data.superset_chart.lookup", "params"),
					resource.TestCheckResourceAttrSet("data.superset_chart.lookup", "query_context"),
				),
			},
		},
	})
}

func testAccCheckChartDatasetAndDatabaseDestroy(state *terraform.State) error {
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
		case "superset_chart":
			_, err = client.GetChart(context.Background(), id)
			if err == nil {
				return fmt.Errorf("Superset chart %d still exists", id)
			}

			if !isSupersetNotFoundError(err) {
				return err
			}
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

func testAccChartResourceConfig(databaseName string, sliceName string, description string, includeDescription bool, includeQueryContext bool) string {
	descriptionLine := ""
	if includeDescription {
		descriptionLine = fmt.Sprintf("  description   = %q\n", description)
	}

	queryContextLine := ""
	cacheTimeoutLine := ""
	if includeQueryContext {
		queryContextLine = `
  query_context = jsonencode({
    datasource = {
      id   = superset_dataset.test.id
      type = "table"
    }
    force = false
    queries = [
      {
        filters             = []
        extras              = { having = "", where = "" }
        applied_time_extras = {}
        columns             = ["id"]
        orderby             = [["id", true]]
        annotation_layers   = []
        row_limit           = 1000
        series_limit        = 0
        order_desc          = true
        url_params          = {}
        custom_params       = {}
        custom_form_data    = {}
      }
    ]
    form_data = {
      datasource = local.datasource_uid
      viz_type   = "table"
    }
    result_format = "json"
    result_type   = "full"
  })
`
		cacheTimeoutLine = "  cache_timeout = 300\n"
	}

	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name  = %q
  sqlalchemy_uri = %q
}

resource "superset_dataset" "test" {
  database_id            = superset_database.test.id
  schema                 = "analytics"
  table_name             = "events"
  main_dttm_col          = "created_at"
  filter_select_enabled  = true
}

locals {
  datasource_uid = format("%%d__table", superset_dataset.test.id)
}

resource "superset_chart" "test" {
  slice_name    = %q
%s  datasource_id = superset_dataset.test.id
  viz_type      = "table"
  params = jsonencode({
    datasource = local.datasource_uid
    viz_type   = "table"
  })
%s%s}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI(), sliceName, descriptionLine, queryContextLine, cacheTimeoutLine)
}

func testAccChartDataSourceConfig(databaseName string) string {
	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name  = %q
  sqlalchemy_uri = %q
}

resource "superset_dataset" "test" {
  database_id            = superset_database.test.id
  schema                 = "analytics"
  table_name             = "events"
  main_dttm_col          = "created_at"
  filter_select_enabled  = true
}

locals {
  datasource_uid = format("%%d__table", superset_dataset.test.id)
}

resource "superset_chart" "test" {
  slice_name    = "Lookup chart"
  datasource_id = superset_dataset.test.id
  viz_type      = "table"
  params = jsonencode({
    datasource = local.datasource_uid
    viz_type   = "table"
  })
  query_context = jsonencode({
    datasource = {
      id   = superset_dataset.test.id
      type = "table"
    }
    force = false
    queries = [
      {
        filters             = []
        extras              = { having = "", where = "" }
        applied_time_extras = {}
        columns             = ["id"]
        orderby             = [["id", true]]
        annotation_layers   = []
        row_limit           = 1000
        series_limit        = 0
        order_desc          = true
        url_params          = {}
        custom_params       = {}
        custom_form_data    = {}
      }
    ]
    form_data = {
      datasource = local.datasource_uid
      viz_type   = "table"
    }
    result_format = "json"
    result_type   = "full"
  })
}

data "superset_chart" "lookup" {
  datasource_id = superset_chart.test.datasource_id
  slice_name    = superset_chart.test.slice_name
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI())
}
