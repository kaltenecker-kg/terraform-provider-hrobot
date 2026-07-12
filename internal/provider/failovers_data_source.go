package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ datasource.DataSource              = (*failoversDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*failoversDataSource)(nil)
)

// NewFailoversDataSource returns the hrobot_failovers data source.
func NewFailoversDataSource() datasource.DataSource {
	return &failoversDataSource{}
}

type failoversDataSource struct {
	client *hrobot.Client
}

type failoversModel struct {
	Failovers []failoverModel `tfsdk:"failovers"`
}

func (d *failoversDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_failovers"
}

func (d *failoversDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Hetzner Robot failover IPs and their routing targets.",
		Attributes: map[string]schema.Attribute{
			"failovers": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "All failover IPs.",
				NestedObject: schema.NestedAttributeObject{Attributes: failoverAttributes(false)},
			},
		},
	}
}

func (d *failoversDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *failoversDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	list, err := d.client.Failover.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError("hrobot failover list failed", err.Error())
		return
	}
	out := failoversModel{Failovers: make([]failoverModel, len(list))}
	for i := range list {
		setFailoverModel(&out.Failovers[i], &list[i])
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
