package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
		"name": schema.StringAttribute{Optional: true, Description: "Rule name."},
		"ip_version": schema.StringAttribute{
			Optional:    true,
			Description: "ipv4 or ipv6. Required for input rules; leave empty for output rules to apply to both.",
			Validators:  []validator.String{stringvalidator.OneOf(string(hrobot.IPv4), string(hrobot.IPv6))},
		},
		"action": schema.StringAttribute{
			Required:    true,
			Description: "accept or discard.",
			Validators:  []validator.String{stringvalidator.OneOf(string(hrobot.ActionAccept), string(hrobot.ActionDiscard))},
		},
		"protocol":  schema.StringAttribute{Optional: true, Description: "tcp, udp, icmp, gre, ipip, ah, esp, ipencap."},
		"src_ip":    schema.StringAttribute{Optional: true, Description: "Source CIDR."},
		"dst_ip":    schema.StringAttribute{Optional: true, Description: "Destination CIDR."},
		"src_port":  schema.StringAttribute{Optional: true, Description: "Source port or range (e.g. 1024-65535)."},
		"dst_port":  schema.StringAttribute{Optional: true, Description: "Destination port or range."},
		"tcp_flags": schema.StringAttribute{Optional: true, Description: "TCP flags expression (e.g. syn|fin)."},
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
				Description: "Server number to attach the firewall configuration to. Changing it replaces the resource, which resets the firewall of the previous server.",
				// In-place would configure the new server and leave the old
				// server's rules behind, no longer tracked by Terraform.
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				Required:    true,
				Description: "active or disabled.",
				Validators:  []validator.String{stringvalidator.OneOf(string(hrobot.FirewallStatusActive), string(hrobot.FirewallStatusDisabled))},
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
				Validators:   []validator.List{listvalidator.SizeAtMost(10)},
			},
			"output_rule": schema.ListNestedAttribute{
				Optional:     true,
				Description:  "Outbound rules, evaluated top to bottom. Hetzner may store rules without `ip_version` expanded into separate ipv4 and ipv6 entries; the provider treats that expanded form as equal to the configured rules. After an import the provider cannot tell which form was configured, so the first plan may show an in-place update that re-applies the equivalent rules.",
				NestedObject: schema.NestedAttributeObject{Attributes: ruleAttrs},
			},
		},
	}
}

func (r *firewallResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *firewallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan firewallModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.Diagnostics, &resp.State, resp.Identity)
}

func (r *firewallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state firewallModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.Firewall.Get(ctx, hrobot.ServerID(state.ServerNumber.ValueInt64()))
	if err != nil {
		// A cancelled server takes its firewall with it.
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("hrobot firewall read failed", err.Error())
		return
	}
	inputRules, outputRules := reconcileRules(&state, cfg.Rules)
	setFirewallModel(&state, cfg)
	state.InputRules = inputRules
	state.OutputRules = outputRules
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	setIdentity(ctx, resp.Identity, firewallIdentity{ServerNumber: state.ServerNumber}, &resp.Diagnostics)
}

func (r *firewallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan firewallModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.Diagnostics, &resp.State, resp.Identity)
}

func (r *firewallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state firewallModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := hrobot.ServerID(state.ServerNumber.ValueInt64())
	if err := r.client.Firewall.Delete(ctx, id); err != nil {
		// The server (and with it the firewall) is already gone.
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("hrobot firewall delete failed", err.Error())
		return
	}
	if err := r.client.Firewall.WaitForFirewallReady(ctx, id); err != nil {
		resp.Diagnostics.AddError("hrobot firewall did not settle after delete", err.Error())
	}
}

func (r *firewallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by identity block: the identity carries the typed key.
	if req.ID == "" {
		var id firewallIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &id)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_number"), id.ServerNumber)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"),
			types.StringValue(strconv.FormatInt(id.ServerNumber.ValueInt64(), 10)))...)
		return
	}
	n, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("invalid import id", "expected the server number as a positive integer")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_number"), n)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.FormatInt(n, 10))...)
}

// apply pushes the planned config to the API and writes the response back to state.
func (r *firewallResource) apply(ctx context.Context, plan *firewallModel, diags *diag.Diagnostics, state *tfsdk.State, identity *tfsdk.ResourceIdentity) {
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

	// Re-fetch after the firewall settles, but only for computed attributes.
	// input_rule/output_rule (and the other config attributes) must keep
	// their planned values in state: they are not Computed, so Terraform
	// requires the applied value to match the plan exactly, while the API
	// may return a normalized ruleset (version-less rules expanded into
	// separate ipv4/ipv6 entries).
	final, err := r.client.Firewall.Get(ctx, id)
	if err != nil {
		diags.AddError("hrobot firewall re-read failed", err.Error())
		return
	}
	if !final.Rules.Equivalent(cfg.Rules) {
		diags.AddWarning(
			"hrobot firewall rules diverged after apply",
			"the api reports a ruleset that differs materially from the applied configuration; the next plan will show it as drift",
		)
	}

	plan.ID = types.StringValue(strconv.Itoa(final.ServerNumber))
	plan.Port = types.StringValue(final.Port)
	diags.Append(state.Set(ctx, plan)...)
	setIdentity(ctx, identity, firewallIdentity{ServerNumber: plan.ServerNumber}, diags)
}

// reconcileRules returns the rule lists to store in state after a refresh:
// the prior state's lists when the live ruleset is merely a normalized form
// of them (the API may expand version-less rules into separate ipv4/ipv6
// entries), or the live lists when they differ materially.
func reconcileRules(prior *firewallModel, live hrobot.FirewallRules) (input, output []ruleModel) {
	priorRules := hrobot.FirewallRules{
		Input:  toAPIRules(prior.InputRules),
		Output: toAPIRules(prior.OutputRules),
	}
	if live.Equivalent(priorRules) {
		return prior.InputRules, prior.OutputRules
	}
	return fromAPIRules(live.Input), fromAPIRules(live.Output)
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
