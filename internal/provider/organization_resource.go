package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &OrganizationResource{}

// OrganizationResource manages organization settings.
type OrganizationResource struct {
	client *Client
}

// OrganizationResourceModel describes the resource data model.
type OrganizationResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Slug          types.String `tfsdk:"slug"`
	UseSites      types.Bool   `tfsdk:"use_sites"`
	SnmpCommunity types.String `tfsdk:"snmp_community"`
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
				Description: "The URL-friendly slug of the organization.",
				Computed:    true,
			},
			"use_sites": schema.BoolAttribute{
				Description: "Whether the organization uses sites to group devices. When true, devices are organized under sites. When false, devices belong directly to the organization.",
				Required:    true,
			},
			"snmp_community": schema.StringAttribute{
				Description: "Default SNMP community string for devices. Can only be set by organization owners.",
				Optional:    true,
				Sensitive:   true,
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

	org := Organization{
		UseSites: data.UseSites.ValueBool(),
	}

	// Include name if provided
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		org.Name = data.Name.ValueString()
	}

	// Include SNMP community if provided
	if !data.SnmpCommunity.IsNull() && !data.SnmpCommunity.IsUnknown() {
		org.SnmpCommunity = data.SnmpCommunity.ValueString()
	}

	updated, err := r.client.UpdateOrganization(org)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update organization", err.Error())
		return
	}

	data.ID = types.StringValue(updated.ID)
	data.Name = types.StringValue(updated.Name)
	data.Slug = types.StringValue(updated.Slug)
	data.UseSites = types.BoolValue(updated.UseSites)
	data.SnmpCommunity = types.StringValue(updated.SnmpCommunity)

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
		resp.Diagnostics.AddError("Failed to read organization", err.Error())
		return
	}

	data.ID = types.StringValue(org.ID)
	data.Name = types.StringValue(org.Name)
	data.Slug = types.StringValue(org.Slug)
	data.UseSites = types.BoolValue(org.UseSites)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OrganizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := Organization{
		UseSites: data.UseSites.ValueBool(),
	}

	// Include name if provided
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		org.Name = data.Name.ValueString()
	}

	// Include SNMP community if provided
	if !data.SnmpCommunity.IsNull() && !data.SnmpCommunity.IsUnknown() {
		org.SnmpCommunity = data.SnmpCommunity.ValueString()
	}

	updated, err := r.client.UpdateOrganization(org)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update organization", err.Error())
		return
	}

	data.ID = types.StringValue(updated.ID)
	data.Name = types.StringValue(updated.Name)
	data.Slug = types.StringValue(updated.Slug)
	data.UseSites = types.BoolValue(updated.UseSites)
	data.SnmpCommunity = types.StringValue(updated.SnmpCommunity)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Organizations cannot be deleted via the API. When this resource is
	// removed from Terraform config, we simply remove it from state.
	// The organization continues to exist in TowerOps.
}
