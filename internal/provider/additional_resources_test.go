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

func TestAccSavedQueryResource(t *testing.T) {
	suffix := time.Now().UnixNano()
	databaseName := fmt.Sprintf("tfacc-saved-query-db-%d", suffix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSavedQueryAndDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSavedQueryResourceConfig(databaseName, "Saved Query One", "Saved query description", "analytics", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_saved_query.test", "id"),
					resource.TestCheckResourceAttrPair("superset_saved_query.test", "database_id", "superset_database.test", "id"),
					resource.TestCheckResourceAttrPair("superset_saved_query.test", "database_name", "superset_database.test", "database_name"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "label", "Saved Query One"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "description", "Saved query description"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "schema", "analytics"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "sql", "select 1 as id"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "template_parameters", "{\"region\":\"emea\"}"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "extra_json", "{\"editor_width\":640}"),
				),
			},
			{
				Config: testAccSavedQueryResourceConfig(databaseName, "Saved Query Two", "", "", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_saved_query.test", "label", "Saved Query Two"),
					resource.TestCheckNoResourceAttr("superset_saved_query.test", "description"),
					resource.TestCheckNoResourceAttr("superset_saved_query.test", "schema"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "sql", "select 2 as id"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "template_parameters", "{\"region\":\"apac\"}"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "extra_json", "{\"editor_width\":480}"),
				),
			},
		},
	})
}

func TestAccCSSTemplateResource(t *testing.T) {
	suffix := time.Now().UnixNano()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCSSTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCSSTemplateResourceConfig(fmt.Sprintf("tfacc-css-%d", suffix), ".dashboard { color: #203040; }"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_css_template.test", "id"),
					resource.TestCheckResourceAttr("superset_css_template.test", "template_name", fmt.Sprintf("tfacc-css-%d", suffix)),
					resource.TestCheckResourceAttr("superset_css_template.test", "css", ".dashboard { color: #203040; }"),
				),
			},
			{
				Config: testAccCSSTemplateResourceConfig(fmt.Sprintf("tfacc-css-%d-updated", suffix), ".dashboard { color: #405060; }"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_css_template.test", "template_name", fmt.Sprintf("tfacc-css-%d-updated", suffix)),
					resource.TestCheckResourceAttr("superset_css_template.test", "css", ".dashboard { color: #405060; }"),
				),
			},
		},
	})
}

func TestAccAnnotationLayerResource(t *testing.T) {
	suffix := time.Now().UnixNano()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAnnotationLayerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAnnotationLayerResourceConfig(fmt.Sprintf("tfacc-layer-%d", suffix), "Deployment markers", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_annotation_layer.test", "id"),
					resource.TestCheckResourceAttr("superset_annotation_layer.test", "name", fmt.Sprintf("tfacc-layer-%d", suffix)),
					resource.TestCheckResourceAttr("superset_annotation_layer.test", "description", "Deployment markers"),
				),
			},
			{
				Config: testAccAnnotationLayerResourceConfig(fmt.Sprintf("tfacc-layer-%d-updated", suffix), "", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_annotation_layer.test", "name", fmt.Sprintf("tfacc-layer-%d-updated", suffix)),
					resource.TestCheckNoResourceAttr("superset_annotation_layer.test", "description"),
				),
			},
		},
	})
}

func testAccCheckSavedQueryAndDatabaseDestroy(state *terraform.State) error {
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
		case "superset_saved_query":
			_, err = client.GetSavedQuery(context.Background(), id)
			if err == nil {
				return fmt.Errorf("Superset saved query %d still exists", id)
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

func testAccCheckCSSTemplateDestroy(state *terraform.State) error {
	client, err := testAccSupersetClient()
	if err != nil {
		return err
	}

	for _, resourceState := range state.RootModule().Resources {
		if resourceState.Type != "superset_css_template" {
			continue
		}

		id, err := strconv.ParseInt(resourceState.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("parse Superset CSS template id %q: %w", resourceState.Primary.ID, err)
		}

		_, err = client.GetCSSTemplate(context.Background(), id)
		if err == nil {
			return fmt.Errorf("Superset CSS template %d still exists", id)
		}

		if !isSupersetNotFoundError(err) {
			return err
		}
	}

	return nil
}

func testAccCheckAnnotationLayerDestroy(state *terraform.State) error {
	client, err := testAccSupersetClient()
	if err != nil {
		return err
	}

	for _, resourceState := range state.RootModule().Resources {
		if resourceState.Type != "superset_annotation_layer" {
			continue
		}

		id, err := strconv.ParseInt(resourceState.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("parse Superset annotation layer id %q: %w", resourceState.Primary.ID, err)
		}

		_, err = client.GetAnnotationLayer(context.Background(), id)
		if err == nil {
			return fmt.Errorf("Superset annotation layer %d still exists", id)
		}

		if !isSupersetNotFoundError(err) {
			return err
		}
	}

	return nil
}

func testAccSavedQueryResourceConfig(databaseName string, label string, description string, schema string, initial bool) string {
	descriptionLine := ""
	schemaLine := ""
	templateParameters := "{\"region\":\"apac\"}"
	extraJSON := "{\"editor_width\":480}"
	sql := "select 2 as id"

	if description != "" {
		descriptionLine = fmt.Sprintf("  description = %q\n", description)
	}

	if schema != "" {
		schemaLine = fmt.Sprintf("  schema      = %q\n", schema)
	}

	if initial {
		templateParameters = "{\"region\":\"emea\"}"
		extraJSON = "{\"editor_width\":640}"
		sql = "select 1 as id"
	}

	return fmt.Sprintf(`
%s

resource "superset_database" "test" {
  database_name  = %q
  sqlalchemy_uri = %q
}

resource "superset_saved_query" "test" {
  database_id = superset_database.test.id
  label       = %q
%s%s  sql         = %q
  template_parameters = %q
  extra_json          = %q
}
`, testAccProviderConfig(), databaseName, testAccWarehouseSQLAlchemyURI(), label, descriptionLine, schemaLine, sql, templateParameters, extraJSON)
}

func testAccCSSTemplateResourceConfig(templateName string, css string) string {
	return fmt.Sprintf(`
%s

resource "superset_css_template" "test" {
  template_name = %q
  css           = %q
}
`, testAccProviderConfig(), templateName, css)
}

func testAccAnnotationLayerResourceConfig(name string, description string, includeDescription bool) string {
	descriptionLine := ""
	if includeDescription {
		descriptionLine = fmt.Sprintf("  description = %q\n", description)
	}

	return fmt.Sprintf(`
%s

resource "superset_annotation_layer" "test" {
  name = %q
%s}
`, testAccProviderConfig(), name, descriptionLine)
}
