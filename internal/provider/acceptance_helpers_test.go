package provider

import "github.com/hashicorp/terraform-plugin-testing/helper/resource"

func testAccImportStateStep(resourceName string, ignore ...string) resource.TestStep {
	return resource.TestStep{
		ResourceName:            resourceName,
		ImportState:             true,
		ImportStateVerify:       true,
		ImportStateVerifyIgnore: ignore,
	}
}
