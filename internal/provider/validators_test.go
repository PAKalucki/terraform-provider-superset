package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNonEmptyTrimmedStringValidator(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		value     types.String
		wantError bool
	}{
		{
			name:      "valid",
			value:     types.StringValue("analytics"),
			wantError: false,
		},
		{
			name:      "blank",
			value:     types.StringValue("   "),
			wantError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			resp := &validator.StringResponse{}
			nonEmptyTrimmedStringValidator().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("name"),
				ConfigValue: testCase.value,
			}, resp)

			if got := resp.Diagnostics.HasError(); got != testCase.wantError {
				t.Fatalf("expected error=%t, got diagnostics %v", testCase.wantError, resp.Diagnostics)
			}
		})
	}
}

func TestPositiveInt64Validator(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		value     types.Int64
		wantError bool
	}{
		{
			name:      "valid",
			value:     types.Int64Value(7),
			wantError: false,
		},
		{
			name:      "zero",
			value:     types.Int64Value(0),
			wantError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			resp := &validator.Int64Response{}
			positiveInt64Validator().ValidateInt64(context.Background(), validator.Int64Request{
				Path:        path.Root("id"),
				ConfigValue: testCase.value,
			}, resp)

			if got := resp.Diagnostics.HasError(); got != testCase.wantError {
				t.Fatalf("expected error=%t, got diagnostics %v", testCase.wantError, resp.Diagnostics)
			}
		})
	}
}

func TestEmailAddressValidator(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		value     types.String
		wantError bool
	}{
		{
			name:      "valid",
			value:     types.StringValue("admin@example.com"),
			wantError: false,
		},
		{
			name:      "invalid",
			value:     types.StringValue("not-an-email"),
			wantError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			resp := &validator.StringResponse{}
			emailAddressValidator().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("email"),
				ConfigValue: testCase.value,
			}, resp)

			if got := resp.Diagnostics.HasError(); got != testCase.wantError {
				t.Fatalf("expected error=%t, got diagnostics %v", testCase.wantError, resp.Diagnostics)
			}
		})
	}
}
