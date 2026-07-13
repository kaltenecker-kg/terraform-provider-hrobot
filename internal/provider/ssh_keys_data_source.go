package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ datasource.DataSource              = (*sshKeysDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*sshKeysDataSource)(nil)
)

// NewSSHKeysDataSource returns the hrobot_ssh_keys data source.
func NewSSHKeysDataSource() datasource.DataSource {
	return &sshKeysDataSource{}
}

type sshKeysDataSource struct {
	client *hrobot.Client
}

type sshKeysModel struct {
	Keys []sshKeyModel `tfsdk:"ssh_keys"`
}

func (d *sshKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_keys"
}

func (d *sshKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all SSH keys stored in the Hetzner Robot account.",
		Attributes: map[string]schema.Attribute{
			"ssh_keys": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "All SSH keys.",
				NestedObject: schema.NestedAttributeObject{Attributes: sshKeyAttributes(false)},
			},
		},
	}
}

func (d *sshKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *sshKeysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	keys, err := d.client.Key.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError("hrobot ssh key list failed", err.Error())
		return
	}
	out := sshKeysModel{Keys: make([]sshKeyModel, len(keys))}
	for i := range keys {
		setSSHKeyModel(&out.Keys[i], &keys[i])
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
