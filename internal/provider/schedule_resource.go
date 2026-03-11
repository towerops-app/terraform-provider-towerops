package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ScheduleResource{}
var _ resource.ResourceWithImportState = &ScheduleResource{}

// ScheduleResource defines the resource implementation.
type ScheduleResource struct {
	client *Client
}

// ScheduleResourceModel describes the resource data model.
type ScheduleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Timezone    types.String `tfsdk:"timezone"`
	InsertedAt  types.String `tfsdk:"inserted_at"`
}

// NewScheduleResource creates a new schedule resource.
func NewScheduleResource() resource.Resource {
	return &ScheduleResource{}
}

func (r *ScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schedule"
}

func (r *ScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TowerOps on-call schedule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the schedule.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the on-call schedule.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the schedule.",
				Optional:    true,
			},
			"timezone": schema.StringAttribute{
				Description: "The timezone for the schedule (e.g. America/Chicago).",
				Required:    true,
			},
			"inserted_at": schema.StringAttribute{
				Description: "The timestamp when the schedule was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ScheduleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedule := OnCallSchedule{
		Name:     data.Name.ValueString(),
		Timezone: data.Timezone.ValueString(),
	}

	if !data.Description.IsNull() {
		desc := data.Description.ValueString()
		schedule.Description = &desc
	}

	created, err := r.client.CreateSchedule(schedule)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create schedule", err.Error())
		return
	}

	data.ID = types.StringValue(created.ID)
	data.InsertedAt = types.StringValue(created.InsertedAt)

	if created.Description != nil {
		data.Description = types.StringValue(*created.Description)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ScheduleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedule, err := r.client.GetSchedule(data.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read schedule", err.Error())
		return
	}

	data.Name = types.StringValue(schedule.Name)
	data.Timezone = types.StringValue(schedule.Timezone)
	data.InsertedAt = types.StringValue(schedule.InsertedAt)

	if schedule.Description != nil {
		data.Description = types.StringValue(*schedule.Description)
	} else {
		data.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ScheduleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedule := OnCallSchedule{
		Name:     data.Name.ValueString(),
		Timezone: data.Timezone.ValueString(),
	}

	if !data.Description.IsNull() {
		desc := data.Description.ValueString()
		schedule.Description = &desc
	}

	updated, err := r.client.UpdateSchedule(data.ID.ValueString(), schedule)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			created, createErr := r.client.CreateSchedule(schedule)
			if createErr != nil {
				resp.Diagnostics.AddError("Failed to create schedule (after 404 on update)", createErr.Error())
				return
			}
			data.ID = types.StringValue(created.ID)
			data.InsertedAt = types.StringValue(created.InsertedAt)
			data.Name = types.StringValue(created.Name)
			data.Timezone = types.StringValue(created.Timezone)
			if created.Description != nil {
				data.Description = types.StringValue(*created.Description)
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError("Failed to update schedule", err.Error())
		return
	}

	data.Name = types.StringValue(updated.Name)
	data.Timezone = types.StringValue(updated.Timezone)

	if updated.Description != nil {
		data.Description = types.StringValue(*updated.Description)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ScheduleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSchedule(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete schedule", err.Error())
		return
	}
}

func (r *ScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
