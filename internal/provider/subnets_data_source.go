package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ datasource.DataSource              = (*subnetsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*subnetsDataSource)(nil)
)

// NewSubnetsDataSource returns the hrobot_subnets data source.
func NewSubnetsDataSource() datasource.DataSource {
	return &subnetsDataSource{}
}

type subnetsDataSource struct {
	client *hrobot.Client
}

type subnetsModel struct {
	Subnets []subnetModel `tfsdk:"subnets"`
}

func (d *subnetsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnets"
}

func (d *subnetsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all subnets managed in Hetzner Robot.",
		Attributes: map[string]schema.Attribute{
			"subnets": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "All subnets.",
				NestedObject: schema.NestedAttributeObject{Attributes: subnetAttributes(false)},
			},
		},
	}
}

func (d *subnetsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *subnetsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	list, err := d.client.Subnet.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError("hrobot subnet list failed", err.Error())
		return
	}
	out := subnetsModel{Subnets: make([]subnetModel, len(list))}
	for i := range list {
		setSubnetModel(&out.Subnets[i], &list[i])
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
