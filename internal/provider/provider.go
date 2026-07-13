// Package provider implements the hrobot Terraform/OpenTofu provider.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var _ provider.Provider = (*hrobotProvider)(nil)

type hrobotProvider struct {
	version string
}

// New returns a constructor for the provider, suitable for providerserver.Serve.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &hrobotProvider{version: version}
	}
}

type providerModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	BaseURL  types.String `tfsdk:"base_url"`
}

func (p *hrobotProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hrobot"
	resp.Version = p.version
}

func (p *hrobotProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provider for the Hetzner Robot API (dedicated servers).",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Description: "Hetzner Robot webservice username. Falls back to HROBOT_USERNAME.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Hetzner Robot webservice password. Falls back to HROBOT_PASSWORD.",
				Optional:    true,
				Sensitive:   true,
			},
			"base_url": schema.StringAttribute{
				Description: "Override API base URL. Falls back to HROBOT_BASE_URL, then the library default.",
				Optional:    true,
			},
		},
	}
}

func (p *hrobotProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := stringOrEnv(cfg.Username, "HROBOT_USERNAME")
	password := stringOrEnv(cfg.Password, "HROBOT_PASSWORD")
	baseURL := stringOrEnv(cfg.BaseURL, "HROBOT_BASE_URL")

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"missing hrobot username",
			"Set the provider `username` attribute or the HROBOT_USERNAME environment variable.",
		)
	}
	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"missing hrobot password",
			"Set the provider `password` attribute or the HROBOT_PASSWORD environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	opts := []hrobot.ClientOption{
		hrobot.WithApplication("terraform-provider-hrobot", p.version),
	}
	if baseURL != "" {
		opts = append(opts, hrobot.WithBaseURL(baseURL))
	}

	client := hrobot.NewClient(username, password, opts...)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *hrobotProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewServerDataSource,
		NewServersDataSource,
		NewSSHKeyDataSource,
		NewSSHKeysDataSource,
		NewRDNSDataSource,
		NewFailoverDataSource,
		NewFailoversDataSource,
		NewVSwitchDataSource,
		NewVSwitchesDataSource,
		NewIPDataSource,
		NewIPsDataSource,
		NewSubnetDataSource,
		NewSubnetsDataSource,
		NewStorageBoxDataSource,
		NewStorageBoxesDataSource,
		NewStorageBoxSubAccountsDataSource,
		NewStorageBoxSnapshotsDataSource,
		NewBootDataSource,
		NewTrafficDataSource,
	}
}

func (p *hrobotProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewFirewallResource,
		NewSSHKeyResource,
		NewRDNSResource,
		NewVSwitchResource,
		NewFailoverIPResource,
		NewStorageBoxSubAccountResource,
		NewStorageBoxSnapshotResource,
		NewStorageBoxSnapshotPlanResource,
	}
}

func stringOrEnv(v types.String, env string) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	return os.Getenv(env)
}
