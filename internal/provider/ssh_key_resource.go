package provider

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

var (
	_ resource.Resource                = (*sshKeyResource)(nil)
	_ resource.ResourceWithConfigure   = (*sshKeyResource)(nil)
	_ resource.ResourceWithImportState = (*sshKeyResource)(nil)
)

// NewSSHKeyResource returns the hrobot_ssh_key resource.
func NewSSHKeyResource() resource.Resource {
	return &sshKeyResource{}
}

type sshKeyResource struct {
	client *hrobot.Client
}

type sshKeyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	Name        types.String `tfsdk:"name"`
	PublicKey   types.String `tfsdk:"public_key"`
	Type        types.String `tfsdk:"type"`
	Size        types.Int64  `tfsdk:"size"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func (r *sshKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func (r *sshKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an SSH key in the Hetzner Robot account. The key can be referenced when activating rescue or Linux installation boot configs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Key fingerprint (mirrors `fingerprint`).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"fingerprint": schema.StringAttribute{
				Computed:      true,
				Description:   "Fingerprint assigned by Hetzner; the identifier used for import.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Key name. Can be changed in place.",
			},
			"public_key": schema.StringAttribute{
				Required:    true,
				Description: "The public key material (e.g. `ssh-ed25519 AAAA...`). Changing it replaces the key; a change in surrounding whitespace only (such as the trailing newline from `file()`) does not.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIf(
					publicKeyMaterialChanged,
					"Replace the key when its material changes, ignoring surrounding whitespace.",
					"Replace the key when its material changes, ignoring surrounding whitespace.",
				)},
			},
			"type":       schema.StringAttribute{Computed: true, Description: "Key algorithm (e.g. ED25519, RSA)."},
			"size":       schema.Int64Attribute{Computed: true, Description: "Key size in bits."},
			"created_at": schema.StringAttribute{Computed: true, Description: "Creation timestamp (RFC 3339)."},
		},
	}
}

func (r *sshKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *sshKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan sshKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	key, err := r.client.Key.Create(ctx, plan.Name.ValueString(), plan.PublicKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("hrobot ssh key create failed", err.Error())
		return
	}
	setSSHKeyResourceModel(&plan, key)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	setIdentity(ctx, resp.Identity, sshKeyIdentityOf(&plan), &resp.Diagnostics)
}

func (r *sshKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state sshKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	key, err := r.client.Key.Get(ctx, state.Fingerprint.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("hrobot ssh key read failed", err.Error())
		return
	}
	setSSHKeyResourceModel(&state, key)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	setIdentity(ctx, resp.Identity, sshKeyIdentityOf(&state), &resp.Diagnostics)
}

func (r *sshKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state sshKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Only the name can change in place; a whitespace-only public_key change
	// (e.g. after an import) needs no API call beyond the re-read.
	var key *hrobot.SSHKey
	var err error
	if plan.Name.Equal(state.Name) {
		key, err = r.client.Key.Get(ctx, state.Fingerprint.ValueString())
	} else {
		key, err = r.client.Key.Rename(ctx, state.Fingerprint.ValueString(), plan.Name.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("hrobot ssh key update failed", err.Error())
		return
	}
	setSSHKeyResourceModel(&plan, key)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	setIdentity(ctx, resp.Identity, sshKeyIdentityOf(&plan), &resp.Diagnostics)
}

func (r *sshKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state sshKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Key.Delete(ctx, state.Fingerprint.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("hrobot ssh key delete failed", err.Error())
	}
}

func (r *sshKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by identity block: the identity carries the typed key.
	if req.ID == "" {
		var id sshKeyIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &id)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("fingerprint"), id.Fingerprint)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id.Fingerprint)...)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("fingerprint"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func setSSHKeyResourceModel(m *sshKeyResourceModel, k *hrobot.SSHKey) {
	m.ID = types.StringValue(k.Fingerprint)
	m.Fingerprint = types.StringValue(k.Fingerprint)
	m.Name = types.StringValue(k.Name)
	// Keep the configured text when it differs from the stored key only in
	// surrounding whitespace: `file("id.pub")` carries a trailing newline
	// that the API drops, and returning the trimmed form would fail the apply
	// as inconsistent (and then force a replacement on every plan).
	if m.PublicKey.IsNull() || m.PublicKey.IsUnknown() ||
		strings.TrimSpace(m.PublicKey.ValueString()) != strings.TrimSpace(k.Data) {
		m.PublicKey = types.StringValue(strings.TrimSpace(k.Data))
	}
	m.Type = types.StringValue(k.Type)
	m.Size = types.Int64Value(int64(k.Size))
	m.CreatedAt = types.StringValue(k.CreatedAt.Format(time.RFC3339))
}

// publicKeyMaterialChanged requires replacement only when the key itself
// changes. The API stores the key trimmed, so after an import the state lacks
// the trailing newline that `file("id.pub")` yields; that alone must not
// destroy and re-create the key.
func publicKeyMaterialChanged(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = strings.TrimSpace(req.StateValue.ValueString()) != strings.TrimSpace(req.PlanValue.ValueString())
}
