package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ datasource.DataSource              = (*rdnsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*rdnsDataSource)(nil)
)

// NewRDNSDataSource returns the hrobot_rdns data source.
func NewRDNSDataSource() datasource.DataSource {
	return &rdnsDataSource{}
}

type rdnsDataSource struct {
	client *hrobot.Client
}

type rdnsModel struct {
	IP  types.String `tfsdk:"ip"`
	PTR types.String `tfsdk:"ptr"`
}

func (d *rdnsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rdns"
}

func (d *rdnsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up the reverse DNS (PTR) entry for an IP address.",
		Attributes: map[string]schema.Attribute{
			"ip":  schema.StringAttribute{Required: true, Description: "The IP address to look up."},
			"ptr": schema.StringAttribute{Computed: true, Description: "The PTR record (reverse DNS name)."},
		},
	}
}

func (d *rdnsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *rdnsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data rdnsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	entry, err := d.client.RDNS.Get(ctx, data.IP.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("hrobot rdns lookup failed", err.Error())
		return
	}
	data.IP = types.StringValue(entry.IP)
	data.PTR = types.StringValue(entry.PTR)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
