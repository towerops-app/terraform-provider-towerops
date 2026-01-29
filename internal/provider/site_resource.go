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

var _ resource.Resource = &SiteResource{}
var _ resource.ResourceWithImportState = &SiteResource{}

// SiteResource defines the resource implementation.
type SiteResource struct {
	client *Client
}

// SiteResourceModel describes the resource data model.
type SiteResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Location      types.String `tfsdk:"location"`
	SNMPCommunity types.String `tfsdk:"snmp_community"`
	InsertedAt    types.String `tfsdk:"inserted_at"`
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
			"location": schema.StringAttribute{
				Description: "The physical location or address of the site.",
				Optional:    true,
			},
			"snmp_community": schema.StringAttribute{
				Description: "The default SNMP community string for devices at this site.",
				Optional:    true,
				Sensitive:   true,
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

	site := Site{
		Name: data.Name.ValueString(),
	}

	if !data.Location.IsNull() {
		location := data.Location.ValueString()
		site.Location = &location
	}

	if !data.SNMPCommunity.IsNull() {
		community := data.SNMPCommunity.ValueString()
		site.SNMPCommunity = &community
	}

	created, err := r.client.CreateSite(site)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create site", err.Error())
		return
	}

	data.ID = types.StringValue(created.ID)
	data.InsertedAt = types.StringValue(created.InsertedAt)

	if created.Location != nil {
		data.Location = types.StringValue(*created.Location)
	}
	if created.SNMPCommunity != nil {
		data.SNMPCommunity = types.StringValue(*created.SNMPCommunity)
	}

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

	data.Name = types.StringValue(site.Name)
	data.InsertedAt = types.StringValue(site.InsertedAt)

	if site.Location != nil {
		data.Location = types.StringValue(*site.Location)
	} else {
		data.Location = types.StringNull()
	}

	if site.SNMPCommunity != nil {
		data.SNMPCommunity = types.StringValue(*site.SNMPCommunity)
	} else {
		data.SNMPCommunity = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SiteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SiteResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := Site{
		Name: data.Name.ValueString(),
	}

	if !data.Location.IsNull() {
		location := data.Location.ValueString()
		site.Location = &location
	}

	if !data.SNMPCommunity.IsNull() {
		community := data.SNMPCommunity.ValueString()
		site.SNMPCommunity = &community
	}

	updated, err := r.client.UpdateSite(data.ID.ValueString(), site)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Site was deleted outside of Terraform, recreate it
			created, createErr := r.client.CreateSite(site)
			if createErr != nil {
				resp.Diagnostics.AddError("Failed to create site (after 404 on update)", createErr.Error())
				return
			}
			data.ID = types.StringValue(created.ID)
			data.InsertedAt = types.StringValue(created.InsertedAt)
			data.Name = types.StringValue(created.Name)
			if created.Location != nil {
				data.Location = types.StringValue(*created.Location)
			}
			if created.SNMPCommunity != nil {
				data.SNMPCommunity = types.StringValue(*created.SNMPCommunity)
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError("Failed to update site", err.Error())
		return
	}

	data.Name = types.StringValue(updated.Name)

	if updated.Location != nil {
		data.Location = types.StringValue(*updated.Location)
	}
	if updated.SNMPCommunity != nil {
		data.SNMPCommunity = types.StringValue(*updated.SNMPCommunity)
	}

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
