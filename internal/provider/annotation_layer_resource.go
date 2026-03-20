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

var _ resource.Resource = &AnnotationLayerResource{}
var _ resource.ResourceWithImportState = &AnnotationLayerResource{}

func NewAnnotationLayerResource() resource.Resource {
	return &AnnotationLayerResource{}
}

type AnnotationLayerResource struct {
	client *supersetclient.Client
}

type annotationLayerModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (r *AnnotationLayerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_annotation_layer"
}

func (r *AnnotationLayerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Superset annotation layer.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Superset annotation layer identifier.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Superset annotation layer name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional annotation layer description.",
			},
		},
	}
}

func (r *AnnotationLayerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AnnotationLayerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importInt64Attributes(ctx, req, resp, "id")
}

func (r *AnnotationLayerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data annotationLayerModel
	var createdID int64
	persistedState := false

	defer func() {
		if createdID == 0 || persistedState {
			return
		}

		if err := r.client.DeleteAnnotationLayer(ctx, createdID); err != nil && !isSupersetNotFoundError(err) {
			resp.Diagnostics.AddWarning(
				"Unable to Roll Back Superset Annotation Layer After Create Failure",
				fmt.Sprintf("The provider created Superset annotation layer %d but could not delete it after the Terraform create operation failed: %v", createdID, err),
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
			"The provider client was not configured for the annotation layer resource.",
		)

		return
	}

	request, diags := expandAnnotationLayerCreateRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	layer, err := r.client.CreateAnnotationLayer(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Superset Annotation Layer",
			err.Error(),
		)

		return
	}

	createdID = layer.ID

	layer, err = r.client.GetAnnotationLayer(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Annotation Layer After Create",
			err.Error(),
		)

		return
	}

	state := flattenAnnotationLayerModel(layer)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	persistedState = true
}

func (r *AnnotationLayerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data annotationLayerModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the annotation layer resource.",
		)

		return
	}

	layer, err := r.client.GetAnnotationLayer(ctx, data.ID.ValueInt64())
	if err != nil {
		if isSupersetNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Unable to Read Superset Annotation Layer",
			err.Error(),
		)

		return
	}

	state := flattenAnnotationLayerModel(layer)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AnnotationLayerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data annotationLayerModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the annotation layer resource.",
		)

		return
	}

	var current annotationLayerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	layer, err := r.client.GetAnnotationLayer(ctx, current.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Annotation Layer Before Update",
			err.Error(),
		)

		return
	}

	request, diags := expandAnnotationLayerUpdateRequest(data, layer)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateAnnotationLayer(ctx, current.ID.ValueInt64(), request); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Superset Annotation Layer",
			err.Error(),
		)

		return
	}

	layer, err = r.client.GetAnnotationLayer(ctx, current.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset Annotation Layer After Update",
			err.Error(),
		)

		return
	}

	state := flattenAnnotationLayerModel(layer)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AnnotationLayerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data annotationLayerModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the annotation layer resource.",
		)

		return
	}

	if err := r.client.DeleteAnnotationLayer(ctx, data.ID.ValueInt64()); err != nil && !isSupersetNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Unable to Delete Superset Annotation Layer",
			err.Error(),
		)
	}
}

func expandAnnotationLayerCreateRequest(data annotationLayerModel) (supersetclient.AnnotationLayerCreateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	name := strings.TrimSpace(stringValue(data.Name))
	if name == "" {
		diags.AddAttributeError(
			path.Root("name"),
			"Missing Annotation Layer Name",
			"`name` must be configured.",
		)
	}

	return supersetclient.AnnotationLayerCreateRequest{
		Name:        name,
		Description: stringPointerValue(data.Description),
	}, diags
}

func expandAnnotationLayerUpdateRequest(data annotationLayerModel, current *supersetclient.AnnotationLayer) (supersetclient.AnnotationLayerUpdateRequest, diag.Diagnostics) {
	createRequest, diags := expandAnnotationLayerCreateRequest(data)

	return supersetclient.AnnotationLayerUpdateRequest{
		Name:               createRequest.Name,
		Description:        createRequest.Description,
		IncludeDescription: includeManagedString(data.Description, stringPointerValueOrEmpty(current.Description)),
	}, diags
}

func flattenAnnotationLayerModel(remote *supersetclient.AnnotationLayer) annotationLayerModel {
	return annotationLayerModel{
		ID:          types.Int64Value(remote.ID),
		Name:        stringTypeValue(remote.Name),
		Description: stringTypeValue(stringPointerValueOrEmpty(remote.Description)),
	}
}
