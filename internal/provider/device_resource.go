package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DeviceResource{}
var _ resource.ResourceWithImportState = &DeviceResource{}

// DeviceResource defines the resource implementation.
type DeviceResource struct {
	client *Client
}

// DeviceResourceModel describes the resource data model.
type DeviceResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	SiteID               types.String `tfsdk:"site_id"`
	OrganizationID       types.String `tfsdk:"organization_id"`
	Name                 types.String `tfsdk:"name"`
	IPAddress            types.String `tfsdk:"ip_address"`
	Description          types.String `tfsdk:"description"`
	MonitoringEnabled    types.Bool   `tfsdk:"monitoring_enabled"`
	SNMPEnabled          types.Bool   `tfsdk:"snmp_enabled"`
	SNMPVersion          types.String `tfsdk:"snmp_version"`
	SNMPPort             types.Int64  `tfsdk:"snmp_port"`
	SNMPv3SecurityLevel  types.String `tfsdk:"snmpv3_security_level"`
	SNMPv3Username       types.String `tfsdk:"snmpv3_username"`
	SNMPv3AuthProtocol   types.String `tfsdk:"snmpv3_auth_protocol"`
	SNMPv3AuthPassword   types.String `tfsdk:"snmpv3_auth_password"`
	SNMPv3PrivProtocol   types.String `tfsdk:"snmpv3_priv_protocol"`
	SNMPv3PrivPassword   types.String `tfsdk:"snmpv3_priv_password"`
	InsertedAt           types.String `tfsdk:"inserted_at"`
}

// NewDeviceResource creates a new device resource.
func NewDeviceResource() resource.Resource {
	return &DeviceResource{}
}

func (r *DeviceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (r *DeviceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TowerOps device. Devices represent network equipment at a site or directly in an organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the device.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": schema.StringAttribute{
				Description: "The ID of the site this device belongs to. Optional if organization_id is provided.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The ID of the organization this device belongs to. Required if site_id is not provided.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the device. If not provided, will be auto-discovered.",
				Optional:    true,
				Computed:    true,
			},
			"ip_address": schema.StringAttribute{
				Description: "The IP address of the device.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the device.",
				Optional:    true,
			},
			"monitoring_enabled": schema.BoolAttribute{
				Description: "Whether monitoring is enabled for this device.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"snmp_enabled": schema.BoolAttribute{
				Description: "Whether SNMP polling is enabled for this device.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"snmp_version": schema.StringAttribute{
				Description: "The SNMP version to use (1, 2c, or 3).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("2c"),
			},
			"snmp_port": schema.Int64Attribute{
				Description: "The SNMP port to use.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(161),
			},
			"snmpv3_security_level": schema.StringAttribute{
				Description: "SNMPv3 security level (noAuthNoPriv, authNoPriv, or authPriv). Only used when snmp_version is '3'.",
				Optional:    true,
			},
			"snmpv3_username": schema.StringAttribute{
				Description: "SNMPv3 username. Only used when snmp_version is '3'.",
				Optional:    true,
			},
			"snmpv3_auth_protocol": schema.StringAttribute{
				Description: "SNMPv3 authentication protocol (MD5, SHA, SHA-224, SHA-256, SHA-384, SHA-512). Only used when snmp_version is '3'.",
				Optional:    true,
			},
			"snmpv3_auth_password": schema.StringAttribute{
				Description: "SNMPv3 authentication password. Only used when snmp_version is '3'.",
				Optional:    true,
				Sensitive:   true,
			},
			"snmpv3_priv_protocol": schema.StringAttribute{
				Description: "SNMPv3 privacy protocol (DES, AES, AES-192, AES-256). Only used when snmp_version is '3'.",
				Optional:    true,
			},
			"snmpv3_priv_password": schema.StringAttribute{
				Description: "SNMPv3 privacy password. Only used when snmp_version is '3'.",
				Optional:    true,
				Sensitive:   true,
			},
			"inserted_at": schema.StringAttribute{
				Description: "The timestamp when the device was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *DeviceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DeviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeviceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	device := Device{
		IPAddress: data.IPAddress.ValueString(),
	}

	if !data.SiteID.IsNull() {
		siteID := data.SiteID.ValueString()
		device.SiteID = &siteID
	}

	if !data.OrganizationID.IsNull() {
		orgID := data.OrganizationID.ValueString()
		device.OrganizationID = &orgID
	}

	if !data.Name.IsNull() {
		name := data.Name.ValueString()
		device.Name = &name
	}

	if !data.Description.IsNull() {
		desc := data.Description.ValueString()
		device.Description = &desc
	}

	if !data.MonitoringEnabled.IsNull() {
		enabled := data.MonitoringEnabled.ValueBool()
		device.MonitoringEnabled = &enabled
	}

	if !data.SNMPEnabled.IsNull() {
		enabled := data.SNMPEnabled.ValueBool()
		device.SNMPEnabled = &enabled
	}

	if !data.SNMPVersion.IsNull() {
		version := data.SNMPVersion.ValueString()
		device.SNMPVersion = &version
	}

	if !data.SNMPPort.IsNull() {
		port := int(data.SNMPPort.ValueInt64())
		device.SNMPPort = &port
	}

	// SNMPv3 fields
	if !data.SNMPv3SecurityLevel.IsNull() {
		level := data.SNMPv3SecurityLevel.ValueString()
		device.SNMPv3SecurityLevel = &level
	}

	if !data.SNMPv3Username.IsNull() {
		username := data.SNMPv3Username.ValueString()
		device.SNMPv3Username = &username
	}

	if !data.SNMPv3AuthProtocol.IsNull() {
		protocol := data.SNMPv3AuthProtocol.ValueString()
		device.SNMPv3AuthProtocol = &protocol
	}

	if !data.SNMPv3AuthPassword.IsNull() {
		password := data.SNMPv3AuthPassword.ValueString()
		device.SNMPv3AuthPassword = &password
	}

	if !data.SNMPv3PrivProtocol.IsNull() {
		protocol := data.SNMPv3PrivProtocol.ValueString()
		device.SNMPv3PrivProtocol = &protocol
	}

	if !data.SNMPv3PrivPassword.IsNull() {
		password := data.SNMPv3PrivPassword.ValueString()
		device.SNMPv3PrivPassword = &password
	}

	created, err := r.client.CreateDevice(device)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create device", err.Error())
		return
	}

	data.ID = types.StringValue(created.ID)
	data.InsertedAt = types.StringValue(created.InsertedAt)

	if created.SiteID != nil {
		data.SiteID = types.StringValue(*created.SiteID)
	} else {
		data.SiteID = types.StringNull()
	}

	if created.OrganizationID != nil {
		data.OrganizationID = types.StringValue(*created.OrganizationID)
	} else {
		data.OrganizationID = types.StringNull()
	}

	if created.Name != nil {
		data.Name = types.StringValue(*created.Name)
	}

	if created.MonitoringEnabled != nil {
		data.MonitoringEnabled = types.BoolValue(*created.MonitoringEnabled)
	}
	if created.SNMPEnabled != nil {
		data.SNMPEnabled = types.BoolValue(*created.SNMPEnabled)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeviceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	device, err := r.client.GetDevice(data.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Device was deleted outside of Terraform, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read device", err.Error())
		return
	}

	if device.SiteID != nil {
		data.SiteID = types.StringValue(*device.SiteID)
	} else {
		data.SiteID = types.StringNull()
	}

	if device.OrganizationID != nil {
		data.OrganizationID = types.StringValue(*device.OrganizationID)
	} else {
		data.OrganizationID = types.StringNull()
	}

	data.IPAddress = types.StringValue(device.IPAddress)

	if device.Name != nil {
		data.Name = types.StringValue(*device.Name)
	} else {
		data.Name = types.StringNull()
	}
	data.InsertedAt = types.StringValue(device.InsertedAt)

	if device.Description != nil {
		data.Description = types.StringValue(*device.Description)
	} else {
		data.Description = types.StringNull()
	}

	if device.MonitoringEnabled != nil {
		data.MonitoringEnabled = types.BoolValue(*device.MonitoringEnabled)
	}

	if device.SNMPEnabled != nil {
		data.SNMPEnabled = types.BoolValue(*device.SNMPEnabled)
	}

	if device.SNMPVersion != nil {
		data.SNMPVersion = types.StringValue(*device.SNMPVersion)
	}

	if device.SNMPPort != nil {
		data.SNMPPort = types.Int64Value(int64(*device.SNMPPort))
	}

	// SNMPv3 fields
	if device.SNMPv3SecurityLevel != nil {
		data.SNMPv3SecurityLevel = types.StringValue(*device.SNMPv3SecurityLevel)
	} else {
		data.SNMPv3SecurityLevel = types.StringNull()
	}

	if device.SNMPv3Username != nil {
		data.SNMPv3Username = types.StringValue(*device.SNMPv3Username)
	} else {
		data.SNMPv3Username = types.StringNull()
	}

	if device.SNMPv3AuthProtocol != nil {
		data.SNMPv3AuthProtocol = types.StringValue(*device.SNMPv3AuthProtocol)
	} else {
		data.SNMPv3AuthProtocol = types.StringNull()
	}

	if device.SNMPv3AuthPassword != nil {
		data.SNMPv3AuthPassword = types.StringValue(*device.SNMPv3AuthPassword)
	} else {
		data.SNMPv3AuthPassword = types.StringNull()
	}

	if device.SNMPv3PrivProtocol != nil {
		data.SNMPv3PrivProtocol = types.StringValue(*device.SNMPv3PrivProtocol)
	} else {
		data.SNMPv3PrivProtocol = types.StringNull()
	}

	if device.SNMPv3PrivPassword != nil {
		data.SNMPv3PrivPassword = types.StringValue(*device.SNMPv3PrivPassword)
	} else {
		data.SNMPv3PrivPassword = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DeviceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	device := Device{
		IPAddress: data.IPAddress.ValueString(),
	}

	if !data.SiteID.IsNull() {
		siteID := data.SiteID.ValueString()
		device.SiteID = &siteID
	}

	if !data.OrganizationID.IsNull() {
		orgID := data.OrganizationID.ValueString()
		device.OrganizationID = &orgID
	}

	if !data.Name.IsNull() {
		name := data.Name.ValueString()
		device.Name = &name
	}

	if !data.Description.IsNull() {
		desc := data.Description.ValueString()
		device.Description = &desc
	}

	if !data.MonitoringEnabled.IsNull() {
		enabled := data.MonitoringEnabled.ValueBool()
		device.MonitoringEnabled = &enabled
	}

	if !data.SNMPEnabled.IsNull() {
		enabled := data.SNMPEnabled.ValueBool()
		device.SNMPEnabled = &enabled
	}

	if !data.SNMPVersion.IsNull() {
		version := data.SNMPVersion.ValueString()
		device.SNMPVersion = &version
	}

	if !data.SNMPPort.IsNull() {
		port := int(data.SNMPPort.ValueInt64())
		device.SNMPPort = &port
	}

	// SNMPv3 fields
	if !data.SNMPv3SecurityLevel.IsNull() {
		level := data.SNMPv3SecurityLevel.ValueString()
		device.SNMPv3SecurityLevel = &level
	}

	if !data.SNMPv3Username.IsNull() {
		username := data.SNMPv3Username.ValueString()
		device.SNMPv3Username = &username
	}

	if !data.SNMPv3AuthProtocol.IsNull() {
		protocol := data.SNMPv3AuthProtocol.ValueString()
		device.SNMPv3AuthProtocol = &protocol
	}

	if !data.SNMPv3AuthPassword.IsNull() {
		password := data.SNMPv3AuthPassword.ValueString()
		device.SNMPv3AuthPassword = &password
	}

	if !data.SNMPv3PrivProtocol.IsNull() {
		protocol := data.SNMPv3PrivProtocol.ValueString()
		device.SNMPv3PrivProtocol = &protocol
	}

	if !data.SNMPv3PrivPassword.IsNull() {
		password := data.SNMPv3PrivPassword.ValueString()
		device.SNMPv3PrivPassword = &password
	}

	updated, err := r.client.UpdateDevice(data.ID.ValueString(), device)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Device was deleted outside of Terraform, recreate it
			created, createErr := r.client.CreateDevice(device)
			if createErr != nil {
				resp.Diagnostics.AddError("Failed to create device (after 404 on update)", createErr.Error())
				return
			}
			data.ID = types.StringValue(created.ID)
			data.InsertedAt = types.StringValue(created.InsertedAt)
			data.IPAddress = types.StringValue(created.IPAddress)

			if created.SiteID != nil {
				data.SiteID = types.StringValue(*created.SiteID)
			} else {
				data.SiteID = types.StringNull()
			}

			if created.OrganizationID != nil {
				data.OrganizationID = types.StringValue(*created.OrganizationID)
			} else {
				data.OrganizationID = types.StringNull()
			}

			if created.Name != nil {
				data.Name = types.StringValue(*created.Name)
			}
			if created.Description != nil {
				data.Description = types.StringValue(*created.Description)
			}
			if created.MonitoringEnabled != nil {
				data.MonitoringEnabled = types.BoolValue(*created.MonitoringEnabled)
			}
			if created.SNMPEnabled != nil {
				data.SNMPEnabled = types.BoolValue(*created.SNMPEnabled)
			}
			if created.SNMPVersion != nil {
				data.SNMPVersion = types.StringValue(*created.SNMPVersion)
			}
			if created.SNMPPort != nil {
				data.SNMPPort = types.Int64Value(int64(*created.SNMPPort))
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError("Failed to update device", err.Error())
		return
	}

	data.IPAddress = types.StringValue(updated.IPAddress)

	if updated.SiteID != nil {
		data.SiteID = types.StringValue(*updated.SiteID)
	} else {
		data.SiteID = types.StringNull()
	}

	if updated.OrganizationID != nil {
		data.OrganizationID = types.StringValue(*updated.OrganizationID)
	} else {
		data.OrganizationID = types.StringNull()
	}

	if updated.Name != nil {
		data.Name = types.StringValue(*updated.Name)
	}
	if updated.Description != nil {
		data.Description = types.StringValue(*updated.Description)
	}
	if updated.MonitoringEnabled != nil {
		data.MonitoringEnabled = types.BoolValue(*updated.MonitoringEnabled)
	}
	if updated.SNMPEnabled != nil {
		data.SNMPEnabled = types.BoolValue(*updated.SNMPEnabled)
	}
	if updated.SNMPVersion != nil {
		data.SNMPVersion = types.StringValue(*updated.SNMPVersion)
	}
	if updated.SNMPPort != nil {
		data.SNMPPort = types.Int64Value(int64(*updated.SNMPPort))
	}

	// SNMPv3 fields
	if updated.SNMPv3SecurityLevel != nil {
		data.SNMPv3SecurityLevel = types.StringValue(*updated.SNMPv3SecurityLevel)
	} else {
		data.SNMPv3SecurityLevel = types.StringNull()
	}

	if updated.SNMPv3Username != nil {
		data.SNMPv3Username = types.StringValue(*updated.SNMPv3Username)
	} else {
		data.SNMPv3Username = types.StringNull()
	}

	if updated.SNMPv3AuthProtocol != nil {
		data.SNMPv3AuthProtocol = types.StringValue(*updated.SNMPv3AuthProtocol)
	} else {
		data.SNMPv3AuthProtocol = types.StringNull()
	}

	if updated.SNMPv3AuthPassword != nil {
		data.SNMPv3AuthPassword = types.StringValue(*updated.SNMPv3AuthPassword)
	} else {
		data.SNMPv3AuthPassword = types.StringNull()
	}

	if updated.SNMPv3PrivProtocol != nil {
		data.SNMPv3PrivProtocol = types.StringValue(*updated.SNMPv3PrivProtocol)
	} else {
		data.SNMPv3PrivProtocol = types.StringNull()
	}

	if updated.SNMPv3PrivPassword != nil {
		data.SNMPv3PrivPassword = types.StringValue(*updated.SNMPv3PrivPassword)
	} else {
		data.SNMPv3PrivPassword = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeviceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDevice(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete device", err.Error())
		return
	}
}

func (r *DeviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
