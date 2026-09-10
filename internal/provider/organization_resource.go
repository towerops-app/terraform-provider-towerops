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

var _ resource.Resource = &OrganizationResource{}
var _ resource.ResourceWithImportState = &OrganizationResource{}

// OrganizationResource manages organization settings.
type OrganizationResource struct {
	client *Client
}

// OrganizationResourceModel describes the resource data model.
//
// The model covers every key the API returns. Two changeset fields are
// deliberately absent: alert_routing and default_escalation_policy_id are
// accepted on write but never returned by the show/update payload, so a
// provider attribute for either one would read back null on every refresh
// and leave the plan permanently dirty.
type OrganizationResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Slug                types.String `tfsdk:"slug"`
	SubscriptionPlan    types.String `tfsdk:"subscription_plan"`
	UseSites            types.Bool   `tfsdk:"use_sites"`
	SNMPCommunity       types.String `tfsdk:"snmp_community"`
	SNMPCommunitySet    types.Bool   `tfsdk:"snmp_community_set"`
	SNMPVersion         types.String `tfsdk:"snmp_version"`
	SNMPPort            types.Int64  `tfsdk:"snmp_port"`
	SNMPTransport       types.String `tfsdk:"snmp_transport"`
	SNMPv3SecurityLevel types.String `tfsdk:"snmpv3_security_level"`
	SNMPv3Username      types.String `tfsdk:"snmpv3_username"`
	SNMPv3AuthProtocol  types.String `tfsdk:"snmpv3_auth_protocol"`
	SNMPv3AuthPassword  types.String `tfsdk:"snmpv3_auth_password"`
	SNMPv3PrivProtocol  types.String `tfsdk:"snmpv3_priv_protocol"`
	SNMPv3PrivPassword  types.String `tfsdk:"snmpv3_priv_password"`
	MikrotikEnabled     types.Bool   `tfsdk:"mikrotik_enabled"`
	MikrotikUsername    types.String `tfsdk:"mikrotik_username"`
	MikrotikPassword    types.String `tfsdk:"mikrotik_password"`
	MikrotikPort        types.Int64  `tfsdk:"mikrotik_port"`
	MikrotikSSHPort     types.Int64  `tfsdk:"mikrotik_ssh_port"`
	MikrotikUseSSL      types.Bool   `tfsdk:"mikrotik_use_ssl"`
	DefaultAgentTokenID types.String `tfsdk:"default_agent_token_id"`
	InsertedAt          types.String `tfsdk:"inserted_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

// NewOrganizationResource creates a new organization resource.
func NewOrganizationResource() resource.Resource {
	return &OrganizationResource{}
}

func (r *OrganizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *OrganizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages organization settings for the organization associated with the API token. There is exactly one organization per token, so this resource manages settings rather than creating or deleting organizations.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the organization. Can only be set by organization owners.",
				Optional:    true,
				Computed:    true,
			},
			"slug": schema.StringAttribute{
				Description: "The URL-friendly slug of the organization. Derived from the name by the API.",
				Computed:    true,
			},
			"subscription_plan": schema.StringAttribute{
				Description: "The billing plan of the organization, either 'free' or 'paid'. Managed through billing, not Terraform.",
				Computed:    true,
			},
			"use_sites": schema.BoolAttribute{
				Description: "Whether the organization uses sites to group devices. When true, devices are organized under sites. When false, devices belong directly to the organization.",
				Required:    true,
			},
			"snmp_community": schema.StringAttribute{
				Description: "Default SNMP community string for devices, used by SNMP v1 and v2c. Write-only: the API accepts it but never returns it, so use 'snmp_community_set' to tell whether a community is stored. Can only be set by organization owners.",
				Optional:    true,
				Sensitive:   true,
			},
			"snmp_community_set": schema.BoolAttribute{
				Description: "Whether a default SNMP community string is stored for the organization.",
				Computed:    true,
			},
			"snmp_version": schema.StringAttribute{
				Description: "Default SNMP version for devices: '1', '2c', or '3'. Defaults to '2c'.",
				Optional:    true,
				Computed:    true,
			},
			"snmp_port": schema.Int64Attribute{
				Description: "Default SNMP port for devices. Defaults to 161.",
				Optional:    true,
				Computed:    true,
			},
			"snmp_transport": schema.StringAttribute{
				Description: "Default SNMP transport for devices, for example 'udp'. Defaults to 'udp'.",
				Optional:    true,
				Computed:    true,
			},
			"snmpv3_security_level": schema.StringAttribute{
				Description: "SNMPv3 security level: 'noAuthNoPriv', 'authNoPriv', or 'authPriv'. The API derives this from the configured protocols, so leave it unset unless you need to pin it.",
				Optional:    true,
				Computed:    true,
			},
			"snmpv3_username": schema.StringAttribute{
				Description: "Default SNMPv3 username. Required by the API when 'snmp_version' is '3'.",
				Optional:    true,
				Computed:    true,
			},
			"snmpv3_auth_protocol": schema.StringAttribute{
				Description: "Default SNMPv3 authentication protocol: 'MD5', 'SHA', 'SHA-224', 'SHA-256', 'SHA-384', or 'SHA-512'. Defaults to 'SHA-256'.",
				Optional:    true,
				Computed:    true,
			},
			"snmpv3_auth_password": schema.StringAttribute{
				Description: "Default SNMPv3 authentication password, at least 8 characters. Write-only: the API accepts it but never returns it.",
				Optional:    true,
				Sensitive:   true,
			},
			"snmpv3_priv_protocol": schema.StringAttribute{
				Description: "Default SNMPv3 privacy protocol: 'DES', 'AES', 'AES-192', 'AES-256', or 'AES-256-C'. Defaults to 'AES'.",
				Optional:    true,
				Computed:    true,
			},
			"snmpv3_priv_password": schema.StringAttribute{
				Description: "Default SNMPv3 privacy password, at least 8 characters. Write-only: the API accepts it but never returns it.",
				Optional:    true,
				Sensitive:   true,
			},
			"mikrotik_enabled": schema.BoolAttribute{
				Description: "Whether MikroTik API polling is enabled by default. Defaults to false.",
				Optional:    true,
				Computed:    true,
			},
			"mikrotik_username": schema.StringAttribute{
				Description: "Default MikroTik API username. Required by the API when MikroTik polling is enabled.",
				Optional:    true,
				Computed:    true,
			},
			"mikrotik_password": schema.StringAttribute{
				Description: "Default MikroTik API password. Write-only: the API accepts it but never returns it.",
				Optional:    true,
				Sensitive:   true,
			},
			"mikrotik_port": schema.Int64Attribute{
				Description: "Default MikroTik API port. Defaults to 8729, the API-SSL port.",
				Optional:    true,
				Computed:    true,
			},
			"mikrotik_ssh_port": schema.Int64Attribute{
				Description: "Default MikroTik SSH port. Defaults to 22.",
				Optional:    true,
				Computed:    true,
			},
			"mikrotik_use_ssl": schema.BoolAttribute{
				Description: "Whether the MikroTik API connection uses SSL. Defaults to true.",
				Optional:    true,
				Computed:    true,
			},
			"default_agent_token_id": schema.StringAttribute{
				Description: "The identifier of the agent token new devices poll through by default.",
				Optional:    true,
				Computed:    true,
			},
			"inserted_at": schema.StringAttribute{
				Description: "The timestamp when the organization was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the organization was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *OrganizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OrganizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Organizations are created out of band, with the account. Applying the
	// resource for the first time pushes the configured settings onto the
	// organization the token belongs to.
	updated, err := r.client.UpdateOrganization(buildOrganizationFromModel(&data))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update organization", err.Error())
		return
	}

	applyOrganizationResponse(&data, updated, true)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OrganizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := r.client.GetOrganization()
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Organization was removed upstream, drop it from state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read organization", err.Error())
		return
	}

	applyOrganizationResponse(&data, org, false)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OrganizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateOrganization(buildOrganizationFromModel(&data))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update organization", err.Error())
		return
	}

	applyOrganizationResponse(&data, updated, true)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Organizations cannot be deleted via the API. When this resource is
	// removed from Terraform config, we simply remove it from state.
	// The organization continues to exist in TowerOps.
}

func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// buildOrganizationFromModel converts the Terraform model to an API
// Organization struct. Only attributes with a known value are sent: an
// unknown Optional+Computed attribute means the practitioner left it out of
// the configuration, and sending its zero value would clobber the stored
// setting.
func buildOrganizationFromModel(data *OrganizationResourceModel) Organization {
	org := Organization{
		UseSites: data.UseSites.ValueBool(),
	}

	if orgKnownString(data.Name) {
		org.Name = data.Name.ValueString()
	}
	if orgKnownString(data.SNMPCommunity) {
		org.SnmpCommunity = data.SNMPCommunity.ValueString()
	}
	org.SNMPVersion = orgStringPtr(data.SNMPVersion)
	org.SNMPPort = orgIntPtr(data.SNMPPort)
	org.SNMPTransport = orgStringPtr(data.SNMPTransport)
	org.SNMPv3SecurityLevel = orgStringPtr(data.SNMPv3SecurityLevel)
	org.SNMPv3Username = orgStringPtr(data.SNMPv3Username)
	org.SNMPv3AuthProtocol = orgStringPtr(data.SNMPv3AuthProtocol)
	org.SNMPv3AuthPassword = orgStringPtr(data.SNMPv3AuthPassword)
	org.SNMPv3PrivProtocol = orgStringPtr(data.SNMPv3PrivProtocol)
	org.SNMPv3PrivPassword = orgStringPtr(data.SNMPv3PrivPassword)
	org.MikrotikEnabled = orgBoolPtr(data.MikrotikEnabled)
	org.MikrotikUsername = orgStringPtr(data.MikrotikUsername)
	org.MikrotikPassword = orgStringPtr(data.MikrotikPassword)
	org.MikrotikPort = orgIntPtr(data.MikrotikPort)
	org.MikrotikSSHPort = orgIntPtr(data.MikrotikSSHPort)
	org.MikrotikUseSSL = orgBoolPtr(data.MikrotikUseSSL)
	org.DefaultAgentTokenID = orgStringPtr(data.DefaultAgentTokenID)

	return org
}

// applyOrganizationResponse maps an API response back onto the model.
//
// planIsAuthoritative is true in Create and Update. There, a value Terraform
// already knows from the plan must survive verbatim: writing anything else
// back produces "Provider produced inconsistent result after apply". Read
// passes false, so real drift reaches state and shows up in the next plan.
//
// The write-only secrets (snmp_community, the SNMPv3 passwords and the
// MikroTik password) are never touched here. The API does not return them,
// so anything this function did with them would erase the configured value.
func applyOrganizationResponse(data *OrganizationResourceModel, org *Organization, planIsAuthoritative bool) {
	data.ID = types.StringValue(org.ID)
	data.Slug = types.StringValue(org.Slug)
	data.SubscriptionPlan = types.StringValue(org.SubscriptionPlan)
	data.SNMPCommunitySet = types.BoolValue(org.SNMPCommunitySet)
	data.InsertedAt = types.StringValue(org.InsertedAt)
	data.UpdatedAt = types.StringValue(org.UpdatedAt)

	if !planIsAuthoritative {
		data.Name = types.StringValue(org.Name)
		data.UseSites = types.BoolValue(org.UseSites)
	} else if !orgKnownString(data.Name) {
		data.Name = types.StringValue(org.Name)
	}

	data.SNMPVersion = orgResolveString(data.SNMPVersion, org.SNMPVersion, planIsAuthoritative)
	data.SNMPPort = orgResolveInt64(data.SNMPPort, org.SNMPPort, planIsAuthoritative)
	data.SNMPTransport = orgResolveString(data.SNMPTransport, org.SNMPTransport, planIsAuthoritative)
	data.SNMPv3SecurityLevel = orgResolveString(data.SNMPv3SecurityLevel, org.SNMPv3SecurityLevel, planIsAuthoritative)
	data.SNMPv3Username = orgResolveString(data.SNMPv3Username, org.SNMPv3Username, planIsAuthoritative)
	data.SNMPv3AuthProtocol = orgResolveString(data.SNMPv3AuthProtocol, org.SNMPv3AuthProtocol, planIsAuthoritative)
	data.SNMPv3PrivProtocol = orgResolveString(data.SNMPv3PrivProtocol, org.SNMPv3PrivProtocol, planIsAuthoritative)
	data.MikrotikEnabled = orgResolveBool(data.MikrotikEnabled, org.MikrotikEnabled, planIsAuthoritative)
	data.MikrotikUsername = orgResolveString(data.MikrotikUsername, org.MikrotikUsername, planIsAuthoritative)
	data.MikrotikPort = orgResolveInt64(data.MikrotikPort, org.MikrotikPort, planIsAuthoritative)
	data.MikrotikSSHPort = orgResolveInt64(data.MikrotikSSHPort, org.MikrotikSSHPort, planIsAuthoritative)
	data.MikrotikUseSSL = orgResolveBool(data.MikrotikUseSSL, org.MikrotikUseSSL, planIsAuthoritative)
	data.DefaultAgentTokenID = orgResolveString(data.DefaultAgentTokenID, org.DefaultAgentTokenID, planIsAuthoritative)
}

func orgKnownString(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown()
}

func orgStringPtr(v types.String) *string {
	if !orgKnownString(v) {
		return nil
	}
	return new(v.ValueString())
}

func orgIntPtr(v types.Int64) *int {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return new(int(v.ValueInt64()))
}

func orgBoolPtr(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return new(v.ValueBool())
}

func orgResolveString(planned types.String, apiValue *string, planIsAuthoritative bool) types.String {
	if planIsAuthoritative && orgKnownString(planned) {
		return planned
	}
	if apiValue == nil {
		return types.StringNull()
	}
	return types.StringValue(*apiValue)
}

func orgResolveInt64(planned types.Int64, apiValue *int, planIsAuthoritative bool) types.Int64 {
	if planIsAuthoritative && !planned.IsNull() && !planned.IsUnknown() {
		return planned
	}
	if apiValue == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*apiValue))
}

func orgResolveBool(planned types.Bool, apiValue *bool, planIsAuthoritative bool) types.Bool {
	if planIsAuthoritative && !planned.IsNull() && !planned.IsUnknown() {
		return planned
	}
	if apiValue == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*apiValue)
}
