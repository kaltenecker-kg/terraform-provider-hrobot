package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ datasource.DataSource              = (*subnetDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*subnetDataSource)(nil)
)

// NewSubnetDataSource returns the hrobot_subnet data source.
func NewSubnetDataSource() datasource.DataSource {
	return &subnetDataSource{}
}

type subnetDataSource struct {
	client *hrobot.Client
}

type subnetModel struct {
	IP              types.String `tfsdk:"ip"`
	Mask            types.Int64  `tfsdk:"mask"`
	Gateway         types.String `tfsdk:"gateway"`
	ServerIP        types.String `tfsdk:"server_ip"`
	ServerNumber    types.Int64  `tfsdk:"server_number"`
	Failover        types.Bool   `tfsdk:"failover"`
	Locked          types.Bool   `tfsdk:"locked"`
	TrafficWarnings types.Bool   `tfsdk:"traffic_warnings"`
	TrafficHourly   types.Int64  `tfsdk:"traffic_hourly"`
	TrafficDaily    types.Int64  `tfsdk:"traffic_daily"`
	TrafficMonthly  types.Int64  `tfsdk:"traffic_monthly"`
}

func subnetAttributes(withIPInput bool) map[string]schema.Attribute {
	ip := schema.StringAttribute{Computed: true, Description: "Subnet network address."}
	if withIPInput {
		ip = schema.StringAttribute{Required: true, Description: "Subnet network address to look up."}
	}
	return map[string]schema.Attribute{
		"ip":               ip,
		"mask":             schema.Int64Attribute{Computed: true, Description: "Subnet mask (CIDR bits)."},
		"gateway":          schema.StringAttribute{Computed: true, Description: "Subnet gateway."},
		"server_ip":        schema.StringAttribute{Computed: true, Description: "Main IP of the owning server."},
		"server_number":    schema.Int64Attribute{Computed: true, Description: "Number of the owning server."},
		"failover":         schema.BoolAttribute{Computed: true, Description: "Whether this is a failover subnet."},
		"locked":           schema.BoolAttribute{Computed: true, Description: "Whether the subnet is locked."},
		"traffic_warnings": schema.BoolAttribute{Computed: true, Description: "Whether traffic warnings are enabled."},
		"traffic_hourly":   schema.Int64Attribute{Computed: true, Description: "Hourly traffic warning threshold (MB)."},
		"traffic_daily":    schema.Int64Attribute{Computed: true, Description: "Daily traffic warning threshold (MB)."},
		"traffic_monthly":  schema.Int64Attribute{Computed: true, Description: "Monthly traffic warning threshold (GB)."},
	}
}

func (d *subnetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a single subnet managed in Hetzner Robot.",
		Attributes:  subnetAttributes(true),
	}
}

func (d *subnetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnet"
}

func (d *subnetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *subnetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data subnetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	s, err := d.client.Subnet.Get(ctx, data.IP.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("hrobot subnet lookup failed", err.Error())
		return
	}
	setSubnetModel(&data, s)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setSubnetModel(m *subnetModel, s *hrobot.SubnetResource) {
	m.IP = types.StringValue(s.IP)
	m.Mask = types.Int64Value(int64(s.Mask))
	m.Gateway = optString(s.Gateway)
	m.ServerIP = optString(s.ServerIP)
	m.ServerNumber = types.Int64Value(int64(s.ServerNumber))
	m.Failover = types.BoolValue(s.Failover)
	m.Locked = types.BoolValue(s.Locked)
	m.TrafficWarnings = types.BoolValue(s.TrafficWarnings)
	m.TrafficHourly = types.Int64Value(int64(s.TrafficHourly))
	m.TrafficDaily = types.Int64Value(int64(s.TrafficDaily))
	m.TrafficMonthly = types.Int64Value(int64(s.TrafficMonthly))
}
