// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccSupersetAPIDataSourceAndPutResource(t *testing.T) {
	suffix := time.Now().UnixNano()
	templateName := fmt.Sprintf("tfacc-api-put-%d", suffix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCSSTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSupersetAPIPutConfig(templateName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_api.update", "id"),
					resource.TestCheckResourceAttrSet("superset_api.update", "response_body"),
					resource.TestCheckResourceAttrSet("data.superset_api.lookup", "response_body"),
					testCheckSupersetAPICSSResponse("data.superset_api.lookup", templateName, ".dashboard { color: #506070; }"),
				),
			},
		},
	})
}

func TestAccSupersetAPIPostResource(t *testing.T) {
	suffix := time.Now().UnixNano()
	templateName := fmt.Sprintf("tfacc-api-post-%d", suffix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAndCleanupSupersetAPIPost,
		Steps: []resource.TestStep{
			{
				Config: testAccSupersetAPIPostConfig(templateName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_api.create", "id", "POST /api/v1/css_template/"),
					resource.TestCheckResourceAttrSet("superset_api.create", "response_body"),
					testCheckSupersetAPICSSResponse("superset_api.create", templateName, ".dashboard { color: #102030; }"),
				),
			},
		},
	})
}

func testCheckSupersetAPICSSResponse(resourceName string, templateName string, css string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(resourceName, "response_body", func(value string) error {
		var response struct {
			ID     int64 `json:"id"`
			Result struct {
				ID           int64  `json:"id"`
				TemplateName string `json:"template_name"`
				CSS          string `json:"css"`
			} `json:"result"`
		}

		if err := json.Unmarshal([]byte(value), &response); err != nil {
			return fmt.Errorf("decode Superset API response: %w", err)
		}

		if response.ID == 0 && response.Result.ID == 0 {
			return fmt.Errorf("Superset API response does not contain a CSS template id: %s", value)
		}

		if response.Result.TemplateName != templateName {
			return fmt.Errorf("expected template name %q, got %q", templateName, response.Result.TemplateName)
		}

		if response.Result.CSS != css {
			return fmt.Errorf("expected CSS %q, got %q", css, response.Result.CSS)
		}

		return nil
	})
}

func testAccCheckAndCleanupSupersetAPIPost(state *terraform.State) error {
	client, err := testAccSupersetClient()
	if err != nil {
		return err
	}

	for _, resourceState := range state.RootModule().Resources {
		if resourceState.Type != "superset_api" || resourceState.Primary.Attributes["method"] != supersetAPIMethodPost {
			continue
		}

		var response struct {
			ID     int64 `json:"id"`
			Result struct {
				ID int64 `json:"id"`
			} `json:"result"`
		}

		if err := json.Unmarshal([]byte(resourceState.Primary.Attributes["response_body"]), &response); err != nil {
			return fmt.Errorf("decode Superset API POST response during cleanup: %w", err)
		}

		id := response.ID
		if id == 0 {
			id = response.Result.ID
		}
		if id == 0 {
			return fmt.Errorf("Superset API POST response does not contain a CSS template id")
		}

		if _, err := client.GetCSSTemplate(context.Background(), id); err != nil {
			return fmt.Errorf("generic Superset API resource unexpectedly removed CSS template %d during destroy: %w", id, err)
		}

		if err := client.DeleteCSSTemplate(context.Background(), id); err != nil && !isSupersetNotFoundError(err) {
			return fmt.Errorf("clean up CSS template %d created by generic Superset API resource: %w", id, err)
		}
	}

	return nil
}

func testAccSupersetAPIPutConfig(templateName string) string {
	return fmt.Sprintf(`
%s

resource "superset_css_template" "target" {
  template_name = %q
  css           = ".dashboard { color: #203040; }"

  lifecycle {
    ignore_changes = [css]
  }
}

resource "superset_api" "update" {
  method = "PUT"
  path   = "/api/v1/css_template/${superset_css_template.target.id}"

  request_body = jsonencode({
    template_name = %q
    css           = ".dashboard { color: #506070; }"
  })
}

data "superset_api" "lookup" {
  path = "/api/v1/css_template/${superset_css_template.target.id}"

  depends_on = [superset_api.update]
}
`, testAccProviderConfig(), templateName, templateName)
}

func testAccSupersetAPIPostConfig(templateName string) string {
	return fmt.Sprintf(`
%s

resource "superset_api" "create" {
  method = "POST"
  path   = "/api/v1/css_template/"

  request_body = jsonencode({
    template_name = %q
    css           = ".dashboard { color: #102030; }"
  })
}
`, testAccProviderConfig(), templateName)
}
