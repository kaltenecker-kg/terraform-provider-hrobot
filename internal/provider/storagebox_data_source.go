package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ datasource.DataSource              = (*storageBoxDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*storageBoxDataSource)(nil)
)

// NewStorageBoxDataSource returns the hrobot_storagebox data source.
func NewStorageBoxDataSource() datasource.DataSource {
	return &storageBoxDataSource{}
}

type storageBoxDataSource struct {
	client *hrobot.Client
}

type storageBoxModel struct {
	ID                   types.Int64  `tfsdk:"id"`
	Login                types.String `tfsdk:"login"`
	Name                 types.String `tfsdk:"name"`
	Product              types.String `tfsdk:"product"`
	Cancelled            types.Bool   `tfsdk:"cancelled"`
	Locked               types.Bool   `tfsdk:"locked"`
	Location             types.String `tfsdk:"location"`
	LinkedServer         types.Int64  `tfsdk:"linked_server"`
	PaidUntil            types.String `tfsdk:"paid_until"`
	DiskQuota            types.Int64  `tfsdk:"disk_quota"`
	DiskUsage            types.Int64  `tfsdk:"disk_usage"`
	DiskUsageData        types.Int64  `tfsdk:"disk_usage_data"`
	DiskUsageSnapshots   types.Int64  `tfsdk:"disk_usage_snapshots"`
	Webdav               types.Bool   `tfsdk:"webdav"`
	Samba                types.Bool   `tfsdk:"samba"`
	SSH                  types.Bool   `tfsdk:"ssh"`
	ExternalReachability types.Bool   `tfsdk:"external_reachability"`
	ZFS                  types.Bool   `tfsdk:"zfs"`
	Server               types.String `tfsdk:"server"`
	HostSystem           types.String `tfsdk:"host_system"`
}

func storageBoxAttributes(withIDInput bool) map[string]schema.Attribute {
	id := schema.Int64Attribute{Computed: true, Description: "Storage Box ID."}
	if withIDInput {
		id = schema.Int64Attribute{Required: true, Description: "Storage Box ID to look up."}
	}
	return map[string]schema.Attribute{
		"id":                    id,
		"login":                 schema.StringAttribute{Computed: true, Description: "Login/username of the Storage Box."},
		"name":                  schema.StringAttribute{Computed: true, Description: "Storage Box name."},
		"product":               schema.StringAttribute{Computed: true, Description: "Product name (e.g. BX11)."},
		"cancelled":             schema.BoolAttribute{Computed: true, Description: "Whether the Storage Box is cancelled."},
		"locked":                schema.BoolAttribute{Computed: true, Description: "Whether the Storage Box is locked."},
		"location":              schema.StringAttribute{Computed: true, Description: "Datacenter location."},
		"linked_server":         schema.Int64Attribute{Computed: true, Description: "Number of the linked server (0 if none)."},
		"paid_until":            schema.StringAttribute{Computed: true, Description: "Paid-until date."},
		"disk_quota":            schema.Int64Attribute{Computed: true, Description: "Total disk quota (MB)."},
		"disk_usage":            schema.Int64Attribute{Computed: true, Description: "Total disk usage (MB)."},
		"disk_usage_data":       schema.Int64Attribute{Computed: true, Description: "Disk usage by data (MB)."},
		"disk_usage_snapshots":  schema.Int64Attribute{Computed: true, Description: "Disk usage by snapshots (MB)."},
		"webdav":                schema.BoolAttribute{Computed: true, Description: "Whether WebDAV is enabled."},
		"samba":                 schema.BoolAttribute{Computed: true, Description: "Whether Samba/CIFS is enabled."},
		"ssh":                   schema.BoolAttribute{Computed: true, Description: "Whether SSH is enabled."},
		"external_reachability": schema.BoolAttribute{Computed: true, Description: "Whether the box is reachable from outside the Hetzner network."},
		"zfs":                   schema.BoolAttribute{Computed: true, Description: "Whether ZFS snapshot directory access is enabled."},
		"server":                schema.StringAttribute{Computed: true, Description: "Hostname clients connect to."},
		"host_system":           schema.StringAttribute{Computed: true, Description: "Host system identifier."},
	}
}

func (d *storageBoxDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Hetzner Storage Box by its ID.",
		Attributes:  storageBoxAttributes(true),
	}
}

func (d *storageBoxDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storagebox"
}

func (d *storageBoxDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *storageBoxDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data storageBoxModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	box, err := d.client.StorageBox.Get(ctx, int(data.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("hrobot storagebox lookup failed", err.Error())
		return
	}
	setStorageBoxModel(&data, box)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setStorageBoxModel(m *storageBoxModel, b *hrobot.StorageBox) {
	m.ID = types.Int64Value(int64(b.ID))
	m.Login = types.StringValue(b.Login)
	m.Name = types.StringValue(b.Name)
	m.Product = types.StringValue(b.Product)
	m.Cancelled = types.BoolValue(b.Cancelled)
	m.Locked = types.BoolValue(b.Locked)
	m.Location = types.StringValue(b.Location)
	m.LinkedServer = types.Int64Value(int64(b.LinkedServer))
	m.PaidUntil = types.StringValue(b.PaidUntil)
	m.DiskQuota = types.Int64Value(int64(b.DiskQuota))
	m.DiskUsage = types.Int64Value(int64(b.DiskUsage))
	m.DiskUsageData = types.Int64Value(int64(b.DiskUsageData))
	m.DiskUsageSnapshots = types.Int64Value(int64(b.DiskUsageSnapshots))
	m.Webdav = types.BoolValue(b.Webdav)
	m.Samba = types.BoolValue(b.Samba)
	m.SSH = types.BoolValue(b.SSH)
	m.ExternalReachability = types.BoolValue(b.ExternalReachability)
	m.ZFS = types.BoolValue(b.ZFS)
	m.Server = types.StringValue(b.Server)
	m.HostSystem = types.StringValue(b.HostSystem)
}
