package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &EscalationPolicyResource{}
var _ resource.ResourceWithImportState = &EscalationPolicyResource{}

// EscalationPolicyResource defines the resource implementation.
type EscalationPolicyResource struct {
	client *Client
}

// EscalationPolicyResourceModel describes the resource data model.
type EscalationPolicyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	RepeatCount types.Int64  `tfsdk:"repeat_count"`
	InsertedAt  types.String `tfsdk:"inserted_at"`
}

// NewEscalationPolicyResource creates a new escalation policy resource.
func NewEscalationPolicyResource() resource.Resource {
	return &EscalationPolicyResource{}
}

func (r *EscalationPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_escalation_policy"
}

// Schema covers exactly the fields format_policy/1 serializes.
//
// handoff_notifications_mode is left out deliberately: EscalationPolicy's
// changeset casts it, but the API never returns it, so Terraform could not
// read the applied value back and every plan would report perpetual drift.
//
// Escalation rules and their targets are managed through separate nested
// endpoints (POST /api/v1/escalation_policies/:id/rules and
// .../rules/:rule_id/targets) that this provider does not expose yet, so a
// policy created here has no rules until they are added in the TowerOps UI.
func (r *EscalationPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TowerOps escalation policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the escalation policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the escalation policy.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the escalation policy.",
				Optional:    true,
			},
			"repeat_count": schema.Int64Attribute{
				Description: "Number of times to repeat the escalation cycle. Defaults to 3.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"inserted_at": schema.StringAttribute{
				Description: "The timestamp when the escalation policy was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *EscalationPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *EscalationPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data EscalationPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy := EscalationPolicyAPI{
		Name: data.Name.ValueString(),
	}

	if !data.Description.IsNull() {
		desc := data.Description.ValueString()
		policy.Description = &desc
	}

	if !data.RepeatCount.IsNull() && !data.RepeatCount.IsUnknown() {
		rc := int(data.RepeatCount.ValueInt64())
		policy.RepeatCount = &rc
	}

	created, err := r.client.CreateEscalationPolicy(policy)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create escalation policy", err.Error())
		return
	}

	data.ID = types.StringValue(created.ID)
	data.InsertedAt = types.StringValue(created.InsertedAt)

	// description is optional without a default, so its state has to match the
	// configuration exactly, and repeat_count is only taken from the response
	// when the plan left it unset. Writing back a value the plan did not carry
	// fails the apply with an inconsistent result.
	if data.RepeatCount.IsNull() || data.RepeatCount.IsUnknown() {
		if created.RepeatCount != nil {
			data.RepeatCount = types.Int64Value(int64(*created.RepeatCount))
		} else {
			data.RepeatCount = types.Int64Null()
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EscalationPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EscalationPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetEscalationPolicy(data.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read escalation policy", err.Error())
		return
	}

	data.Name = types.StringValue(policy.Name)
	data.InsertedAt = types.StringValue(policy.InsertedAt)

	if policy.Description != nil {
		data.Description = types.StringValue(*policy.Description)
	} else {
		data.Description = types.StringNull()
	}

	if policy.RepeatCount != nil {
		data.RepeatCount = types.Int64Value(int64(*policy.RepeatCount))
	} else {
		data.RepeatCount = types.Int64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EscalationPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data EscalationPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy := EscalationPolicyAPI{
		Name: data.Name.ValueString(),
	}

	if !data.Description.IsNull() {
		desc := data.Description.ValueString()
		policy.Description = &desc
	}

	if !data.RepeatCount.IsNull() && !data.RepeatCount.IsUnknown() {
		rc := int(data.RepeatCount.ValueInt64())
		policy.RepeatCount = &rc
	}

	updated, err := r.client.UpdateEscalationPolicy(data.ID.ValueString(), policy)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			created, createErr := r.client.CreateEscalationPolicy(policy)
			if createErr != nil {
				resp.Diagnostics.AddError("Failed to create escalation policy (after 404 on update)", createErr.Error())
				return
			}
			data.ID = types.StringValue(created.ID)
			data.InsertedAt = types.StringValue(created.InsertedAt)
			if data.RepeatCount.IsNull() || data.RepeatCount.IsUnknown() {
				if created.RepeatCount != nil {
					data.RepeatCount = types.Int64Value(int64(*created.RepeatCount))
				} else {
					data.RepeatCount = types.Int64Null()
				}
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError("Failed to update escalation policy", err.Error())
		return
	}

	// name and description already hold their planned values, which are
	// authoritative after apply, so only a repeat_count the plan left unset
	// falls back to the response.
	if data.RepeatCount.IsNull() || data.RepeatCount.IsUnknown() {
		if updated.RepeatCount != nil {
			data.RepeatCount = types.Int64Value(int64(*updated.RepeatCount))
		} else {
			data.RepeatCount = types.Int64Null()
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EscalationPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EscalationPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteEscalationPolicy(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete escalation policy", err.Error())
		return
	}
}

func (r *EscalationPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
