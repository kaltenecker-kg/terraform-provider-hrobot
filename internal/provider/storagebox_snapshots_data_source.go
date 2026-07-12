package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ datasource.DataSource              = (*storageBoxSnapshotsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*storageBoxSnapshotsDataSource)(nil)
)

// NewStorageBoxSnapshotsDataSource returns the hrobot_storagebox_snapshots data source.
func NewStorageBoxSnapshotsDataSource() datasource.DataSource {
	return &storageBoxSnapshotsDataSource{}
}

type storageBoxSnapshotsDataSource struct {
	client *hrobot.Client
}

type snapshotModel struct {
	Name           types.String `tfsdk:"name"`
	Timestamp      types.String `tfsdk:"timestamp"`
	Size           types.Int64  `tfsdk:"size"`
	FilesystemSize types.Int64  `tfsdk:"filesystem_size"`
	Automatic      types.Bool   `tfsdk:"automatic"`
	Comment        types.String `tfsdk:"comment"`
}

type storageBoxSnapshotsModel struct {
	StorageBoxID types.Int64     `tfsdk:"storagebox_id"`
	Snapshots    []snapshotModel `tfsdk:"snapshots"`
}

func (d *storageBoxSnapshotsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storagebox_snapshots"
}

func (d *storageBoxSnapshotsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all snapshots of a Hetzner Storage Box.",
		Attributes: map[string]schema.Attribute{
			"storagebox_id": schema.Int64Attribute{Required: true, Description: "Storage Box ID."},
			"snapshots": schema.ListNestedAttribute{
				Computed:    true,
				Description: "All snapshots of the Storage Box.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"name":            schema.StringAttribute{Computed: true, Description: "Snapshot name."},
					"timestamp":       schema.StringAttribute{Computed: true, Description: "Snapshot creation timestamp."},
					"size":            schema.Int64Attribute{Computed: true, Description: "Snapshot size (bytes)."},
					"filesystem_size": schema.Int64Attribute{Computed: true, Description: "Filesystem size at snapshot time (bytes)."},
					"automatic":       schema.BoolAttribute{Computed: true, Description: "Whether the snapshot was created automatically by a snapshot plan."},
					"comment":         schema.StringAttribute{Computed: true, Description: "Free-form comment."},
				}},
			},
		},
	}
}

func (d *storageBoxSnapshotsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *storageBoxSnapshotsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data storageBoxSnapshotsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.StorageBox.ListSnapshots(ctx, int(data.StorageBoxID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("hrobot storagebox snapshot list failed", err.Error())
		return
	}
	data.Snapshots = make([]snapshotModel, len(list))
	for i, s := range list {
		data.Snapshots[i] = snapshotModel{
			Name:           types.StringValue(s.Name),
			Timestamp:      types.StringValue(s.Timestamp),
			Size:           types.Int64Value(int64(s.Size)),
			FilesystemSize: types.Int64Value(int64(s.FilesystemSize)),
			Automatic:      types.BoolValue(s.Automatic),
			Comment:        optString(s.Comment),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
