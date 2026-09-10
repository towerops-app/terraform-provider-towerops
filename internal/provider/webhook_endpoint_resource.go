package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &WebhookEndpointResource{}
var _ resource.ResourceWithImportState = &WebhookEndpointResource{}

// WebhookEndpointResource defines the resource implementation.
type WebhookEndpointResource struct {
	client *Client
}

// WebhookEndpointResourceModel describes the resource data model.
type WebhookEndpointResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	URL                 types.String `tfsdk:"url"`
	Secret              types.String `tfsdk:"secret"`
	Events              types.List   `tfsdk:"events"`
	Active              types.Bool   `tfsdk:"active"`
	ConsecutiveFailures types.Int64  `tfsdk:"consecutive_failures"`
	LastDeliveryAt      types.String `tfsdk:"last_delivery_at"`
	LastDeliveryStatus  types.String `tfsdk:"last_delivery_status"`
	CreatedAt           types.String `tfsdk:"created_at"`
}

// NewWebhookEndpointResource creates a new webhook endpoint resource.
func NewWebhookEndpointResource() resource.Resource {
	return &WebhookEndpointResource{}
}

func (r *WebhookEndpointResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_endpoint"
}

func (r *WebhookEndpointResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TowerOps outbound webhook endpoint. The endpoint receives HMAC-SHA256 signed POSTs when the subscribed organization events fire. Every action on this resource requires an API token with the 'webhooks:manage' scope.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the webhook endpoint.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Description: "The HTTPS URL that receives the signed event payloads. The API rejects URLs that resolve to private or link-local addresses.",
				Required:    true,
			},
			"events": schema.ListAttribute{
				Description: "The event types this endpoint subscribes to. Must contain at least one entry. Valid values: alert.triggered, alert.resolved, alert.acknowledged, device.status_changed, config.changed, subscriber_impact.detected, maintenance.started, maintenance.ended.",
				Required:    true,
				ElementType: types.StringType,
			},
			"name": schema.StringAttribute{
				Description: "A human readable label for the endpoint. Maximum 255 characters.",
				Optional:    true,
			},
			"secret": schema.StringAttribute{
				Description: "The shared secret used to sign outgoing requests, 16 to 256 characters. Omit it to have the server generate one. The API returns the secret only once, on the create response, so it is captured then and never refreshed afterwards. Removing it from the configuration does not rotate it; set a new value to rotate.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"active": schema.BoolAttribute{
				Description: "Whether the endpoint receives deliveries. Defaults to true. The server latches this to false when the delivery circuit breaker trips, so an endpoint that should stay off must be declared with active = false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"consecutive_failures": schema.Int64Attribute{
				Description: "The number of consecutive terminal delivery failures. Server managed: it is reset when the endpoint is re-enabled.",
				Computed:    true,
			},
			"last_delivery_at": schema.StringAttribute{
				Description: "The timestamp of the most recent delivery attempt.",
				Computed:    true,
			},
			"last_delivery_status": schema.StringAttribute{
				Description: "The status of the most recent delivery attempt.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the webhook endpoint was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *WebhookEndpointResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebhookEndpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WebhookEndpointResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint, diags := buildWebhookEndpointFromModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateWebhookEndpoint(endpoint)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create webhook endpoint", err.Error())
		return
	}

	data.ID = types.StringValue(created.ID)
	data.CreatedAt = types.StringValue(created.CreatedAt)
	setWebhookEndpointDeliveryFields(&data, created)
	// url, name and events are authoritative from the plan, so the response
	// never overwrites them: a differing value would make the apply
	// inconsistent. Drift in those fields is reported by Read instead.
	if data.Active.IsNull() || data.Active.IsUnknown() {
		data.Active = webhookEndpointActive(created)
	}
	setWebhookEndpointSecret(&data, created)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookEndpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WebhookEndpointResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint, err := r.client.GetWebhookEndpoint(data.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Endpoint was deleted outside of Terraform, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read webhook endpoint", err.Error())
		return
	}

	data.URL = types.StringValue(endpoint.URL)
	data.CreatedAt = types.StringValue(endpoint.CreatedAt)
	data.Active = webhookEndpointActive(endpoint)
	setWebhookEndpointDeliveryFields(&data, endpoint)

	if endpoint.Name != nil {
		data.Name = types.StringValue(*endpoint.Name)
	} else {
		data.Name = types.StringNull()
	}

	events, diags := webhookEndpointEvents(ctx, endpoint)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Events = events

	// The secret is deliberately left untouched. A show response never carries
	// it, so copying the response would wipe the value the practitioner
	// configured (or the generated one captured at create time) out of state.

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookEndpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WebhookEndpointResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint, diags := buildWebhookEndpointFromModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateWebhookEndpoint(data.ID.ValueString(), endpoint)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Deliberately not recreated here: a new endpoint would be signed
			// with a new server generated secret when the configuration does
			// not pin one, silently breaking the receiver.
			resp.Diagnostics.AddError(
				"Webhook endpoint no longer exists",
				"The webhook endpoint was deleted outside of Terraform. Run 'terraform apply -refresh-only' to drop it from state, then apply again to recreate it.",
			)
			return
		}
		resp.Diagnostics.AddError("Failed to update webhook endpoint", err.Error())
		return
	}

	data.CreatedAt = types.StringValue(updated.CreatedAt)
	setWebhookEndpointDeliveryFields(&data, updated)
	// As in Create, the planned url, name, events and active values win.
	if data.Active.IsNull() || data.Active.IsUnknown() {
		data.Active = webhookEndpointActive(updated)
	}
	// An update response never carries the secret, so the planned value (which
	// falls back to the prior state for an unconfigured secret) stays put.
	setWebhookEndpointSecret(&data, updated)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookEndpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WebhookEndpointResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteWebhookEndpoint(data.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Already gone upstream, nothing to delete.
			return
		}
		resp.Diagnostics.AddError("Failed to delete webhook endpoint", err.Error())
		return
	}
}

func (r *WebhookEndpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The imported state carries a null secret: the API cannot return it.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// buildWebhookEndpointFromModel converts the Terraform model to an API
// WebhookEndpoint struct.
func buildWebhookEndpointFromModel(ctx context.Context, data *WebhookEndpointResourceModel) (WebhookEndpoint, diag.Diagnostics) {
	endpoint := WebhookEndpoint{
		URL: data.URL.ValueString(),
	}

	var diags diag.Diagnostics
	if !data.Events.IsNull() && !data.Events.IsUnknown() {
		events := make([]string, 0, len(data.Events.Elements()))
		diags.Append(data.Events.ElementsAs(ctx, &events, false)...)
		endpoint.Events = events
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		v := data.Name.ValueString()
		endpoint.Name = &v
	}
	if !data.Secret.IsNull() && !data.Secret.IsUnknown() {
		v := data.Secret.ValueString()
		endpoint.Secret = &v
	}
	if !data.Active.IsNull() && !data.Active.IsUnknown() {
		v := data.Active.ValueBool()
		endpoint.Active = &v
	}

	return endpoint, diags
}

// setWebhookEndpointSecret captures a secret the API is willing to disclose.
//
// Two halves of one rule. When the practitioner configured a secret, the
// planned value is authoritative and is left alone. When they did not, the
// planned value is unknown, the server generated a secret, and the create
// response is the only chance to see it, so it is stored then. A response that
// omits the secret (every read and update) never clears state.
func setWebhookEndpointSecret(data *WebhookEndpointResourceModel, endpoint *WebhookEndpoint) {
	if !data.Secret.IsNull() && !data.Secret.IsUnknown() {
		return
	}
	if endpoint.Secret != nil {
		data.Secret = types.StringValue(*endpoint.Secret)
		return
	}
	data.Secret = types.StringNull()
}

// setWebhookEndpointDeliveryFields maps the server managed circuit breaker and
// delivery telemetry back to the Terraform model.
func setWebhookEndpointDeliveryFields(data *WebhookEndpointResourceModel, endpoint *WebhookEndpoint) {
	// The counter is never null upstream: the column defaults to 0 and the
	// serializer always emits it.
	if endpoint.ConsecutiveFailures != nil {
		data.ConsecutiveFailures = types.Int64Value(int64(*endpoint.ConsecutiveFailures))
	} else {
		data.ConsecutiveFailures = types.Int64Value(0)
	}

	if endpoint.LastDeliveryAt != nil {
		data.LastDeliveryAt = types.StringValue(*endpoint.LastDeliveryAt)
	} else {
		data.LastDeliveryAt = types.StringNull()
	}
	if endpoint.LastDeliveryStatus != nil {
		data.LastDeliveryStatus = types.StringValue(*endpoint.LastDeliveryStatus)
	} else {
		data.LastDeliveryStatus = types.StringNull()
	}
}

// webhookEndpointActive reads the active flag, falling back to the schema
// default when the response omits it.
func webhookEndpointActive(endpoint *WebhookEndpoint) types.Bool {
	if endpoint.Active != nil {
		return types.BoolValue(*endpoint.Active)
	}
	return types.BoolValue(true)
}

// webhookEndpointEvents converts the API event list to a Terraform list.
func webhookEndpointEvents(ctx context.Context, endpoint *WebhookEndpoint) (types.List, diag.Diagnostics) {
	if endpoint.Events == nil {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, endpoint.Events)
}
