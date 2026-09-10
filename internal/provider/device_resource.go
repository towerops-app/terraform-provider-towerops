package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	ID                    types.String `tfsdk:"id"`
	SiteID                types.String `tfsdk:"site_id"`
	OrganizationID        types.String `tfsdk:"organization_id"`
	Name                  types.String `tfsdk:"name"`
	IPAddress             types.String `tfsdk:"ip_address"`
	Description           types.String `tfsdk:"description"`
	MonitoringEnabled     types.Bool   `tfsdk:"monitoring_enabled"`
	CheckIntervalSeconds  types.Int64  `tfsdk:"check_interval_seconds"`
	SNMPEnabled           types.Bool   `tfsdk:"snmp_enabled"`
	SNMPVersion           types.String `tfsdk:"snmp_version"`
	SNMPPort              types.Int64  `tfsdk:"snmp_port"`
	DeviceRole            types.String `tfsdk:"device_role"`
	SNMPv3SecurityLevel   types.String `tfsdk:"snmpv3_security_level"`
	SNMPv3Username        types.String `tfsdk:"snmpv3_username"`
	SNMPv3AuthProtocol    types.String `tfsdk:"snmpv3_auth_protocol"`
	SNMPv3AuthPassword    types.String `tfsdk:"snmpv3_auth_password"`
	SNMPv3AuthPasswordSet types.Bool   `tfsdk:"snmpv3_auth_password_set"`
	SNMPv3PrivProtocol    types.String `tfsdk:"snmpv3_priv_protocol"`
	SNMPv3PrivPassword    types.String `tfsdk:"snmpv3_priv_password"`
	SNMPv3PrivPasswordSet types.Bool   `tfsdk:"snmpv3_priv_password_set"`
	InsertedAt            types.String `tfsdk:"inserted_at"`
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
					stringplanmodifier.UseStateForUnknown(),
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
			"check_interval_seconds": schema.Int64Attribute{
				Description: "How often the device is checked, in seconds. The API accepts 300 to 3600 inclusive and rejects anything outside that range.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(300),
			},
			"snmp_enabled": schema.BoolAttribute{
				Description: "Whether SNMP polling is enabled for this device.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
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
			"device_role": schema.StringAttribute{
				Description: "The device role. Valid values: router, switch, access_point, backhaul, server, other. Defaults to 'other' if not specified.",
				Optional:    true,
				Computed:    true,
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
				Description: "SNMPv3 authentication password. Only used when snmp_version is '3'. Write-only: the API accepts it but never returns it, so its state always mirrors the configuration. Watch snmpv3_auth_password_set to see whether the API holds one.",
				Optional:    true,
				Sensitive:   true,
			},
			"snmpv3_auth_password_set": schema.BoolAttribute{
				Description: "Whether the API currently holds an SNMPv3 authentication password for this device.",
				Computed:    true,
			},
			"snmpv3_priv_protocol": schema.StringAttribute{
				Description: "SNMPv3 privacy protocol (DES, AES, AES-192, AES-256). Only used when snmp_version is '3'.",
				Optional:    true,
			},
			"snmpv3_priv_password": schema.StringAttribute{
				Description: "SNMPv3 privacy password. Only used when snmp_version is '3'. Write-only: the API accepts it but never returns it, so its state always mirrors the configuration. Watch snmpv3_priv_password_set to see whether the API holds one.",
				Optional:    true,
				Sensitive:   true,
			},
			"snmpv3_priv_password_set": schema.BoolAttribute{
				Description: "Whether the API currently holds an SNMPv3 privacy password for this device.",
				Computed:    true,
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
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// deviceValueKnown reports whether an attribute carries a real value, as
// opposed to null or a value Terraform has not resolved yet.
func deviceValueKnown(value attr.Value) bool {
	return !value.IsNull() && !value.IsUnknown()
}

// deviceStringOrNull turns an optional API string into an attribute value.
func deviceStringOrNull(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

// toAPI builds the request payload from the planned values. An unknown value
// is never sent: the API would read it as an empty string or a zero.
func (m *DeviceResourceModel) toAPI() Device {
	device := Device{
		IPAddress: m.IPAddress.ValueString(),
	}

	if deviceValueKnown(m.SiteID) {
		siteID := m.SiteID.ValueString()
		device.SiteID = &siteID
	}

	if deviceValueKnown(m.OrganizationID) {
		orgID := m.OrganizationID.ValueString()
		device.OrganizationID = &orgID
	}

	if deviceValueKnown(m.Name) {
		name := m.Name.ValueString()
		device.Name = &name
	}

	if deviceValueKnown(m.Description) {
		desc := m.Description.ValueString()
		device.Description = &desc
	}

	if deviceValueKnown(m.DeviceRole) {
		role := m.DeviceRole.ValueString()
		device.DeviceRole = &role
	}

	if deviceValueKnown(m.MonitoringEnabled) {
		enabled := m.MonitoringEnabled.ValueBool()
		device.MonitoringEnabled = &enabled
	}

	if deviceValueKnown(m.CheckIntervalSeconds) {
		interval := int(m.CheckIntervalSeconds.ValueInt64())
		device.CheckIntervalSeconds = &interval
	}

	if deviceValueKnown(m.SNMPEnabled) {
		enabled := m.SNMPEnabled.ValueBool()
		device.SNMPEnabled = &enabled
	}

	// Only send SNMP config when SNMP is enabled.
	if m.SNMPEnabled.ValueBool() {
		if deviceValueKnown(m.SNMPVersion) {
			version := m.SNMPVersion.ValueString()
			device.SNMPVersion = &version
		}

		if deviceValueKnown(m.SNMPPort) {
			port := int(m.SNMPPort.ValueInt64())
			device.SNMPPort = &port
		}

		// SNMPv3 fields
		if deviceValueKnown(m.SNMPv3SecurityLevel) {
			level := m.SNMPv3SecurityLevel.ValueString()
			device.SNMPv3SecurityLevel = &level
		}

		if deviceValueKnown(m.SNMPv3Username) {
			username := m.SNMPv3Username.ValueString()
			device.SNMPv3Username = &username
		}

		if deviceValueKnown(m.SNMPv3AuthProtocol) {
			protocol := m.SNMPv3AuthProtocol.ValueString()
			device.SNMPv3AuthProtocol = &protocol
		}

		if deviceValueKnown(m.SNMPv3AuthPassword) {
			password := m.SNMPv3AuthPassword.ValueString()
			device.SNMPv3AuthPassword = &password
		}

		if deviceValueKnown(m.SNMPv3PrivProtocol) {
			protocol := m.SNMPv3PrivProtocol.ValueString()
			device.SNMPv3PrivProtocol = &protocol
		}

		if deviceValueKnown(m.SNMPv3PrivPassword) {
			password := m.SNMPv3PrivPassword.ValueString()
			device.SNMPv3PrivPassword = &password
		}
	}

	return device
}

// applyWriteResponse folds a create or update response into the model.
//
// A planned value that is already known is authoritative: Terraform fails the
// apply with "Provider produced inconsistent result after apply" if the
// provider stores anything else, so only attributes whose planned value is
// still null or unknown take their value from the response. Drift is reported
// by Read instead, which is free to differ from the prior state.
//
// ip_address, description and the SNMPv3 strings are never copied: their
// planned value is always known. The two SNMPv3 passwords are write-only, so
// the response carries only the companion *_set booleans.
func (m *DeviceResourceModel) applyWriteResponse(device *Device) {
	if !deviceValueKnown(m.ID) {
		m.ID = types.StringValue(device.ID)
	}

	if !deviceValueKnown(m.InsertedAt) {
		m.InsertedAt = types.StringValue(device.InsertedAt)
	}

	if !deviceValueKnown(m.SiteID) {
		m.SiteID = deviceStringOrNull(device.SiteID)
	}

	if !deviceValueKnown(m.OrganizationID) {
		m.OrganizationID = deviceStringOrNull(device.OrganizationID)
	}

	if !deviceValueKnown(m.Name) {
		m.Name = deviceStringOrNull(device.Name)
	}

	if !deviceValueKnown(m.DeviceRole) {
		m.DeviceRole = deviceStringOrNull(device.DeviceRole)
	}

	if !deviceValueKnown(m.MonitoringEnabled) && device.MonitoringEnabled != nil {
		m.MonitoringEnabled = types.BoolValue(*device.MonitoringEnabled)
	}

	// The create response (format_device/1) leaves check_interval_seconds out,
	// so a nil here means "not reported", not "zero".
	if !deviceValueKnown(m.CheckIntervalSeconds) && device.CheckIntervalSeconds != nil {
		m.CheckIntervalSeconds = types.Int64Value(int64(*device.CheckIntervalSeconds))
	}

	if !deviceValueKnown(m.SNMPEnabled) && device.SNMPEnabled != nil {
		m.SNMPEnabled = types.BoolValue(*device.SNMPEnabled)
	}

	if !deviceValueKnown(m.SNMPVersion) && device.SNMPVersion != nil {
		m.SNMPVersion = types.StringValue(*device.SNMPVersion)
	}

	if !deviceValueKnown(m.SNMPPort) && device.SNMPPort != nil {
		m.SNMPPort = types.Int64Value(int64(*device.SNMPPort))
	}

	if !deviceValueKnown(m.SNMPv3AuthPasswordSet) {
		m.SNMPv3AuthPasswordSet = types.BoolValue(device.SNMPv3AuthPasswordSet)
	}

	if !deviceValueKnown(m.SNMPv3PrivPasswordSet) {
		m.SNMPv3PrivPasswordSet = types.BoolValue(device.SNMPv3PrivPasswordSet)
	}
}

func (r *DeviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeviceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateDevice(data.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create device", err.Error())
		return
	}

	data.applyWriteResponse(created)

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

	data.SiteID = deviceStringOrNull(device.SiteID)
	data.OrganizationID = deviceStringOrNull(device.OrganizationID)
	data.IPAddress = types.StringValue(device.IPAddress)
	data.Name = deviceStringOrNull(device.Name)
	data.Description = deviceStringOrNull(device.Description)
	data.InsertedAt = types.StringValue(device.InsertedAt)
	data.DeviceRole = deviceStringOrNull(device.DeviceRole)

	if device.MonitoringEnabled != nil {
		data.MonitoringEnabled = types.BoolValue(*device.MonitoringEnabled)
	}

	if device.CheckIntervalSeconds != nil {
		data.CheckIntervalSeconds = types.Int64Value(int64(*device.CheckIntervalSeconds))
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

	// SNMPv3 fields. The two passwords are write-only: the API never returns
	// them, so the configured value stays untouched and the *_set booleans
	// carry the drift instead.
	data.SNMPv3SecurityLevel = deviceStringOrNull(device.SNMPv3SecurityLevel)
	data.SNMPv3Username = deviceStringOrNull(device.SNMPv3Username)
	data.SNMPv3AuthProtocol = deviceStringOrNull(device.SNMPv3AuthProtocol)
	data.SNMPv3AuthPasswordSet = types.BoolValue(device.SNMPv3AuthPasswordSet)
	data.SNMPv3PrivProtocol = deviceStringOrNull(device.SNMPv3PrivProtocol)
	data.SNMPv3PrivPasswordSet = types.BoolValue(device.SNMPv3PrivPasswordSet)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DeviceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	device := data.toAPI()

	updated, err := r.client.UpdateDevice(data.ID.ValueString(), device)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Device was deleted outside of Terraform, recreate it
			created, createErr := r.client.CreateDevice(device)
			if createErr != nil {
				resp.Diagnostics.AddError("Failed to create device (after 404 on update)", createErr.Error())
				return
			}
			data.applyWriteResponse(created)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError("Failed to update device", err.Error())
		return
	}

	data.applyWriteResponse(updated)

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
