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

func TestAccDashboardResource(t *testing.T) {
	databaseName := fmt.Sprintf("tfacc-dashboard-db-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDashboardChartDatasetAndDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardResourceConfig(databaseName, "Operations Dashboard", "operations-dashboard", true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_dashboard.test", "id"),
					resource.TestCheckResourceAttrSet("superset_dashboard.test", "uuid"),
					resource.TestCheckResourceAttr("superset_dashboard.test", "dashboard_title", "Operations Dashboard"),
					resource.TestCheckResourceAttr("superset_dashboard.test", "slug", "operations-dashboard"),
					resource.TestCheckResourceAttr("superset_dashboard.test", "css", ".dashboard { background: #f5f3ea; }"),
					resource.TestCheckResourceAttr("superset_dashboard.test", "published", "true"),
					resource.TestCheckResourceAttr("superset_dashboard.test", "chart_ids.#", "1"),
					resource.TestCheckResourceAttrSet("superset_dashboard.test", "position_json"),
					resource.TestCheckResourceAttrSet("superset_dashboard.test", "url"),
				),
			},
			{
				Config: testAccDashboardResourceConfig(databaseName, "Operations Dashboard Updated", "", false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_dashboard.test", "dashboard_title", "Operations Dashboard Updated"),
					resource.TestCheckNoResourceAttr("superset_dashboard.test", "slug"),
					resource.TestCheckNoResourceAttr("superset_dashboard.test", "css"),
					resource.TestCheckNoResourceAttr("superset_dashboard.test", "published"),
					resource.TestCheckNoResourceAttr("superset_dashboard.test", "chart_ids.#"),
					resource.TestCheckNoResourceAttr("superset_dashboard.test", "position_json"),
				),
			},
			testAccImportStateStep("superset_dashboard.test"),
		},
	})
}

func TestAccDashboardDataSource(t *testing.T) {
	databaseName := fmt.Sprintf("tfacc-dashboard-ds-db-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDashboardChartDatasetAndDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardDataSourceConfig(databaseName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.superset_dashboard.lookup", "id", "superset_dashboard.test", "id"),
					resource.TestCheckResourceAttrPair("data.superset_dashboard.lookup", "uuid", "superset_dashboard.test", "uuid"),
					resource.TestCheckResourceAttrPair("data.superset_dashboard.lookup", "dashboard_title", "superset_dashboard.test", "dashboard_title"),
					resource.TestCheckResourceAttrPair("data.superset_dashboard.lookup", "slug", "superset_dashboard.test", "slug"),
					resource.TestCheckResourceAttr("data.superset_dashboard.lookup", "published", "true"),
					resource.TestCheckResourceAttr("data.superset_dashboard.lookup", "chart_ids.#", "1"),
					resource.TestCheckResourceAttrSet("data.superset_dashboard.lookup", "position_json"),
					resource.TestCheckResourceAttrSet("data.superset_dashboard.lookup", "url"),
				),
			},
		},
	})
}

func TestAccDashboardResourceNativeFilters(t *testing.T) {
	databaseName := fmt.Sprintf("tfacc-dashboard-filters-db-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDashboardChartDatasetAndDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardResourceNativeFiltersConfig(databaseName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_dashboard.test", "id"),
					resource.TestCheckResourceAttr("superset_dashboard.test", "show_native_filters", "true"),
					resource.TestCheckResourceAttrSet("superset_dashboard.test", "native_filter_configuration"),
				),
			},
			{
				Config: testAccDashboardResourceNativeFiltersConfig(databaseName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("superset_dashboard.test", "show_native_filters"),
					resource.TestCheckNoResourceAttr("superset_dashboard.test", "native_filter_configuration"),
				),
			},
		},
	})
}

func testAccCheckDashboardChartDatasetAndDatabaseDestroy(state *terraform.State) error {
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
		case "superset_dashboard":
			_, err = client.GetDashboard(context.Background(), strconv.FormatInt(id, 10))
			if err == nil {
				return fmt.Errorf("Superset dashboard %d still exists", id)
			}

			if !isSupersetNotFoundError(err) {
				return err
			}
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

func testAccDashboardResourceConfig(databaseName string, dashboardTitle string, slug string, includeLayout bool, includeSettings bool) string {
	slugLine := ""
	cssLine := ""
	publishedLine := ""
	chartIDsLine := ""
	positionJSONLine := ""

	if includeSettings {
		slugLine = fmt.Sprintf("  slug      = %q\n", slug)
		cssLine = "  css       = \".dashboard { background: #f5f3ea; }\"\n"
		publishedLine = "  published = true\n"
	}

	if includeLayout {
		chartIDsLine = "  chart_ids = [superset_chart.test.id]\n"
		positionJSONLine = fmt.Sprintf(`
  position_json = jsonencode({
    DASHBOARD_VERSION_KEY = "v2"
    ROOT_ID = {
      children = ["GRID_ID"]
      id       = "ROOT_ID"
      type     = "ROOT"
    }
    GRID_ID = {
      children = ["ROW-1"]
      id       = "GRID_ID"
      parents  = ["ROOT_ID"]
      type     = "GRID"
    }
    HEADER_ID = {
      id   = "HEADER_ID"
      meta = { text = %q }
      type = "HEADER"
    }
    "ROW-1" = {
      children = ["CHART-1"]
      id       = "ROW-1"
      meta     = { "0" = "ROOT_ID", background = "BACKGROUND_TRANSPARENT" }
      parents  = ["ROOT_ID", "GRID_ID"]
      type     = "ROW"
    }
    "CHART-1" = {
      children = []
      id       = "CHART-1"
      meta = {
        chartId   = superset_chart.test.id
        height    = 50
        sliceName = superset_chart.test.slice_name
        uuid      = superset_chart.test.uuid
        width     = 4
      }
      parents = ["ROOT_ID", "GRID_ID", "ROW-1"]
      type    = "CHART"
    }
  })
`, dashboardTitle)
	}

	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name  = %q
  sqlalchemy_uri = %q
}

resource "superset_dataset" "test" {
  database_id           = superset_database.test.id
  schema                = "analytics"
  table_name            = "events"
  main_dttm_col         = "created_at"
  filter_select_enabled = true
}

locals {
  datasource_uid = format("%%d__table", superset_dataset.test.id)
}

resource "superset_chart" "test" {
  slice_name    = "Dashboard chart"
  datasource_id = superset_dataset.test.id
  viz_type      = "table"
  params = jsonencode({
    datasource = local.datasource_uid
    viz_type   = "table"
  })
}

resource "superset_dashboard" "test" {
  dashboard_title = %q
%s%s%s%s}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI(), dashboardTitle, slugLine, cssLine, publishedLine, chartIDsLine+positionJSONLine)
}

func testAccDashboardDataSourceConfig(databaseName string) string {
	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name  = %q
  sqlalchemy_uri = %q
}

resource "superset_dataset" "test" {
  database_id           = superset_database.test.id
  schema                = "analytics"
  table_name            = "events"
  main_dttm_col         = "created_at"
  filter_select_enabled = true
}

locals {
  datasource_uid = format("%%d__table", superset_dataset.test.id)
}

resource "superset_chart" "test" {
  slice_name    = "Dashboard lookup chart"
  datasource_id = superset_dataset.test.id
  viz_type      = "table"
  params = jsonencode({
    datasource = local.datasource_uid
    viz_type   = "table"
  })
}

resource "superset_dashboard" "test" {
  dashboard_title = "Dashboard lookup"
  slug            = "dashboard-lookup"
  published       = true
  chart_ids       = [superset_chart.test.id]
}

data "superset_dashboard" "lookup" {
  slug = superset_dashboard.test.slug
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI())
}

func testAccDashboardResourceNativeFiltersConfig(databaseName string, includeNativeFilters bool) string {
	nativeFiltersBlock := ""
	if includeNativeFilters {
		nativeFiltersBlock = `
  show_native_filters = true
  native_filter_configuration = jsonencode([
    {
      id = "NATIVE_FILTER-event-name"
      controlValues = {
        enableEmptyFilter  = false
        defaultToFirstItem = false
        multiSelect        = true
        searchAllOptions   = false
        inverseSelection   = false
      }
      name       = "Event Name"
      filterType = "filter_select"
      targets = [
        {
          datasetId = superset_dataset.test.id
          column = {
            name = "event_name"
          }
        }
      ]
      defaultDataMask = {
        extraFormData = {}
        filterState   = {}
        ownState      = {}
      }
      cascadeParentIds = []
      scope = {
        rootPath = ["ROOT_ID"]
        excluded = []
      }
      type          = "NATIVE_FILTER"
      description   = ""
      chartsInScope = [superset_chart.test.id]
      tabsInScope   = []
    },
    {
      id = "NATIVE_FILTER-created-at"
      controlValues = {
        enableEmptyFilter = false
      }
      name       = "Created At"
      filterType = "filter_time"
      targets    = [{}]
      defaultDataMask = {
        extraFormData = {}
        filterState   = {}
        ownState      = {}
      }
      cascadeParentIds = []
      scope = {
        rootPath = ["ROOT_ID"]
        excluded = []
      }
      type          = "NATIVE_FILTER"
      description   = ""
      chartsInScope = [superset_chart.test.id]
      tabsInScope   = []
    }
  ])
`
	}

	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name  = %q
  sqlalchemy_uri = %q
}

resource "superset_dataset" "test" {
  database_id           = superset_database.test.id
  schema                = "analytics"
  table_name            = "events"
  main_dttm_col         = "created_at"
  filter_select_enabled = true
}

locals {
  datasource_uid = format("%%d__table", superset_dataset.test.id)
}

resource "superset_chart" "test" {
  slice_name    = "Dashboard filters chart"
  datasource_id = superset_dataset.test.id
  viz_type      = "table"
  params = jsonencode({
    datasource = local.datasource_uid
    viz_type   = "table"
  })
}

resource "superset_dashboard" "test" {
  dashboard_title = %q
  slug            = %q
  published       = true
  chart_ids       = [superset_chart.test.id]
%s}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI(), databaseName, databaseName, nativeFiltersBlock)
}
