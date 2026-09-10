package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SiteResource{}
var _ resource.ResourceWithImportState = &SiteResource{}

// SiteResource defines the resource implementation.
type SiteResource struct {
	client *Client
}

// SiteResourceModel describes the resource data model.
type SiteResourceModel struct {
	ID               types.String  `tfsdk:"id"`
	Name             types.String  `tfsdk:"name"`
	Description      types.String  `tfsdk:"description"`
	Location         types.String  `tfsdk:"location"`
	Address          types.String  `tfsdk:"address"`
	Latitude         types.Float64 `tfsdk:"latitude"`
	Longitude        types.Float64 `tfsdk:"longitude"`
	DisplayOrder     types.Int64   `tfsdk:"display_order"`
	SNMPCommunity    types.String  `tfsdk:"snmp_community"`
	SNMPCommunitySet types.Bool    `tfsdk:"snmp_community_set"`
	SNMPVersion      types.String  `tfsdk:"snmp_version"`
	SNMPPort         types.Int64   `tfsdk:"snmp_port"`
	SNMPTransport    types.String  `tfsdk:"snmp_transport"`
	AgentTokenID     types.String  `tfsdk:"agent_token_id"`
	ParentSiteID     types.String  `tfsdk:"parent_site_id"`
	InsertedAt       types.String  `tfsdk:"inserted_at"`
}

// NewSiteResource creates a new site resource.
func NewSiteResource() resource.Resource {
	return &SiteResource{}
}

func (r *SiteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (r *SiteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TowerOps site. Sites represent physical locations that contain devices.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the site.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the site. Must be between 2 and 200 characters.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A longer description of the site. Maximum 1000 characters.",
				Optional:    true,
			},
			"location": schema.StringAttribute{
				Description: "A short description of the physical location. Maximum 200 characters.",
				Optional:    true,
			},
			"address": schema.StringAttribute{
				Description: "The street address of the site. Maximum 500 characters.",
				Optional:    true,
			},
			"latitude": schema.Float64Attribute{
				Description: "The latitude of the site (-90 to 90).",
				Optional:    true,
				Computed:    true,
			},
			"longitude": schema.Float64Attribute{
				Description: "The longitude of the site (-180 to 180).",
				Optional:    true,
				Computed:    true,
			},
			"display_order": schema.Int64Attribute{
				Description: "The sort position of the site in TowerOps listings.",
				Optional:    true,
				Computed:    true,
			},
			"snmp_community": schema.StringAttribute{
				Description: "The default SNMP community string for devices at this site. Write only: the API accepts this value but never returns it, so use snmp_community_set to tell whether a community is stored.",
				Optional:    true,
				Sensitive:   true,
			},
			"snmp_community_set": schema.BoolAttribute{
				Description: "Whether the site has an SNMP community string stored.",
				Computed:    true,
			},
			"snmp_version": schema.StringAttribute{
				Description: "The default SNMP version for devices at this site. One of \"1\", \"2c\", or \"3\".",
				Optional:    true,
				Computed:    true,
			},
			"snmp_port": schema.Int64Attribute{
				Description: "The default SNMP port for devices at this site (1 to 65535).",
				Optional:    true,
				Computed:    true,
			},
			"snmp_transport": schema.StringAttribute{
				Description: "The default SNMP transport for devices at this site, for example \"udp\".",
				Optional:    true,
				Computed:    true,
			},
			"agent_token_id": schema.StringAttribute{
				Description: "The ID of the agent token that polls this site.",
				Optional:    true,
				Computed:    true,
			},
			"parent_site_id": schema.StringAttribute{
				Description: "The ID of the parent site, for nesting sites into a hierarchy.",
				Optional:    true,
			},
			"inserted_at": schema.StringAttribute{
				Description: "The timestamp when the site was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SiteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SiteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SiteResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := buildSiteFromModel(&data)

	created, err := r.client.CreateSite(site)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create site", err.Error())
		return
	}

	applySiteResponseToPlan(&data, created)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SiteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SiteResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site, err := r.client.GetSite(data.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Site was deleted outside of Terraform, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read site", err.Error())
		return
	}

	applySiteResponse(&data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SiteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SiteResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := buildSiteFromModel(&data)

	updated, err := r.client.UpdateSite(data.ID.ValueString(), site)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Site was deleted outside of Terraform, recreate it
			created, createErr := r.client.CreateSite(site)
			if createErr != nil {
				resp.Diagnostics.AddError("Failed to create site (after 404 on update)", createErr.Error())
				return
			}
			applySiteResponseToPlan(&data, created)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError("Failed to update site", err.Error())
		return
	}

	applySiteResponseToPlan(&data, updated)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SiteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SiteResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSite(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete site", err.Error())
		return
	}
}

func (r *SiteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// buildSiteFromModel converts the Terraform model to an API Site struct.
func buildSiteFromModel(data *SiteResourceModel) Site {
	site := Site{
		Name: data.Name.ValueString(),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		v := data.Description.ValueString()
		site.Description = &v
	}
	if !data.Location.IsNull() && !data.Location.IsUnknown() {
		v := data.Location.ValueString()
		site.Location = &v
	}
	if !data.Address.IsNull() && !data.Address.IsUnknown() {
		v := data.Address.ValueString()
		site.Address = &v
	}
	if !data.Latitude.IsNull() && !data.Latitude.IsUnknown() {
		v := data.Latitude.ValueFloat64()
		site.Latitude = &v
	}
	if !data.Longitude.IsNull() && !data.Longitude.IsUnknown() {
		v := data.Longitude.ValueFloat64()
		site.Longitude = &v
	}
	if !data.DisplayOrder.IsNull() && !data.DisplayOrder.IsUnknown() {
		v := int(data.DisplayOrder.ValueInt64())
		site.DisplayOrder = &v
	}
	if !data.SNMPCommunity.IsNull() && !data.SNMPCommunity.IsUnknown() {
		v := data.SNMPCommunity.ValueString()
		site.SNMPCommunity = &v
	}
	if !data.SNMPVersion.IsNull() && !data.SNMPVersion.IsUnknown() {
		v := data.SNMPVersion.ValueString()
		site.SNMPVersion = &v
	}
	if !data.SNMPPort.IsNull() && !data.SNMPPort.IsUnknown() {
		v := int(data.SNMPPort.ValueInt64())
		site.SNMPPort = &v
	}
	if !data.SNMPTransport.IsNull() && !data.SNMPTransport.IsUnknown() {
		v := data.SNMPTransport.ValueString()
		site.SNMPTransport = &v
	}
	if !data.AgentTokenID.IsNull() && !data.AgentTokenID.IsUnknown() {
		v := data.AgentTokenID.ValueString()
		site.AgentTokenID = &v
	}
	if !data.ParentSiteID.IsNull() && !data.ParentSiteID.IsUnknown() {
		v := data.ParentSiteID.ValueString()
		site.ParentSiteID = &v
	}

	return site
}

// applySiteResponse maps every returned field of an API response onto the
// model. It is used by Read, where a value that differs from the configuration
// is real drift and must land in state.
//
// SNMPCommunity is never touched: the API does not return it, so the prior
// state value stays and only SNMPCommunitySet reflects the server.
func applySiteResponse(data *SiteResourceModel, site *Site) {
	data.ID = types.StringValue(site.ID)
	data.Name = types.StringValue(site.Name)
	data.InsertedAt = types.StringValue(site.InsertedAt)
	data.SNMPCommunitySet = types.BoolValue(site.SNMPCommunitySet)

	data.Description = siteStringState(site.Description)
	data.Location = siteStringState(site.Location)
	data.Address = siteStringState(site.Address)
	data.Latitude = siteFloat64State(site.Latitude)
	data.Longitude = siteFloat64State(site.Longitude)
	data.DisplayOrder = siteInt64State(site.DisplayOrder)
	data.SNMPVersion = siteStringState(site.SNMPVersion)
	data.SNMPPort = siteInt64State(site.SNMPPort)
	data.SNMPTransport = siteStringState(site.SNMPTransport)
	data.AgentTokenID = siteStringState(site.AgentTokenID)
	data.ParentSiteID = siteStringState(site.ParentSiteID)
}

// applySiteResponseToPlan fills in the values Terraform does not already know,
// and only those. Create and Update must return exactly the planned value for
// every attribute whose plan is known, otherwise Terraform reports "Provider
// produced inconsistent result after apply". An Optional and Computed
// attribute therefore only takes its value from the response when the plan
// left it null or unknown; purely computed attributes always take it.
//
// SNMPCommunity is write only and never assigned from a response.
func applySiteResponseToPlan(data *SiteResourceModel, site *Site) {
	data.ID = types.StringValue(site.ID)
	data.InsertedAt = types.StringValue(site.InsertedAt)
	data.SNMPCommunitySet = types.BoolValue(site.SNMPCommunitySet)

	if siteValueUnset(data.Latitude) {
		data.Latitude = siteFloat64State(site.Latitude)
	}
	if siteValueUnset(data.Longitude) {
		data.Longitude = siteFloat64State(site.Longitude)
	}
	if siteValueUnset(data.DisplayOrder) {
		data.DisplayOrder = siteInt64State(site.DisplayOrder)
	}
	if siteValueUnset(data.SNMPVersion) {
		data.SNMPVersion = siteStringState(site.SNMPVersion)
	}
	if siteValueUnset(data.SNMPPort) {
		data.SNMPPort = siteInt64State(site.SNMPPort)
	}
	if siteValueUnset(data.SNMPTransport) {
		data.SNMPTransport = siteStringState(site.SNMPTransport)
	}
	if siteValueUnset(data.AgentTokenID) {
		data.AgentTokenID = siteStringState(site.AgentTokenID)
	}
}

// siteValueUnset reports whether Terraform has no known value for an
// attribute, which is the only case where a response value may be adopted
// during Create or Update.
func siteValueUnset(value attr.Value) bool {
	return value.IsNull() || value.IsUnknown()
}

func siteStringState(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func siteFloat64State(value *float64) types.Float64 {
	if value == nil {
		return types.Float64Null()
	}
	return types.Float64Value(*value)
}

func siteInt64State(value *int) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*value))
}
