package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &ToweropsProvider{}

// ToweropsProvider defines the provider implementation.
type ToweropsProvider struct {
	version string
}

// ToweropsProviderModel describes the provider data model.
type ToweropsProviderModel struct {
	Token types.String `tfsdk:"token"`
}

// New creates a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ToweropsProvider{
			version: version,
		}
	}
}

func (p *ToweropsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "towerops"
	resp.Version = p.version
}

func (p *ToweropsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The TowerOps provider allows you to manage TowerOps resources such as sites and devices.",
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Description: "The API token for authenticating with TowerOps. This token determines which organization's resources are accessible.",
				Required:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *ToweropsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ToweropsProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Token.IsUnknown() {
		resp.Diagnostics.AddError(
			"Unknown TowerOps API Token",
			"The provider cannot create the TowerOps API client as there is an unknown configuration value for the API token.",
		)
		return
	}

	if config.Token.IsNull() || config.Token.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing TowerOps API Token",
			"The provider requires a token to authenticate with the TowerOps API.",
		)
		return
	}

	client := NewClient(config.Token.ValueString())

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ToweropsProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSiteResource,
		NewDeviceResource,
	}
}

func (p *ToweropsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
