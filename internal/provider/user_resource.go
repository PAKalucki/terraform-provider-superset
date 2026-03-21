package provider

import (
	"context"
	"fmt"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithImportState = &UserResource{}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResource struct {
	client *supersetclient.Client
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Superset user for auth backends that allow user administration through the Superset REST API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Superset user identifier.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Superset username.",
				Validators: []validator.String{
					nonEmptyTrimmedStringValidator(),
				},
			},
			"first_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Superset first name.",
				Validators: []validator.String{
					nonEmptyTrimmedStringValidator(),
				},
			},
			"last_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Superset last name.",
				Validators: []validator.String{
					nonEmptyTrimmedStringValidator(),
				},
			},
			"email": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Superset email address.",
				Validators: []validator.String{
					nonEmptyTrimmedStringValidator(),
					emailAddressValidator(),
				},
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the Superset user is active.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Superset password. This is required on create. When omitted later, the provider leaves the existing password unchanged.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role_ids": schema.SetAttribute{
				Required:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Authoritative set of Superset role identifiers assigned to the user.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importInt64Attributes(ctx, req, resp, "id")
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data userModel
	var createdID int64
	persistedState := false

	defer func() {
		if createdID == 0 || persistedState {
			return
		}

		if err := r.client.DeleteUser(ctx, createdID); err != nil && !isSupersetNotFoundError(err) {
			resp.Diagnostics.AddWarning(
				"Unable to Roll Back Superset User After Create Failure",
				fmt.Sprintf("The provider created Superset user %d but could not delete it after the Terraform create operation failed: %v", createdID, err),
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
			"The provider client was not configured for the user resource.",
		)

		return
	}

	request, diags := expandUserCreateRequest(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.CreateUser(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Superset User",
			err.Error(),
		)

		return
	}

	createdID = user.ID

	user, err = r.client.GetUser(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset User After Create",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenUserModel(ctx, data, user)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	persistedState = true
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data userModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the user resource.",
		)

		return
	}

	user, err := r.client.GetUser(ctx, data.ID.ValueInt64())
	if err != nil {
		if isSupersetNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Unable to Read Superset User",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenUserModel(ctx, data, user)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data userModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the user resource.",
		)

		return
	}

	var current userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request, diags := expandUserUpdateRequest(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateUser(ctx, current.ID.ValueInt64(), request); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Superset User",
			err.Error(),
		)

		return
	}

	user, err := r.client.GetUser(ctx, current.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset User After Update",
			err.Error(),
		)

		return
	}

	state, stateDiags := flattenUserModel(ctx, data, user)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data userModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Superset Client",
			"The provider client was not configured for the user resource.",
		)

		return
	}

	if err := r.client.DeleteUser(ctx, data.ID.ValueInt64()); err != nil && !isSupersetNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Unable to Delete Superset User",
			err.Error(),
		)
	}
}
