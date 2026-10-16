package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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

func TestAccVirtualDatasetResourceUpdatesSQLInPlace(t *testing.T) {
	databaseName := fmt.Sprintf("tfacc-virtual-dataset-db-%d", time.Now().UnixNano())

	initialSQL := "SELECT id, created_at FROM analytics.events"
	updatedSQL := "SELECT id, event_name, created_at FROM analytics.events WHERE event_name IS NOT NULL"

	var createdDatasetID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDatasetAndDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualDatasetResourceConfig(databaseName, initialSQL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_dataset.test", "id"),
					resource.TestCheckResourceAttr("superset_dataset.test", "table_name", "events_virtual"),
					resource.TestCheckResourceAttr("superset_dataset.test", "sql", initialSQL),
					testAccCaptureResourceID("superset_dataset.test", &createdDatasetID),
				),
			},
			{
				Config: testAccVirtualDatasetResourceConfig(databaseName, updatedSQL),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("superset_dataset.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_dataset.test", "sql", updatedSQL),
					testAccCheckResourceIDEquals("superset_dataset.test", &createdDatasetID),
				),
			},
			testAccImportStateStep("superset_dataset.test"),
		},
	})
}

// TestAccVirtualDatasetResourceImportPreservesSQL adopts a pre-existing virtual
// dataset with `terraform import`, keeps the imported state, and verifies that
// a configuration matching the remote query plans no changes. If import drops
// the remote SQL, the sql plan modifier treats the configured query as a
// physical-to-virtual conversion and forces an unnecessary replacement.
func TestAccVirtualDatasetResourceImportPreservesSQL(t *testing.T) {
	databaseName := fmt.Sprintf("tfacc-virtual-dataset-import-db-%d", time.Now().UnixNano())
	tableName := "events_virtual_adopted"
	sql := "SELECT id, created_at FROM analytics.events"

	var databaseID string
	var existingDatasetID string

	config := testAccAdoptedVirtualDatasetResourceConfig(databaseName, tableName, sql)

	// The imported dataset only records its dependency on the database once an
	// apply succeeds. If the test fails before that, Terraform may destroy the
	// database before the dataset and leave both behind, so clean up explicitly.
	t.Cleanup(func() { testAccDeleteDatasetAndDatabase(t, &existingDatasetID, &databaseID) })

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDatasetAndDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				// Manage only the database so the dataset can be created outside Terraform.
				Config: testAccDatabaseOnlyConfig(databaseName),
				Check:  testAccCaptureResourceID("superset_database.test", &databaseID),
			},
			{
				// Create the virtual dataset directly in Superset, then adopt it with terraform import.
				PreConfig: func() {
					existingDatasetID = testAccCreateVirtualDataset(t, databaseID, "analytics", tableName, sql)
				},
				Config:       config,
				ResourceName: "superset_dataset.adopted",
				ImportState:  true,
				ImportStateIdFunc: func(*terraform.State) (string, error) {
					return existingDatasetID, nil
				},
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					for _, state := range states {
						if state.Ephemeral.Type == "superset_dataset" && state.ID == existingDatasetID {
							return nil
						}
					}

					return fmt.Errorf("imported state does not contain superset_dataset %s", existingDatasetID)
				},
			},
			{
				// The configuration matches the imported dataset, so the plan must be a no-op.
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckResourceIDEquals("superset_dataset.adopted", &existingDatasetID),
					resource.TestCheckResourceAttr("superset_dataset.adopted", "table_name", tableName),
					resource.TestCheckResourceAttr("superset_dataset.adopted", "sql", sql),
				),
			},
		},
	})
}

func TestAccDatasetResourcePreservesOmittedColumnBoolsOnCreate(t *testing.T) {
	databaseName := fmt.Sprintf("tfacc-dataset-omitted-bools-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDatasetAndDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasetResourceOmittedColumnBoolsConfig(databaseName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_dataset.test", "id"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.#", "2"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.0.column_name", "id"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.0.filterable", "true"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.0.groupby", "true"),
					resource.TestCheckResourceAttr("superset_dataset.test", "columns.1.column_name", "pipeline_duration_s"),
					resource.TestCheckNoResourceAttr("superset_dataset.test", "columns.1.filterable"),
					resource.TestCheckNoResourceAttr("superset_dataset.test", "columns.1.groupby"),
				),
			},
		},
	})
}

func TestAccDatasetResourcePreservesAutomaticDatetime(t *testing.T) {
	databaseName := fmt.Sprintf("tfacc-dataset-datetime-%d", time.Now().UnixNano())
	checkDatetime := func(expected string) resource.TestCheckFunc {
		return func(state *terraform.State) error {
			client, err := testAccSupersetClient()
			if err != nil {
				return err
			}
			id, err := strconv.ParseInt(state.RootModule().Resources["superset_dataset.test"].Primary.ID, 10, 64)
			if err != nil {
				return err
			}
			dataset, err := client.GetDataset(context.Background(), id)
			if err != nil {
				return err
			}
			if dataset.MainDttmCol != expected {
				return fmt.Errorf("expected remote main_dttm_col %q, got %q", expected, dataset.MainDttmCol)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDatasetAndDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasetResourceAutomaticDatetimeConfig(databaseName, "Initial description", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("superset_dataset.test", "main_dttm_col"),
					checkDatetime("created_at"),
				),
			},
			{
				Config: testAccDatasetResourceAutomaticDatetimeConfig(databaseName, "Updated description", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_dataset.test", "description", "Updated description"),
					resource.TestCheckNoResourceAttr("superset_dataset.test", "main_dttm_col"),
					checkDatetime("created_at"),
				),
			},
			{
				Config: testAccDatasetResourceAutomaticDatetimeConfig(databaseName, "Updated description", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_dataset.test", "main_dttm_col", "created_at"),
					checkDatetime("created_at"),
				),
			},
			{
				Config: testAccDatasetResourceAutomaticDatetimeConfig(databaseName, "Updated description", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("superset_dataset.test", "main_dttm_col"),
					checkDatetime(""),
				),
			},
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

func testAccVirtualDatasetResourceConfig(databaseName string, sql string) string {
	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name  = %q
  sqlalchemy_uri = %q
}

resource "superset_dataset" "test" {
  database_id = superset_database.test.id
  schema      = "analytics"
  table_name  = "events_virtual"
  sql         = %q
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI(), sql)
}

func testAccDatabaseOnlyConfig(databaseName string) string {
	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name  = %q
  sqlalchemy_uri = %q
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI())
}

func testAccAdoptedVirtualDatasetResourceConfig(databaseName string, tableName string, sql string) string {
	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name  = %q
  sqlalchemy_uri = %q
}

resource "superset_dataset" "adopted" {
  database_id = superset_database.test.id
  schema      = "analytics"
  table_name  = %q
  sql         = %q
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI(), tableName, sql)
}

// testAccCreateVirtualDataset creates a virtual dataset directly through the
// Superset API, simulating a dataset that pre-dates Terraform management.
func testAccCreateVirtualDataset(t *testing.T, databaseID string, schema string, tableName string, sql string) string {
	t.Helper()

	client, err := testAccSupersetClient()
	if err != nil {
		t.Fatalf("configure Superset client: %v", err)
	}

	parsedDatabaseID, err := strconv.ParseInt(databaseID, 10, 64)
	if err != nil {
		t.Fatalf("parse database id %q: %v", databaseID, err)
	}

	datasetID, err := client.CreateDataset(context.Background(), supersetclient.DatasetCreateRequest{
		Database:  parsedDatabaseID,
		TableName: tableName,
		Schema:    &schema,
		SQL:       &sql,
	})
	if err != nil {
		t.Fatalf("create virtual dataset outside Terraform: %v", err)
	}

	return strconv.FormatInt(datasetID, 10)
}

// testAccDeleteDatasetAndDatabase removes the dataset and then the database if
// they still exist. Resources already removed by Terraform are ignored.
func testAccDeleteDatasetAndDatabase(t *testing.T, datasetID *string, databaseID *string) {
	t.Helper()

	client, err := testAccSupersetClient()
	if err != nil {
		t.Logf("cleanup: configure Superset client: %v", err)
		return
	}

	deleteByID := func(kind string, rawID string, deleteFunc func(context.Context, int64) error) {
		if rawID == "" {
			return
		}

		id, err := strconv.ParseInt(rawID, 10, 64)
		if err != nil {
			t.Logf("cleanup: parse %s id %q: %v", kind, rawID, err)
			return
		}

		if err := deleteFunc(context.Background(), id); err != nil && !isSupersetNotFoundError(err) {
			t.Logf("cleanup: delete %s %d: %v", kind, id, err)
		}
	}

	deleteByID("dataset", *datasetID, client.DeleteDataset)
	deleteByID("database", *databaseID, client.DeleteDatabase)
}

func testAccDatasetResourceOmittedColumnBoolsConfig(databaseName string) string {
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
  normalize_columns     = true
  filter_select_enabled = true

  columns = [
    {
      column_name = "id"
      filterable  = true
      groupby     = true
      is_active   = true
      type        = "INTEGER"
    },
    {
      column_name = "pipeline_duration_s"
      is_active   = true
      type        = "DOUBLE"
    }
  ]

  metrics = [
    {
      metric_name = "run_count"
      expression  = "COUNT(*)"
      metric_type = "count"
    }
  ]
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI())
}

func testAccDatasetResourceAutomaticDatetimeConfig(databaseName string, description string, manageDatetime bool) string {
	mainDttmCol := ""
	if manageDatetime {
		mainDttmCol = `main_dttm_col = "created_at"`
	}
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
  description = %q
  %s
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI(), description, mainDttmCol)
}
