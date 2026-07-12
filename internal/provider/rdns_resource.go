package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

var (
	_ resource.Resource                = (*rdnsResource)(nil)
	_ resource.ResourceWithConfigure   = (*rdnsResource)(nil)
	_ resource.ResourceWithImportState = (*rdnsResource)(nil)
)

// NewRDNSResource returns the hrobot_rdns resource.
func NewRDNSResource() resource.Resource {
	return &rdnsResource{}
}

type rdnsResource struct {
	client *hrobot.Client
}

type rdnsResourceModel struct {
	ID  types.String `tfsdk:"id"`
	IP  types.String `tfsdk:"ip"`
	PTR types.String `tfsdk:"ptr"`
}

func (r *rdnsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rdns"
}

func (r *rdnsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the reverse DNS (PTR) entry for a single IP address.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The IP address (mirrors `ip`).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ip": schema.StringAttribute{
				Required:      true,
				Description:   "The IP address to set the PTR record for. Changing it replaces the resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ptr": schema.StringAttribute{
				Required:    true,
				Description: "The PTR record (reverse DNS hostname).",
			},
		},
	}
}

func (r *rdnsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *rdnsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan rdnsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	entry, err := r.client.RDNS.Create(ctx, plan.IP.ValueString(), plan.PTR.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("hrobot rdns create failed", err.Error())
		return
	}
	setRDNSResourceModel(&plan, entry)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *rdnsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state rdnsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	entry, err := r.client.RDNS.Get(ctx, state.IP.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("hrobot rdns read failed", err.Error())
		return
	}
	setRDNSResourceModel(&state, entry)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *rdnsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan rdnsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	entry, err := r.client.RDNS.Update(ctx, plan.IP.ValueString(), plan.PTR.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("hrobot rdns update failed", err.Error())
		return
	}
	setRDNSResourceModel(&plan, entry)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *rdnsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state rdnsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RDNS.Delete(ctx, state.IP.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("hrobot rdns delete failed", err.Error())
	}
}

func (r *rdnsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ip"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func setRDNSResourceModel(m *rdnsResourceModel, e *hrobot.RDNS) {
	m.ID = types.StringValue(e.IP)
	m.IP = types.StringValue(e.IP)
	m.PTR = types.StringValue(e.PTR)
}
