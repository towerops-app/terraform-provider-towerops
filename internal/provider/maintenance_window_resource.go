package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &MaintenanceWindowResource{}
var _ resource.ResourceWithImportState = &MaintenanceWindowResource{}

// MaintenanceWindowResource defines the resource implementation.
type MaintenanceWindowResource struct {
	client *Client
}

// MaintenanceWindowResourceModel describes the resource data model.
type MaintenanceWindowResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Reason         types.String `tfsdk:"reason"`
	StartsAt       types.String `tfsdk:"starts_at"`
	EndsAt         types.String `tfsdk:"ends_at"`
	SuppressAlerts types.Bool   `tfsdk:"suppress_alerts"`
	SiteID         types.String `tfsdk:"site_id"`
	DeviceID       types.String `tfsdk:"device_id"`
	InsertedAt     types.String `tfsdk:"inserted_at"`
}

// NewMaintenanceWindowResource creates a new maintenance window resource.
func NewMaintenanceWindowResource() resource.Resource {
	return &MaintenanceWindowResource{}
}

func (r *MaintenanceWindowResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_window"
}

func (r *MaintenanceWindowResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TowerOps maintenance window. Maintenance windows suppress alerts during planned work periods.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the maintenance window.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the maintenance window.",
				Required:    true,
			},
			"reason": schema.StringAttribute{
				Description: "The reason for the maintenance window.",
				Optional:    true,
			},
			"starts_at": schema.StringAttribute{
				Description: "The start time of the maintenance window in ISO 8601 format (e.g. 2024-01-15T02:00:00Z).",
				Required:    true,
			},
			"ends_at": schema.StringAttribute{
				Description: "The end time of the maintenance window in ISO 8601 format (e.g. 2024-01-15T06:00:00Z).",
				Required:    true,
			},
			"suppress_alerts": schema.BoolAttribute{
				Description: "Whether to suppress alerts during the maintenance window. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"site_id": schema.StringAttribute{
				Description: "The site to apply the maintenance window to. If omitted, applies to all sites.",
				Optional:    true,
			},
			"device_id": schema.StringAttribute{
				Description: "The device to apply the maintenance window to. If omitted, applies to all devices.",
				Optional:    true,
			},
			"inserted_at": schema.StringAttribute{
				Description: "The timestamp when the maintenance window was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *MaintenanceWindowResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MaintenanceWindowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MaintenanceWindowResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	window := MaintenanceWindowAPI{
		Name:     data.Name.ValueString(),
		StartsAt: data.StartsAt.ValueString(),
		EndsAt:   data.EndsAt.ValueString(),
	}

	if !data.Reason.IsNull() {
		reason := data.Reason.ValueString()
		window.Reason = &reason
	}

	if !data.SuppressAlerts.IsNull() {
		suppress := data.SuppressAlerts.ValueBool()
		window.SuppressAlerts = &suppress
	}

	if !data.SiteID.IsNull() {
		siteID := data.SiteID.ValueString()
		window.SiteID = &siteID
	}

	if !data.DeviceID.IsNull() {
		deviceID := data.DeviceID.ValueString()
		window.DeviceID = &deviceID
	}

	created, err := r.client.CreateMaintenanceWindow(window)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create maintenance window", err.Error())
		return
	}

	data.ID = types.StringValue(created.ID)
	data.InsertedAt = types.StringValue(created.InsertedAt)

	if created.SuppressAlerts != nil {
		data.SuppressAlerts = types.BoolValue(*created.SuppressAlerts)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MaintenanceWindowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MaintenanceWindowResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	window, err := r.client.GetMaintenanceWindow(data.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read maintenance window", err.Error())
		return
	}

	data.Name = types.StringValue(window.Name)
	data.StartsAt = types.StringValue(window.StartsAt)
	data.EndsAt = types.StringValue(window.EndsAt)
	data.InsertedAt = types.StringValue(window.InsertedAt)

	if window.Reason != nil {
		data.Reason = types.StringValue(*window.Reason)
	} else {
		data.Reason = types.StringNull()
	}

	if window.SuppressAlerts != nil {
		data.SuppressAlerts = types.BoolValue(*window.SuppressAlerts)
	}

	if window.SiteID != nil {
		data.SiteID = types.StringValue(*window.SiteID)
	} else {
		data.SiteID = types.StringNull()
	}

	if window.DeviceID != nil {
		data.DeviceID = types.StringValue(*window.DeviceID)
	} else {
		data.DeviceID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MaintenanceWindowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MaintenanceWindowResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	window := MaintenanceWindowAPI{
		Name:     data.Name.ValueString(),
		StartsAt: data.StartsAt.ValueString(),
		EndsAt:   data.EndsAt.ValueString(),
	}

	if !data.Reason.IsNull() {
		reason := data.Reason.ValueString()
		window.Reason = &reason
	}

	if !data.SuppressAlerts.IsNull() {
		suppress := data.SuppressAlerts.ValueBool()
		window.SuppressAlerts = &suppress
	}

	if !data.SiteID.IsNull() {
		siteID := data.SiteID.ValueString()
		window.SiteID = &siteID
	}

	if !data.DeviceID.IsNull() {
		deviceID := data.DeviceID.ValueString()
		window.DeviceID = &deviceID
	}

	updated, err := r.client.UpdateMaintenanceWindow(data.ID.ValueString(), window)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			created, createErr := r.client.CreateMaintenanceWindow(window)
			if createErr != nil {
				resp.Diagnostics.AddError("Failed to create maintenance window (after 404 on update)", createErr.Error())
				return
			}
			data.ID = types.StringValue(created.ID)
			data.InsertedAt = types.StringValue(created.InsertedAt)
			if created.SuppressAlerts != nil {
				data.SuppressAlerts = types.BoolValue(*created.SuppressAlerts)
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError("Failed to update maintenance window", err.Error())
		return
	}

	data.Name = types.StringValue(updated.Name)
	data.StartsAt = types.StringValue(updated.StartsAt)
	data.EndsAt = types.StringValue(updated.EndsAt)

	if updated.SuppressAlerts != nil {
		data.SuppressAlerts = types.BoolValue(*updated.SuppressAlerts)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MaintenanceWindowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MaintenanceWindowResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMaintenanceWindow(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete maintenance window", err.Error())
		return
	}
}

func (r *MaintenanceWindowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
