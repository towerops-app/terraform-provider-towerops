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

var _ resource.Resource = &CheckResource{}
var _ resource.ResourceWithImportState = &CheckResource{}

// CheckResource defines the resource implementation.
type CheckResource struct {
	client *Client
}

// CheckResourceModel describes the resource data model.
type CheckResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	CheckType            types.String `tfsdk:"check_type"`
	Description          types.String `tfsdk:"description"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	Alerting             types.Bool   `tfsdk:"alerting"`
	DeviceID             types.String `tfsdk:"device_id"`
	AgentTokenID         types.String `tfsdk:"agent_token_id"`
	IntervalSeconds      types.Int64  `tfsdk:"interval_seconds"`
	TimeoutMs            types.Int64  `tfsdk:"timeout_ms"`
	RetryIntervalSeconds types.Int64  `tfsdk:"retry_interval_seconds"`
	MaxCheckAttempts     types.Int64  `tfsdk:"max_check_attempts"`

	// HTTP config
	URL             types.String `tfsdk:"url"`
	Method          types.String `tfsdk:"method"`
	ExpectedStatus  types.Int64  `tfsdk:"expected_status"`
	VerifySSL       types.Bool   `tfsdk:"verify_ssl"`
	FollowRedirects types.Bool   `tfsdk:"follow_redirects"`
	ContentMatch    types.String `tfsdk:"content_match"`

	// TCP config
	Host         types.String `tfsdk:"host"`
	Port         types.Int64  `tfsdk:"port"`
	SendString   types.String `tfsdk:"send_string"`
	ExpectString types.String `tfsdk:"expect_string"`

	// DNS config
	Hostname       types.String `tfsdk:"hostname"`
	DNSServer      types.String `tfsdk:"dns_server"`
	RecordType     types.String `tfsdk:"record_type"`
	ExpectedResult types.String `tfsdk:"expected_result"`

	// Ping config
	PingCount types.Int64 `tfsdk:"ping_count"`

	// Computed
	CurrentState     types.Int64  `tfsdk:"current_state"`
	CurrentStateType types.String `tfsdk:"current_state_type"`
	LastCheckAt      types.String `tfsdk:"last_check_at"`
	InsertedAt       types.String `tfsdk:"inserted_at"`
}

// NewCheckResource creates a new check resource.
func NewCheckResource() resource.Resource {
	return &CheckResource{}
}

func (r *CheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check"
}

func (r *CheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TowerOps service check. Supports HTTP, TCP, DNS, and ping check types.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the check.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the check.",
				Required:    true,
			},
			"check_type": schema.StringAttribute{
				Description: "The type of check. The REST API accepts only http, tcp, dns, and ping; any other value is rejected with a 400 bad_request.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of the check.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the check is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"alerting": schema.BoolAttribute{
				Description: "Whether the check raises alerts. The API ignores this at create time (a check is armed by default) and only honors it on update, so setting it to false on a new check takes effect on the following apply.",
				Optional:    true,
				Computed:    true,
			},
			"device_id": schema.StringAttribute{
				Description: "The ID of the device this check is associated with.",
				Optional:    true,
			},
			"agent_token_id": schema.StringAttribute{
				Description: "The ID of the agent token that executes this check.",
				Optional:    true,
			},
			"interval_seconds": schema.Int64Attribute{
				Description: "How often the check runs, in seconds.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(60),
			},
			"timeout_ms": schema.Int64Attribute{
				Description: "Timeout for the check execution, in milliseconds.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5000),
			},
			"retry_interval_seconds": schema.Int64Attribute{
				Description: "Interval between retries after a failure, in seconds.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
			},
			"max_check_attempts": schema.Int64Attribute{
				Description: "Number of consecutive failures before transitioning to hard state.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3),
			},

			// HTTP config
			"url": schema.StringAttribute{
				Description: "The URL to check. Required for HTTP checks.",
				Optional:    true,
			},
			"method": schema.StringAttribute{
				Description: "HTTP method to use. Defaults to GET.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("GET"),
			},
			"expected_status": schema.Int64Attribute{
				Description: "Expected HTTP status code. Defaults to 200.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(200),
			},
			"verify_ssl": schema.BoolAttribute{
				Description: "Whether to verify SSL certificates. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"follow_redirects": schema.BoolAttribute{
				Description: "Whether to follow HTTP redirects. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"content_match": schema.StringAttribute{
				Description: "Regex pattern to match against HTTP response body.",
				Optional:    true,
			},

			// TCP config
			"host": schema.StringAttribute{
				Description: "The host to check. Used for TCP and ping checks.",
				Optional:    true,
			},
			"port": schema.Int64Attribute{
				Description: "The TCP port to check. Required for TCP checks.",
				Optional:    true,
			},
			"send_string": schema.StringAttribute{
				Description: "Data to send after TCP connection.",
				Optional:    true,
			},
			"expect_string": schema.StringAttribute{
				Description: "Expected response string for TCP checks.",
				Optional:    true,
			},

			// DNS config
			"hostname": schema.StringAttribute{
				Description: "The hostname to resolve. Required for DNS checks.",
				Optional:    true,
			},
			"dns_server": schema.StringAttribute{
				Description: "The DNS server to query. Defaults to the resolver of the executing host.",
				Optional:    true,
			},
			"record_type": schema.StringAttribute{
				Description: "DNS record type (A, AAAA, CNAME, MX, TXT, NS, PTR). Defaults to A.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("A"),
			},
			"expected_result": schema.StringAttribute{
				Description: "Expected DNS resolution result.",
				Optional:    true,
			},

			// Ping config
			"ping_count": schema.Int64Attribute{
				Description: "Number of ping packets to send. Defaults to 3.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3),
			},

			// Computed
			"current_state": schema.Int64Attribute{
				Description: "Current check state (0=OK, 1=WARNING, 2=CRITICAL, 3=UNKNOWN).",
				Computed:    true,
			},
			"current_state_type": schema.StringAttribute{
				Description: "Current state type (soft or hard).",
				Computed:    true,
			},
			"last_check_at": schema.StringAttribute{
				Description: "Timestamp of the last check execution.",
				Computed:    true,
			},
			"inserted_at": schema.StringAttribute{
				Description: "The timestamp when the check was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *CheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check := Check{
		Name:      data.Name.ValueString(),
		CheckType: data.CheckType.ValueString(),
		Config:    buildConfig(data),
	}

	if !data.Description.IsNull() {
		desc := data.Description.ValueString()
		check.Description = &desc
	}

	if !data.Enabled.IsNull() {
		enabled := data.Enabled.ValueBool()
		check.Enabled = &enabled
	}

	if !data.DeviceID.IsNull() {
		id := data.DeviceID.ValueString()
		check.DeviceID = &id
	}

	if !data.AgentTokenID.IsNull() {
		id := data.AgentTokenID.ValueString()
		check.AgentTokenID = &id
	}

	if !data.IntervalSeconds.IsNull() {
		v := int(data.IntervalSeconds.ValueInt64())
		check.IntervalSeconds = &v
	}

	if !data.TimeoutMs.IsNull() {
		v := int(data.TimeoutMs.ValueInt64())
		check.TimeoutMs = &v
	}

	if !data.RetryIntervalSeconds.IsNull() {
		v := int(data.RetryIntervalSeconds.ValueInt64())
		check.RetryIntervalSeconds = &v
	}

	if !data.MaxCheckAttempts.IsNull() {
		v := int(data.MaxCheckAttempts.ValueInt64())
		check.MaxCheckAttempts = &v
	}

	created, err := r.client.CreateCheck(check)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create check", err.Error())
		return
	}

	applyCheckToPlan(created, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check, err := r.client.GetCheck(data.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read check", err.Error())
		return
	}

	mapCheckToState(check, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CheckResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check := Check{
		Name:      data.Name.ValueString(),
		CheckType: data.CheckType.ValueString(),
		Config:    buildConfig(data),
	}

	if !data.Description.IsNull() {
		desc := data.Description.ValueString()
		check.Description = &desc
	}

	if !data.Enabled.IsNull() {
		enabled := data.Enabled.ValueBool()
		check.Enabled = &enabled
	}

	// The create endpoint drops `alerting`, the update endpoint pops it out of
	// the `check` object and routes it through its own arming path, so this is
	// the only request that may carry it.
	if !data.Alerting.IsNull() && !data.Alerting.IsUnknown() {
		alerting := data.Alerting.ValueBool()
		check.Alerting = &alerting
	}

	if !data.DeviceID.IsNull() {
		id := data.DeviceID.ValueString()
		check.DeviceID = &id
	}

	if !data.AgentTokenID.IsNull() {
		id := data.AgentTokenID.ValueString()
		check.AgentTokenID = &id
	}

	if !data.IntervalSeconds.IsNull() {
		v := int(data.IntervalSeconds.ValueInt64())
		check.IntervalSeconds = &v
	}

	if !data.TimeoutMs.IsNull() {
		v := int(data.TimeoutMs.ValueInt64())
		check.TimeoutMs = &v
	}

	if !data.RetryIntervalSeconds.IsNull() {
		v := int(data.RetryIntervalSeconds.ValueInt64())
		check.RetryIntervalSeconds = &v
	}

	if !data.MaxCheckAttempts.IsNull() {
		v := int(data.MaxCheckAttempts.ValueInt64())
		check.MaxCheckAttempts = &v
	}

	updated, err := r.client.UpdateCheck(data.ID.ValueString(), check)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Check deleted outside Terraform, recreate
			created, createErr := r.client.CreateCheck(check)
			if createErr != nil {
				resp.Diagnostics.AddError("Failed to create check (after 404 on update)", createErr.Error())
				return
			}
			applyCheckToPlan(created, &data)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError("Failed to update check", err.Error())
		return
	}

	applyCheckToPlan(updated, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CheckResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCheck(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete check", err.Error())
		return
	}
}

func (r *CheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// buildConfig constructs the config map from flat Terraform attributes based on check_type.
func buildConfig(data CheckResourceModel) map[string]any {
	config := map[string]any{}
	checkType := data.CheckType.ValueString()

	switch checkType {
	case "http":
		if !data.URL.IsNull() {
			config["url"] = data.URL.ValueString()
		}
		if !data.Method.IsNull() {
			config["method"] = data.Method.ValueString()
		}
		if !data.ExpectedStatus.IsNull() {
			config["expected_status"] = int(data.ExpectedStatus.ValueInt64())
		}
		if !data.VerifySSL.IsNull() {
			config["verify_ssl"] = data.VerifySSL.ValueBool()
		}
		if !data.FollowRedirects.IsNull() {
			config["follow_redirects"] = data.FollowRedirects.ValueBool()
		}
		if !data.ContentMatch.IsNull() {
			config["regex"] = data.ContentMatch.ValueString()
		}

	case "tcp":
		if !data.Host.IsNull() {
			config["host"] = data.Host.ValueString()
		}
		if !data.Port.IsNull() {
			config["port"] = int(data.Port.ValueInt64())
		}
		if !data.SendString.IsNull() {
			config["send"] = data.SendString.ValueString()
		}
		if !data.ExpectString.IsNull() {
			config["expect"] = data.ExpectString.ValueString()
		}

	case "dns":
		if !data.Hostname.IsNull() {
			config["hostname"] = data.Hostname.ValueString()
		}
		if !data.RecordType.IsNull() {
			config["record_type"] = data.RecordType.ValueString()
		}
		if !data.DNSServer.IsNull() {
			config["server"] = data.DNSServer.ValueString()
		}
		if !data.ExpectedResult.IsNull() {
			config["expected"] = data.ExpectedResult.ValueString()
		}

	case "ping":
		if !data.Host.IsNull() {
			config["host"] = data.Host.ValueString()
		}
		if !data.PingCount.IsNull() {
			config["count"] = int(data.PingCount.ValueInt64())
		}
	}

	return config
}

// mapCheckToState maps an API Check response back into the Terraform state model.
func mapCheckToState(check *Check, data *CheckResourceModel) {
	data.ID = types.StringValue(check.ID)
	data.Name = types.StringValue(check.Name)
	data.CheckType = types.StringValue(check.CheckType)
	data.InsertedAt = types.StringValue(check.InsertedAt)

	if check.Description != nil {
		data.Description = types.StringValue(*check.Description)
	} else {
		data.Description = types.StringNull()
	}

	if check.Enabled != nil {
		data.Enabled = types.BoolValue(*check.Enabled)
	}

	if check.Alerting != nil {
		data.Alerting = types.BoolValue(*check.Alerting)
	}

	if check.DeviceID != nil {
		data.DeviceID = types.StringValue(*check.DeviceID)
	} else {
		data.DeviceID = types.StringNull()
	}

	if check.AgentTokenID != nil {
		data.AgentTokenID = types.StringValue(*check.AgentTokenID)
	} else {
		data.AgentTokenID = types.StringNull()
	}

	if check.IntervalSeconds != nil {
		data.IntervalSeconds = types.Int64Value(int64(*check.IntervalSeconds))
	}

	if check.TimeoutMs != nil {
		data.TimeoutMs = types.Int64Value(int64(*check.TimeoutMs))
	}

	if check.RetryIntervalSeconds != nil {
		data.RetryIntervalSeconds = types.Int64Value(int64(*check.RetryIntervalSeconds))
	}

	if check.MaxCheckAttempts != nil {
		data.MaxCheckAttempts = types.Int64Value(int64(*check.MaxCheckAttempts))
	}

	setCheckRuntimeState(check, data)

	// Unpack config into flat fields based on check_type
	unpackConfig(check.Config, check.CheckType, data)
}

// applyCheckToPlan fills the computed attributes of a planned model from an API
// response. Planned values are authoritative: Terraform rejects an apply whose
// result differs from a planned value that was already known, so nothing the
// plan decided is overwritten here. Detecting drift is Read's job.
func applyCheckToPlan(check *Check, data *CheckResourceModel) {
	data.ID = types.StringValue(check.ID)
	data.InsertedAt = types.StringValue(check.InsertedAt)

	// `alerting` is the one optional attribute the API decides on its own: the
	// create endpoint ignores whatever was sent and arms the check, so an
	// unconfigured value is unknown in the plan and has to come from the
	// response. A configured value stays as planned.
	if data.Alerting.IsUnknown() || data.Alerting.IsNull() {
		if check.Alerting != nil {
			data.Alerting = types.BoolValue(*check.Alerting)
		} else {
			// Checks are armed unless something disarms them.
			data.Alerting = types.BoolValue(true)
		}
	}

	setCheckRuntimeState(check, data)
}

// setCheckRuntimeState maps the read-only execution state of a check.
func setCheckRuntimeState(check *Check, data *CheckResourceModel) {
	if check.CurrentState != nil {
		data.CurrentState = types.Int64Value(int64(*check.CurrentState))
	} else {
		data.CurrentState = types.Int64Null()
	}

	if check.CurrentStateType != nil {
		data.CurrentStateType = types.StringValue(*check.CurrentStateType)
	} else {
		data.CurrentStateType = types.StringNull()
	}

	if check.LastCheckAt != nil {
		data.LastCheckAt = types.StringValue(*check.LastCheckAt)
	} else {
		data.LastCheckAt = types.StringNull()
	}
}

// unpackConfig reads the config map and sets flat Terraform attributes.
func unpackConfig(config map[string]any, checkType string, data *CheckResourceModel) {
	if config == nil {
		return
	}

	switch checkType {
	case "http":
		if v, ok := config["url"].(string); ok {
			data.URL = types.StringValue(v)
		}
		if v, ok := config["method"].(string); ok {
			data.Method = types.StringValue(v)
		}
		if v, ok := configInt(config, "expected_status"); ok {
			data.ExpectedStatus = types.Int64Value(int64(v))
		}
		if v, ok := config["verify_ssl"].(bool); ok {
			data.VerifySSL = types.BoolValue(v)
		}
		if v, ok := config["follow_redirects"].(bool); ok {
			data.FollowRedirects = types.BoolValue(v)
		}
		if v, ok := config["regex"].(string); ok {
			data.ContentMatch = types.StringValue(v)
		}

	case "tcp":
		if v, ok := config["host"].(string); ok {
			data.Host = types.StringValue(v)
		}
		if v, ok := configInt(config, "port"); ok {
			data.Port = types.Int64Value(int64(v))
		}
		if v, ok := config["send"].(string); ok {
			data.SendString = types.StringValue(v)
		}
		if v, ok := config["expect"].(string); ok {
			data.ExpectString = types.StringValue(v)
		}

	case "dns":
		if v, ok := config["hostname"].(string); ok {
			data.Hostname = types.StringValue(v)
		}
		if v, ok := config["server"].(string); ok {
			data.DNSServer = types.StringValue(v)
		}
		if v, ok := config["record_type"].(string); ok {
			data.RecordType = types.StringValue(v)
		}
		if v, ok := config["expected"].(string); ok {
			data.ExpectedResult = types.StringValue(v)
		}

	case "ping":
		if v, ok := config["host"].(string); ok {
			data.Host = types.StringValue(v)
		}
		if v, ok := configInt(config, "count"); ok {
			data.PingCount = types.Int64Value(int64(v))
		}
	}
}

// configInt extracts an integer from a config map, handling JSON's float64 encoding.
func configInt(config map[string]any, key string) (int, bool) {
	v, ok := config[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}
