package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ resource.Resource                = (*storageBoxSubAccountResource)(nil)
	_ resource.ResourceWithConfigure   = (*storageBoxSubAccountResource)(nil)
	_ resource.ResourceWithImportState = (*storageBoxSubAccountResource)(nil)
)

// NewStorageBoxSubAccountResource returns the hrobot_storagebox_subaccount resource.
func NewStorageBoxSubAccountResource() resource.Resource {
	return &storageBoxSubAccountResource{}
}

type storageBoxSubAccountResource struct {
	client *hrobot.Client
}

type storageBoxSubAccountResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	StorageBoxID         types.Int64  `tfsdk:"storagebox_id"`
	Username             types.String `tfsdk:"username"`
	Password             types.String `tfsdk:"password"`
	HomeDirectory        types.String `tfsdk:"home_directory"`
	Samba                types.Bool   `tfsdk:"samba"`
	SSH                  types.Bool   `tfsdk:"ssh"`
	ExternalReachability types.Bool   `tfsdk:"external_reachability"`
	Webdav               types.Bool   `tfsdk:"webdav"`
	ReadOnly             types.Bool   `tfsdk:"readonly"`
	Comment              types.String `tfsdk:"comment"`
	AccountID            types.String `tfsdk:"account_id"`
	Server               types.String `tfsdk:"server"`
	CreateTime           types.String `tfsdk:"create_time"`
}

func (r *storageBoxSubAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storagebox_subaccount"
}

func (r *storageBoxSubAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computedBool := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Description: desc, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	resp.Schema = schema.Schema{
		Description: "Manages a sub-account on a Hetzner Storage Box. The Storage Box itself is ordered outside Terraform and referenced by `storagebox_id`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Composite identifier `<storagebox_id>/<username>`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"storagebox_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the Storage Box that owns the sub-account. Changing it replaces the resource.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"username": schema.StringAttribute{
				Computed:      true,
				Description:   "Sub-account username, assigned by Hetzner.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"password": schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "Auto-generated password, returned only at creation.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"home_directory": schema.StringAttribute{
				Required:    true,
				Description: "Home directory of the sub-account, relative to the Storage Box root.",
			},
			"samba":                 computedBool("Enable Samba/CIFS access."),
			"ssh":                   computedBool("Enable SSH access."),
			"external_reachability": computedBool("Allow access from outside the Hetzner network."),
			"webdav":                computedBool("Enable WebDAV access."),
			"readonly":              computedBool("Make the sub-account read-only."),
			"comment":               schema.StringAttribute{Optional: true, Description: "Free-form comment."},
			"account_id":            schema.StringAttribute{Computed: true, Description: "Sub-account ID.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"server":                schema.StringAttribute{Computed: true, Description: "Hostname clients connect to.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"create_time":           schema.StringAttribute{Computed: true, Description: "Creation timestamp.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *storageBoxSubAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func subAccountInput(m *storageBoxSubAccountResourceModel) hrobot.StorageBoxSubAccountInput {
	return hrobot.StorageBoxSubAccountInput{
		HomeDirectory:        strPtr(m.HomeDirectory),
		Samba:                boolPtr(m.Samba),
		SSH:                  boolPtr(m.SSH),
		ExternalReachability: boolPtr(m.ExternalReachability),
		Webdav:               boolPtr(m.Webdav),
		ReadOnly:             boolPtr(m.ReadOnly),
		Comment:              strPtr(m.Comment),
	}
}

func (r *storageBoxSubAccountResource) find(ctx context.Context, boxID int, username string) (*hrobot.StorageBoxSubAccount, error) {
	list, err := r.client.StorageBox.ListSubAccounts(ctx, boxID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].Username == username {
			return &list[i], nil
		}
	}
	return nil, nil
}

func (r *storageBoxSubAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan storageBoxSubAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	boxID := int(plan.StorageBoxID.ValueInt64())

	created, err := r.client.StorageBox.CreateSubAccount(ctx, boxID, subAccountInput(&plan))
	if err != nil {
		resp.Diagnostics.AddError("hrobot storagebox subaccount create failed", err.Error())
		return
	}
	sub, err := r.find(ctx, boxID, created.Username)
	if err != nil {
		resp.Diagnostics.AddError("hrobot storagebox subaccount re-read failed", err.Error())
		return
	}
	if sub == nil {
		resp.Diagnostics.AddError("hrobot storagebox subaccount not found after create", "the created sub-account was not returned by the list endpoint")
		return
	}
	setSubAccountResourceModel(&plan, boxID, sub)
	plan.Password = types.StringValue(created.Password)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageBoxSubAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state storageBoxSubAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	boxID := int(state.StorageBoxID.ValueInt64())
	sub, err := r.find(ctx, boxID, state.Username.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("hrobot storagebox subaccount read failed", err.Error())
		return
	}
	if sub == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	// Password is only returned at creation; preserve the stored value.
	setSubAccountResourceModel(&state, boxID, sub)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *storageBoxSubAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state storageBoxSubAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	boxID := int(state.StorageBoxID.ValueInt64())
	username := state.Username.ValueString()

	if err := r.client.StorageBox.UpdateSubAccount(ctx, boxID, username, subAccountInput(&plan)); err != nil {
		resp.Diagnostics.AddError("hrobot storagebox subaccount update failed", err.Error())
		return
	}
	sub, err := r.find(ctx, boxID, username)
	if err != nil || sub == nil {
		resp.Diagnostics.AddError("hrobot storagebox subaccount re-read failed", fmt.Sprintf("%v", err))
		return
	}
	setSubAccountResourceModel(&plan, boxID, sub)
	plan.Password = state.Password
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageBoxSubAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state storageBoxSubAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.StorageBox.DeleteSubAccount(ctx, int(state.StorageBoxID.ValueInt64()), state.Username.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("hrobot storagebox subaccount delete failed", err.Error())
	}
}

func (r *storageBoxSubAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	boxID, username, ok := splitStorageBoxChildID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("invalid import id", "expected `<storagebox_id>/<username>`")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("storagebox_id"), boxID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), username)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func setSubAccountResourceModel(m *storageBoxSubAccountResourceModel, boxID int, s *hrobot.StorageBoxSubAccount) {
	m.ID = types.StringValue(fmt.Sprintf("%d/%s", boxID, s.Username))
	m.StorageBoxID = types.Int64Value(int64(boxID))
	m.Username = types.StringValue(s.Username)
	m.HomeDirectory = types.StringValue(s.HomeDirectory)
	m.Samba = types.BoolValue(s.Samba)
	m.SSH = types.BoolValue(s.SSH)
	m.ExternalReachability = types.BoolValue(s.ExternalReachability)
	m.Webdav = types.BoolValue(s.Webdav)
	m.ReadOnly = types.BoolValue(s.ReadOnly)
	m.Comment = optString(s.Comment)
	m.AccountID = types.StringValue(s.AccountID)
	m.Server = types.StringValue(s.Server)
	m.CreateTime = types.StringValue(s.CreateTime)
}

// splitStorageBoxChildID parses a `<storagebox_id>/<name>` composite import ID.
func splitStorageBoxChildID(id string) (boxID int64, name string, ok bool) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, "", false
	}
	n, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", false
	}
	return n, parts[1], true
}
