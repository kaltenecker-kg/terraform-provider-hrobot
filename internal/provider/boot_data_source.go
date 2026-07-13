package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ datasource.DataSource              = (*bootDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*bootDataSource)(nil)
)

// NewBootDataSource returns the hrobot_boot data source.
func NewBootDataSource() datasource.DataSource {
	return &bootDataSource{}
}

type bootDataSource struct {
	client *hrobot.Client
}

type bootModel struct {
	ServerNumber types.Int64        `tfsdk:"server_number"`
	Rescue       *bootRescueModel   `tfsdk:"rescue"`
	Linux        *bootLinuxModel    `tfsdk:"linux"`
	VNC          *bootLinuxModel    `tfsdk:"vnc"`
	Windows      *bootWindowsModel  `tfsdk:"windows"`
	Plesk        *bootHostnameModel `tfsdk:"plesk"`
	CPanel       *bootHostnameModel `tfsdk:"cpanel"`
}

type bootRescueModel struct {
	Active types.Bool   `tfsdk:"active"`
	OS     types.String `tfsdk:"os"`
	Arch   types.Int64  `tfsdk:"arch"`
}

type bootLinuxModel struct {
	Active types.Bool   `tfsdk:"active"`
	Dist   types.String `tfsdk:"dist"`
	Arch   types.Int64  `tfsdk:"arch"`
	Lang   types.String `tfsdk:"lang"`
}

type bootWindowsModel struct {
	Active types.Bool   `tfsdk:"active"`
	OS     types.String `tfsdk:"os"`
	Lang   types.String `tfsdk:"lang"`
}

type bootHostnameModel struct {
	Active   types.Bool   `tfsdk:"active"`
	Hostname types.String `tfsdk:"hostname"`
}

func (d *bootDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_boot"
}

func (d *bootDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	langArch := func(withDist bool) map[string]schema.Attribute {
		m := map[string]schema.Attribute{
			"active": schema.BoolAttribute{Computed: true, Description: "Whether this boot configuration is currently active."},
			"arch":   schema.Int64Attribute{Computed: true, Description: "Active architecture (32 or 64)."},
			"lang":   schema.StringAttribute{Computed: true, Description: "Active language/keyboard layout."},
		}
		if withDist {
			m["dist"] = schema.StringAttribute{Computed: true, Description: "Active distribution."}
		}
		return m
	}
	resp.Schema = schema.Schema{
		Description: "Read the boot configuration of a Hetzner Robot server (which rescue/linux/vnc/windows/plesk/cpanel option, if any, is active). Boot activation is a one-shot operation and is intentionally not a managed resource.",
		Attributes: map[string]schema.Attribute{
			"server_number": schema.Int64Attribute{Required: true, Description: "Server number."},
			"rescue": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Rescue system boot configuration.",
				Attributes: map[string]schema.Attribute{
					"active": schema.BoolAttribute{Computed: true, Description: "Whether the rescue system is active for the next boot."},
					"os":     schema.StringAttribute{Computed: true, Description: "Active rescue OS."},
					"arch":   schema.Int64Attribute{Computed: true, Description: "Active architecture (32 or 64)."},
				},
			},
			"linux": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Linux installation boot configuration.",
				Attributes:  langArch(true),
			},
			"vnc": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "VNC installation boot configuration.",
				Attributes:  langArch(true),
			},
			"windows": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Windows installation boot configuration.",
				Attributes: map[string]schema.Attribute{
					"active": schema.BoolAttribute{Computed: true, Description: "Whether the Windows installer is active for the next boot."},
					"os":     schema.StringAttribute{Computed: true, Description: "Active Windows edition."},
					"lang":   schema.StringAttribute{Computed: true, Description: "Active language."},
				},
			},
			"plesk": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Plesk installation boot configuration.",
				Attributes: map[string]schema.Attribute{
					"active":   schema.BoolAttribute{Computed: true, Description: "Whether the Plesk installer is active."},
					"hostname": schema.StringAttribute{Computed: true, Description: "Configured hostname."},
				},
			},
			"cpanel": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "cPanel installation boot configuration.",
				Attributes: map[string]schema.Attribute{
					"active":   schema.BoolAttribute{Computed: true, Description: "Whether the cPanel installer is active."},
					"hostname": schema.StringAttribute{Computed: true, Description: "Configured hostname."},
				},
			},
		},
	}
}

func (d *bootDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *bootDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data bootModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := d.client.Boot.Get(ctx, hrobot.ServerID(data.ServerNumber.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("hrobot boot lookup failed", err.Error())
		return
	}
	setBootModel(&data, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setBootModel(m *bootModel, c *hrobot.BootConfig) {
	if c.Rescue != nil {
		m.Rescue = &bootRescueModel{
			Active: types.BoolValue(c.Rescue.Active),
			OS:     optString(c.Rescue.ActiveOS()),
			Arch:   types.Int64Value(int64(c.Rescue.ActiveArch())),
		}
	}
	if c.Linux != nil {
		m.Linux = &bootLinuxModel{
			Active: types.BoolValue(c.Linux.Active),
			Dist:   optString(c.Linux.ActiveDist()),
			Arch:   types.Int64Value(int64(c.Linux.ActiveArch())),
			Lang:   optString(c.Linux.ActiveLang()),
		}
	}
	if c.VNC != nil {
		m.VNC = &bootLinuxModel{
			Active: types.BoolValue(c.VNC.Active),
			Dist:   optString(c.VNC.ActiveDist()),
			Arch:   types.Int64Value(int64(c.VNC.ActiveArch())),
			Lang:   optString(c.VNC.ActiveLang()),
		}
	}
	if c.Windows != nil {
		m.Windows = &bootWindowsModel{
			Active: types.BoolValue(c.Windows.Active),
			OS:     optString(c.Windows.ActiveOS()),
			Lang:   optString(c.Windows.ActiveLang()),
		}
	}
	if c.Plesk != nil {
		m.Plesk = &bootHostnameModel{
			Active:   types.BoolValue(c.Plesk.Active),
			Hostname: optString(c.Plesk.Hostname),
		}
	}
	if c.CPanel != nil {
		m.CPanel = &bootHostnameModel{
			Active:   types.BoolValue(c.CPanel.Active),
			Hostname: optString(c.CPanel.Hostname),
		}
	}
}
