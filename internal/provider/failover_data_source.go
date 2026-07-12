package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ datasource.DataSource              = (*failoverDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*failoverDataSource)(nil)
)

// NewFailoverDataSource returns the hrobot_failover data source.
func NewFailoverDataSource() datasource.DataSource {
	return &failoverDataSource{}
}

type failoverDataSource struct {
	client *hrobot.Client
}

type failoverModel struct {
	IP             types.String `tfsdk:"ip"`
	Netmask        types.String `tfsdk:"netmask"`
	ServerIP       types.String `tfsdk:"server_ip"`
	ServerIPv6Net  types.String `tfsdk:"server_ipv6_net"`
	ServerNumber   types.Int64  `tfsdk:"server_number"`
	ActiveServerIP types.String `tfsdk:"active_server_ip"`
}

func failoverAttributes(withIPInput bool) map[string]schema.Attribute {
	ip := schema.StringAttribute{Computed: true, Description: "The failover IP address."}
	if withIPInput {
		ip = schema.StringAttribute{Required: true, Description: "The failover IP address to look up."}
	}
	return map[string]schema.Attribute{
		"ip":               ip,
		"netmask":          schema.StringAttribute{Computed: true, Description: "Netmask of the failover IP."},
		"server_ip":        schema.StringAttribute{Computed: true, Description: "Main IP of the server the failover IP belongs to."},
		"server_ipv6_net":  schema.StringAttribute{Computed: true, Description: "IPv6 network of the server the failover IP belongs to."},
		"server_number":    schema.Int64Attribute{Computed: true, Description: "Number of the server the failover IP belongs to."},
		"active_server_ip": schema.StringAttribute{Computed: true, Description: "Main IP of the server the failover IP is currently routed to."},
	}
}

func (d *failoverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Hetzner Robot failover IP and its current routing target.",
		Attributes:  failoverAttributes(true),
	}
}

func (d *failoverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_failover"
}

func (d *failoverDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *failoverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data failoverModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fo, err := d.client.Failover.Get(ctx, data.IP.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("hrobot failover lookup failed", err.Error())
		return
	}
	setFailoverModel(&data, fo)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setFailoverModel(m *failoverModel, f *hrobot.Failover) {
	m.IP = types.StringValue(f.IP)
	m.Netmask = types.StringValue(f.Netmask)
	m.ServerIP = types.StringValue(f.ServerIP)
	m.ServerIPv6Net = optString(f.ServerIPv6Net)
	m.ServerNumber = types.Int64Value(int64(f.ServerNumber))
	if f.ActiveServerIP != nil {
		m.ActiveServerIP = types.StringValue(*f.ActiveServerIP)
	} else {
		m.ActiveServerIP = types.StringNull()
	}
}
