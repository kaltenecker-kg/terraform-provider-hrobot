package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ datasource.DataSource              = (*ipsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*ipsDataSource)(nil)
)

// NewIPsDataSource returns the hrobot_ips data source.
func NewIPsDataSource() datasource.DataSource {
	return &ipsDataSource{}
}

type ipsDataSource struct {
	client *hrobot.Client
}

type ipsModel struct {
	IPs []ipModel `tfsdk:"ips"`
}

func (d *ipsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ips"
}

func (d *ipsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all single IP addresses managed in Hetzner Robot.",
		Attributes: map[string]schema.Attribute{
			"ips": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "All IP addresses.",
				NestedObject: schema.NestedAttributeObject{Attributes: ipAttributes(false)},
			},
		},
	}
}

func (d *ipsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *ipsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	list, err := d.client.IP.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError("hrobot ip list failed", err.Error())
		return
	}
	out := ipsModel{IPs: make([]ipModel, len(list))}
	for i := range list {
		setIPModel(&out.IPs[i], &list[i])
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
