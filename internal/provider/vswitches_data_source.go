package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ datasource.DataSource              = (*vswitchesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*vswitchesDataSource)(nil)
)

// NewVSwitchesDataSource returns the hrobot_vswitches data source.
func NewVSwitchesDataSource() datasource.DataSource {
	return &vswitchesDataSource{}
}

type vswitchesDataSource struct {
	client *hrobot.Client
}

type vswitchSummaryModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	VLAN      types.Int64  `tfsdk:"vlan"`
	Cancelled types.Bool   `tfsdk:"cancelled"`
}

type vswitchesModel struct {
	VSwitches []vswitchSummaryModel `tfsdk:"vswitches"`
}

func (d *vswitchesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vswitches"
}

func (d *vswitchesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Hetzner Robot vSwitches (summary view; use hrobot_vswitch for attached servers/subnets).",
		Attributes: map[string]schema.Attribute{
			"vswitches": schema.ListNestedAttribute{
				Computed:    true,
				Description: "All vSwitches.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":        schema.Int64Attribute{Computed: true, Description: "vSwitch ID."},
					"name":      schema.StringAttribute{Computed: true, Description: "vSwitch name."},
					"vlan":      schema.Int64Attribute{Computed: true, Description: "VLAN ID."},
					"cancelled": schema.BoolAttribute{Computed: true, Description: "Whether the vSwitch is cancelled."},
				}},
			},
		},
	}
}

func (d *vswitchesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *vswitchesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	list, err := d.client.VSwitch.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError("hrobot vswitch list failed", err.Error())
		return
	}
	out := vswitchesModel{VSwitches: make([]vswitchSummaryModel, len(list))}
	for i, vs := range list {
		out.VSwitches[i] = vswitchSummaryModel{
			ID:        types.Int64Value(int64(vs.ID)),
			Name:      types.StringValue(vs.Name),
			VLAN:      types.Int64Value(int64(vs.VLAN)),
			Cancelled: types.BoolValue(vs.Cancelled),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
