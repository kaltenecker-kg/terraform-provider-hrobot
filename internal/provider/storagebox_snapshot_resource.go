package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ resource.Resource                = (*storageBoxSnapshotResource)(nil)
	_ resource.ResourceWithConfigure   = (*storageBoxSnapshotResource)(nil)
	_ resource.ResourceWithImportState = (*storageBoxSnapshotResource)(nil)
)

// NewStorageBoxSnapshotResource returns the hrobot_storagebox_snapshot resource.
func NewStorageBoxSnapshotResource() resource.Resource {
	return &storageBoxSnapshotResource{}
}

type storageBoxSnapshotResource struct {
	client *hrobot.Client
}

type storageBoxSnapshotResourceModel struct {
	ID             types.String `tfsdk:"id"`
	StorageBoxID   types.Int64  `tfsdk:"storagebox_id"`
	Name           types.String `tfsdk:"name"`
	Comment        types.String `tfsdk:"comment"`
	Timestamp      types.String `tfsdk:"timestamp"`
	Size           types.Int64  `tfsdk:"size"`
	FilesystemSize types.Int64  `tfsdk:"filesystem_size"`
	Automatic      types.Bool   `tfsdk:"automatic"`
}

func (r *storageBoxSnapshotResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storagebox_snapshot"
}

func (r *storageBoxSnapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a manual snapshot of a Hetzner Storage Box. Creating the resource takes a snapshot immediately; the snapshot name is assigned by Hetzner. (Snapshot restore is a one-shot action and is not modeled.)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Composite identifier `<storagebox_id>/<name>`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"storagebox_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the Storage Box to snapshot. Changing it replaces the resource.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Computed:      true,
				Description:   "Snapshot name, assigned by Hetzner.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"comment":         schema.StringAttribute{Optional: true, Description: "Free-form comment on the snapshot."},
			"timestamp":       schema.StringAttribute{Computed: true, Description: "Creation timestamp.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"size":            schema.Int64Attribute{Computed: true, Description: "Snapshot size (bytes)."},
			"filesystem_size": schema.Int64Attribute{Computed: true, Description: "Filesystem size at snapshot time (bytes)."},
			"automatic":       schema.BoolAttribute{Computed: true, Description: "Whether the snapshot was created automatically by a snapshot plan (always false for this resource)."},
		},
	}
}

func (r *storageBoxSnapshotResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *storageBoxSnapshotResource) find(ctx context.Context, boxID int, name string) (*hrobot.StorageBoxSnapshot, error) {
	list, err := r.client.StorageBox.ListSnapshots(ctx, boxID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].Name == name {
			return &list[i], nil
		}
	}
	return nil, nil
}

func (r *storageBoxSnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan storageBoxSnapshotResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	boxID := int(plan.StorageBoxID.ValueInt64())

	snap, err := r.client.StorageBox.CreateSnapshot(ctx, boxID)
	if err != nil {
		resp.Diagnostics.AddError("hrobot storagebox snapshot create failed", err.Error())
		return
	}
	if c := strPtr(plan.Comment); c != nil {
		if err := r.client.StorageBox.SetSnapshotComment(ctx, boxID, snap.Name, *c); err != nil {
			resp.Diagnostics.AddError("hrobot storagebox snapshot comment failed", err.Error())
			return
		}
	}
	if err := r.reread(ctx, boxID, snap.Name, &plan); err != nil {
		resp.Diagnostics.AddError("hrobot storagebox snapshot re-read failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageBoxSnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state storageBoxSnapshotResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	boxID := int(state.StorageBoxID.ValueInt64())
	snap, err := r.find(ctx, boxID, state.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("hrobot storagebox snapshot read failed", err.Error())
		return
	}
	if snap == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	setSnapshotResourceModel(&state, boxID, snap)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *storageBoxSnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state storageBoxSnapshotResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	boxID := int(state.StorageBoxID.ValueInt64())
	name := state.Name.ValueString()

	comment := ""
	if c := strPtr(plan.Comment); c != nil {
		comment = *c
	}
	if err := r.client.StorageBox.SetSnapshotComment(ctx, boxID, name, comment); err != nil {
		resp.Diagnostics.AddError("hrobot storagebox snapshot comment failed", err.Error())
		return
	}
	if err := r.reread(ctx, boxID, name, &plan); err != nil {
		resp.Diagnostics.AddError("hrobot storagebox snapshot re-read failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageBoxSnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state storageBoxSnapshotResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.StorageBox.DeleteSnapshot(ctx, int(state.StorageBoxID.ValueInt64()), state.Name.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("hrobot storagebox snapshot delete failed", err.Error())
	}
}

func (r *storageBoxSnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	boxID, name, ok := splitStorageBoxChildID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("invalid import id", "expected `<storagebox_id>/<snapshot_name>`")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("storagebox_id"), boxID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *storageBoxSnapshotResource) reread(ctx context.Context, boxID int, name string, m *storageBoxSnapshotResourceModel) error {
	snap, err := r.find(ctx, boxID, name)
	if err != nil {
		return err
	}
	if snap == nil {
		return fmt.Errorf("snapshot %q not found on storage box %d", name, boxID)
	}
	setSnapshotResourceModel(m, boxID, snap)
	return nil
}

func setSnapshotResourceModel(m *storageBoxSnapshotResourceModel, boxID int, s *hrobot.StorageBoxSnapshot) {
	m.ID = types.StringValue(fmt.Sprintf("%d/%s", boxID, s.Name))
	m.StorageBoxID = types.Int64Value(int64(boxID))
	m.Name = types.StringValue(s.Name)
	m.Comment = optString(s.Comment)
	m.Timestamp = types.StringValue(s.Timestamp)
	m.Size = types.Int64Value(int64(s.Size))
	m.FilesystemSize = types.Int64Value(int64(s.FilesystemSize))
	m.Automatic = types.BoolValue(s.Automatic)
}
