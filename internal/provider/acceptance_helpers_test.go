package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccImportStateStep(resourceName string, ignore ...string) resource.TestStep {
	return resource.TestStep{
		ResourceName:            resourceName,
		ImportState:             true,
		ImportStateVerify:       true,
		ImportStateVerifyIgnore: ignore,
	}
}

func testAccCaptureResourceID(resourceName string, target *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %q not found in state", resourceName)
		}

		*target = resourceState.Primary.ID

		return nil
	}
}

func testAccCheckResourceIDEquals(resourceName string, expected *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %q not found in state", resourceName)
		}

		if resourceState.Primary.ID != *expected {
			return fmt.Errorf("resource %q was replaced: ID changed from %q to %q", resourceName, *expected, resourceState.Primary.ID)
		}

		return nil
	}
}
