---
page_title: "towerops_coverage Resource - TowerOps"
description: |-
  Manages a TowerOps RF coverage prediction.
---

# towerops_coverage (Resource)

Manages a TowerOps RF coverage prediction. A coverage describes one transmit antenna at a site (pattern, height, azimuth, frequency, power) plus the prediction extent and the receiver assumptions, and the TowerOps coverage worker renders it into a signal heatmap.

## Example Usage

### Basic

```terraform
resource "towerops_site" "tower" {
  name      = "Verona Tower"
  latitude  = 33.4356
  longitude = -96.0028
}

resource "towerops_coverage" "north_sector" {
  name          = "North sector 5 GHz"
  site_id       = towerops_site.tower.id
  antenna_slug  = "rf-elements-tp-sh-30"
  frequency_mhz = 5800
  tx_power_dbm  = 22.0
  height_agl_m  = 30.0
  azimuth_deg   = 0
  radius_m      = 6437
}
```

### Tuned sector

```terraform
resource "towerops_coverage" "south_sector" {
  name          = "South sector 3 GHz"
  site_id       = towerops_site.tower.id
  device_id     = towerops_device.south_ap.id
  antenna_slug  = "rf-elements-tp-sh-30"
  frequency_mhz = 3650
  tx_power_dbm  = 27.0
  height_agl_m  = 45.0
  azimuth_deg   = 180
  downtilt_deg  = 3.0
  radius_m      = 12000
  cell_size_m   = 15

  cable_loss_db     = 1.5
  sm_gain_dbi       = 19.0
  receiver_height_m = 6.0
  rx_threshold_dbm  = -85.0
  foliage_tuning    = 25
}
```

## Asynchronous compute

Creating a coverage enqueues a compute job and returns immediately, so `status` is `queued` right after `terraform apply`. The worker moves it through `computing` to `ready` or `failed`. The statuses are `draft`, `queued`, `computing`, `ready` and `failed`.

The provider never blocks and never polls. It records whatever `status` and `progress_pct` the API returned, and a later `terraform refresh` (or any subsequent plan) observes the terminal status along with `computed_at`, `png_path`, `raster_path` and `bbox`. If you must gate other work on `status = "ready"`, do the waiting outside the resource, for example with a `terraform_data` resource and a provisioner.

`PATCH /api/v1/coverages/:id` deliberately does not recompute the heatmap, so an in-place update changes the parameters and leaves the previous raster and status alone. The API also exposes `POST /api/v1/coverages/:id/recompute`, but re-running the pipeline is an imperative action, so this resource never calls it. Trigger a recompute out of band when you want fresh outputs.

## SI units only

The API accepts imperial virtual inputs (`height_agl_ft`, `radius_mi`, `frequency_ghz`, `receiver_height_ft`, `tx_clearance_ft`, `height_above_rooftop_ft`) for the web form, and its changeset converts them to SI before persisting. A configuration written in imperial units would therefore read back in metric and every plan would be permanently dirty, so this resource exposes the SI fields only. Convert in HCL if your source data is imperial, for example `radius_m = 4 * 1609` or `height_agl_m = 100 * 0.3048`.

## Schema

### Required

- `name` (String) - The name of the coverage. Must be between 2 and 100 characters, and unique within the site.
- `site_id` (String) - The identifier of the site the transmit antenna is mounted at.
- `antenna_slug` (String) - The slug of the antenna pattern to model, for example `rf-elements-tp-sh-30`. Must match a known antenna.
- `frequency_mhz` (Number) - The transmit centre frequency in MHz. Must be between 700 and 90000.
- `tx_power_dbm` (Float) - The transmit power in dBm. Must be between -10.0 and 50.0.
- `height_agl_m` (Float) - The antenna height above ground level in metres. Must be between 1.0 and 200.0.
- `azimuth_deg` (Float) - The antenna bearing in degrees true. Must be between 0.0 and 360.0.
- `radius_m` (Number) - The prediction radius in metres. Must be between 500 and 40000.

### Optional

- `device_id` (String) - Optional identifier of the device this coverage models.
- `cell_size_m` (Number) - The raster cell size in metres. Must be between 1 and 50. Omit it and the server derives a cell size from the radius that fits the pixel budget.
- `downtilt_deg` (Float) - The mechanical downtilt in degrees. Must be between -10.0 and 30.0. Defaults to `0.0`.
- `cable_loss_db` (Float) - The feedline loss in dB. Must be between 0.0 and 20.0. Defaults to `0.0`.
- `sm_gain_dbi` (Float) - The subscriber module antenna gain in dBi. Must be between 0.0 and 40.0. Defaults to `0.0`.
- `height_above_rooftop_m` (Float) - The antenna height above the rooftop in metres. Must be between 0.0 and 100.0. Defaults to `0.0`.
- `tx_clearance_m` (Float) - The required Fresnel clearance at the transmitter in metres. Must be between 0.0 and 1000.0. The API accepts this value but does not return it, so the configured value is kept in state exactly as written.
- `receiver_height_m` (Float) - The assumed receiver height above ground in metres. Must be between 0.5 and 100.0. Defaults to `3.0`.
- `rx_threshold_dbm` (Float) - The receive threshold in dBm below which a cell counts as no coverage. Must be between -130.0 and 0.0. Defaults to `-90.0`.
- `foliage_tuning` (Number) - The foliage attenuation tuning factor. Must be between 0 and 100. Defaults to `0`.
- `latitude_override` (Float) - Overrides the transmit latitude instead of using the parent site. Must be between -90.0 and 90.0.
- `longitude_override` (Float) - Overrides the transmit longitude instead of using the parent site. Must be between -180.0 and 180.0.

### Read-Only

- `id` (String) - The unique identifier of the coverage.
- `organization_id` (String) - The organization that owns the coverage. Derived from the API token.
- `status` (String) - The compute status: `draft`, `queued`, `computing`, `ready` or `failed`.
- `progress_pct` (Number) - The compute progress in percent, 0 to 100.
- `error_message` (String) - The failure reason when `status` is `failed`.
- `computed_at` (String) - The timestamp of the last successful compute.
- `png_path` (String) - The relative path of the rendered heatmap PNG.
- `raster_path` (String) - The relative path of the raw signal raster.
- `bbox` (Attributes) - The geographic extent of the computed heatmap. Null until the compute succeeds.
  - `min_lat` (Float) - The southern edge of the heatmap.
  - `max_lat` (Float) - The northern edge of the heatmap.
  - `min_lon` (Float) - The western edge of the heatmap.
  - `max_lon` (Float) - The eastern edge of the heatmap.
- `inserted_at` (String) - The timestamp when the coverage was created.
- `updated_at` (String) - The timestamp when the coverage was last updated.

## Import

Coverages can be imported using their UUID:

```shell
terraform import towerops_coverage.north_sector 550e8400-e29b-41d4-a716-446655440000
```
