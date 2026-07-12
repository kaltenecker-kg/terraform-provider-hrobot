package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ datasource.DataSource              = (*storageBoxesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*storageBoxesDataSource)(nil)
)

// NewStorageBoxesDataSource returns the hrobot_storageboxes data source.
func NewStorageBoxesDataSource() datasource.DataSource {
	return &storageBoxesDataSource{}
}

type storageBoxesDataSource struct {
	client *hrobot.Client
}

type storageBoxesModel struct {
	StorageBoxes []storageBoxModel `tfsdk:"storageboxes"`
}

func (d *storageBoxesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storageboxes"
}

func (d *storageBoxesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Hetzner Storage Boxes on the account.",
		Attributes: map[string]schema.Attribute{
			"storageboxes": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "All Storage Boxes.",
				NestedObject: schema.NestedAttributeObject{Attributes: storageBoxAttributes(false)},
			},
		},
	}
}

func (d *storageBoxesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *storageBoxesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	list, err := d.client.StorageBox.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError("hrobot storagebox list failed", err.Error())
		return
	}
	out := storageBoxesModel{StorageBoxes: make([]storageBoxModel, len(list))}
	for i := range list {
		setStorageBoxModel(&out.StorageBoxes[i], &list[i])
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
