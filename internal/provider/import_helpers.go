package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func importInt64Attributes(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse, attributeNames ...string) {
	if req.ID == "" {
		resp.Diagnostics.AddError(
			"Missing Import Identifier",
			"Terraform did not supply an import identifier for this resource.",
		)

		return
	}

	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import Identifier",
			fmt.Sprintf("Expected a numeric Superset identifier, got %q: %v", req.ID, err),
		)

		return
	}

	for _, attributeName := range attributeNames {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attributeName), id)...)
	}
}
