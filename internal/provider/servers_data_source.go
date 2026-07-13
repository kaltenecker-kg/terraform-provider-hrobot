package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ datasource.DataSource              = (*serversDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*serversDataSource)(nil)
)

func NewServersDataSource() datasource.DataSource {
	return &serversDataSource{}
}

type serversDataSource struct {
	client *hrobot.Client
}

type serversModel struct {
	Servers []serverModel `tfsdk:"servers"`
}

func (d *serversDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_servers"
}

func (d *serversDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Hetzner Robot dedicated servers on the account.",
		Attributes: map[string]schema.Attribute{
			"servers": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.Int64Attribute{Computed: true, Description: "Server number."},
						"server_ip":   schema.StringAttribute{Computed: true, Description: "Main (primary) IP address of the server."},
						"server_name": schema.StringAttribute{Computed: true, Description: "Server name set in Robot."},
						"product":     schema.StringAttribute{Computed: true, Description: "Product name (e.g. AX41-NVMe)."},
						"dc":          schema.StringAttribute{Computed: true, Description: "Datacenter the server is located in."},
						"status":      schema.StringAttribute{Computed: true, Description: "Server status (ready, in process, or cancelled)."},
						"cancelled":   schema.BoolAttribute{Computed: true, Description: "Whether the server is cancelled."},
						"paid_until":  schema.StringAttribute{Computed: true, Description: "Date the server is paid until."},
						"ips":         schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "All single IP addresses assigned to the server."},
					},
				},
			},
		},
	}
}

func (d *serversDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serversDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	servers, err := d.client.Server.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError("hrobot server list failed", err.Error())
		return
	}

	out := serversModel{Servers: make([]serverModel, len(servers))}
	for i := range servers {
		resp.Diagnostics.Append(setServerModel(ctx, &out.Servers[i], &servers[i])...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
