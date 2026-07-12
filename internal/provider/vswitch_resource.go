package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ resource.Resource                = (*vswitchResource)(nil)
	_ resource.ResourceWithConfigure   = (*vswitchResource)(nil)
	_ resource.ResourceWithImportState = (*vswitchResource)(nil)
)

// NewVSwitchResource returns the hrobot_vswitch resource.
func NewVSwitchResource() resource.Resource {
	return &vswitchResource{}
}

type vswitchResource struct {
	client *hrobot.Client
}

type vswitchResourceModel struct {
	ID               types.Int64  `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	VLAN             types.Int64  `tfsdk:"vlan"`
	ServerNumbers    types.Set    `tfsdk:"server_numbers"`
	CancellationDate types.String `tfsdk:"cancellation_date"`
	Cancelled        types.Bool   `tfsdk:"cancelled"`
}

func (r *vswitchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vswitch"
}

func (r *vswitchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hetzner Robot vSwitch and the set of servers attached to it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:      true,
				Description:   "vSwitch ID.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "vSwitch name.",
			},
			"vlan": schema.Int64Attribute{
				Required:      true,
				Description:   "VLAN ID (4000-4091). Changing it replaces the vSwitch.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"server_numbers": schema.SetAttribute{
				Optional:      true,
				Computed:      true,
				ElementType:   types.Int64Type,
				Description:   "Server numbers attached to the vSwitch. Omit to leave the current membership unchanged; set to `[]` to detach all.",
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"cancellation_date": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("now"),
				Description: "Cancellation date used when the vSwitch is destroyed (`now` for immediate, or `YYYY-MM-DD`).",
			},
			"cancelled": schema.BoolAttribute{Computed: true, Description: "Whether the vSwitch is cancelled."},
		},
	}
}

func (r *vswitchResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *vswitchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vswitchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vs, err := r.client.VSwitch.Create(ctx, plan.Name.ValueString(), int(plan.VLAN.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("hrobot vswitch create failed", err.Error())
		return
	}
	id := vs.ID

	desired, d := setToInt64s(ctx, plan.ServerNumbers)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(desired) > 0 {
		if err := r.client.VSwitch.AddServers(ctx, id, int64sToStrings(desired)); err != nil {
			resp.Diagnostics.AddError("hrobot vswitch add servers failed", err.Error())
			return
		}
		if err := r.client.VSwitch.WaitForVSwitchReady(ctx, id); err != nil {
			resp.Diagnostics.AddError("hrobot vswitch did not settle", err.Error())
			return
		}
	}

	r.refresh(ctx, id, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vswitchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vswitchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	vs, err := r.client.VSwitch.Get(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("hrobot vswitch read failed", err.Error())
		return
	}
	resp.Diagnostics.Append(setVSwitchResourceModel(ctx, &state, vs)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *vswitchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vswitchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := int(state.ID.ValueInt64())

	if !plan.Name.Equal(state.Name) {
		if err := r.client.VSwitch.Update(ctx, id, plan.Name.ValueString(), int(state.VLAN.ValueInt64())); err != nil {
			resp.Diagnostics.AddError("hrobot vswitch update failed", err.Error())
			return
		}
	}

	desired, d := setToInt64s(ctx, plan.ServerNumbers)
	resp.Diagnostics.Append(d...)
	current, d2 := setToInt64s(ctx, state.ServerNumbers)
	resp.Diagnostics.Append(d2...)
	if resp.Diagnostics.HasError() {
		return
	}
	toAdd, toRemove := diffInt64Sets(desired, current)
	changed := false
	if len(toAdd) > 0 {
		if err := r.client.VSwitch.AddServers(ctx, id, int64sToStrings(toAdd)); err != nil {
			resp.Diagnostics.AddError("hrobot vswitch add servers failed", err.Error())
			return
		}
		changed = true
	}
	if len(toRemove) > 0 {
		if err := r.client.VSwitch.RemoveServers(ctx, id, int64sToStrings(toRemove)); err != nil {
			resp.Diagnostics.AddError("hrobot vswitch remove servers failed", err.Error())
			return
		}
		changed = true
	}
	if changed {
		if err := r.client.VSwitch.WaitForVSwitchReady(ctx, id); err != nil {
			resp.Diagnostics.AddError("hrobot vswitch did not settle", err.Error())
			return
		}
	}

	r.refresh(ctx, id, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vswitchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vswitchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.VSwitch.Delete(ctx, int(state.ID.ValueInt64()), state.CancellationDate.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("hrobot vswitch delete failed", err.Error())
	}
}

func (r *vswitchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	n, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("invalid import id", "expected the vSwitch ID as an integer")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), n)...)
}

// refresh re-reads the vSwitch and writes it into m, preserving the plan's
// cancellation_date (which is Terraform-only and not returned by the API).
func (r *vswitchResource) refresh(ctx context.Context, id int, m *vswitchResourceModel, diags *diag.Diagnostics) {
	vs, err := r.client.VSwitch.Get(ctx, id)
	if err != nil {
		diags.AddError("hrobot vswitch re-read failed", err.Error())
		return
	}
	cancellationDate := m.CancellationDate
	diags.Append(setVSwitchResourceModel(ctx, m, vs)...)
	m.CancellationDate = cancellationDate
}

func setVSwitchResourceModel(ctx context.Context, m *vswitchResourceModel, vs *hrobot.VSwitch) diag.Diagnostics {
	var diags diag.Diagnostics
	m.ID = types.Int64Value(int64(vs.ID))
	m.Name = types.StringValue(vs.Name)
	m.VLAN = types.Int64Value(int64(vs.VLAN))
	m.Cancelled = types.BoolValue(vs.Cancelled)

	nums := make([]int64, 0, len(vs.Servers))
	for _, s := range vs.Servers {
		nums = append(nums, int64(s.ServerNumber))
	}
	set, d := types.SetValueFrom(ctx, types.Int64Type, nums)
	diags.Append(d...)
	m.ServerNumbers = set
	return diags
}

func setToInt64s(ctx context.Context, s types.Set) ([]int64, diag.Diagnostics) {
	if s.IsNull() || s.IsUnknown() {
		return nil, nil
	}
	var out []int64
	diags := s.ElementsAs(ctx, &out, false)
	return out, diags
}

func diffInt64Sets(desired, current []int64) (toAdd, toRemove []int64) {
	cur := make(map[int64]bool, len(current))
	for _, n := range current {
		cur[n] = true
	}
	des := make(map[int64]bool, len(desired))
	for _, n := range desired {
		des[n] = true
		if !cur[n] {
			toAdd = append(toAdd, n)
		}
	}
	for _, n := range current {
		if !des[n] {
			toRemove = append(toRemove, n)
		}
	}
	return toAdd, toRemove
}

func int64sToStrings(in []int64) []string {
	out := make([]string, len(in))
	for i, n := range in {
		out[i] = strconv.FormatInt(n, 10)
	}
	return out
}
