package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ resource.Resource                = (*failoverIPResource)(nil)
	_ resource.ResourceWithConfigure   = (*failoverIPResource)(nil)
	_ resource.ResourceWithImportState = (*failoverIPResource)(nil)
)

// NewFailoverIPResource returns the hrobot_failover_ip resource.
func NewFailoverIPResource() resource.Resource {
	return &failoverIPResource{}
}

type failoverIPResource struct {
	client *hrobot.Client
}

type failoverResourceModel struct {
	ID             types.String `tfsdk:"id"`
	IP             types.String `tfsdk:"ip"`
	ActiveServerIP types.String `tfsdk:"active_server_ip"`
	Netmask        types.String `tfsdk:"netmask"`
	ServerIP       types.String `tfsdk:"server_ip"`
	ServerNumber   types.Int64  `tfsdk:"server_number"`
}

func (r *failoverIPResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_failover_ip"
}

func (r *failoverIPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Routes a Hetzner failover IP to a server. The failover IP itself is ordered outside Terraform; this resource manages which server it is routed to. Destroying the resource unroutes the IP.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The failover IP (mirrors `ip`).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ip": schema.StringAttribute{
				Required:      true,
				Description:   "The failover IP to route. Changing it replaces the resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"active_server_ip": schema.StringAttribute{
				Required:    true,
				Description: "Main IP of the server the failover IP should be routed to.",
			},
			"netmask":       schema.StringAttribute{Computed: true, Description: "Netmask of the failover IP."},
			"server_ip":     schema.StringAttribute{Computed: true, Description: "Main IP of the server the failover IP belongs to."},
			"server_number": schema.Int64Attribute{Computed: true, Description: "Number of the server the failover IP belongs to."},
		},
	}
}

func (r *failoverIPResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *failoverIPResource) route(ctx context.Context, plan *failoverResourceModel, diags *diag.Diagnostics) bool {
	fo, err := r.client.Failover.Update(ctx, plan.IP.ValueString(), plan.ActiveServerIP.ValueString())
	if err != nil {
		diags.AddError("hrobot failover route failed", err.Error())
		return false
	}
	setFailoverResourceModel(plan, fo)
	return true
}

func (r *failoverIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan failoverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.route(ctx, &plan, &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *failoverIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state failoverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fo, err := r.client.Failover.Get(ctx, state.IP.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("hrobot failover read failed", err.Error())
		return
	}
	setFailoverResourceModel(&state, fo)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *failoverIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan failoverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.route(ctx, &plan, &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *failoverIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state failoverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Failover.Delete(ctx, state.IP.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("hrobot failover unroute failed", err.Error())
	}
}

func (r *failoverIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ip"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func setFailoverResourceModel(m *failoverResourceModel, f *hrobot.Failover) {
	m.ID = types.StringValue(f.IP)
	m.IP = types.StringValue(f.IP)
	if f.ActiveServerIP != nil {
		m.ActiveServerIP = types.StringValue(*f.ActiveServerIP)
	} else {
		m.ActiveServerIP = types.StringNull()
	}
	m.Netmask = types.StringValue(f.Netmask)
	m.ServerIP = types.StringValue(f.ServerIP)
	m.ServerNumber = types.Int64Value(int64(f.ServerNumber))
}
