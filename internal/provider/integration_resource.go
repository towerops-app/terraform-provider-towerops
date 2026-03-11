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

var _ resource.Resource = &IntegrationResource{}
var _ resource.ResourceWithImportState = &IntegrationResource{}

// IntegrationResource defines the resource implementation.
type IntegrationResource struct {
	client *Client
}

// IntegrationResourceModel describes the resource data model.
type IntegrationResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Provider            types.String `tfsdk:"provider"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	SyncIntervalMinutes types.Int64  `tfsdk:"sync_interval_minutes"`
	InsertedAt          types.String `tfsdk:"inserted_at"`
}

// NewIntegrationResource creates a new integration resource.
func NewIntegrationResource() resource.Resource {
	return &IntegrationResource{}
}

func (r *IntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

func (r *IntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TowerOps integration with third-party services.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the integration.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"provider": schema.StringAttribute{
				Description: "The integration provider type (e.g. pagerduty, slack, webhook).",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the integration is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"sync_interval_minutes": schema.Int64Attribute{
				Description: "How often the integration syncs, in minutes.",
				Optional:    true,
			},
			"inserted_at": schema.StringAttribute{
				Description: "The timestamp when the integration was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *IntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	integration := integrationWithCredentials{
		Provider: data.Provider.ValueString(),
	}

	if !data.Enabled.IsNull() {
		enabled := data.Enabled.ValueBool()
		integration.Enabled = &enabled
	}

	if !data.SyncIntervalMinutes.IsNull() {
		interval := int(data.SyncIntervalMinutes.ValueInt64())
		integration.SyncIntervalMinutes = &interval
	}

	created, err := r.client.CreateIntegration(integration)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create integration", err.Error())
		return
	}

	data.ID = types.StringValue(created.ID)
	data.InsertedAt = types.StringValue(created.InsertedAt)

	if created.Enabled != nil {
		data.Enabled = types.BoolValue(*created.Enabled)
	}
	if created.SyncIntervalMinutes != nil {
		data.SyncIntervalMinutes = types.Int64Value(int64(*created.SyncIntervalMinutes))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	integration, err := r.client.GetIntegration(data.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read integration", err.Error())
		return
	}

	data.Provider = types.StringValue(integration.Provider)
	data.InsertedAt = types.StringValue(integration.InsertedAt)

	if integration.Enabled != nil {
		data.Enabled = types.BoolValue(*integration.Enabled)
	}
	if integration.SyncIntervalMinutes != nil {
		data.SyncIntervalMinutes = types.Int64Value(int64(*integration.SyncIntervalMinutes))
	} else {
		data.SyncIntervalMinutes = types.Int64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	integration := integrationWithCredentials{
		Provider: data.Provider.ValueString(),
	}

	if !data.Enabled.IsNull() {
		enabled := data.Enabled.ValueBool()
		integration.Enabled = &enabled
	}

	if !data.SyncIntervalMinutes.IsNull() {
		interval := int(data.SyncIntervalMinutes.ValueInt64())
		integration.SyncIntervalMinutes = &interval
	}

	updated, err := r.client.UpdateIntegration(data.ID.ValueString(), integration)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			created, createErr := r.client.CreateIntegration(integration)
			if createErr != nil {
				resp.Diagnostics.AddError("Failed to create integration (after 404 on update)", createErr.Error())
				return
			}
			data.ID = types.StringValue(created.ID)
			data.InsertedAt = types.StringValue(created.InsertedAt)
			if created.Enabled != nil {
				data.Enabled = types.BoolValue(*created.Enabled)
			}
			if created.SyncIntervalMinutes != nil {
				data.SyncIntervalMinutes = types.Int64Value(int64(*created.SyncIntervalMinutes))
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError("Failed to update integration", err.Error())
		return
	}

	data.Provider = types.StringValue(updated.Provider)

	if updated.Enabled != nil {
		data.Enabled = types.BoolValue(*updated.Enabled)
	}
	if updated.SyncIntervalMinutes != nil {
		data.SyncIntervalMinutes = types.Int64Value(int64(*updated.SyncIntervalMinutes))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteIntegration(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete integration", err.Error())
		return
	}
}

func (r *IntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
