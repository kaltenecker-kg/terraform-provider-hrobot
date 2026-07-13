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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ resource.Resource                = (*storageBoxSnapshotPlanResource)(nil)
	_ resource.ResourceWithConfigure   = (*storageBoxSnapshotPlanResource)(nil)
	_ resource.ResourceWithImportState = (*storageBoxSnapshotPlanResource)(nil)
)

// NewStorageBoxSnapshotPlanResource returns the hrobot_storagebox_snapshot_plan resource.
func NewStorageBoxSnapshotPlanResource() resource.Resource {
	return &storageBoxSnapshotPlanResource{}
}

type storageBoxSnapshotPlanResource struct {
	client *hrobot.Client
}

type storageBoxSnapshotPlanResourceModel struct {
	ID           types.String `tfsdk:"id"`
	StorageBoxID types.Int64  `tfsdk:"storagebox_id"`
	Status       types.String `tfsdk:"status"`
	Minute       types.Int64  `tfsdk:"minute"`
	Hour         types.Int64  `tfsdk:"hour"`
	DayOfWeek    types.Int64  `tfsdk:"day_of_week"`
	DayOfMonth   types.Int64  `tfsdk:"day_of_month"`
	Month        types.Int64  `tfsdk:"month"`
	MaxSnapshots types.Int64  `tfsdk:"max_snapshots"`
}

func (r *storageBoxSnapshotPlanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storagebox_snapshot_plan"
}

func (r *storageBoxSnapshotPlanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the automatic snapshot plan of a Hetzner Storage Box. There is one plan per box; destroying the resource disables it. Schedule fields left unset act as wildcards.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Storage Box ID (mirrors `storagebox_id`).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"storagebox_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the Storage Box. Changing it replaces the resource.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("enabled"),
				Description: "Plan status: `enabled` or `disabled`.",
			},
			"minute":        schema.Int64Attribute{Optional: true, Description: "Minute (0-59). Unset = every minute."},
			"hour":          schema.Int64Attribute{Optional: true, Description: "Hour (0-23). Unset = every hour."},
			"day_of_week":   schema.Int64Attribute{Optional: true, Description: "Day of week (1-7). Unset = every day."},
			"day_of_month":  schema.Int64Attribute{Optional: true, Description: "Day of month (1-31). Unset = every day."},
			"month":         schema.Int64Attribute{Optional: true, Description: "Month (1-12). Unset = every month."},
			"max_snapshots": schema.Int64Attribute{Required: true, Description: "Maximum number of automatic snapshots to retain."},
		},
	}
}

func (r *storageBoxSnapshotPlanResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func planFromModel(m *storageBoxSnapshotPlanResourceModel) hrobot.StorageBoxSnapshotPlan {
	return hrobot.StorageBoxSnapshotPlan{
		Status:       m.Status.ValueString(),
		Minute:       intPtr(m.Minute),
		Hour:         intPtr(m.Hour),
		DayOfWeek:    intPtr(m.DayOfWeek),
		DayOfMonth:   intPtr(m.DayOfMonth),
		Month:        intPtr(m.Month),
		MaxSnapshots: int(m.MaxSnapshots.ValueInt64()),
	}
}

func (r *storageBoxSnapshotPlanResource) apply(ctx context.Context, plan *storageBoxSnapshotPlanResourceModel, diags *diag.Diagnostics) bool {
	boxID := int(plan.StorageBoxID.ValueInt64())
	got, err := r.client.StorageBox.SetSnapshotPlan(ctx, boxID, planFromModel(plan))
	if err != nil {
		diags.AddError("hrobot storagebox snapshot plan set failed", err.Error())
		return false
	}
	setSnapshotPlanResourceModel(plan, boxID, got)
	return true
}

func (r *storageBoxSnapshotPlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan storageBoxSnapshotPlanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.apply(ctx, &plan, &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageBoxSnapshotPlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state storageBoxSnapshotPlanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	boxID := int(state.StorageBoxID.ValueInt64())
	got, err := r.client.StorageBox.GetSnapshotPlan(ctx, boxID)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("hrobot storagebox snapshot plan read failed", err.Error())
		return
	}
	setSnapshotPlanResourceModel(&state, boxID, got)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *storageBoxSnapshotPlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan storageBoxSnapshotPlanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.apply(ctx, &plan, &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageBoxSnapshotPlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state storageBoxSnapshotPlanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// A snapshot plan cannot be deleted, only disabled.
	state.Status = types.StringValue("disabled")
	if _, err := r.client.StorageBox.SetSnapshotPlan(ctx, int(state.StorageBoxID.ValueInt64()), planFromModel(&state)); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("hrobot storagebox snapshot plan disable failed", err.Error())
	}
}

func (r *storageBoxSnapshotPlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	n, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("invalid import id", "expected the Storage Box ID as an integer")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("storagebox_id"), n)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.FormatInt(n, 10))...)
}

func setSnapshotPlanResourceModel(m *storageBoxSnapshotPlanResourceModel, boxID int, p *hrobot.StorageBoxSnapshotPlan) {
	m.ID = types.StringValue(strconv.Itoa(boxID))
	m.StorageBoxID = types.Int64Value(int64(boxID))
	m.Status = types.StringValue(p.Status)
	m.Minute = int64FromPtr(p.Minute)
	m.Hour = int64FromPtr(p.Hour)
	m.DayOfWeek = int64FromPtr(p.DayOfWeek)
	m.DayOfMonth = int64FromPtr(p.DayOfMonth)
	m.Month = int64FromPtr(p.Month)
	m.MaxSnapshots = types.Int64Value(int64(p.MaxSnapshots))
}
