package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ datasource.DataSource              = (*trafficDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*trafficDataSource)(nil)
)

// NewTrafficDataSource returns the hrobot_traffic data source.
func NewTrafficDataSource() datasource.DataSource {
	return &trafficDataSource{}
}

type trafficDataSource struct {
	client *hrobot.Client
}

type trafficModel struct {
	Type types.String `tfsdk:"type"`
	From types.String `tfsdk:"from"`
	To   types.String `tfsdk:"to"`
	IP   types.String `tfsdk:"ip"`
	Data types.Map    `tfsdk:"data"`
}

type trafficStatsModel struct {
	In  types.Float64 `tfsdk:"in"`
	Out types.Float64 `tfsdk:"out"`
	Sum types.Float64 `tfsdk:"sum"`
}

var trafficStatsAttrTypes = map[string]attr.Type{
	"in":  types.Float64Type,
	"out": types.Float64Type,
	"sum": types.Float64Type,
}

func (d *trafficDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_traffic"
}

func (d *trafficDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Query traffic statistics for an IP address over a time range.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{Required: true, Description: "Aggregation granularity: day, month, or year."},
			"from": schema.StringAttribute{Required: true, Description: "Start of the range (e.g. 2026-07-01T00 for day, 2026-07 for month, 2026 for year)."},
			"to":   schema.StringAttribute{Required: true, Description: "End of the range, same format as `from`."},
			"ip":   schema.StringAttribute{Required: true, Description: "IP address to report traffic for."},
			"data": schema.MapNestedAttribute{
				Computed:    true,
				Description: "Traffic totals keyed by IP address; values are in/out/sum in the API's reported unit.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"in":  schema.Float64Attribute{Computed: true, Description: "Inbound traffic."},
					"out": schema.Float64Attribute{Computed: true, Description: "Outbound traffic."},
					"sum": schema.Float64Attribute{Computed: true, Description: "Total traffic."},
				}},
			},
		},
	}
}

func (d *trafficDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *trafficDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data trafficModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	td, err := d.client.Traffic.Get(ctx, hrobot.TrafficGetParams{
		Type: hrobot.TrafficType(data.Type.ValueString()),
		From: data.From.ValueString(),
		To:   data.To.ValueString(),
		IP:   data.IP.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("hrobot traffic query failed", err.Error())
		return
	}

	goMap := make(map[string]trafficStatsModel, len(td.Data))
	for k, v := range td.Data {
		goMap[k] = trafficStatsModel{
			In:  types.Float64Value(v.In),
			Out: types.Float64Value(v.Out),
			Sum: types.Float64Value(v.Sum),
		}
	}
	m, diags := types.MapValueFrom(ctx, types.ObjectType{AttrTypes: trafficStatsAttrTypes}, goMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Data = m
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
