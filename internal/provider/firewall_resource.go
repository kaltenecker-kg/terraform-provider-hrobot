package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ resource.Resource                = (*firewallResource)(nil)
	_ resource.ResourceWithConfigure   = (*firewallResource)(nil)
	_ resource.ResourceWithImportState = (*firewallResource)(nil)
)

func NewFirewallResource() resource.Resource {
	return &firewallResource{}
}

type firewallResource struct {
	client *hrobot.Client
}

type firewallModel struct {
	ID           types.String `tfsdk:"id"`
	ServerNumber types.Int64  `tfsdk:"server_number"`
	Status       types.String `tfsdk:"status"`
	FilterIPv6   types.Bool   `tfsdk:"filter_ipv6"`
	WhitelistHOS types.Bool   `tfsdk:"whitelist_hos"`
	Port         types.String `tfsdk:"port"`
	InputRules   []ruleModel  `tfsdk:"input_rule"`
	OutputRules  []ruleModel  `tfsdk:"output_rule"`
}

type ruleModel struct {
	Name       types.String `tfsdk:"name"`
	IPVersion  types.String `tfsdk:"ip_version"`
	Action     types.String `tfsdk:"action"`
	Protocol   types.String `tfsdk:"protocol"`
	SourceIP   types.String `tfsdk:"src_ip"`
	DestIP     types.String `tfsdk:"dst_ip"`
	SourcePort types.String `tfsdk:"src_port"`
	DestPort   types.String `tfsdk:"dst_port"`
	TCPFlags   types.String `tfsdk:"tcp_flags"`
}

func (r *firewallResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall"
}

func (r *firewallResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	ruleAttrs := map[string]schema.Attribute{
		"name":       schema.StringAttribute{Optional: true, Description: "Rule name."},
		"ip_version": schema.StringAttribute{Optional: true, Description: "ipv4 or ipv6. Required for input rules; leave empty for output rules to apply to both."},
		"action":     schema.StringAttribute{Required: true, Description: "accept or discard."},
		"protocol":   schema.StringAttribute{Optional: true, Description: "tcp, udp, icmp, gre, ipip, ah, esp, ipencap."},
		"src_ip":     schema.StringAttribute{Optional: true, Description: "Source CIDR."},
		"dst_ip":     schema.StringAttribute{Optional: true, Description: "Destination CIDR."},
		"src_port":   schema.StringAttribute{Optional: true, Description: "Source port or range (e.g. 1024-65535)."},
		"dst_port":   schema.StringAttribute{Optional: true, Description: "Destination port or range."},
		"tcp_flags":  schema.StringAttribute{Optional: true, Description: "TCP flags expression (e.g. syn|fin)."},
	}

	resp.Schema = schema.Schema{
		Description: "Manages the firewall configuration of a Hetzner Robot server. Hetzner allows at most 10 input rules.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Server number (mirrors `server_number`).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"server_number": schema.Int64Attribute{
				Required:    true,
				Description: "Server number to attach the firewall configuration to.",
			},
			"status": schema.StringAttribute{
				Required:    true,
				Description: "active or disabled.",
			},
			"filter_ipv6": schema.BoolAttribute{
				Optional: true, Computed: true,
				Default:     booldefault.StaticBool(false),
				Description: "Apply the rules to IPv6 traffic too.",
			},
			"whitelist_hos": schema.BoolAttribute{
				Optional: true, Computed: true,
				Default:     booldefault.StaticBool(false),
				Description: "Allow Hetzner online services regardless of rules.",
			},
			"port": schema.StringAttribute{
				Computed:    true,
				Description: "Switch port the firewall is attached to (informational).",
			},
			"input_rule": schema.ListNestedAttribute{
				Optional:     true,
				Description:  "Inbound rules, evaluated top to bottom. Hetzner enforces a maximum of 10.",
				NestedObject: schema.NestedAttributeObject{Attributes: ruleAttrs},
			},
			"output_rule": schema.ListNestedAttribute{
				Optional:     true,
				Description:  "Outbound rules, evaluated top to bottom.",
				NestedObject: schema.NestedAttributeObject{Attributes: ruleAttrs},
			},
		},
	}
}

func (r *firewallResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*hrobot.Client)
	if !ok {
		resp.Diagnostics.AddError("unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *firewallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan firewallModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.Diagnostics, &resp.State)
}

func (r *firewallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state firewallModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.Firewall.Get(ctx, hrobot.ServerID(state.ServerNumber.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("hrobot firewall read failed", err.Error())
		return
	}
	setFirewallModel(&state, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *firewallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan firewallModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.Diagnostics, &resp.State)
}

func (r *firewallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state firewallModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := hrobot.ServerID(state.ServerNumber.ValueInt64())
	if err := r.client.Firewall.Delete(ctx, id); err != nil {
		resp.Diagnostics.AddError("hrobot firewall delete failed", err.Error())
		return
	}
	if err := r.client.Firewall.WaitForFirewallReady(ctx, id); err != nil {
		resp.Diagnostics.AddError("hrobot firewall did not settle after delete", err.Error())
	}
}

func (r *firewallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	n, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("invalid import id", "expected the server number as a positive integer")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_number"), n)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.FormatInt(n, 10))...)
}

// apply pushes the planned config to the API and writes the response back to state.
func (r *firewallResource) apply(ctx context.Context, plan *firewallModel, diags *diag.Diagnostics, state *tfsdk.State) {
	id := hrobot.ServerID(plan.ServerNumber.ValueInt64())

	// status is Required and filter_ipv6/whitelist_hos are Computed with static
	// defaults, so all three always hold a known value here. UpdateConfig takes
	// pointers to distinguish "unset" from zero, so take addresses of locals.
	status := hrobot.FirewallStatus(plan.Status.ValueString())
	filterIPv6 := plan.FilterIPv6.ValueBool()
	whitelistHOS := plan.WhitelistHOS.ValueBool()

	cfg := hrobot.UpdateConfig{
		Status:       &status,
		FilterIPv6:   &filterIPv6,
		WhitelistHOS: &whitelistHOS,
		Rules: hrobot.FirewallRules{
			Input:  toAPIRules(plan.InputRules),
			Output: toAPIRules(plan.OutputRules),
		},
	}

	if _, err := r.client.Firewall.Update(ctx, id, cfg); err != nil {
		diags.AddError("hrobot firewall update failed", err.Error())
		return
	}
	if err := r.client.Firewall.WaitForFirewallReady(ctx, id); err != nil {
		diags.AddError("hrobot firewall did not settle after update", err.Error())
		return
	}

	// Re-fetch after the firewall settles so state reflects API normalization.
	final, err := r.client.Firewall.Get(ctx, id)
	if err != nil {
		diags.AddError("hrobot firewall re-read failed", err.Error())
		return
	}

	setFirewallModel(plan, final)
	diags.Append(state.Set(ctx, plan)...)
}

func setFirewallModel(m *firewallModel, cfg *hrobot.FirewallConfig) {
	m.ID = types.StringValue(strconv.Itoa(cfg.ServerNumber))
	m.ServerNumber = types.Int64Value(int64(cfg.ServerNumber))
	m.Status = types.StringValue(string(cfg.Status))
	m.FilterIPv6 = types.BoolValue(cfg.FilterIPv6)
	m.WhitelistHOS = types.BoolValue(cfg.WhitelistHOS)
	m.Port = types.StringValue(cfg.Port)
	m.InputRules = fromAPIRules(cfg.Rules.Input)
	m.OutputRules = fromAPIRules(cfg.Rules.Output)
}

func toAPIRules(in []ruleModel) []hrobot.FirewallRule {
	if len(in) == 0 {
		return nil
	}
	out := make([]hrobot.FirewallRule, len(in))
	for i, r := range in {
		out[i] = hrobot.FirewallRule{
			Name:       r.Name.ValueString(),
			IPVersion:  hrobot.IPVersion(r.IPVersion.ValueString()),
			Action:     hrobot.Action(r.Action.ValueString()),
			Protocol:   hrobot.Protocol(r.Protocol.ValueString()),
			SourceIP:   r.SourceIP.ValueString(),
			DestIP:     r.DestIP.ValueString(),
			SourcePort: r.SourcePort.ValueString(),
			DestPort:   r.DestPort.ValueString(),
			TCPFlags:   r.TCPFlags.ValueString(),
		}
	}
	return out
}

func fromAPIRules(in []hrobot.FirewallRule) []ruleModel {
	if len(in) == 0 {
		return nil
	}
	out := make([]ruleModel, len(in))
	for i, r := range in {
		out[i] = ruleModel{
			Name:       optString(r.Name),
			IPVersion:  optString(string(r.IPVersion)),
			Action:     types.StringValue(string(r.Action)),
			Protocol:   optString(string(r.Protocol)),
			SourceIP:   optString(r.SourceIP),
			DestIP:     optString(r.DestIP),
			SourcePort: optString(r.SourcePort),
			DestPort:   optString(r.DestPort),
			TCPFlags:   optString(r.TCPFlags),
		}
	}
	return out
}

func optString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
