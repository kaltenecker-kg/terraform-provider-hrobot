package provider

import (
	"context"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ datasource.DataSource              = (*ipDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*ipDataSource)(nil)
)

// NewIPDataSource returns the hrobot_ip data source.
func NewIPDataSource() datasource.DataSource {
	return &ipDataSource{}
}

type ipDataSource struct {
	client *hrobot.Client
}

type ipModel struct {
	IP              types.String `tfsdk:"ip"`
	Gateway         types.String `tfsdk:"gateway"`
	Mask            types.Int64  `tfsdk:"mask"`
	Broadcast       types.String `tfsdk:"broadcast"`
	ServerIP        types.String `tfsdk:"server_ip"`
	ServerNumber    types.Int64  `tfsdk:"server_number"`
	Locked          types.Bool   `tfsdk:"locked"`
	SeparateMAC     types.String `tfsdk:"separate_mac"`
	TrafficWarnings types.Bool   `tfsdk:"traffic_warnings"`
	TrafficHourly   types.Int64  `tfsdk:"traffic_hourly"`
	TrafficDaily    types.Int64  `tfsdk:"traffic_daily"`
	TrafficMonthly  types.Int64  `tfsdk:"traffic_monthly"`
}

func ipAttributes(withIPInput bool) map[string]schema.Attribute {
	ip := schema.StringAttribute{Computed: true, Description: "The IP address."}
	if withIPInput {
		ip = schema.StringAttribute{Required: true, Description: "The IP address to look up."}
	}
	return map[string]schema.Attribute{
		"ip":               ip,
		"gateway":          schema.StringAttribute{Computed: true, Description: "Gateway address."},
		"mask":             schema.Int64Attribute{Computed: true, Description: "Netmask (CIDR bits)."},
		"broadcast":        schema.StringAttribute{Computed: true, Description: "Broadcast address."},
		"server_ip":        schema.StringAttribute{Computed: true, Description: "Main IP of the owning server."},
		"server_number":    schema.Int64Attribute{Computed: true, Description: "Number of the owning server."},
		"locked":           schema.BoolAttribute{Computed: true, Description: "Whether the IP is locked."},
		"separate_mac":     schema.StringAttribute{Computed: true, Description: "Separate MAC address, if one is set."},
		"traffic_warnings": schema.BoolAttribute{Computed: true, Description: "Whether traffic warnings are enabled."},
		"traffic_hourly":   schema.Int64Attribute{Computed: true, Description: "Hourly traffic warning threshold (MB)."},
		"traffic_daily":    schema.Int64Attribute{Computed: true, Description: "Daily traffic warning threshold (MB)."},
		"traffic_monthly":  schema.Int64Attribute{Computed: true, Description: "Monthly traffic warning threshold (GB)."},
	}
}

func (d *ipDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a single IP address managed in Hetzner Robot.",
		Attributes:  ipAttributes(true),
	}
}

func (d *ipDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip"
}

func (d *ipDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *ipDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ipModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	parsed := net.ParseIP(data.IP.ValueString())
	if parsed == nil {
		resp.Diagnostics.AddError("invalid ip", "the `ip` attribute is not a valid IP address")
		return
	}
	a, err := d.client.IP.Get(ctx, parsed)
	if err != nil {
		resp.Diagnostics.AddError("hrobot ip lookup failed", err.Error())
		return
	}
	setIPModel(&data, a)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setIPModel(m *ipModel, a *hrobot.IPAddress) {
	m.IP = ipToString(a.IP)
	m.Gateway = ipToString(a.Gateway)
	m.Mask = types.Int64Value(int64(a.Mask))
	m.Broadcast = ipToString(a.Broadcast)
	m.ServerIP = ipToString(a.ServerIP)
	m.ServerNumber = types.Int64Value(int64(a.ServerNumber))
	m.Locked = types.BoolValue(a.Locked)
	m.SeparateMAC = optString(a.SeparateMac)
	m.TrafficWarnings = types.BoolValue(a.TrafficWarnings)
	m.TrafficHourly = types.Int64Value(int64(a.TrafficHourly))
	m.TrafficDaily = types.Int64Value(int64(a.TrafficDaily))
	m.TrafficMonthly = types.Int64Value(int64(a.TrafficMonthly))
}
