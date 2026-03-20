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

func TestAccUserResource(t *testing.T) {
	suffix := time.Now().UnixNano()
	username := fmt.Sprintf("tfacc_user_%d", suffix)
	emailOne := fmt.Sprintf("tfacc_user_%d@example.com", suffix)
	emailTwo := fmt.Sprintf("tfacc_user_%d_updated@example.com", suffix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig(username, "Terraform", "User", emailOne, true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_user.test", "id"),
					resource.TestCheckResourceAttr("superset_user.test", "username", username),
					resource.TestCheckResourceAttr("superset_user.test", "first_name", "Terraform"),
					resource.TestCheckResourceAttr("superset_user.test", "last_name", "User"),
					resource.TestCheckResourceAttr("superset_user.test", "email", emailOne),
					resource.TestCheckResourceAttr("superset_user.test", "active", "true"),
					resource.TestCheckResourceAttr("superset_user.test", "role_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair("superset_user.test", "role_ids.*", "data.superset_role.alpha", "id"),
				),
			},
			{
				Config: testAccUserResourceConfig(username, "TerraformUpdated", "User", emailTwo, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_user.test", "first_name", "TerraformUpdated"),
					resource.TestCheckResourceAttr("superset_user.test", "email", emailTwo),
					resource.TestCheckResourceAttr("superset_user.test", "active", "false"),
					resource.TestCheckResourceAttr("superset_user.test", "role_ids.#", "2"),
					resource.TestCheckTypeSetElemAttrPair("superset_user.test", "role_ids.*", "data.superset_role.alpha", "id"),
					resource.TestCheckTypeSetElemAttrPair("superset_user.test", "role_ids.*", "data.superset_role.gamma", "id"),
				),
			},
		},
	})
}

func testAccCheckUserDestroy(state *terraform.State) error {
	client, err := testAccSupersetClient()
	if err != nil {
		return err
	}

	for _, resourceState := range state.RootModule().Resources {
		if resourceState.Type != "superset_user" {
			continue
		}

		id, err := strconv.ParseInt(resourceState.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("parse Superset user id %q: %w", resourceState.Primary.ID, err)
		}

		_, err = client.GetUser(context.Background(), id)
		if err == nil {
			return fmt.Errorf("Superset user %d still exists", id)
		}

		if !isSupersetNotFoundError(err) {
			return err
		}
	}

	return nil
}

func testAccUserResourceConfig(username string, firstName string, lastName string, email string, active bool, alphaOnly bool) string {
	roleIDs := "data.superset_role.alpha.id,\n    data.superset_role.gamma.id"
	if alphaOnly {
		roleIDs = "data.superset_role.alpha.id"
	}

	return fmt.Sprintf(`
%s

data "superset_role" "alpha" {
  name = "Alpha"
}

data "superset_role" "gamma" {
  name = "Gamma"
}

resource "superset_user" "test" {
  username   = %q
  first_name = %q
  last_name  = %q
  email      = %q
  active     = %t
  password   = "Terraform123!"
  role_ids = [
    %s
  ]
}
`, testAccProviderConfig(), username, firstName, lastName, email, active, roleIDs)
}
