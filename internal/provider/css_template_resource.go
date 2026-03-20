package provider

import (
	"context"
	"fmt"
	"strings"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &CSSTemplateResource{}
var _ resource.ResourceWithImportState = &CSSTemplateResource{}

func NewCSSTemplateResource() resource.Resource {
	return &CSSTemplateResource{}
}

type CSSTemplateResource struct {
	client *supersetclient.Client
}

type cssTemplateModel struct {
	ID           types.Int64  `tfsdk:"id"`
	TemplateName types.String `tfsdk:"template_name"`
	CSS          types.String `tfsdk:"css"`
}

func (r *CSSTemplateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_css_template"
}

func (r *CSSTemplateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Superset CSS template.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Superset CSS template identifier.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"template_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Superset CSS template name.",
			},
			"css": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "CSS text stored in the template.",
			},
		},
	}
}

func (r *CSSTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*supersetclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *CSSTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importInt64Attributes(ctx, req, resp, "id")
}

func (r *CSSTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data cssTemplateModel
	var createdID int64
	persistedState := false

	defer func() {
		if createdID == 0 || persistedState {
			return
		}

		if err := r.client.DeleteCSSTemplate(ctx, createdID); err != nil && !isSupersetNotFoundError(err) {
			resp.Diagnostics.AddWarning(
				"Unable to Roll Back Superset CSS Template After Create Failure",
				fmt.Sprintf("The provider created Superset CSS template %d but could not delete it after the Terraform create operation failed: %v", createdID, err),
			)
		}
	}()

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the CSS template resource.",
		)

		return
	}

	request, diags := expandCSSTemplateRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cssTemplate, err := r.client.CreateCSSTemplate(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Superset CSS Template",
			err.Error(),
		)

		return
	}

	createdID = cssTemplate.ID

	cssTemplate, err = r.client.GetCSSTemplate(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset CSS Template After Create",
			err.Error(),
		)

		return
	}

	state := flattenCSSTemplateModel(cssTemplate)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	persistedState = true
}

func (r *CSSTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data cssTemplateModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the CSS template resource.",
		)

		return
	}

	cssTemplate, err := r.client.GetCSSTemplate(ctx, data.ID.ValueInt64())
	if err != nil {
		if isSupersetNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Unable to Read Superset CSS Template",
			err.Error(),
		)

		return
	}

	state := flattenCSSTemplateModel(cssTemplate)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CSSTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data cssTemplateModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the CSS template resource.",
		)

		return
	}

	var current cssTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request, diags := expandCSSTemplateRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateCSSTemplate(ctx, current.ID.ValueInt64(), request); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Superset CSS Template",
			err.Error(),
		)

		return
	}

	cssTemplate, err := r.client.GetCSSTemplate(ctx, current.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset CSS Template After Update",
			err.Error(),
		)

		return
	}

	state := flattenCSSTemplateModel(cssTemplate)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CSSTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data cssTemplateModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the CSS template resource.",
		)

		return
	}

	if err := r.client.DeleteCSSTemplate(ctx, data.ID.ValueInt64()); err != nil && !isSupersetNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Unable to Delete Superset CSS Template",
			err.Error(),
		)
	}
}

func expandCSSTemplateRequest(data cssTemplateModel) (supersetclient.CSSTemplateCreateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	templateName := strings.TrimSpace(stringValue(data.TemplateName))
	if templateName == "" {
		diags.AddAttributeError(
			path.Root("template_name"),
			"Missing CSS Template Name",
			"`template_name` must be configured.",
		)
	}

	css := stringValue(data.CSS)
	if strings.TrimSpace(css) == "" {
		diags.AddAttributeError(
			path.Root("css"),
			"Missing CSS Template Content",
			"`css` must be configured.",
		)
	}

	return supersetclient.CSSTemplateCreateRequest{
		TemplateName: templateName,
		CSS:          css,
	}, diags
}

func flattenCSSTemplateModel(remote *supersetclient.CSSTemplate) cssTemplateModel {
	return cssTemplateModel{
		ID:           types.Int64Value(remote.ID),
		TemplateName: stringTypeValue(remote.TemplateName),
		CSS:          stringTypeValue(remote.CSS),
	}
}
