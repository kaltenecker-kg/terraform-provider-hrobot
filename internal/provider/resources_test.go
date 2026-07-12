package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/kaltenecker-kg/hrobot-go"
)

func rsSchema(t *testing.T, r resource.Resource) rschema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %+v", resp.Diagnostics)
	}
	return resp.Schema
}

func mustSetResourceState(t *testing.T, s rschema.Schema, model any) {
	t.Helper()
	st := tfsdk.State{Schema: s}
	if diags := st.Set(context.Background(), model); diags.HasError() {
		t.Fatalf("state.Set diagnostics: %+v", diags)
	}
}

func TestSetSSHKeyResourceModel(t *testing.T) {
	var m sshKeyResourceModel
	setSSHKeyResourceModel(&m, &hrobot.SSHKey{Name: "k", Fingerprint: "aa:bb", Type: "RSA", Size: 4096, Data: "ssh-rsa AAAA\n"})
	if m.ID.ValueString() != "aa:bb" || m.PublicKey.ValueString() != "ssh-rsa AAAA" {
		t.Errorf("mapping wrong: id=%q public_key=%q", m.ID.ValueString(), m.PublicKey.ValueString())
	}
	mustSetResourceState(t, rsSchema(t, &sshKeyResource{}), &m)
}

func TestSetRDNSResourceModel(t *testing.T) {
	var m rdnsResourceModel
	setRDNSResourceModel(&m, &hrobot.RDNS{IP: "1.2.3.4", PTR: "host.example"})
	if m.ID.ValueString() != "1.2.3.4" {
		t.Errorf("id = %q", m.ID.ValueString())
	}
	mustSetResourceState(t, rsSchema(t, &rdnsResource{}), &m)
}

func TestSetFailoverResourceModel(t *testing.T) {
	active := "9.9.9.9"
	var m failoverResourceModel
	setFailoverResourceModel(&m, &hrobot.Failover{IP: "1.2.3.4", Netmask: "255.255.255.255", ServerIP: "5.6.7.8", ServerNumber: 1, ActiveServerIP: &active})
	if m.ActiveServerIP.ValueString() != active {
		t.Errorf("active_server_ip = %q", m.ActiveServerIP.ValueString())
	}
	mustSetResourceState(t, rsSchema(t, &failoverIPResource{}), &m)
}

func TestSetVSwitchResourceModel(t *testing.T) {
	ctx := context.Background()
	var m vswitchResourceModel
	diags := setVSwitchResourceModel(ctx, &m, &hrobot.VSwitch{
		ID: 42, Name: "vsw", VLAN: 4001,
		Servers: []hrobot.VSwitchServer{{ServerNumber: 111}, {ServerNumber: 222}},
	})
	if diags.HasError() {
		t.Fatalf("setVSwitchResourceModel diagnostics: %+v", diags)
	}
	if m.ID.ValueInt64() != 42 || len(m.ServerNumbers.Elements()) != 2 {
		t.Errorf("mapping wrong: id=%d servers=%d", m.ID.ValueInt64(), len(m.ServerNumbers.Elements()))
	}
	mustSetResourceState(t, rsSchema(t, &vswitchResource{}), &m)
}

func TestDiffInt64Sets(t *testing.T) {
	add, remove := diffInt64Sets([]int64{1, 2, 3}, []int64{2, 3, 4})
	if len(add) != 1 || add[0] != 1 {
		t.Errorf("toAdd = %v, want [1]", add)
	}
	if len(remove) != 1 || remove[0] != 4 {
		t.Errorf("toRemove = %v, want [4]", remove)
	}
}
