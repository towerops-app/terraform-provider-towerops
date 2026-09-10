package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &CoverageResource{}
var _ resource.ResourceWithImportState = &CoverageResource{}

// CoverageResource defines the resource implementation.
type CoverageResource struct {
	client *Client
}

// CoverageResourceModel describes the resource data model.
//
// Only SI inputs are exposed. The API also accepts imperial virtual fields
// (height_agl_ft, radius_mi, frequency_ghz, receiver_height_ft,
// tx_clearance_ft, height_above_rooftop_ft) but the changeset converts them
// to SI before persisting, so a config written in imperial units would read
// back in metric and every plan would be permanently dirty.
type CoverageResourceModel struct {
	ID                  types.String  `tfsdk:"id"`
	Name                types.String  `tfsdk:"name"`
	OrganizationID      types.String  `tfsdk:"organization_id"`
	SiteID              types.String  `tfsdk:"site_id"`
	DeviceID            types.String  `tfsdk:"device_id"`
	AntennaSlug         types.String  `tfsdk:"antenna_slug"`
	FrequencyMHz        types.Int64   `tfsdk:"frequency_mhz"`
	TxPowerDBm          types.Float64 `tfsdk:"tx_power_dbm"`
	CableLossDB         types.Float64 `tfsdk:"cable_loss_db"`
	SmGainDBi           types.Float64 `tfsdk:"sm_gain_dbi"`
	HeightAGLM          types.Float64 `tfsdk:"height_agl_m"`
	HeightAboveRooftopM types.Float64 `tfsdk:"height_above_rooftop_m"`
	TxClearanceM        types.Float64 `tfsdk:"tx_clearance_m"`
	AzimuthDeg          types.Float64 `tfsdk:"azimuth_deg"`
	DowntiltDeg         types.Float64 `tfsdk:"downtilt_deg"`
	RadiusM             types.Int64   `tfsdk:"radius_m"`
	CellSizeM           types.Int64   `tfsdk:"cell_size_m"`
	ReceiverHeightM     types.Float64 `tfsdk:"receiver_height_m"`
	RxThresholdDBm      types.Float64 `tfsdk:"rx_threshold_dbm"`
	FoliageTuning       types.Int64   `tfsdk:"foliage_tuning"`
	LatitudeOverride    types.Float64 `tfsdk:"latitude_override"`
	LongitudeOverride   types.Float64 `tfsdk:"longitude_override"`

	Status       types.String `tfsdk:"status"`
	ProgressPct  types.Int64  `tfsdk:"progress_pct"`
	ErrorMessage types.String `tfsdk:"error_message"`
	ComputedAt   types.String `tfsdk:"computed_at"`
	PNGPath      types.String `tfsdk:"png_path"`
	RasterPath   types.String `tfsdk:"raster_path"`
	BBox         types.Object `tfsdk:"bbox"`
	InsertedAt   types.String `tfsdk:"inserted_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

// coverageBBoxAttrTypes is the object type of the computed bbox attribute.
var coverageBBoxAttrTypes = map[string]attr.Type{
	"min_lat": types.Float64Type,
	"max_lat": types.Float64Type,
	"min_lon": types.Float64Type,
	"max_lon": types.Float64Type,
}

// NewCoverageResource creates a new coverage resource.
func NewCoverageResource() resource.Resource {
	return &CoverageResource{}
}

func (r *CoverageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_coverage"
}

func (r *CoverageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TowerOps RF coverage prediction. Creating a coverage enqueues an asynchronous " +
			"compute job, so a fresh coverage starts with status \"queued\" and reaches \"ready\" or \"failed\" " +
			"later. The provider never blocks or polls: run \"terraform refresh\" (or any later plan) to observe " +
			"the terminal status. Inputs are SI only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the coverage.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the coverage. Must be between 2 and 100 characters, and unique within the site.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization that owns the coverage. Derived from the API token, never sent.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": schema.StringAttribute{
				Description: "The identifier of the site the transmit antenna is mounted at.",
				Required:    true,
			},
			"device_id": schema.StringAttribute{
				Description: "Optional identifier of the device this coverage models.",
				Optional:    true,
			},
			"antenna_slug": schema.StringAttribute{
				Description: "The slug of the antenna pattern to model, for example \"rf-elements-tp-sh-30\". Must match a known antenna.",
				Required:    true,
			},
			"frequency_mhz": schema.Int64Attribute{
				Description: "The transmit centre frequency in MHz. Must be between 700 and 90000.",
				Required:    true,
			},
			"tx_power_dbm": schema.Float64Attribute{
				Description: "The transmit power in dBm. Must be between -10.0 and 50.0.",
				Required:    true,
			},
			"height_agl_m": schema.Float64Attribute{
				Description: "The antenna height above ground level in metres. Must be between 1.0 and 200.0.",
				Required:    true,
			},
			"azimuth_deg": schema.Float64Attribute{
				Description: "The antenna bearing in degrees true. Must be between 0.0 and 360.0.",
				Required:    true,
			},
			"radius_m": schema.Int64Attribute{
				Description: "The prediction radius in metres. Must be between 500 and 40000.",
				Required:    true,
			},
			"cell_size_m": schema.Int64Attribute{
				Description: "The raster cell size in metres. Must be between 1 and 50. Omit it and the server " +
					"derives a cell size from the radius that fits the pixel budget.",
				Optional: true,
				Computed: true,
			},
			"downtilt_deg": schema.Float64Attribute{
				Description: "The mechanical downtilt in degrees. Must be between -10.0 and 30.0. Defaults to 0.0.",
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(0.0),
			},
			"cable_loss_db": schema.Float64Attribute{
				Description: "The feedline loss in dB. Must be between 0.0 and 20.0. Defaults to 0.0.",
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(0.0),
			},
			"sm_gain_dbi": schema.Float64Attribute{
				Description: "The subscriber module antenna gain in dBi. Must be between 0.0 and 40.0. Defaults to 0.0.",
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(0.0),
			},
			"height_above_rooftop_m": schema.Float64Attribute{
				Description: "The antenna height above the rooftop in metres. Must be between 0.0 and 100.0. Defaults to 0.0.",
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(0.0),
			},
			"tx_clearance_m": schema.Float64Attribute{
				Description: "The required Fresnel clearance at the transmitter in metres. Must be between 0.0 " +
					"and 1000.0. The API accepts this value but does not return it, so the configured value is " +
					"kept in state as written.",
				Optional: true,
			},
			"receiver_height_m": schema.Float64Attribute{
				Description: "The assumed receiver height above ground in metres. Must be between 0.5 and 100.0. Defaults to 3.0.",
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(3.0),
			},
			"rx_threshold_dbm": schema.Float64Attribute{
				Description: "The receive threshold in dBm below which a cell counts as no coverage. Must be " +
					"between -130.0 and 0.0. Defaults to -90.0.",
				Optional: true,
				Computed: true,
				Default:  float64default.StaticFloat64(-90.0),
			},
			"foliage_tuning": schema.Int64Attribute{
				Description: "The foliage attenuation tuning factor. Must be between 0 and 100. Defaults to 0.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"latitude_override": schema.Float64Attribute{
				Description: "Overrides the transmit latitude instead of using the parent site. Must be between -90.0 and 90.0.",
				Optional:    true,
			},
			"longitude_override": schema.Float64Attribute{
				Description: "Overrides the transmit longitude instead of using the parent site. Must be between -180.0 and 180.0.",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "The compute status: draft, queued, computing, ready or failed. A newly created " +
					"coverage is queued; refresh later to observe the terminal status.",
				Computed: true,
			},
			"progress_pct": schema.Int64Attribute{
				Description: "The compute progress in percent, 0 to 100.",
				Computed:    true,
			},
			"error_message": schema.StringAttribute{
				Description: "The failure reason when status is failed.",
				Computed:    true,
			},
			"computed_at": schema.StringAttribute{
				Description: "The timestamp of the last successful compute.",
				Computed:    true,
			},
			"png_path": schema.StringAttribute{
				Description: "The relative path of the rendered heatmap PNG.",
				Computed:    true,
			},
			"raster_path": schema.StringAttribute{
				Description: "The relative path of the raw signal raster.",
				Computed:    true,
			},
			"bbox": schema.SingleNestedAttribute{
				Description: "The geographic extent of the computed heatmap. Null until the compute succeeds.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"min_lat": schema.Float64Attribute{
						Description: "The southern edge of the heatmap.",
						Computed:    true,
					},
					"max_lat": schema.Float64Attribute{
						Description: "The northern edge of the heatmap.",
						Computed:    true,
					},
					"min_lon": schema.Float64Attribute{
						Description: "The western edge of the heatmap.",
						Computed:    true,
					},
					"max_lon": schema.Float64Attribute{
						Description: "The eastern edge of the heatmap.",
						Computed:    true,
					},
				},
			},
			"inserted_at": schema.StringAttribute{
				Description: "The timestamp when the coverage was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the coverage was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *CoverageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CoverageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CoverageResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	coverage := buildCoverageFromModel(&data)

	// Create enqueues the compute job and returns immediately. Whatever
	// status the response carries (normally "queued") is recorded as is; a
	// later refresh observes "ready" or "failed".
	created, err := r.client.CreateCoverage(coverage)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create coverage", err.Error())
		return
	}

	applyCoveragePlan(&data, created)
	applyCoverageComputed(&data, created)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CoverageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CoverageResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	coverage, err := r.client.GetCoverage(data.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Coverage was deleted outside of Terraform, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read coverage", err.Error())
		return
	}

	applyCoverageRead(&data, coverage)
	applyCoverageComputed(&data, coverage)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CoverageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CoverageResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	coverage := buildCoverageFromModel(&data)

	// PATCH deliberately does not recompute the heatmap. The API exposes
	// POST /api/v1/coverages/:id/recompute for that, but re-running the
	// pipeline is an imperative action, not a declarative one, so this
	// resource never calls it.
	updated, err := r.client.UpdateCoverage(data.ID.ValueString(), coverage)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update coverage", err.Error())
		return
	}

	applyCoveragePlan(&data, updated)
	applyCoverageComputed(&data, updated)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CoverageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CoverageResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCoverage(data.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Failed to delete coverage", err.Error())
		return
	}
}

func (r *CoverageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// buildCoverageFromModel converts the Terraform model to an API Coverage
// struct. Only SI inputs are sent, and only values Terraform actually knows:
// an unknown Optional plus Computed value is left out so the server can
// derive it.
func buildCoverageFromModel(data *CoverageResourceModel) Coverage {
	coverage := Coverage{
		Name:        data.Name.ValueString(),
		SiteID:      data.SiteID.ValueString(),
		AntennaSlug: data.AntennaSlug.ValueString(),
	}

	if v, ok := covKnownString(data.DeviceID); ok {
		coverage.DeviceID = &v
	}
	if v, ok := covKnownInt(data.FrequencyMHz); ok {
		coverage.FrequencyMHz = &v
	}
	if v, ok := covKnownFloat(data.TxPowerDBm); ok {
		coverage.TxPowerDBm = &v
	}
	if v, ok := covKnownFloat(data.CableLossDB); ok {
		coverage.CableLossDB = &v
	}
	if v, ok := covKnownFloat(data.SmGainDBi); ok {
		coverage.SmGainDBi = &v
	}
	if v, ok := covKnownFloat(data.HeightAGLM); ok {
		coverage.HeightAGLM = &v
	}
	if v, ok := covKnownFloat(data.HeightAboveRooftopM); ok {
		coverage.HeightAboveRooftopM = &v
	}
	if v, ok := covKnownFloat(data.TxClearanceM); ok {
		coverage.TxClearanceM = &v
	}
	if v, ok := covKnownFloat(data.AzimuthDeg); ok {
		coverage.AzimuthDeg = &v
	}
	if v, ok := covKnownFloat(data.DowntiltDeg); ok {
		coverage.DowntiltDeg = &v
	}
	if v, ok := covKnownInt(data.RadiusM); ok {
		coverage.RadiusM = &v
	}
	if v, ok := covKnownInt(data.CellSizeM); ok {
		coverage.CellSizeM = &v
	}
	if v, ok := covKnownFloat(data.ReceiverHeightM); ok {
		coverage.ReceiverHeightM = &v
	}
	if v, ok := covKnownFloat(data.RxThresholdDBm); ok {
		coverage.RxThresholdDBm = &v
	}
	if v, ok := covKnownInt(data.FoliageTuning); ok {
		coverage.FoliageTuning = &v
	}
	if v, ok := covKnownFloat(data.LatitudeOverride); ok {
		coverage.LatitudeOverride = &v
	}
	if v, ok := covKnownFloat(data.LongitudeOverride); ok {
		coverage.LongitudeOverride = &v
	}

	return coverage
}

// applyCoveragePlan reconciles the configurable attributes after a create or
// an update. The planned value wins whenever Terraform knows it, because
// writing a different value back would fail the "Provider produced
// inconsistent result after apply" check. Only a value the plan left unknown
// (an Optional plus Computed attribute the server derives, such as
// cell_size_m) is taken from the API response. Drift is reported by Read.
func applyCoveragePlan(data *CoverageResourceModel, c *Coverage) {
	if data.DeviceID.IsUnknown() {
		data.DeviceID = covStringOrNull(c.DeviceID)
	}
	if data.FrequencyMHz.IsUnknown() {
		data.FrequencyMHz = covIntOrNull(c.FrequencyMHz)
	}
	if data.TxPowerDBm.IsUnknown() {
		data.TxPowerDBm = covFloatOrNull(c.TxPowerDBm)
	}
	if data.CableLossDB.IsUnknown() {
		data.CableLossDB = covFloatOrNull(c.CableLossDB)
	}
	if data.SmGainDBi.IsUnknown() {
		data.SmGainDBi = covFloatOrNull(c.SmGainDBi)
	}
	if data.HeightAGLM.IsUnknown() {
		data.HeightAGLM = covFloatOrNull(c.HeightAGLM)
	}
	if data.HeightAboveRooftopM.IsUnknown() {
		data.HeightAboveRooftopM = covFloatOrNull(c.HeightAboveRooftopM)
	}
	if data.TxClearanceM.IsUnknown() {
		data.TxClearanceM = covFloatOrNull(c.TxClearanceM)
	}
	if data.AzimuthDeg.IsUnknown() {
		data.AzimuthDeg = covFloatOrNull(c.AzimuthDeg)
	}
	if data.DowntiltDeg.IsUnknown() {
		data.DowntiltDeg = covFloatOrNull(c.DowntiltDeg)
	}
	if data.RadiusM.IsUnknown() {
		data.RadiusM = covIntOrNull(c.RadiusM)
	}
	if data.CellSizeM.IsUnknown() {
		data.CellSizeM = covIntOrNull(c.CellSizeM)
	}
	if data.ReceiverHeightM.IsUnknown() {
		data.ReceiverHeightM = covFloatOrNull(c.ReceiverHeightM)
	}
	if data.RxThresholdDBm.IsUnknown() {
		data.RxThresholdDBm = covFloatOrNull(c.RxThresholdDBm)
	}
	if data.FoliageTuning.IsUnknown() {
		data.FoliageTuning = covIntOrNull(c.FoliageTuning)
	}
	if data.LatitudeOverride.IsUnknown() {
		data.LatitudeOverride = covFloatOrNull(c.LatitudeOverride)
	}
	if data.LongitudeOverride.IsUnknown() {
		data.LongitudeOverride = covFloatOrNull(c.LongitudeOverride)
	}
	if data.Name.IsUnknown() {
		data.Name = types.StringValue(c.Name)
	}
	if data.SiteID.IsUnknown() {
		data.SiteID = types.StringValue(c.SiteID)
	}
	if data.AntennaSlug.IsUnknown() {
		data.AntennaSlug = types.StringValue(c.AntennaSlug)
	}
}

// applyCoverageRead overwrites the configurable attributes from an API
// response so that a refresh reports real drift.
func applyCoverageRead(data *CoverageResourceModel, c *Coverage) {
	data.Name = types.StringValue(c.Name)
	if c.SiteID != "" {
		data.SiteID = types.StringValue(c.SiteID)
	}
	data.AntennaSlug = types.StringValue(c.AntennaSlug)
	data.DeviceID = covStringOrNull(c.DeviceID)
	data.FrequencyMHz = covIntOrNull(c.FrequencyMHz)
	data.TxPowerDBm = covFloatOrNull(c.TxPowerDBm)
	data.CableLossDB = covFloatOrNull(c.CableLossDB)
	data.SmGainDBi = covFloatOrNull(c.SmGainDBi)
	data.HeightAGLM = covFloatOrNull(c.HeightAGLM)
	data.HeightAboveRooftopM = covFloatOrNull(c.HeightAboveRooftopM)
	data.AzimuthDeg = covFloatOrNull(c.AzimuthDeg)
	data.DowntiltDeg = covFloatOrNull(c.DowntiltDeg)
	data.RadiusM = covIntOrNull(c.RadiusM)
	data.CellSizeM = covIntOrNull(c.CellSizeM)
	data.ReceiverHeightM = covFloatOrNull(c.ReceiverHeightM)
	data.RxThresholdDBm = covFloatOrNull(c.RxThresholdDBm)
	data.FoliageTuning = covIntOrNull(c.FoliageTuning)
	data.LatitudeOverride = covFloatOrNull(c.LatitudeOverride)
	data.LongitudeOverride = covFloatOrNull(c.LongitudeOverride)

	// tx_clearance_m is accepted on write but absent from every response, so
	// the prior value stays untouched. Clearing it here would create a
	// permanent diff against the configuration.
}

// applyCoverageComputed maps the server-owned attributes onto the model.
func applyCoverageComputed(data *CoverageResourceModel, c *Coverage) {
	data.ID = types.StringValue(c.ID)
	data.OrganizationID = types.StringValue(c.OrganizationID)
	data.Status = types.StringValue(c.Status)
	data.ProgressPct = covIntOrNull(c.ProgressPct)
	data.ErrorMessage = covStringOrNull(c.ErrorMessage)
	data.ComputedAt = covStringOrNull(c.ComputedAt)
	data.PNGPath = covStringOrNull(c.PNGPath)
	data.RasterPath = covStringOrNull(c.RasterPath)
	data.InsertedAt = types.StringValue(c.InsertedAt)
	data.UpdatedAt = types.StringValue(c.UpdatedAt)
	data.BBox = coverageBBoxObject(c.BBox)
}

// coverageBBoxObject converts the API bbox into its object value. The API
// always emits the key but leaves the members null until a compute succeeds,
// so an all-null bbox collapses to a null object.
func coverageBBoxObject(bbox *CoverageBBox) types.Object {
	if bbox == nil {
		return types.ObjectNull(coverageBBoxAttrTypes)
	}
	if bbox.MinLat == nil && bbox.MaxLat == nil && bbox.MinLon == nil && bbox.MaxLon == nil {
		return types.ObjectNull(coverageBBoxAttrTypes)
	}

	return types.ObjectValueMust(coverageBBoxAttrTypes, map[string]attr.Value{
		"min_lat": covFloatOrNull(bbox.MinLat),
		"max_lat": covFloatOrNull(bbox.MaxLat),
		"min_lon": covFloatOrNull(bbox.MinLon),
		"max_lon": covFloatOrNull(bbox.MaxLon),
	})
}

func covKnownString(v types.String) (string, bool) {
	if v.IsNull() || v.IsUnknown() {
		return "", false
	}
	return v.ValueString(), true
}

func covKnownFloat(v types.Float64) (float64, bool) {
	if v.IsNull() || v.IsUnknown() {
		return 0, false
	}
	return v.ValueFloat64(), true
}

func covKnownInt(v types.Int64) (int, bool) {
	if v.IsNull() || v.IsUnknown() {
		return 0, false
	}
	return int(v.ValueInt64()), true
}

func covStringOrNull(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

func covFloatOrNull(v *float64) types.Float64 {
	if v == nil {
		return types.Float64Null()
	}
	return types.Float64Value(*v)
}

func covIntOrNull(v *int) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}
