package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &AgentResource{}
var _ resource.ResourceWithImportState = &AgentResource{}

// AgentResource defines the resource implementation.
type AgentResource struct {
	client *Client
}

// AgentResourceModel describes the resource data model.
type AgentResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Token       types.String `tfsdk:"token"`
	LastIP      types.String `tfsdk:"last_ip"`
	Metadata    types.Map    `tfsdk:"metadata"`
	DeviceCount types.Int64  `tfsdk:"device_count"`
	InsertedAt  types.String `tfsdk:"inserted_at"`
}

// NewAgentResource creates a new agent resource.
func NewAgentResource() resource.Resource {
	return &AgentResource{}
}

func (r *AgentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (r *AgentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TowerOps agent token. Agents are deployed on customer networks to poll devices via SNMP, ping, and SSH.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the agent.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the agent. Changing this forces a new resource to be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				Description: "The bearer token for this agent. Only available after creation and cannot be retrieved again.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_ip": schema.StringAttribute{
				Description: "The source IP address the agent last checked in from.",
				Computed:    true,
			},
			"metadata": schema.MapAttribute{
				Description: "Metadata the agent reported on its last check-in, such as its version and host details. Values the API sends as numbers, booleans, objects or arrays are stored as their JSON encoding.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"device_count": schema.Int64Attribute{
				Description: "The number of devices currently assigned to this agent.",
				Computed:    true,
			},
			"inserted_at": schema.StringAttribute{
				Description: "The timestamp when the agent was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AgentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AgentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	agent := Agent{
		Name: data.Name.ValueString(),
	}

	created, err := r.client.CreateAgent(agent)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create agent", err.Error())
		return
	}

	data.ID = types.StringValue(created.ID)
	data.Token = types.StringValue(created.Token)
	data.InsertedAt = types.StringValue(created.InsertedAt)

	if created.LastIP != nil {
		data.LastIP = types.StringValue(*created.LastIP)
	} else {
		data.LastIP = types.StringNull()
	}

	metadata, diags := agentMetadataToMap(ctx, created.Metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Metadata = metadata

	// The create response carries no device_count: only the show action adds
	// it. A freshly issued token has nothing assigned to it yet, and the
	// attribute is computed, so it needs a known value before state is written.
	if created.DeviceCount != nil {
		data.DeviceCount = types.Int64Value(int64(*created.DeviceCount))
	} else {
		data.DeviceCount = types.Int64Value(0)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AgentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	agent, err := r.client.GetAgent(data.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read agent", err.Error())
		return
	}

	data.Name = types.StringValue(agent.Name)
	data.InsertedAt = types.StringValue(agent.InsertedAt)
	// Token is only returned by the create response, so the state value stands.

	if agent.LastIP != nil {
		data.LastIP = types.StringValue(*agent.LastIP)
	} else {
		data.LastIP = types.StringNull()
	}

	metadata, diags := agentMetadataToMap(ctx, agent.Metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Metadata = metadata

	if agent.DeviceCount != nil {
		data.DeviceCount = types.Int64Value(int64(*agent.DeviceCount))
	} else {
		data.DeviceCount = types.Int64Value(0)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is unreachable in practice. The TowerOps API has no agent update
// action (the agents controller implements index, create, show and delete
// only), and name, the sole configurable attribute, requires replacement, so
// Terraform destroys and recreates the agent instead of calling this. The
// diagnostic exists so a future writable attribute cannot fail silently.
func (r *AgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Agent Update Not Supported",
		"The TowerOps API does not expose an update action for agent tokens. Changing the name replaces the agent, which issues a new token; every other attribute is read-only.",
	)
}

func (r *AgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AgentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAgent(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete agent", err.Error())
		return
	}
}

func (r *AgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// agentMetadataToMap converts the arbitrary JSON object the API reports as
// agent metadata into a map of strings. Nested and non-string values keep
// their JSON encoding so nothing is lost, and absent metadata becomes a null
// map rather than an empty one.
func agentMetadataToMap(ctx context.Context, metadata map[string]any) (types.Map, diag.Diagnostics) {
	if len(metadata) == 0 {
		return types.MapNull(types.StringType), nil
	}

	flat := make(map[string]string, len(metadata))
	for key, value := range metadata {
		switch typed := value.(type) {
		case nil:
			flat[key] = ""
		case string:
			flat[key] = typed
		default:
			encoded, err := json.Marshal(typed)
			if err != nil {
				flat[key] = fmt.Sprintf("%v", typed)
				continue
			}
			flat[key] = string(encoded)
		}
	}

	return types.MapValueFrom(ctx, types.StringType, flat)
}
