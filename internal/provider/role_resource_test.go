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

func TestAccRoleResource(t *testing.T) {
	roleNameOne := fmt.Sprintf("tfacc-role-%d", time.Now().UnixNano())
	roleNameTwo := fmt.Sprintf("tfacc-role-%d-updated", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResourceConfig(roleNameOne),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("superset_role.test", "id"),
					resource.TestCheckResourceAttr("superset_role.test", "name", roleNameOne),
				),
			},
			{
				Config: testAccRoleResourceConfig(roleNameTwo),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_role.test", "name", roleNameTwo),
				),
			},
			testAccImportStateStep("superset_role.test"),
		},
	})
}

func TestAccRoleDataSource(t *testing.T) {
	roleName := fmt.Sprintf("tfacc-role-ds-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleDataSourceConfig(roleName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.superset_role.lookup", "id", "superset_role.test", "id"),
					resource.TestCheckResourceAttrPair("data.superset_role.lookup", "name", "superset_role.test", "name"),
					resource.TestCheckResourceAttr("data.superset_role.lookup", "user_ids.#", "0"),
					resource.TestCheckResourceAttr("data.superset_role.lookup", "group_ids.#", "0"),
					resource.TestCheckResourceAttr("data.superset_role.lookup", "permission_ids.#", "0"),
				),
			},
		},
	})
}

func TestAccPermissionDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPermissionDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.superset_permission.dashboard_read", "id"),
					resource.TestCheckResourceAttr("data.superset_permission.dashboard_read", "permission_name", "can_read"),
					resource.TestCheckResourceAttr("data.superset_permission.dashboard_read", "view_menu_name", "Dashboard"),
					resource.TestCheckResourceAttrPair("data.superset_permission.dashboard_read_by_id", "id", "data.superset_permission.dashboard_read", "id"),
					resource.TestCheckResourceAttrPair("data.superset_permission.dashboard_read_by_id", "permission_name", "data.superset_permission.dashboard_read", "permission_name"),
					resource.TestCheckResourceAttrPair("data.superset_permission.dashboard_read_by_id", "view_menu_name", "data.superset_permission.dashboard_read", "view_menu_name"),
				),
			},
		},
	})
}

func TestAccRolePermissionResource(t *testing.T) {
	roleName := fmt.Sprintf("tfacc-role-permissions-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRolePermissionResourceConfig(roleName, true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("superset_role_permission.test", "id", "superset_role.test", "id"),
					resource.TestCheckResourceAttrPair("superset_role_permission.test", "role_name", "superset_role.test", "name"),
					resource.TestCheckResourceAttr("superset_role_permission.test", "permission_ids.#", "2"),
					resource.TestCheckTypeSetElemAttrPair("superset_role_permission.test", "permission_ids.*", "data.superset_permission.dashboard_read", "id"),
					resource.TestCheckTypeSetElemAttrPair("superset_role_permission.test", "permission_ids.*", "data.superset_permission.log_read", "id"),
					resource.TestCheckResourceAttrPair("data.superset_role.lookup", "id", "superset_role.test", "id"),
					resource.TestCheckResourceAttr("data.superset_role.lookup", "permission_ids.#", "2"),
					resource.TestCheckTypeSetElemAttrPair("data.superset_role.lookup", "permission_ids.*", "data.superset_permission.dashboard_read", "id"),
					resource.TestCheckTypeSetElemAttrPair("data.superset_role.lookup", "permission_ids.*", "data.superset_permission.log_read", "id"),
				),
			},
			{
				Config: testAccRolePermissionResourceConfig(roleName, true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_role_permission.test", "permission_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair("superset_role_permission.test", "permission_ids.*", "data.superset_permission.dashboard_read", "id"),
					resource.TestCheckResourceAttr("data.superset_role.lookup", "permission_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair("data.superset_role.lookup", "permission_ids.*", "data.superset_permission.dashboard_read", "id"),
				),
			},
			{
				Config: testAccRolePermissionResourceConfig(roleName, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_role_permission.test", "permission_ids.#", "0"),
					resource.TestCheckResourceAttr("data.superset_role.lookup", "permission_ids.#", "0"),
				),
			},
			testAccImportStateStep("superset_role_permission.test"),
		},
	})
}

func testAccCheckRoleDestroy(state *terraform.State) error {
	client, err := testAccSupersetClient()
	if err != nil {
		return err
	}

	for _, resourceState := range state.RootModule().Resources {
		if resourceState.Type != "superset_role" {
			continue
		}

		id, err := strconv.ParseInt(resourceState.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("parse Superset role id %q: %w", resourceState.Primary.ID, err)
		}

		_, err = client.GetRole(context.Background(), id)
		if err == nil {
			return fmt.Errorf("Superset role %d still exists", id)
		}

		if !isSupersetNotFoundError(err) {
			return err
		}
	}

	return nil
}

func testAccRoleResourceConfig(roleName string) string {
	return fmt.Sprintf(`
%s

resource "superset_role" "test" {
  name = %q
}
`, testAccProviderConfig(), roleName)
}

func testAccRoleDataSourceConfig(roleName string) string {
	return fmt.Sprintf(`
%s

resource "superset_role" "test" {
  name = %q
}

data "superset_role" "lookup" {
  name = superset_role.test.name
}
`, testAccProviderConfig(), roleName)
}

func testAccPermissionDataSourceConfig() string {
	return fmt.Sprintf(`
%s

data "superset_permission" "dashboard_read" {
  permission_name = "can_read"
  view_menu_name  = "Dashboard"
}

data "superset_permission" "dashboard_read_by_id" {
  id = data.superset_permission.dashboard_read.id
}
`, testAccProviderConfig())
}

func testAccRolePermissionResourceConfig(roleName string, includeDashboardRead bool, includeLogRead bool) string {
	permissionIDs := ""

	switch {
	case includeDashboardRead && includeLogRead:
		permissionIDs = "data.superset_permission.dashboard_read.id,\n    data.superset_permission.log_read.id"
	case includeDashboardRead:
		permissionIDs = "data.superset_permission.dashboard_read.id"
	case includeLogRead:
		permissionIDs = "data.superset_permission.log_read.id"
	}

	return fmt.Sprintf(`
%s

data "superset_permission" "dashboard_read" {
  permission_name = "can_read"
  view_menu_name  = "Dashboard"
}

data "superset_permission" "log_read" {
  permission_name = "can_read"
  view_menu_name  = "Log"
}

resource "superset_role" "test" {
  name = %q
}

resource "superset_role_permission" "test" {
  role_id = superset_role.test.id
  permission_ids = [%s]
}

data "superset_role" "lookup" {
  id         = superset_role.test.id
  depends_on = [superset_role_permission.test]
}
`, testAccProviderConfig(), roleName, permissionIDs)
}
