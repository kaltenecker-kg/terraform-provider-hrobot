package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ datasource.DataSource              = (*vswitchDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*vswitchDataSource)(nil)
)

// NewVSwitchDataSource returns the hrobot_vswitch data source.
func NewVSwitchDataSource() datasource.DataSource {
	return &vswitchDataSource{}
}

type vswitchDataSource struct {
	client *hrobot.Client
}

type vswitchModel struct {
	ID            types.Int64          `tfsdk:"id"`
	Name          types.String         `tfsdk:"name"`
	VLAN          types.Int64          `tfsdk:"vlan"`
	Cancelled     types.Bool           `tfsdk:"cancelled"`
	Servers       []vswitchServerModel `tfsdk:"servers"`
	Subnets       []vswitchSubnetModel `tfsdk:"subnets"`
	CloudNetworks []cloudNetworkModel  `tfsdk:"cloud_networks"`
}

type vswitchServerModel struct {
	ServerIP      types.String `tfsdk:"server_ip"`
	ServerIPv6Net types.String `tfsdk:"server_ipv6_net"`
	ServerNumber  types.Int64  `tfsdk:"server_number"`
	Status        types.String `tfsdk:"status"`
}

type vswitchSubnetModel struct {
	IP      types.String `tfsdk:"ip"`
	Mask    types.Int64  `tfsdk:"mask"`
	Gateway types.String `tfsdk:"gateway"`
}

type cloudNetworkModel struct {
	ID      types.Int64  `tfsdk:"id"`
	IP      types.String `tfsdk:"ip"`
	Mask    types.Int64  `tfsdk:"mask"`
	Gateway types.String `tfsdk:"gateway"`
}

func (d *vswitchDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vswitch"
}

func (d *vswitchDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Hetzner Robot vSwitch by its ID, including attached servers and subnets.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.Int64Attribute{Required: true, Description: "vSwitch ID."},
			"name":      schema.StringAttribute{Computed: true, Description: "vSwitch name."},
			"vlan":      schema.Int64Attribute{Computed: true, Description: "VLAN ID (4000-4091)."},
			"cancelled": schema.BoolAttribute{Computed: true, Description: "Whether the vSwitch is cancelled."},
			"servers": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Servers attached to the vSwitch.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"server_ip":       schema.StringAttribute{Computed: true, Description: "Main IP of the server."},
					"server_ipv6_net": schema.StringAttribute{Computed: true, Description: "IPv6 network of the server."},
					"server_number":   schema.Int64Attribute{Computed: true, Description: "Server number."},
					"status":          schema.StringAttribute{Computed: true, Description: "Connection status (ready, in process, failed)."},
				}},
			},
			"subnets": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Subnets routed over the vSwitch.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"ip":      schema.StringAttribute{Computed: true, Description: "Subnet network address."},
					"mask":    schema.Int64Attribute{Computed: true, Description: "Subnet mask (CIDR bits)."},
					"gateway": schema.StringAttribute{Computed: true, Description: "Subnet gateway."},
				}},
			},
			"cloud_networks": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Attached Hetzner Cloud networks.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":      schema.Int64Attribute{Computed: true, Description: "Cloud network ID."},
					"ip":      schema.StringAttribute{Computed: true, Description: "Cloud network address."},
					"mask":    schema.Int64Attribute{Computed: true, Description: "Cloud network mask (CIDR bits)."},
					"gateway": schema.StringAttribute{Computed: true, Description: "Cloud network gateway."},
				}},
			},
		},
	}
}

func (d *vswitchDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *vswitchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data vswitchModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	vs, err := d.client.VSwitch.Get(ctx, int(data.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("hrobot vswitch lookup failed", err.Error())
		return
	}
	setVSwitchModel(&data, vs)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setVSwitchModel(m *vswitchModel, vs *hrobot.VSwitch) {
	m.ID = types.Int64Value(int64(vs.ID))
	m.Name = types.StringValue(vs.Name)
	m.VLAN = types.Int64Value(int64(vs.VLAN))
	m.Cancelled = types.BoolValue(vs.Cancelled)

	m.Servers = make([]vswitchServerModel, len(vs.Servers))
	for i, s := range vs.Servers {
		m.Servers[i] = vswitchServerModel{
			ServerIP:      optString(s.ServerIP),
			ServerIPv6Net: optString(s.ServerIPv6Net),
			ServerNumber:  types.Int64Value(int64(s.ServerNumber)),
			Status:        optString(s.Status),
		}
	}
	m.Subnets = make([]vswitchSubnetModel, len(vs.Subnets))
	for i, s := range vs.Subnets {
		m.Subnets[i] = vswitchSubnetModel{
			IP:      types.StringValue(s.IP),
			Mask:    types.Int64Value(int64(s.Mask)),
			Gateway: optString(s.Gateway),
		}
	}
	m.CloudNetworks = make([]cloudNetworkModel, len(vs.CloudNetwork))
	for i, c := range vs.CloudNetwork {
		m.CloudNetworks[i] = cloudNetworkModel{
			ID:      types.Int64Value(int64(c.ID)),
			IP:      types.StringValue(c.IP),
			Mask:    types.Int64Value(int64(c.Mask)),
			Gateway: optString(c.Gateway),
		}
	}
}
