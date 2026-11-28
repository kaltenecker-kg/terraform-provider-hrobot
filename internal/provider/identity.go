package provider

// Resource identity (Terraform 1.12+, OpenTofu 1.12+): every resource
// reports the attributes that identify its remote object, so it can be
// imported with an `import { identity = { ... } }` block instead of a
// composite string ID. Import by string ID keeps working, and older clients
// that predate identity never request it.
//
// An identity must never change once set (the framework rejects that), so
// each one consists only of attributes that force replacement or are
// assigned by Hetzner at creation.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.ResourceWithIdentity = (*sshKeyResource)(nil)
	_ resource.ResourceWithIdentity = (*rdnsResource)(nil)
	_ resource.ResourceWithIdentity = (*failoverIPResource)(nil)
	_ resource.ResourceWithIdentity = (*firewallResource)(nil)
	_ resource.ResourceWithIdentity = (*vswitchResource)(nil)
	_ resource.ResourceWithIdentity = (*storageBoxSubAccountResource)(nil)
	_ resource.ResourceWithIdentity = (*storageBoxSnapshotResource)(nil)
	_ resource.ResourceWithIdentity = (*storageBoxSnapshotPlanResource)(nil)
)

// setIdentity stores v as the resource identity. The response identity is
// nil only when the resource has no identity schema; the guard keeps callers
// simple.
func setIdentity(ctx context.Context, id *tfsdk.ResourceIdentity, v any, diags *diag.Diagnostics) {
	if id == nil || diags.HasError() {
		return
	}
	diags.Append(id.Set(ctx, v)...)
}

func identityString(desc string) identityschema.StringAttribute {
	return identityschema.StringAttribute{RequiredForImport: true, Description: desc}
}

func identityInt64(desc string) identityschema.Int64Attribute {
	return identityschema.Int64Attribute{RequiredForImport: true, Description: desc}
}

const storageBoxIDDesc = "ID of the Storage Box."

// --- hrobot_ssh_key ---

type sshKeyIdentity struct {
	Fingerprint types.String `tfsdk:"fingerprint"`
}

func (r *sshKeyResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"fingerprint": identityString("Key fingerprint."),
	}}
}

func sshKeyIdentityOf(m *sshKeyResourceModel) sshKeyIdentity {
	return sshKeyIdentity{Fingerprint: m.Fingerprint}
}

// --- hrobot_rdns and hrobot_failover_ip ---

type ipIdentity struct {
	IP types.String `tfsdk:"ip"`
}

func (r *rdnsResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"ip": identityString("The IP address the PTR record belongs to."),
	}}
}

func (r *failoverIPResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"ip": identityString("The failover IP."),
	}}
}

// --- hrobot_firewall ---

type firewallIdentity struct {
	ServerNumber types.Int64 `tfsdk:"server_number"`
}

func (r *firewallResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"server_number": identityInt64("Number of the server the firewall belongs to."),
	}}
}

// --- hrobot_vswitch ---

type vswitchIdentity struct {
	ID types.Int64 `tfsdk:"id"`
}

func (r *vswitchResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityInt64("vSwitch ID."),
	}}
}

// --- hrobot_storagebox_subaccount ---

type subAccountIdentity struct {
	StorageBoxID types.Int64  `tfsdk:"storagebox_id"`
	Username     types.String `tfsdk:"username"`
}

func (r *storageBoxSubAccountResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"storagebox_id": identityInt64(storageBoxIDDesc),
		"username":      identityString("Sub-account username."),
	}}
}

func subAccountIdentityOf(m *storageBoxSubAccountResourceModel) subAccountIdentity {
	return subAccountIdentity{StorageBoxID: m.StorageBoxID, Username: m.Username}
}

// --- hrobot_storagebox_snapshot ---

type snapshotIdentity struct {
	StorageBoxID types.Int64  `tfsdk:"storagebox_id"`
	Name         types.String `tfsdk:"name"`
}

func (r *storageBoxSnapshotResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"storagebox_id": identityInt64(storageBoxIDDesc),
		"name":          identityString("Snapshot name."),
	}}
}

func snapshotIdentityOf(m *storageBoxSnapshotResourceModel) snapshotIdentity {
	return snapshotIdentity{StorageBoxID: m.StorageBoxID, Name: m.Name}
}

// --- hrobot_storagebox_snapshot_plan ---

type snapshotPlanIdentity struct {
	StorageBoxID types.Int64 `tfsdk:"storagebox_id"`
}

func (r *storageBoxSnapshotPlanResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"storagebox_id": identityInt64(storageBoxIDDesc),
	}}
}
