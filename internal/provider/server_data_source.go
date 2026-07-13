package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ datasource.DataSource              = (*serverDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*serverDataSource)(nil)
)

func NewServerDataSource() datasource.DataSource {
	return &serverDataSource{}
}

type serverDataSource struct {
	client *hrobot.Client
}

type serverModel struct {
	ID         types.Int64  `tfsdk:"id"`
	ServerIP   types.String `tfsdk:"server_ip"`
	ServerName types.String `tfsdk:"server_name"`
	Product    types.String `tfsdk:"product"`
	DC         types.String `tfsdk:"dc"`
	Status     types.String `tfsdk:"status"`
	Cancelled  types.Bool   `tfsdk:"cancelled"`
	PaidUntil  types.String `tfsdk:"paid_until"`
	IPs        types.List   `tfsdk:"ips"`
}

func (d *serverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *serverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Hetzner Robot dedicated server by its server number.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.Int64Attribute{Required: true, Description: "Server number."},
			"server_ip":   schema.StringAttribute{Computed: true, Description: "Main (primary) IP address of the server."},
			"server_name": schema.StringAttribute{Computed: true, Description: "Server name set in Robot."},
			"product":     schema.StringAttribute{Computed: true, Description: "Product name (e.g. AX41-NVMe)."},
			"dc":          schema.StringAttribute{Computed: true, Description: "Datacenter the server is located in."},
			"status":      schema.StringAttribute{Computed: true, Description: "Server status (ready, in process, or cancelled)."},
			"cancelled":   schema.BoolAttribute{Computed: true, Description: "Whether the server is cancelled."},
			"paid_until":  schema.StringAttribute{Computed: true, Description: "Date the server is paid until."},
			"ips":         schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "All single IP addresses assigned to the server."},
		},
	}
}

func (d *serverDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*hrobot.Client)
	if !ok {
		resp.Diagnostics.AddError("unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	srv, err := d.client.Server.Get(ctx, hrobot.ServerID(data.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("hrobot server lookup failed", err.Error())
		return
	}

	resp.Diagnostics.Append(setServerModel(ctx, &data, srv)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setServerModel(ctx context.Context, m *serverModel, srv *hrobot.Server) diag.Diagnostics {
	var diags diag.Diagnostics
	m.ID = types.Int64Value(int64(srv.ServerNumber))
	m.ServerIP = types.StringValue(srv.ServerIP.String())
	m.ServerName = types.StringValue(srv.ServerName)
	m.Product = types.StringValue(srv.Product)
	m.DC = types.StringValue(srv.DC)
	m.Status = types.StringValue(string(srv.Status))
	m.Cancelled = types.BoolValue(srv.Cancelled)
	m.PaidUntil = types.StringValue(srv.PaidUntil)

	ips := make([]string, 0, len(srv.IP))
	for _, ip := range srv.IP {
		ips = append(ips, ip.String())
	}
	list, d := types.ListValueFrom(ctx, types.StringType, ips)
	diags.Append(d...)
	m.IPs = list
	return diags
}
