package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ datasource.DataSource              = (*storageBoxSubAccountsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*storageBoxSubAccountsDataSource)(nil)
)

// NewStorageBoxSubAccountsDataSource returns the hrobot_storagebox_subaccounts data source.
func NewStorageBoxSubAccountsDataSource() datasource.DataSource {
	return &storageBoxSubAccountsDataSource{}
}

type storageBoxSubAccountsDataSource struct {
	client *hrobot.Client
}

type subAccountModel struct {
	Username             types.String `tfsdk:"username"`
	AccountID            types.String `tfsdk:"account_id"`
	Server               types.String `tfsdk:"server"`
	HomeDirectory        types.String `tfsdk:"home_directory"`
	Samba                types.Bool   `tfsdk:"samba"`
	SSH                  types.Bool   `tfsdk:"ssh"`
	ExternalReachability types.Bool   `tfsdk:"external_reachability"`
	Webdav               types.Bool   `tfsdk:"webdav"`
	ReadOnly             types.Bool   `tfsdk:"readonly"`
	CreateTime           types.String `tfsdk:"create_time"`
	Comment              types.String `tfsdk:"comment"`
}

type storageBoxSubAccountsModel struct {
	StorageBoxID types.Int64       `tfsdk:"storagebox_id"`
	SubAccounts  []subAccountModel `tfsdk:"subaccounts"`
}

func (d *storageBoxSubAccountsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storagebox_subaccounts"
}

func (d *storageBoxSubAccountsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all sub-accounts of a Hetzner Storage Box.",
		Attributes: map[string]schema.Attribute{
			"storagebox_id": schema.Int64Attribute{Required: true, Description: "Storage Box ID."},
			"subaccounts": schema.ListNestedAttribute{
				Computed:    true,
				Description: "All sub-accounts of the Storage Box.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"username":              schema.StringAttribute{Computed: true, Description: "Sub-account username."},
					"account_id":            schema.StringAttribute{Computed: true, Description: "Sub-account ID."},
					"server":                schema.StringAttribute{Computed: true, Description: "Hostname clients connect to."},
					"home_directory":        schema.StringAttribute{Computed: true, Description: "Home directory of the sub-account."},
					"samba":                 schema.BoolAttribute{Computed: true, Description: "Whether Samba/CIFS is enabled."},
					"ssh":                   schema.BoolAttribute{Computed: true, Description: "Whether SSH is enabled."},
					"external_reachability": schema.BoolAttribute{Computed: true, Description: "Whether the sub-account is externally reachable."},
					"webdav":                schema.BoolAttribute{Computed: true, Description: "Whether WebDAV is enabled."},
					"readonly":              schema.BoolAttribute{Computed: true, Description: "Whether the sub-account is read-only."},
					"create_time":           schema.StringAttribute{Computed: true, Description: "Creation timestamp."},
					"comment":               schema.StringAttribute{Computed: true, Description: "Free-form comment."},
				}},
			},
		},
	}
}

func (d *storageBoxSubAccountsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *storageBoxSubAccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data storageBoxSubAccountsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.StorageBox.ListSubAccounts(ctx, int(data.StorageBoxID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("hrobot storagebox subaccount list failed", err.Error())
		return
	}
	data.SubAccounts = make([]subAccountModel, len(list))
	for i, s := range list {
		data.SubAccounts[i] = subAccountModel{
			Username:             types.StringValue(s.Username),
			AccountID:            types.StringValue(s.AccountID),
			Server:               types.StringValue(s.Server),
			HomeDirectory:        types.StringValue(s.HomeDirectory),
			Samba:                types.BoolValue(s.Samba),
			SSH:                  types.BoolValue(s.SSH),
			ExternalReachability: types.BoolValue(s.ExternalReachability),
			Webdav:               types.BoolValue(s.Webdav),
			ReadOnly:             types.BoolValue(s.ReadOnly),
			CreateTime:           types.StringValue(s.CreateTime),
			Comment:              optString(s.Comment),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
