package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ datasource.DataSource              = (*sshKeyDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*sshKeyDataSource)(nil)
)

// NewSSHKeyDataSource returns the hrobot_ssh_key data source.
func NewSSHKeyDataSource() datasource.DataSource {
	return &sshKeyDataSource{}
}

type sshKeyDataSource struct {
	client *hrobot.Client
}

type sshKeyModel struct {
	Fingerprint types.String `tfsdk:"fingerprint"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	Size        types.Int64  `tfsdk:"size"`
	Data        types.String `tfsdk:"data"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func (d *sshKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func sshKeyAttributes(withFingerprintInput bool) map[string]schema.Attribute {
	fp := schema.StringAttribute{Computed: true, Description: "Key fingerprint (the identifier used by the Robot API)."}
	if withFingerprintInput {
		fp = schema.StringAttribute{Required: true, Description: "Key fingerprint (the identifier used by the Robot API)."}
	}
	return map[string]schema.Attribute{
		"fingerprint": fp,
		"name":        schema.StringAttribute{Computed: true, Description: "Key name."},
		"type":        schema.StringAttribute{Computed: true, Description: "Key algorithm (e.g. ECDSA, RSA, ED25519)."},
		"size":        schema.Int64Attribute{Computed: true, Description: "Key size in bits."},
		"data":        schema.StringAttribute{Computed: true, Description: "The public key material."},
		"created_at":  schema.StringAttribute{Computed: true, Description: "Creation timestamp (RFC 3339)."},
	}
}

func (d *sshKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Hetzner Robot SSH key by its fingerprint.",
		Attributes:  sshKeyAttributes(true),
	}
}

func (d *sshKeyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *sshKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sshKeyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	key, err := d.client.Key.Get(ctx, data.Fingerprint.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("hrobot ssh key lookup failed", err.Error())
		return
	}
	setSSHKeyModel(&data, key)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setSSHKeyModel(m *sshKeyModel, k *hrobot.SSHKey) {
	m.Fingerprint = types.StringValue(k.Fingerprint)
	m.Name = types.StringValue(k.Name)
	m.Type = types.StringValue(k.Type)
	m.Size = types.Int64Value(int64(k.Size))
	m.Data = types.StringValue(k.Data)
	m.CreatedAt = types.StringValue(k.CreatedAt.Format(time.RFC3339))
}
